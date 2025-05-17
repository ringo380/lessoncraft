package store

import (
	"github.com/ringo380/lessoncraft/lesson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

// MockLessonStore is a mock implementation of the LessonStore interface
type MockLessonStore struct {
	mock.Mock
}

func (m *MockLessonStore) ListLessons(opts ListOptions) (*ListResult, error) {
	args := m.Called(opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ListResult), args.Error(1)
}

func (m *MockLessonStore) ListAllLessons() ([]lesson.Lesson, error) {
	args := m.Called()
	return args.Get(0).([]lesson.Lesson), args.Error(1)
}

func (m *MockLessonStore) GetLesson(id string) (*lesson.Lesson, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*lesson.Lesson), args.Error(1)
}

func (m *MockLessonStore) CreateLesson(l *lesson.Lesson) error {
	args := m.Called(l)
	return args.Error(0)
}

func (m *MockLessonStore) UpdateLesson(id string, l *lesson.Lesson) error {
	args := m.Called(id, l)
	return args.Error(0)
}

func (m *MockLessonStore) DeleteLesson(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func TestInMemoryCache(t *testing.T) {
	cache := NewInMemoryCache()

	// Test Set and Get
	cache.Set("key1", "value1", 1*time.Hour)
	value, found := cache.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "value1", value)

	// Test Get with non-existent key
	value, found = cache.Get("non-existent")
	assert.False(t, found)
	assert.Nil(t, value)

	// Test Delete
	cache.Delete("key1")
	value, found = cache.Get("key1")
	assert.False(t, found)
	assert.Nil(t, value)

	// Test Clear
	cache.Set("key2", "value2", 1*time.Hour)
	cache.Clear()
	value, found = cache.Get("key2")
	assert.False(t, found)
	assert.Nil(t, value)

	// Test expiration
	cache.Set("key3", "value3", 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	value, found = cache.Get("key3")
	assert.False(t, found)
	assert.Nil(t, value)
}

func TestCachedLessonStore_GetLesson(t *testing.T) {
	mockStore := new(MockLessonStore)
	cache := NewInMemoryCache()
	cachedStore := NewCachedLessonStore(mockStore, cache, 1*time.Hour)

	// Create a test lesson
	testLesson := &lesson.Lesson{
		ID:    "test-id",
		Title: "Test Lesson",
	}

	// Set up expectations
	mockStore.On("GetLesson", "test-id").Return(testLesson, nil).Once()

	// First call should hit the underlying store
	result, err := cachedStore.GetLesson("test-id")
	assert.NoError(t, err)
	assert.Equal(t, testLesson, result)

	// Second call should hit the cache and not the underlying store
	result, err = cachedStore.GetLesson("test-id")
	assert.NoError(t, err)
	assert.Equal(t, testLesson, result)

	// Verify expectations
	mockStore.AssertExpectations(t)
}

func TestCachedLessonStore_ListLessons(t *testing.T) {
	mockStore := new(MockLessonStore)
	cache := NewInMemoryCache()
	cachedStore := NewCachedLessonStore(mockStore, cache, 1*time.Hour)

	// Create test data
	opts := DefaultListOptions()
	testLessons := []lesson.Lesson{
		{ID: "1", Title: "Lesson 1"},
		{ID: "2", Title: "Lesson 2"},
	}
	testResult := &ListResult{
		Items:      testLessons,
		TotalItems: 2,
		TotalPages: 1,
		Page:       1,
		PageSize:   20,
	}

	// Set up expectations
	mockStore.On("ListLessons", opts).Return(testResult, nil).Once()

	// First call should hit the underlying store
	result, err := cachedStore.ListLessons(opts)
	assert.NoError(t, err)
	assert.Equal(t, testResult, result)

	// Second call should hit the cache and not the underlying store
	result, err = cachedStore.ListLessons(opts)
	assert.NoError(t, err)
	assert.Equal(t, testResult, result)

	// Verify expectations
	mockStore.AssertExpectations(t)
}

func TestCachedLessonStore_CreateLesson(t *testing.T) {
	mockStore := new(MockLessonStore)
	cache := NewInMemoryCache()
	cachedStore := NewCachedLessonStore(mockStore, cache, 1*time.Hour)

	// Create a test lesson
	testLesson := &lesson.Lesson{
		Title: "New Lesson",
	}

	// Set up expectations
	mockStore.On("CreateLesson", testLesson).Return(nil).Once()

	// Create the lesson
	err := cachedStore.CreateLesson(testLesson)
	assert.NoError(t, err)

	// Verify expectations
	mockStore.AssertExpectations(t)
}

func TestCachedLessonStore_UpdateLesson(t *testing.T) {
	mockStore := new(MockLessonStore)
	cache := NewInMemoryCache()
	cachedStore := NewCachedLessonStore(mockStore, cache, 1*time.Hour)

	// Create a test lesson
	testLesson := &lesson.Lesson{
		ID:    "test-id",
		Title: "Updated Lesson",
	}

	// Set up the cache with the original lesson
	originalLesson := &lesson.Lesson{
		ID:    "test-id",
		Title: "Original Lesson",
	}
	cache.Set("lesson:test-id", originalLesson, 1*time.Hour)

	// Set up expectations
	mockStore.On("UpdateLesson", "test-id", testLesson).Return(nil).Once()

	// Update the lesson
	err := cachedStore.UpdateLesson("test-id", testLesson)
	assert.NoError(t, err)

	// The cache should be invalidated
	_, found := cache.Get("lesson:test-id")
	assert.False(t, found)

	// Verify expectations
	mockStore.AssertExpectations(t)
}

func TestCachedLessonStore_DeleteLesson(t *testing.T) {
	mockStore := new(MockLessonStore)
	cache := NewInMemoryCache()
	cachedStore := NewCachedLessonStore(mockStore, cache, 1*time.Hour)

	// Set up the cache with a lesson
	testLesson := &lesson.Lesson{
		ID:    "test-id",
		Title: "Test Lesson",
	}
	cache.Set("lesson:test-id", testLesson, 1*time.Hour)

	// Set up expectations
	mockStore.On("DeleteLesson", "test-id").Return(nil).Once()

	// Delete the lesson
	err := cachedStore.DeleteLesson("test-id")
	assert.NoError(t, err)

	// The cache should be invalidated
	_, found := cache.Get("lesson:test-id")
	assert.False(t, found)

	// Verify expectations
	mockStore.AssertExpectations(t)
}
