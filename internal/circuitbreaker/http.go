package circuitbreaker

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HTTPClient is an interface for HTTP clients
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// CircuitBreakerHTTPClient wraps an HTTP client with a circuit breaker
type CircuitBreakerHTTPClient struct {
	client         HTTPClient
	circuitBreaker *CircuitBreaker
}

// NewHTTPClient creates a new HTTP client with a circuit breaker
func NewHTTPClient(client HTTPClient, options Options) *CircuitBreakerHTTPClient {
	if options.Name == "" {
		options.Name = "http-client"
	}

	return &CircuitBreakerHTTPClient{
		client:         client,
		circuitBreaker: NewCircuitBreaker(options),
	}
}

// Do executes an HTTP request with circuit breaker protection
func (c *CircuitBreakerHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response

	err := c.circuitBreaker.Execute(func() error {
		var err error
		resp, err = c.client.Do(req)
		if err != nil {
			return err
		}

		// Consider 5xx responses as failures
		if resp.StatusCode >= 500 {
			return fmt.Errorf("server error: %d %s", resp.StatusCode, resp.Status)
		}

		return nil
	})

	if err == ErrCircuitOpen {
		return nil, fmt.Errorf("circuit breaker is open: %w", err)
	}

	return resp, err
}

// DoWithContext executes an HTTP request with circuit breaker protection and context
func (c *CircuitBreakerHTTPClient) DoWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	return c.Do(req)
}

// DoWithTimeout executes an HTTP request with circuit breaker protection and timeout
func (c *CircuitBreakerHTTPClient) DoWithTimeout(req *http.Request, timeout time.Duration) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), timeout)
	defer cancel()

	return c.DoWithContext(ctx, req)
}

// CircuitBreaker returns the underlying circuit breaker
func (c *CircuitBreakerHTTPClient) CircuitBreaker() *CircuitBreaker {
	return c.circuitBreaker
}

// WrapDefaultClient wraps the default http.Client with a circuit breaker
func WrapDefaultClient(options Options) *CircuitBreakerHTTPClient {
	return NewHTTPClient(http.DefaultClient, options)
}

// WrapTransport wraps an http.RoundTripper with a circuit breaker
func WrapTransport(transport http.RoundTripper, options Options) http.RoundTripper {
	if options.Name == "" {
		options.Name = "http-transport"
	}

	cb := NewCircuitBreaker(options)

	return &circuitBreakerTransport{
		transport:      transport,
		circuitBreaker: cb,
	}
}

type circuitBreakerTransport struct {
	transport      http.RoundTripper
	circuitBreaker *CircuitBreaker
}

func (t *circuitBreakerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response

	err := t.circuitBreaker.Execute(func() error {
		var err error
		resp, err = t.transport.RoundTrip(req)
		if err != nil {
			return err
		}

		// Consider 5xx responses as failures
		if resp.StatusCode >= 500 {
			return fmt.Errorf("server error: %d %s", resp.StatusCode, resp.Status)
		}

		return nil
	})

	if err == ErrCircuitOpen {
		return nil, fmt.Errorf("circuit breaker is open: %w", err)
	}

	return resp, err
}
