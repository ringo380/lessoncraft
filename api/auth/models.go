package auth

import (
	"time"

	"github.com/ringo380/lessoncraft/pwd/types"
)

// Role represents a user role in the system
type Role string

const (
	// RoleAdmin is the administrator role with full access
	RoleAdmin Role = "admin"
	// RoleEducator is the educator role with access to create and manage lessons
	RoleEducator Role = "educator"
	// RoleLearner is the learner role with access to view and complete lessons
	RoleLearner Role = "learner"
)

// UserWithAuth extends the base User type with authentication and authorization fields
type UserWithAuth struct {
	// Embed the base User type
	types.User
	// PasswordHash stores the hashed password for local authentication
	PasswordHash string `json:"-" bson:"password_hash"`
	// Roles defines the roles assigned to the user
	Roles []Role `json:"roles" bson:"roles"`
	// LastLogin records the last time the user logged in
	LastLogin time.Time `json:"last_login" bson:"last_login"`
	// AccountStatus indicates whether the account is active, suspended, etc.
	AccountStatus string `json:"account_status" bson:"account_status"`
	// EmailVerified indicates whether the user's email has been verified
	EmailVerified bool `json:"email_verified" bson:"email_verified"`
	// CreatedAt records when the user account was created
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	// UpdatedAt records when the user account was last updated
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// TokenClaims represents the claims in a JWT token
type TokenClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Roles  []Role `json:"roles"`
	// Standard JWT claims
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
}

// LoginResponse represents the response sent to the client after successful login
type LoginResponse struct {
	Token        string     `json:"token"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresAt    time.Time  `json:"expires_at"`
	User         types.User `json:"user"`
}

// LoginRequest represents a request to log in with email and password
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterRequest represents a request to register a new user
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshTokenRequest represents a request to refresh an access token
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// HasRole checks if a user has a specific role
func (u *UserWithAuth) HasRole(role Role) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsAdmin checks if a user has the admin role
func (u *UserWithAuth) IsAdmin() bool {
	return u.HasRole(RoleAdmin)
}

// IsEducator checks if a user has the educator role
func (u *UserWithAuth) IsEducator() bool {
	return u.HasRole(RoleEducator)
}

// IsLearner checks if a user has the learner role
func (u *UserWithAuth) IsLearner() bool {
	return u.HasRole(RoleLearner)
}
