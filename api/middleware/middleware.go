package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"
	"time"

	"golang.org/x/time/rate"
	"github.com/sirupsen/logrus"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"lessoncraft/api/metrics"
)

var tracer opentracing.Tracer

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

func init() {
	cfg := &config.Configuration{
		ServiceName: "lessoncraft",
		Sampler: &config.SamplerConfig{
			Type:  "adaptive",  // Use adaptive sampling
			Param: 0.01,       // Base sampling rate
			MaxOperations: 100, // Max number of operations to track
		},
		Reporter: &config.ReporterConfig{
			LogSpans: true,
			BufferFlushInterval: 1 * time.Second,
			QueueSize: 1000,
			LocalAgentHostPort: "jaeger:6831",
		},
		Tags: []opentracing.Tag{
			{Key: "environment", Value: "production"},
			{Key: "version", Value: "1.0.0"},
		},
	}

	opts := []config.Option{
		config.Logger(jaeger.StdLogger),
		config.Metrics(metrics.NewPrometheusFactory(prometheus.DefaultRegisterer)),
	}

	t, closer, err := cfg.NewTracer(opts...)
	if err != nil {
		log.Fatalf("Could not initialize tracer: %s", err)
	}
	defer closer.Close()

	tracer = t
	opentracing.SetGlobalTracer(tracer)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		span, ctx := opentracing.StartSpanFromContext(r.Context(), "http_request")
		defer span.Finish()

		// Add trace ID to request context
		traceID := span.Context().(jaeger.SpanContext).TraceID()
		r = r.WithContext(ctx)

		// Create a custom response writer to capture the status code
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		// Record metrics
		metrics.RequestDuration.WithLabelValues(
			r.URL.Path,
			r.Method,
			string(rw.status),
		).Observe(duration.Seconds())

		logger.WithFields(logrus.Fields{
			"method":     r.Method,
			"path":      r.URL.Path,
			"status":    rw.status,
			"duration":  duration,
			"trace_id":  traceID.String(),
			"user_agent": r.UserAgent(),
			"request_id": r.Header.Get("X-Request-ID"),
			"remote_ip": r.RemoteAddr,
			"host":     r.Host,
			"protocol": r.Proto,
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
