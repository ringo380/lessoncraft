package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/ringo380/lessoncraft/api/store"
	"github.com/ringo380/lessoncraft/lesson"
	"github.com/stretchr/testify/assert"
)

// TestIntegrationLessonHandlers tests the integration between API handlers and storage
func TestIntegrationLessonHandlers(t *testing.T) {
	// Create a memory store for testing
	memoryStore := store.NewMemoryLessonStore()

	// Create a lesson handler with the memory store
	handler := NewLessonHandler(memoryStore)

	// Create a router and register the routes
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Test creating a lesson
	t.Run("Create Lesson", func(t *testing.T) {
		// Create a test lesson
		testLesson := lesson.Lesson{
			Title:       "Integration Test Lesson",
			Description: "This is a test lesson for integration testing",
			Steps: []lesson.LessonStep{
				{
					ID:       "step-1",
					Content:  "Step 1 content",
					Commands: []string{"echo 'Hello, World!'"},
					Expected: "Hello, World!",
					Timeout:  5 * time.Minute,
				},
			},
		}

		// Convert lesson to JSON
		body, err := json.Marshal(testLesson)
		assert.NoError(t, err)

		// Create a request
		req, err := http.NewRequest("POST", "/api/lessons", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create a response recorder
		rr := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(rr, req)

		// Check the status code
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Parse the response
		var responseLesson lesson.Lesson
		err = json.Unmarshal(rr.Body.Bytes(), &responseLesson)
		assert.NoError(t, err)

		// Check the response
		assert.NotEmpty(t, responseLesson.ID)
		assert.Equal(t, testLesson.Title, responseLesson.Title)
		assert.Equal(t, testLesson.Description, responseLesson.Description)
		assert.Len(t, responseLesson.Steps, 1)

		// Save the lesson ID for later tests
		lessonID := responseLesson.ID

		// Test getting the lesson
		t.Run("Get Lesson", func(t *testing.T) {
			// Create a request
			req, err := http.NewRequest("GET", "/api/lessons/"+lessonID, nil)
			assert.NoError(t, err)

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(rr, req)

			// Check the status code
			assert.Equal(t, http.StatusOK, rr.Code)

			// Parse the response
			var responseLesson lesson.Lesson
			err = json.Unmarshal(rr.Body.Bytes(), &responseLesson)
			assert.NoError(t, err)

			// Check the response
			assert.Equal(t, lessonID, responseLesson.ID)
			assert.Equal(t, testLesson.Title, responseLesson.Title)
			assert.Equal(t, testLesson.Description, responseLesson.Description)
			assert.Len(t, responseLesson.Steps, 1)
		})

		// Test updating the lesson
		t.Run("Update Lesson", func(t *testing.T) {
			// Modify the test lesson
			testLesson.ID = lessonID
			testLesson.Title = "Updated Integration Test Lesson"

			// Convert lesson to JSON
			body, err := json.Marshal(testLesson)
			assert.NoError(t, err)

			// Create a request
			req, err := http.NewRequest("PUT", "/api/lessons/"+lessonID, bytes.NewBuffer(body))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(rr, req)

			// Check the status code
			assert.Equal(t, http.StatusOK, rr.Code)

			// Parse the response
			var responseLesson lesson.Lesson
			err = json.Unmarshal(rr.Body.Bytes(), &responseLesson)
			assert.NoError(t, err)

			// Check the response
			assert.Equal(t, lessonID, responseLesson.ID)
			assert.Equal(t, "Updated Integration Test Lesson", responseLesson.Title)
			assert.Equal(t, testLesson.Description, responseLesson.Description)
			assert.Len(t, responseLesson.Steps, 1)
		})

		// Test listing lessons
		t.Run("List Lessons", func(t *testing.T) {
			// Create a request
			req, err := http.NewRequest("GET", "/api/lessons", nil)
			assert.NoError(t, err)

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(rr, req)

			// Check the status code
			assert.Equal(t, http.StatusOK, rr.Code)

			// Parse the response
			var responseLessons []lesson.Lesson
			err = json.Unmarshal(rr.Body.Bytes(), &responseLessons)
			assert.NoError(t, err)

			// Check the response
			assert.Len(t, responseLessons, 1)
			assert.Equal(t, lessonID, responseLessons[0].ID)
			assert.Equal(t, "Updated Integration Test Lesson", responseLessons[0].Title)
		})

		// Test starting a lesson
		t.Run("Start Lesson", func(t *testing.T) {
			// Create a request
			req, err := http.NewRequest("POST", "/api/lessons/"+lessonID+"/start", nil)
			assert.NoError(t, err)

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(rr, req)

			// Check the status code
			assert.Equal(t, http.StatusOK, rr.Code)

			// Parse the response
			var responseLesson lesson.Lesson
			err = json.Unmarshal(rr.Body.Bytes(), &responseLesson)
			assert.NoError(t, err)

			// Check the response
			assert.Equal(t, lessonID, responseLesson.ID)
			assert.Equal(t, 0, responseLesson.CurrentStep)
		})

		// Test completing a step
		t.Run("Complete Step", func(t *testing.T) {
			// Create a request body with expected output
			body := bytes.NewBufferString(`{"output":"Hello, World!"}`)

			// Create a request
			req, err := http.NewRequest("POST", "/api/lessons/"+lessonID+"/steps/0/complete", body)
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(rr, req)

			// Check the status code
			assert.Equal(t, http.StatusOK, rr.Code)

			// Parse the response
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Check the response
			assert.Equal(t, true, response["valid"])
			assert.Equal(t, "Step completed successfully", response["message"])
			assert.Equal(t, float64(1), response["current_step"])
		})

		// Test validating a step
		t.Run("Validate Step", func(t *testing.T) {
			// Create a request body with expected output
			body := bytes.NewBufferString(`{"output":"Hello, World!"}`)

			// Create a request
			req, err := http.NewRequest("POST", "/api/lessons/"+lessonID+"/validate", body)
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Add route variables
			vars := map[string]string{
				"id":   lessonID,
				"step": "0",
			}
			req = SetURLVars(req, vars)

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Call the handler directly since we need to set route variables
			handler.validateStep(rr, req)

			// Check the status code
			assert.Equal(t, http.StatusOK, rr.Code)

			// Parse the response
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Check the response
			assert.Equal(t, true, response["valid"])
			assert.Equal(t, "Step completed successfully", response["message"])
		})

		// Test deleting the lesson
		t.Run("Delete Lesson", func(t *testing.T) {
			// Create a request
			req, err := http.NewRequest("DELETE", "/api/lessons/"+lessonID, nil)
			assert.NoError(t, err)

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(rr, req)

			// Check the status code
			assert.Equal(t, http.StatusNoContent, rr.Code)

			// Verify the lesson was deleted
			req, err = http.NewRequest("GET", "/api/lessons/"+lessonID, nil)
			assert.NoError(t, err)

			rr = httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusNotFound, rr.Code)
		})
	})
}

// TestIntegrationLessonParsingAndValidation tests the integration between lesson parsing and validation
func TestIntegrationLessonParsingAndValidation(t *testing.T) {
	// Create a memory store for testing
	memoryStore := store.NewMemoryLessonStore()

	// Create a lesson handler with the memory store
	handler := NewLessonHandler(memoryStore)

	// Create a router and register the routes
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Test creating a lesson with invalid data
	t.Run("Create Invalid Lesson", func(t *testing.T) {
		// Create an invalid test lesson (missing title)
		testLesson := lesson.Lesson{
			Description: "This is a test lesson with missing title",
			Steps: []lesson.LessonStep{
				{
					ID:       "step-1",
					Content:  "Step 1 content",
					Commands: []string{"echo 'Hello, World!'"},
					Expected: "Hello, World!",
					Timeout:  5 * time.Minute,
				},
			},
		}

		// Convert lesson to JSON
		body, err := json.Marshal(testLesson)
		assert.NoError(t, err)

		// Create a request
		req, err := http.NewRequest("POST", "/api/lessons", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create a response recorder
		rr := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(rr, req)

		// Check the status code
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		// Parse the response
		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Check the response
		assert.Equal(t, "ValidationError", response["error"])
		assert.Equal(t, "Lesson validation failed", response["message"])
		assert.Equal(t, "lesson title is required", response["details"])
	})

	// Test creating a lesson with invalid step data
	t.Run("Create Lesson with Invalid Step", func(t *testing.T) {
		// Create a test lesson with invalid step (missing ID)
		testLesson := lesson.Lesson{
			Title:       "Test Lesson with Invalid Step",
			Description: "This is a test lesson with an invalid step",
			Steps: []lesson.LessonStep{
				{
					Content:  "Step 1 content",
					Commands: []string{"echo 'Hello, World!'"},
					Expected: "Hello, World!",
					Timeout:  5 * time.Minute,
				},
			},
		}

		// Convert lesson to JSON
		body, err := json.Marshal(testLesson)
		assert.NoError(t, err)

		// Create a request
		req, err := http.NewRequest("POST", "/api/lessons", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create a response recorder
		rr := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(rr, req)

		// Check the status code
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		// Parse the response
		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Check the response
		assert.Equal(t, "ValidationError", response["error"])
		assert.Equal(t, "Lesson validation failed", response["message"])
		assert.Equal(t, "step 1 ID is required", response["details"])
	})
}
