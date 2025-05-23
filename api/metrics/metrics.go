package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/uber/jaeger-lib/metrics"
)

// MockFactory is a simple implementation for testing
type MockFactory struct{}

// Counter implements the metrics.Factory interface
func (f *MockFactory) Counter(options metrics.Options) metrics.Counter {
	return &noopCounter{}
}

// Gauge implements the metrics.Factory interface
func (f *MockFactory) Gauge(options metrics.Options) metrics.Gauge {
	return &noopGauge{}
}

// Timer implements the metrics.Factory interface
func (f *MockFactory) Timer(options metrics.TimerOptions) metrics.Timer {
	return &noopTimer{}
}

// Histogram implements the metrics.Factory interface
func (f *MockFactory) Histogram(options metrics.HistogramOptions) metrics.Histogram {
	return &noopHistogram{}
}

// Namespace implements the metrics.Factory interface
func (f *MockFactory) Namespace(scope metrics.NSOptions) metrics.Factory {
	return f
}

// noopCounter is a no-op implementation of metrics.Counter
type noopCounter struct{}

func (c *noopCounter) Inc(int64) {}

// noopGauge is a no-op implementation of metrics.Gauge
type noopGauge struct{}

func (g *noopGauge) Update(int64) {}

// noopTimer is a no-op implementation of metrics.Timer
type noopTimer struct{}

func (t *noopTimer) Record(time.Duration) {}

// noopHistogram is a no-op implementation of metrics.Histogram
type noopHistogram struct{}

func (h *noopHistogram) Record(float64) {}

// NewPrometheusFactory creates a new metrics factory for testing
func NewPrometheusFactory(registerer prometheus.Registerer) metrics.Factory {
	return &MockFactory{}
}

var (
	// Request metrics
	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lessoncraft_http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"handler", "method", "status"})

	RequestErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lessoncraft_http_errors_total",
		Help: "Total number of HTTP errors",
	}, []string{"handler", "code", "error_type"})

	// Lesson metrics
	LessonCreationTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lessoncraft_lessons_created_total",
		Help: "Total number of lessons created",
	})

	ActiveLessons = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "lessoncraft_active_lessons",
		Help: "Number of currently active lessons",
	})

	LessonDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lessoncraft_lesson_duration_seconds",
		Help:    "Duration of lesson completions",
		Buckets: prometheus.ExponentialBuckets(60, 2, 10), // From 1min to ~17hrs
	}, []string{"lesson_id"})

	// Step metrics
	StepCompletions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lessoncraft_step_completions_total",
		Help: "Total number of lesson step completions",
	}, []string{"lesson_id", "step_index", "success"})

	StepDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lessoncraft_step_duration_seconds",
		Help:    "Duration of step completions",
		Buckets: prometheus.ExponentialBuckets(10, 2, 8), // From 10s to ~42min
	}, []string{"lesson_id", "step_index"})

	StepRetries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lessoncraft_step_retries_total",
		Help: "Number of step retry attempts",
	}, []string{"lesson_id", "step_index"})

	// System metrics
	ActiveUsers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "lessoncraft_active_users",
		Help: "Number of currently active users",
	})

	SystemMemoryUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "lessoncraft_memory_bytes",
		Help: "Current system memory usage in bytes",
	})

	DockerOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lessoncraft_docker_operations_total",
		Help: "Number of Docker operations performed",
	}, []string{"operation", "status"})
)
