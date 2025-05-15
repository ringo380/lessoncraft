package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "lessoncraft_http_request_duration_seconds",
		Help: "Duration of HTTP requests in seconds",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"handler", "method", "status"})

	LessonCreationTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lessoncraft_lessons_created_total",
		Help: "Total number of lessons created",
	})

	ActiveLessons = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "lessoncraft_active_lessons",
		Help: "Number of currently active lessons",
	})

	StepCompletions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lessoncraft_step_completions_total",
		Help: "Total number of lesson step completions",
	}, []string{"lesson_id", "step_index", "success"})
)
