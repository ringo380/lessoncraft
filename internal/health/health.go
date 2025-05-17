package health

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/ringo380/lessoncraft/internal/circuitbreaker"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	
)

// Status represents the health status of a component
type Status string

const (
	// StatusUp indicates that the component is healthy
	StatusUp Status = "UP"
	// StatusDown indicates that the component is unhealthy
	StatusDown Status = "DOWN"
	// StatusUnknown indicates that the health of the component is unknown
	StatusUnknown Status = "UNKNOWN"
)

// ComponentHealth represents the health of a component
type ComponentHealth struct {
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
}

// Health represents the overall health of the application
type Health struct {
	Status     Status                     `json:"status"`
	Components map[string]ComponentHealth `json:"components"`
	Timestamp  time.Time                  `json:"timestamp"`
}

// Service provides health check functionality
type Service struct {
	mu                       sync.RWMutex
	health                   Health
	mongoClient              *mongo.Client
	dockerClient             *client.Client
	kubernetesClient         *kubernetes.Clientset
	mongoCircuitBreaker      *circuitbreaker.CircuitBreaker
	dockerCircuitBreaker     *circuitbreaker.CircuitBreaker
	kubernetesCircuitBreaker *circuitbreaker.CircuitBreaker
	checkInterval            time.Duration
	timeout                  time.Duration
}

// NewService creates a new health check service
func NewService(
	mongoClient *mongo.Client,
	dockerClient *client.Client,
	kubernetesClient *kubernetes.Clientset,
	mongoCircuitBreaker *circuitbreaker.CircuitBreaker,
	dockerCircuitBreaker *circuitbreaker.CircuitBreaker,
	kubernetesCircuitBreaker *circuitbreaker.CircuitBreaker,
) *Service {
	service := &Service{
		health: Health{
			Status:     StatusUnknown,
			Components: make(map[string]ComponentHealth),
			Timestamp:  time.Now(),
		},
		mongoClient:              mongoClient,
		dockerClient:             dockerClient,
		kubernetesClient:         kubernetesClient,
		mongoCircuitBreaker:      mongoCircuitBreaker,
		dockerCircuitBreaker:     dockerCircuitBreaker,
		kubernetesCircuitBreaker: kubernetesCircuitBreaker,
		checkInterval:            30 * time.Second,
		timeout:                  5 * time.Second,
	}

	// Initialize component statuses
	service.health.Components["mongodb"] = ComponentHealth{Status: StatusUnknown}
	service.health.Components["docker"] = ComponentHealth{Status: StatusUnknown}
	service.health.Components["kubernetes"] = ComponentHealth{Status: StatusUnknown}

	// Initialize circuit breaker statuses
	service.health.Components["mongodb-circuit-breaker"] = ComponentHealth{Status: StatusUnknown}
	service.health.Components["docker-circuit-breaker"] = ComponentHealth{Status: StatusUnknown}
	service.health.Components["kubernetes-circuit-breaker"] = ComponentHealth{Status: StatusUnknown}

	// Start background health checks
	go service.startBackgroundChecks()

	return service
}

// startBackgroundChecks starts periodic health checks in the background
func (s *Service) startBackgroundChecks() {
	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	// Perform initial health check
	s.checkHealth()

	for range ticker.C {
		s.checkHealth()
	}
}

// checkHealth checks the health of all components
func (s *Service) checkHealth() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update timestamp
	s.health.Timestamp = time.Now()

	// Check MongoDB health
	s.checkMongoDBHealth()

	// Check Docker health
	s.checkDockerHealth()

	// Check Kubernetes health
	s.checkKubernetesHealth()

	// Check circuit breaker states
	s.checkCircuitBreakerStates()

	// Update overall status
	s.updateOverallStatus()
}

// checkMongoDBHealth checks the health of MongoDB
func (s *Service) checkMongoDBHealth() {
	if s.mongoClient == nil {
		s.health.Components["mongodb"] = ComponentHealth{
			Status:  StatusUnknown,
			Message: "MongoDB client not configured",
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	err := s.mongoClient.Ping(ctx, readpref.Primary())
	if err != nil {
		s.health.Components["mongodb"] = ComponentHealth{
			Status:  StatusDown,
			Message: err.Error(),
		}
		log.Printf("MongoDB health check failed: %v", err)
	} else {
		s.health.Components["mongodb"] = ComponentHealth{
			Status: StatusUp,
		}
	}
}

// checkDockerHealth checks the health of Docker
func (s *Service) checkDockerHealth() {
	if s.dockerClient == nil {
		s.health.Components["docker"] = ComponentHealth{
			Status:  StatusUnknown,
			Message: "Docker client not configured",
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	_, err := s.dockerClient.Ping(ctx)
	if err != nil {
		s.health.Components["docker"] = ComponentHealth{
			Status:  StatusDown,
			Message: err.Error(),
		}
		log.Printf("Docker health check failed: %v", err)
	} else {
		s.health.Components["docker"] = ComponentHealth{
			Status: StatusUp,
		}
	}
}

// checkKubernetesHealth checks the health of the Kubernetes API
func (s *Service) checkKubernetesHealth() {
	if s.kubernetesClient == nil {
		s.health.Components["kubernetes"] = ComponentHealth{
			Status:  StatusUnknown,
			Message: "Kubernetes client not configured",
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// Try to list nodes as a simple health check
	_, err := s.kubernetesClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		s.health.Components["kubernetes"] = ComponentHealth{
			Status:  StatusDown,
			Message: err.Error(),
		}
		log.Printf("Kubernetes health check failed: %v", err)
	} else {
		s.health.Components["kubernetes"] = ComponentHealth{
			Status: StatusUp,
		}
	}
}

// checkCircuitBreakerStates checks the state of all circuit breakers
func (s *Service) checkCircuitBreakerStates() {
	// Check MongoDB circuit breaker
	if s.mongoCircuitBreaker != nil {
		state := s.mongoCircuitBreaker.State()
		status := StatusUnknown
		message := ""

		switch state {
		case circuitbreaker.StateClosed:
			status = StatusUp
			message = "Circuit is closed (normal operation)"
		case circuitbreaker.StateOpen:
			status = StatusDown
			message = "Circuit is open (service is failing)"
		case circuitbreaker.StateHalfOpen:
			status = StatusDown
			message = "Circuit is half-open (testing if service has recovered)"
		}

		s.health.Components["mongodb-circuit-breaker"] = ComponentHealth{
			Status:  status,
			Message: message,
		}
	}

	// Check Docker circuit breaker
	if s.dockerCircuitBreaker != nil {
		state := s.dockerCircuitBreaker.State()
		status := StatusUnknown
		message := ""

		switch state {
		case circuitbreaker.StateClosed:
			status = StatusUp
			message = "Circuit is closed (normal operation)"
		case circuitbreaker.StateOpen:
			status = StatusDown
			message = "Circuit is open (service is failing)"
		case circuitbreaker.StateHalfOpen:
			status = StatusDown
			message = "Circuit is half-open (testing if service has recovered)"
		}

		s.health.Components["docker-circuit-breaker"] = ComponentHealth{
			Status:  status,
			Message: message,
		}
	}

	// Check Kubernetes circuit breaker
	if s.kubernetesCircuitBreaker != nil {
		state := s.kubernetesCircuitBreaker.State()
		status := StatusUnknown
		message := ""

		switch state {
		case circuitbreaker.StateClosed:
			status = StatusUp
			message = "Circuit is closed (normal operation)"
		case circuitbreaker.StateOpen:
			status = StatusDown
			message = "Circuit is open (service is failing)"
		case circuitbreaker.StateHalfOpen:
			status = StatusDown
			message = "Circuit is half-open (testing if service has recovered)"
		}

		s.health.Components["kubernetes-circuit-breaker"] = ComponentHealth{
			Status:  status,
			Message: message,
		}
	}
}

// updateOverallStatus updates the overall health status based on component statuses
func (s *Service) updateOverallStatus() {
	allUp := true
	anyDown := false

	for _, component := range s.health.Components {
		if component.Status == StatusDown {
			anyDown = true
		}
		if component.Status != StatusUp {
			allUp = false
		}
	}

	if allUp {
		s.health.Status = StatusUp
	} else if anyDown {
		s.health.Status = StatusDown
	} else {
		s.health.Status = StatusUnknown
	}
}

// GetHealth returns the current health status
func (s *Service) GetHealth() Health {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy to avoid race conditions
	health := Health{
		Status:     s.health.Status,
		Components: make(map[string]ComponentHealth, len(s.health.Components)),
		Timestamp:  s.health.Timestamp,
	}

	for k, v := range s.health.Components {
		health.Components[k] = v
	}

	return health
}

// SetCheckInterval sets the interval for background health checks
func (s *Service) SetCheckInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkInterval = interval
}

// SetTimeout sets the timeout for health checks
func (s *Service) SetTimeout(timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.timeout = timeout
}
