package api

import (
	"context"
	"net/http"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// varsKey is the key for URL variables in the request context
const varsKey contextKey = "vars"

// SetURLVars sets URL variables for testing
func SetURLVars(r *http.Request, vars map[string]string) *http.Request {
	ctx := context.WithValue(r.Context(), varsKey, vars)
	return r.WithContext(ctx)
}

// GetURLVars gets URL variables from the request context
func GetURLVars(r *http.Request) map[string]string {
	if vars, ok := r.Context().Value(varsKey).(map[string]string); ok {
		return vars
	}
	return map[string]string{}
}
