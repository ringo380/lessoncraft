package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"
	"time"

	"golang.org/x/time/rate"
	"github.com/sirupsen/logrus"
)

type ErrorResponse struct {
	Error       string      `json:"error"`
	Code        int         `json:"code"`
	Message     string      `json:"message"`
	Details     interface{} `json:"details,omitempty"`
	RequestID   string      `json:"request_id,omitempty"`
	Stack       string      `json:"stack,omitempty"`
	TimeStamp   time.Time   `json:"timestamp"`
}

var logger = logrus.New()

func init() {
	logger.SetFormatter(&logrus.JSONFormatter{})
}

func ErrorHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Capture stack trace
				buf := make([]byte, 1024)
				n := runtime.Stack(buf, false)
				stackTrace := string(buf[:n])

				logger.WithFields(logrus.Fields{
					"error": err,
					"stack": stackTrace,
					"path":  r.URL.Path,
				}).Error("Panic recovered in request handler")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(ErrorResponse{
					Error:     "InternalServerError",
					Code:     500,
					Message:  "An unexpected error occurred",
					Stack:    stackTrace,
					RequestID: r.Header.Get("X-Request-ID"),
					TimeStamp: time.Now(),
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func RateLimitMiddleware(next http.Handler) http.Handler {
	limiter := rate.NewLimiter(rate.Every(time.Second), 10) // 10 requests per second

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error:     "RateLimitExceeded",
				Code:     429,
				Message:  "Too many requests",
				TimeStamp: time.Now(),
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom response writer to capture the status code
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)

		logger.WithFields(logrus.Fields{
			"method":     r.Method,
			"path":       r.URL.Path,
			"status":     rw.status,
			"duration":   time.Since(start),
			"user_agent": r.UserAgent(),
			"request_id": r.Header.Get("X-Request-ID"),
		}).Info("Request completed")
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error:   "Unauthorized",
				Code:    401,
				Message: "Missing authorization token",
			})
			return
		}
		// TODO: Implement token validation
		next.ServeHTTP(w, r)
	})
}
