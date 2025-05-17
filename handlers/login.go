package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/ringo380/lessoncraft/config"
	"github.com/ringo380/lessoncraft/pwd/types"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

func LoggedInUser(rw http.ResponseWriter, req *http.Request) {
	cookie, err := ReadCookie(req)
	if err != nil {
		log.Println("Cannot read cookie")
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	user, err := core.UserGet(cookie.Id)
	if err != nil {
		log.Printf("Couldn't get user with id %s. Got: %v\n", cookie.Id, err)
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}
	json.NewEncoder(rw).Encode(user)
}

func ListProviders(rw http.ResponseWriter, req *http.Request) {
	playground := core.PlaygroundFindByDomain(req.Host)
	if playground == nil {
		log.Printf("Playground for domain %s was not found!", req.Host)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	providers := []string{}
	for name := range config.Providers[playground.Id] {
		providers = append(providers, name)
	}
	json.NewEncoder(rw).Encode(providers)
}

func Login(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	providerName := vars["provider"]
	playground := core.PlaygroundFindByDomain(req.Host)
	if playground == nil {
		log.Printf("Playground for domain %s was not found!", req.Host)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	provider, found := config.Providers[playground.Id][providerName]
	if !found {
		log.Printf("Could not find provider %s\n", providerName)
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	loginRequest, err := core.UserNewLoginRequest(providerName)
	if err != nil {
		log.Printf("Could not start a new user login request for provider %s. Got: %v\n", providerName, err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if playground.AuthRedirectBase != "" {
		provider.RedirectURL = fmt.Sprintf("%s/oauth/providers/%s/callback", playground.AuthRedirectBase, providerName)
	} else {
		scheme := "http"
		if req.TLS != nil {
			scheme = "https"
		}
		host := "localhost"
		if req.Host != "" {
			host = req.Host
		}
		provider.RedirectURL = fmt.Sprintf("%s://%s/oauth/providers/%s/callback", scheme, host, providerName)
	}

	url := provider.AuthCodeURL(loginRequest.Id, oauth2.SetAuthURLParam("nonce", uuid.NewV4().String()))

	http.Redirect(rw, req, url, http.StatusFound)
}

func LoginCallback(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	providerName := vars["provider"]
	playground := core.PlaygroundFindByDomain(req.Host)
	if playground == nil {
		log.Printf("Playground for domain %s was not found!", req.Host)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	provider, found := config.Providers[playground.Id][providerName]
	if !found {
		log.Printf("Could not find provider %s\n", providerName)
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	query := req.URL.Query()

	code := query.Get("code")
	loginRequestId := query.Get("state")

	loginRequest, err := core.UserGetLoginRequest(loginRequestId)
	if err != nil {
		log.Printf("Could not get login request %s for provider %s. Got: %v\n", loginRequestId, providerName, err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx := req.Context()
	tok, err := provider.Exchange(ctx, code)
	if err != nil {
		log.Printf("Could not exchage code for access token for provider %s. Got: %v\n", providerName, err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	user := &types.User{Provider: providerName}
	if providerName == "github" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: tok.AccessToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		client := github.NewClient(tc)
		u, _, err := client.Users.Get(ctx, "")
		if err != nil {
			log.Printf("Could not get user from github. Got: %v\n", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		user.ProviderUserId = strconv.Itoa(u.GetID())
		user.Name = u.GetName()
		user.Avatar = u.GetAvatarURL()
		user.Email = u.GetEmail()
	} else if providerName == "google" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: tok.AccessToken},
		)
		tc := oauth2.NewClient(ctx, ts)

		p, err := people.NewService(ctx, option.WithHTTPClient(tc))
		if err != nil {
			log.Printf("Could not initialize people service . Got: %v\n", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		person, err := p.People.Get("people/me").PersonFields("emailAddresses,names").Do()
		if err != nil {
			log.Printf("Could not initialize people service . Got: %v\n", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		user.Email = person.EmailAddresses[0].Value
		user.Name = person.Names[0].GivenName
		user.ProviderUserId = person.ResourceName

	} else if providerName == "facebook" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: tok.AccessToken},
		)
		tc := oauth2.NewClient(ctx, ts)

		// Facebook Graph API to get user info
		resp, err := tc.Get("https://graph.facebook.com/me?fields=id,name,email,picture.type(large)")
		if err != nil {
			log.Printf("Could not get user from Facebook. Got: %v\n", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		fbUser := map[string]interface{}{}
		if err := json.NewDecoder(resp.Body).Decode(&fbUser); err != nil {
			log.Printf("Could not decode Facebook user info. Got: %v\n", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		user.ProviderUserId = fbUser["id"].(string)
		user.Name = fbUser["name"].(string)
		user.Email = fbUser["email"].(string)

		// Get profile picture URL
		if picture, ok := fbUser["picture"].(map[string]interface{}); ok {
			if data, ok := picture["data"].(map[string]interface{}); ok {
				if url, ok := data["url"].(string); ok {
					user.Avatar = url
				}
			}
		}

	} else if providerName == "docker" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: tok.AccessToken},
		)
		tc := oauth2.NewClient(ctx, ts)

		endpoint := getDockerEndpoint(playground)
		resp, err := tc.Get(fmt.Sprintf("https://%s/userinfo", endpoint))
		if err != nil {
			log.Printf("Could not get user from docker. Got: %v\n", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		userInfo := map[string]interface{}{}
		if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
			log.Printf("Could not decode user info. Got: %v\n", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		user.ProviderUserId = strings.Split(userInfo["sub"].(string), "|")[1]
		user.Name = userInfo["https://hub.docker.com"].(map[string]interface{})["username"].(string)
		user.Email = userInfo["https://hub.docker.com"].(map[string]interface{})["email"].(string)
		// Since DockerID doesn't return a user avatar, we try with twitter through avatars.io
		// Worst case we get a generic avatar
		user.Avatar = fmt.Sprintf("https://avatars.io/twitter/%s", user.Name)
	}

	user, err = core.UserLogin(loginRequest, user)
	if err != nil {
		log.Printf("Could not login user. Got: %v\n", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	cookieData := CookieID{Id: user.Id, UserName: user.Name, UserAvatar: user.Avatar, ProviderId: user.ProviderUserId}

	host := "localhost"
	if req.Host != "" {
		// we get the parent domain so cookie is set
		// in all subdomain and siblings
		host = getParentDomain(req.Host)
	}

	if err := cookieData.SetCookie(rw, host); err != nil {
		log.Printf("Could not encode cookie. Got: %v\n", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	r, _ := playground.Extras.GetString("LoginRedirect")

	fmt.Fprintf(rw, `
<!DOCTYPE html>
<html lang="en">
    <head>
        <meta charset="utf-8">
        <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
        <title>Login Successful</title>
        <link rel="stylesheet" href="https://unpkg.com/bootstrap@4.0.0-beta/dist/css/bootstrap.min.css">
        <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/5.15.3/css/all.min.css">
        <style>
            body {
                display: flex;
                justify-content: center;
                align-items: center;
                height: 100vh;
                background-color: #f8f9fa;
            }
            .login-success {
                text-align: center;
                padding: 2rem;
                background-color: white;
                border-radius: 8px;
                box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
                max-width: 400px;
                width: 100%%;
            }
            .success-icon {
                font-size: 48px;
                color: #28a745;
                margin-bottom: 1rem;
            }
            .countdown {
                font-size: 14px;
                color: #6c757d;
                margin-top: 1rem;
            }
            .provider-icon {
                font-size: 24px;
                margin-right: 10px;
                vertical-align: middle;
            }
            .github-color { color: #24292e; }
            .google-color { color: #4285F4; }
            .facebook-color { color: #3b5998; }
            .docker-color { color: #0db7ed; }
        </style>
    </head>
    <body>
        <div class="login-success">
            <i class="fas fa-check-circle success-icon"></i>
            <h3>Login Successful!</h3>

            <p class="mt-3">
                <i class="fab 
                    %s
                    provider-icon"></i>
                You've successfully logged in with <strong>%s</strong>.
            </p>

            <div class="countdown">
                <span id="countdown">3</span> seconds until this window closes...
            </div>
        </div>

        <script>
            // Countdown timer
            var seconds = 3;
            var countdownEl = document.getElementById('countdown');
            var interval = setInterval(function() {
                seconds--;
                countdownEl.textContent = seconds;
                if (seconds <= 0) {
                    clearInterval(interval);
                    closeOrRedirect();
                }
            }, 1000);

            // Close window or redirect
            function closeOrRedirect() {
                if (window.opener && !window.opener.closed) {
                    try {
                        window.opener.postMessage('done','*');
                    }
                    catch(e) { }
                    window.close();
                } else {
                    window.location = '%s';
                }
            }
        </script>
    </body>
</html>`,
		getProviderIconClass(providerName),
		strings.Title(providerName),
		r)
}

// getParentDomain returns the parent domain (if available)
// of the currend domain
func getParentDomain(host string) string {
	levels := strings.Split(host, ".")
	if len(levels) > 2 {
		return strings.Join(levels[1:], ".")
	}
	return host
}

func getDockerEndpoint(p *types.Playground) string {
	if len(p.DockerHost) > 0 {
		return p.DockerHost
	}
	return "login.docker.com"
}

// getProviderIconClass returns the appropriate Font Awesome icon class for a provider
func getProviderIconClass(provider string) string {
	switch provider {
	case "github":
		return "fa-github github-color"
	case "google":
		return "fa-google google-color"
	case "facebook":
		return "fa-facebook-f facebook-color"
	case "docker":
		return "fa-docker docker-color"
	default:
		return "fa-user-circle"
	}
}
