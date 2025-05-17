package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/ringo380/lessoncraft/lesson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// This function is now defined in test_helpers.go

// MockLessonStore is a mock implementation of the LessonStore interface
type MockLessonStore struct {
	mock.Mock
}

func (m *MockLessonStore) ListLessons() ([]lesson.Lesson, error) {
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

// Helper function to create a test lesson
func createTestLesson() lesson.Lesson {
	return lesson.Lesson{
		ID:          "test-id",
		Title:       "Test Lesson",
		Description: "This is a test lesson",
		Steps: []lesson.LessonStep{
			{
				ID:       "step-1",
				Content:  "Step 1 content",
				Commands: []string{"echo 'Hello, World!'"},
				Expected: "Hello, World!",
				Timeout:  5 * time.Minute,
			},
		},
		CreatedAt:   time.Now(),
		CurrentStep: 0,
	}
}

// Test listLessons handler
func TestListLessons(t *testing.T) {
	// Create a mock store
	mockStore := new(MockLessonStore)

	// Create test lessons
	lessons := []lesson.Lesson{createTestLesson(), createTestLesson()}

	// Set up expectations
	mockStore.On("ListLessons").Return(lessons, nil)

	// Create handler with mock store
	handler := NewLessonHandler(mockStore)

	// Create a request
	req, err := http.NewRequest("GET", "/api/lessons", nil)
	assert.NoError(t, err)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.listLessons(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body
	var responseBody []lesson.Lesson
	err = json.Unmarshal(rr.Body.Bytes(), &responseBody)
	assert.NoError(t, err)
	assert.Len(t, responseBody, 2)

	// Verify that expectations were met
	mockStore.AssertExpectations(t)
}

// Test listLessons handler with database error
func TestListLessonsError(t *testing.T) {
	// Create a mock store
	mockStore := new(MockLessonStore)

	// Set up expectations
	mockStore.On("ListLessons").Return([]lesson.Lesson{}, errors.New("database error"))

	// Create handler with mock store
	handler := NewLessonHandler(mockStore)

	// Create a request
	req, err := http.NewRequest("GET", "/api/lessons", nil)
	assert.NoError(t, err)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.listLessons(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	// Verify that expectations were met
	mockStore.AssertExpectations(t)
}

// Test getLesson handler
func TestGetLesson(t *testing.T) {
	// Create a mock store
	mockStore := new(MockLessonStore)

	// Create a test lesson
	testLesson := createTestLesson()

	// Set up expectations
	mockStore.On("GetLesson", "test-id").Return(&testLesson, nil)

	// Create handler with mock store
	handler := NewLessonHandler(mockStore)

	// Create a request
	req, err := http.NewRequest("GET", "/api/lessons/test-id", nil)
	assert.NoError(t, err)

	// Add route variables
	vars := map[string]string{
		"id": "test-id",
	}
	req = SetURLVars(req, vars)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.getLesson(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body
	var responseBody lesson.Lesson
	err = json.Unmarshal(rr.Body.Bytes(), &responseBody)
	assert.NoError(t, err)
	assert.Equal(t, testLesson.ID, responseBody.ID)
	assert.Equal(t, testLesson.Title, responseBody.Title)

	// Verify that expectations were met
	mockStore.AssertExpectations(t)
}

// Test getLesson handler with not found error
func TestGetLessonNotFound(t *testing.T) {
	// Create a mock store
	mockStore := new(MockLessonStore)

	// Set up expectations
	mockStore.On("GetLesson", "non-existent-id").Return(nil, errors.New("lesson not found"))

	// Create handler with mock store
	handler := NewLessonHandler(mockStore)

	// Create a request
	req, err := http.NewRequest("GET", "/api/lessons/non-existent-id", nil)
	assert.NoError(t, err)

	// Add route variables
	vars := map[string]string{
		"id": "non-existent-id",
	}
	req = SetURLVars(req, vars)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.getLesson(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusNotFound, rr.Code)

	// Verify that expectations were met
	mockStore.AssertExpectations(t)
}

// Test createLesson handler
func TestCreateLesson(t *testing.T) {
	// Create a mock store
	mockStore := new(MockLessonStore)

	// Create a test lesson
	testLesson := createTestLesson()

	// Set up expectations
	mockStore.On("CreateLesson", mock.AnythingOfType("*lesson.Lesson")).Return(nil)

	// Create handler with mock store
	handler := NewLessonHandler(mockStore)

	// Create a request body
	body, err := json.Marshal(testLesson)
	assert.NoError(t, err)

	// Create a request
	req, err := http.NewRequest("POST", "/api/lessons", bytes.NewBuffer(body))
	assert.NoError(t, err)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.createLesson(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Verify that expectations were met
	mockStore.AssertExpectations(t)
}

// Test createLesson handler with invalid lesson
func TestCreateLessonInvalidLesson(t *testing.T) {
	// Create a mock store
	mockStore := new(MockLessonStore)

	// Create an invalid lesson (missing title)
	invalidLesson := createTestLesson()
	invalidLesson.Title = ""

	// Create handler with mock store
	handler := NewLessonHandler(mockStore)

	// Create a request body
	body, err := json.Marshal(invalidLesson)
	assert.NoError(t, err)

	// Create a request
	req, err := http.NewRequest("POST", "/api/lessons", bytes.NewBuffer(body))
	assert.NoError(t, err)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.createLesson(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// Test updateLesson handler
func TestUpdateLesson(t *testing.T) {
	// Create a mock store
	mockStore := new(MockLessonStore)

	// Create a test lesson
	testLesson := createTestLesson()

	// Set up expectations
	mockStore.On("UpdateLesson", "test-id", mock.AnythingOfType("*lesson.Lesson")).Return(nil)

	// Create handler with mock store
	handler := NewLessonHandler(mockStore)

	// Create a request body
	body, err := json.Marshal(testLesson)
	assert.NoError(t, err)

	// Create a request
	req, err := http.NewRequest("PUT", "/api/lessons/test-id", bytes.NewBuffer(body))
	assert.NoError(t, err)

	// Add route variables
	vars := map[string]string{
		"id": "test-id",
	}
	req = SetURLVars(req, vars)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.updateLesson(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify that expectations were met
	mockStore.AssertExpectations(t)
}

// Test deleteLesson handler
func TestDeleteLesson(t *testing.T) {
	// Create a mock store
	mockStore := new(MockLessonStore)

	// Create a test lesson
	testLesson := createTestLesson()

	// Set up expectations
	mockStore.On("GetLesson", "test-id").Return(&testLesson, nil)
	mockStore.On("DeleteLesson", "test-id").Return(nil)

	// Create handler with mock store
	handler := NewLessonHandler(mockStore)

	// Create a request
	req, err := http.NewRequest("DELETE", "/api/lessons/test-id", nil)
	assert.NoError(t, err)

	// Add route variables
	vars := map[string]string{
		"id": "test-id",
	}
	req = SetURLVars(req, vars)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.deleteLesson(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Verify that expectations were met
	mockStore.AssertExpectations(t)
}

// Test startLesson handler
func TestStartLesson(t *testing.T) {
	// Create a mock store
	mockStore := new(MockLessonStore)

	// Create a test lesson
	testLesson := createTestLesson()

	// Set up expectations
	mockStore.On("GetLesson", "test-id").Return(&testLesson, nil)
	mockStore.On("UpdateLesson", "test-id", mock.AnythingOfType("*lesson.Lesson")).Return(nil)

	// Create handler with mock store
	handler := NewLessonHandler(mockStore)

	// Create a request
	req, err := http.NewRequest("POST", "/api/lessons/test-id/start", nil)
	assert.NoError(t, err)

	// Add route variables
	vars := map[string]string{
		"id": "test-id",
	}
	req = SetURLVars(req, vars)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.startLesson(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify that expectations were met
	mockStore.AssertExpectations(t)
}

// Test completeStep handler
func TestCompleteStep(t *testing.T) {
	// Create a mock store
	mockStore := new(MockLessonStore)

	// Create a test lesson
	testLesson := createTestLesson()

	// Set up expectations
	mockStore.On("GetLesson", "test-id").Return(&testLesson, nil)
	mockStore.On("UpdateLesson", "test-id", mock.AnythingOfType("*lesson.Lesson")).Return(nil)

	// Create handler with mock store
	handler := NewLessonHandler(mockStore)

	// Create a request body with expected output
	body := bytes.NewBufferString(`{"output":"Hello, World!"}`)

	// Create a request
	req, err := http.NewRequest("POST", "/api/lessons/test-id/steps/0/complete", body)
	assert.NoError(t, err)

	// Add route variables
	vars := map[string]string{
		"id":   "test-id",
		"step": "0",
	}
	req = SetURLVars(req, vars)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.completeStep(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify that expectations were met
	mockStore.AssertExpectations(t)
}

// Test validateStep handler
func TestValidateStep(t *testing.T) {
	// Create a mock store
	mockStore := new(MockLessonStore)

	// Create a test lesson
	testLesson := createTestLesson()

	// Set up expectations
	mockStore.On("GetLesson", "test-id").Return(&testLesson, nil)

	// Create handler with mock store
	handler := NewLessonHandler(mockStore)

	// Create a request body with expected output
	body := bytes.NewBufferString(`{"output":"Hello, World!"}`)

	// Create a request
	req, err := http.NewRequest("POST", "/api/lessons/test-id/validate", body)
	assert.NoError(t, err)

	// Add route variables
	vars := map[string]string{
		"id":   "test-id",
		"step": "0",
	}
	req = SetURLVars(req, vars)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.validateStep(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify that expectations were met
	mockStore.AssertExpectations(t)
}
