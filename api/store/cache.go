package store

import (
	"github.com/ringo380/lessoncraft/lesson"
	"strconv"
	"sync"
	"time"
)

// Cache interface defines the methods for a generic cache
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, expiration time.Duration)
	Delete(key string)
	Clear()
}

// InMemoryCache implements a simple in-memory cache with expiration
type InMemoryCache struct {
	items map[string]cacheItem
	mu    sync.RWMutex
}

// cacheItem represents an item in the cache
type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// NewInMemoryCache creates a new in-memory cache
func NewInMemoryCache() *InMemoryCache {
	cache := &InMemoryCache{
		items: make(map[string]cacheItem),
	}

	// Start a background goroutine to clean up expired items
	go cache.startCleanupTimer()

	return cache
}

// Get retrieves an item from the cache
func (c *InMemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}

	// Check if the item has expired
	if item.expiration.Before(time.Now()) {
		return nil, false
	}

	return item.value, true
}

// Set adds an item to the cache with an expiration time
func (c *InMemoryCache) Set(key string, value interface{}, expiration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheItem{
		value:      value,
		expiration: time.Now().Add(expiration),
	}
}

// Delete removes an item from the cache
func (c *InMemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *InMemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]cacheItem)
}

// startCleanupTimer starts a timer to clean up expired items
func (c *InMemoryCache) startCleanupTimer() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		}
	}
}

// cleanupExpired removes expired items from the cache
func (c *InMemoryCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if item.expiration.Before(now) {
			delete(c.items, key)
		}
	}
}

// CachedLessonStore wraps a LessonStore with caching functionality
type CachedLessonStore struct {
	store LessonStore
	cache Cache
	ttl   time.Duration
}

// NewCachedLessonStore creates a new CachedLessonStore
func NewCachedLessonStore(store LessonStore, cache Cache, ttl time.Duration) *CachedLessonStore {
	return &CachedLessonStore{
		store: store,
		cache: cache,
		ttl:   ttl,
	}
}

// GetLesson retrieves a lesson by ID, using the cache if available
func (s *CachedLessonStore) GetLesson(id string) (*lesson.Lesson, error) {
	// Try to get from cache first
	cacheKey := "lesson:" + id
	if cached, found := s.cache.Get(cacheKey); found {
		if lesson, ok := cached.(*lesson.Lesson); ok {
			return lesson, nil
		}
	}

	// If not in cache, get from store
	lesson, err := s.store.GetLesson(id)
	if err != nil {
		return nil, err
	}

	// Store in cache for future requests
	s.cache.Set(cacheKey, lesson, s.ttl)

	return lesson, nil
}

// ListLessons retrieves lessons with pagination, using the cache if available
func (s *CachedLessonStore) ListLessons(opts ListOptions) (*ListResult, error) {
	// Generate cache key based on options
	cacheKey := generateListCacheKey(opts)

	// Try to get from cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if result, ok := cached.(*ListResult); ok {
			return result, nil
		}
	}

	// If not in cache, get from store
	result, err := s.store.ListLessons(opts)
	if err != nil {
		return nil, err
	}

	// Store in cache for future requests
	s.cache.Set(cacheKey, result, s.ttl)

	return result, nil
}

// ListAllLessons retrieves all lessons, using the cache if available
func (s *CachedLessonStore) ListAllLessons() ([]lesson.Lesson, error) {
	// Try to get from cache first
	cacheKey := "lessons:all"
	if cached, found := s.cache.Get(cacheKey); found {
		if lessons, ok := cached.([]lesson.Lesson); ok {
			return lessons, nil
		}
	}

	// If not in cache, get from store
	lessons, err := s.store.ListAllLessons()
	if err != nil {
		return nil, err
	}

	// Store in cache for future requests
	s.cache.Set(cacheKey, lessons, s.ttl)

	return lessons, nil
}

// CreateLesson creates a new lesson and invalidates relevant caches
func (s *CachedLessonStore) CreateLesson(l *lesson.Lesson) error {
	err := s.store.CreateLesson(l)
	if err != nil {
		return err
	}

	// Invalidate list caches
	s.cache.Delete("lessons:all")

	return nil
}

// UpdateLesson updates a lesson and invalidates relevant caches
func (s *CachedLessonStore) UpdateLesson(id string, l *lesson.Lesson) error {
	err := s.store.UpdateLesson(id, l)
	if err != nil {
		return err
	}

	// Invalidate caches
	s.cache.Delete("lesson:" + id)
	s.cache.Delete("lessons:all")

	return nil
}

// DeleteLesson deletes a lesson and invalidates relevant caches
func (s *CachedLessonStore) DeleteLesson(id string) error {
	err := s.store.DeleteLesson(id)
	if err != nil {
		return err
	}

	// Invalidate caches
	s.cache.Delete("lesson:" + id)
	s.cache.Delete("lessons:all")

	return nil
}

// generateListCacheKey generates a cache key for list options
func generateListCacheKey(opts ListOptions) string {
	// Simple implementation - in a real system, you might want to hash the options
	return "lessons:list:page:" + strconv.FormatInt(opts.Page, 10) + ":size:" + strconv.FormatInt(opts.PageSize, 10)
}
