package k8s

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ringo380/lessoncraft/internal/circuitbreaker"
	"github.com/ringo380/lessoncraft/pwd/types"
	"github.com/ringo380/lessoncraft/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type localCachedFactory struct {
	rw              sync.Mutex
	irw             sync.Mutex
	sessionClient   *kubernetes.Clientset
	instanceClients map[string]*instanceEntry
	storage         storage.StorageApi
	cb              *circuitbreaker.CircuitBreaker
}

type instanceEntry struct {
	rw            sync.Mutex
	client        *kubernetes.Clientset
	kubeletClient *KubeletClient
}

func (f *localCachedFactory) GetForInstance(instance *types.Instance) (*kubernetes.Clientset, error) {
	key := instance.Name

	f.irw.Lock()
	c, found := f.instanceClients[key]
	if !found {
		c := &instanceEntry{}
		f.instanceClients[key] = c
	}
	c = f.instanceClients[key]
	f.irw.Unlock()

	c.rw.Lock()
	defer c.rw.Unlock()

	if c.client == nil {
		kc, err := NewClient(instance, "l2:443")
		if err != nil {
			log.Printf("Failed to create Kubernetes client for instance %s: %v", instance.Name, err)
			// Return a degraded client or fallback behavior
			return nil, fmt.Errorf("Kubernetes client creation failed, service degraded: %w", err)
		}
		c.client = kc
	}

	err := f.check(func() error {
		_, err := c.client.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
		return err
	})
	if err != nil {
		if err.Error() == "Kubernetes API circuit breaker is open, too many failures detected" {
			log.Printf("Kubernetes API is unavailable for instance %s, operating in degraded mode", instance.Name)
			// Return the client anyway, but with a warning that it might not work
			return c.client, fmt.Errorf("Kubernetes API is unavailable, operating in degraded mode: %w", err)
		}
		return nil, err
	}

	return c.client, nil
}

func (f *localCachedFactory) GetKubeletForInstance(instance *types.Instance) (*KubeletClient, error) {
	key := instance.Name

	f.irw.Lock()
	c, found := f.instanceClients[key]
	if !found {
		c := &instanceEntry{}
		f.instanceClients[key] = c
	}
	c = f.instanceClients[key]
	f.irw.Unlock()

	c.rw.Lock()
	defer c.rw.Unlock()

	if c.kubeletClient == nil {
		kc, err := NewKubeletClient(instance, "l2:443")
		if err != nil {
			log.Printf("Failed to create Kubelet client for instance %s: %v", instance.Name, err)
			// Return a degraded client or fallback behavior
			return nil, fmt.Errorf("Kubelet client creation failed, service degraded: %w", err)
		}
		c.kubeletClient = kc
	}

	err := f.check(func() error {
		r, err := c.kubeletClient.Get("/pods")
		if err != nil {
			return err
		}
		defer r.Body.Close()
		return nil
	})
	if err != nil {
		if err.Error() == "Kubernetes API circuit breaker is open, too many failures detected" {
			log.Printf("Kubelet API is unavailable for instance %s, operating in degraded mode", instance.Name)
			// Return the client anyway, but with a warning that it might not work
			return c.kubeletClient, fmt.Errorf("Kubelet API is unavailable, operating in degraded mode: %w", err)
		}
		return nil, err
	}

	return c.kubeletClient, nil
}

func (f *localCachedFactory) check(fn func() error) error {
	// Use the circuit breaker to protect against repeated failures
	err := f.cb.Execute(func() error {
		// Preserve the existing retry logic within the circuit breaker
		ok := false
		for i := 0; i < 5; i++ {
			err := fn()
			if err != nil {
				log.Printf("Connection to k8s api has failed, maybe instance is not ready yet, sleeping and retrying in 1 second. Try #%d. Got: %v\n", i+1, err)
				time.Sleep(time.Second)
				continue
			}
			ok = true
			break
		}
		if !ok {
			return fmt.Errorf("Connection to k8s api was not established")
		}
		return nil
	})

	// If the circuit is open, return a more descriptive error
	if err == circuitbreaker.ErrCircuitOpen {
		return fmt.Errorf("Kubernetes API circuit breaker is open, too many failures detected")
	}

	return err
}

func NewLocalCachedFactory(s storage.StorageApi) *localCachedFactory {
	// Create a circuit breaker for Kubernetes API connections
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Options{
		Name:                     "kubernetes-api",
		FailureThreshold:         3,
		ResetTimeout:             10 * time.Second,
		HalfOpenSuccessThreshold: 1,
		OnStateChange: func(name string, from, to circuitbreaker.State) {
			log.Printf("Kubernetes API circuit breaker state changed from %v to %v", from, to)
		},
	})

	return &localCachedFactory{
		instanceClients: make(map[string]*instanceEntry),
		storage:         s,
		cb:              cb,
	}
}
