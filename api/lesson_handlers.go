package api

import (
	"encoding/json"
	"fmt"
	"github.com/ringo380/lessoncraft/api/middleware"
	"github.com/ringo380/lessoncraft/lesson"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// LessonHandler handles HTTP requests related to lessons.
// It provides endpoints for creating, retrieving, updating, and deleting lessons,
// as well as starting lessons, completing steps, and validating step outputs.
type LessonHandler struct {
	parser lesson.Parser // Parser for converting markdown to lessons
	store  LessonStore   // Storage for lessons
}

// LessonStore defines the interface for lesson storage operations.
// Implementations of this interface handle the persistence of lessons
// in various storage backends (e.g., MongoDB, in-memory).
type LessonStore interface {
	// ListLessons retrieves all lessons from the store.
	ListLessons() ([]lesson.Lesson, error)

	// GetLesson retrieves a lesson by its ID.
	GetLesson(id string) (*lesson.Lesson, error)

	// CreateLesson adds a new lesson to the store.
	CreateLesson(l *lesson.Lesson) error

	// UpdateLesson updates an existing lesson in the store.
	UpdateLesson(id string, l *lesson.Lesson) error

	// DeleteLesson removes a lesson from the store.
	DeleteLesson(id string) error
}

// NewLessonHandler creates a new LessonHandler with the provided store.
// It initializes a new parser for converting markdown to lessons.
//
// Parameters:
//   - store: An implementation of the LessonStore interface
//
// Returns:
//   - A pointer to a new LessonHandler
func NewLessonHandler(store LessonStore) *LessonHandler {
	return &LessonHandler{
		parser: lesson.NewParser(),
		store:  store,
	}
}

// RegisterRoutes registers the lesson-related routes with the provided router.
// It sets up the following endpoints:
//   - GET /api/lessons: List all lessons
//   - GET /api/lessons/{id}: Get a specific lesson
//   - POST /api/lessons: Create a new lesson
//   - PUT /api/lessons/{id}: Update an existing lesson
//   - DELETE /api/lessons/{id}: Delete a lesson
//   - POST /api/lessons/{id}/start: Start a lesson
//   - POST /api/lessons/{id}/steps/{step}/complete: Complete a step in a lesson
//   - POST /api/lessons/{id}/validate: Validate a step in a lesson
//
// Parameters:
//   - r: A mux.Router to register the routes with
func (h *LessonHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/lessons", h.listLessons).Methods("GET")
	r.HandleFunc("/api/lessons/{id}", h.getLesson).Methods("GET")
	r.HandleFunc("/api/lessons", h.createLesson).Methods("POST")
	r.HandleFunc("/api/lessons/{id}", h.updateLesson).Methods("PUT")
	r.HandleFunc("/api/lessons/{id}", h.deleteLesson).Methods("DELETE")
	r.HandleFunc("/api/lessons/{id}/start", h.startLesson).Methods("POST")
	r.HandleFunc("/api/lessons/{id}/steps/{step}/complete", h.completeStep).Methods("POST")
	r.HandleFunc("/api/lessons/{id}/validate", h.validateStep).Methods("POST")

	// New endpoints for lesson editor
	r.HandleFunc("/api/lessons/parse", h.parseMarkdown).Methods("POST")
	r.HandleFunc("/api/lessons/validate", h.validateLesson).Methods("POST")
}

func (h *LessonHandler) listLessons(w http.ResponseWriter, r *http.Request) {
	lessons, err := h.store.ListLessons()
	if err != nil {
		writeError(w, "DatabaseError", http.StatusInternalServerError, "Failed to retrieve lessons", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lessons)
}

func (h *LessonHandler) getLesson(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if len(vars) == 0 {
		// For testing, try to get vars from context
		vars = GetURLVars(r)
	}
	id := vars["id"]

	lesson, err := h.store.GetLesson(id)
	if err != nil {
		writeError(w, "NotFound", http.StatusNotFound, "Lesson not found", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lesson)
}

func (h *LessonHandler) createLesson(w http.ResponseWriter, r *http.Request) {
	var lesson lesson.Lesson
	if err := json.NewDecoder(r.Body).Decode(&lesson); err != nil {
		writeError(w, "InvalidRequest", http.StatusBadRequest, "Invalid lesson format", err)
		return
	}

	if err := validateLesson(&lesson); err != nil {
		writeError(w, "ValidationError", http.StatusBadRequest, "Lesson validation failed", err)
		return
	}

	if err := h.store.CreateLesson(&lesson); err != nil {
		writeError(w, "DatabaseError", http.StatusInternalServerError, "Failed to create lesson", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(lesson)
}

// validateLesson checks if a lesson meets all the validation requirements.
// It validates the lesson's title, description, and steps, ensuring they meet
// the required format and constraints.
//
// Validation rules include:
// - Title must be present and less than 100 characters
// - Description must be present and less than 500 characters
// - Lesson must have at least one step and no more than 50 steps
// - Each step must have a unique ID
// - Step content must be present and less than 5000 characters
// - If a step has expected output, it must have at least one command
// - Each step can have at most 10 commands
// - Each command must be less than 500 characters and valid
// - Step timeout must be between 0 and 1 hour
//
// Parameters:
//   - l: The lesson to validate
//
// Returns:
//   - An error if validation fails, nil otherwise
func validateLesson(l *lesson.Lesson) error {
	if l.Title == "" {
		return fmt.Errorf("lesson title is required")
	}
	if len(l.Title) > 100 {
		return fmt.Errorf("lesson title must be less than 100 characters")
	}
	if l.Description == "" {
		return fmt.Errorf("lesson description is required")
	}
	if len(l.Description) > 500 {
		return fmt.Errorf("lesson description must be less than 500 characters")
	}
	if len(l.Steps) == 0 {
		return fmt.Errorf("lesson must have at least one step")
	}
	if len(l.Steps) > 50 {
		return fmt.Errorf("lesson cannot have more than 50 steps")
	}

	seenIDs := make(map[string]bool)
	for i, step := range l.Steps {
		if step.ID == "" {
			return fmt.Errorf("step %d ID is required", i+1)
		}
		if seenIDs[step.ID] {
			return fmt.Errorf("duplicate step ID: %s", step.ID)
		}
		seenIDs[step.ID] = true

		if step.Content == "" {
			return fmt.Errorf("step %d content is required", i+1)
		}
		if len(step.Content) > 5000 {
			return fmt.Errorf("step %d content must be less than 5000 characters", i+1)
		}
		if step.Expected != "" && len(step.Commands) == 0 {
			return fmt.Errorf("step %d has expected output but no commands", i+1)
		}
		if len(step.Commands) > 10 {
			return fmt.Errorf("step %d cannot have more than 10 commands", i+1)
		}
		for j, cmd := range step.Commands {
			if len(cmd) > 500 {
				return fmt.Errorf("step %d command %d must be less than 500 characters", i+1, j+1)
			}
			if !isValidCommand(cmd) {
				return fmt.Errorf("step %d command %d contains invalid characters or syntax", i+1, j+1)
			}
		}
		if step.Timeout < 0 || step.Timeout > time.Hour {
			return fmt.Errorf("step %d timeout must be between 0 and 1 hour", i+1)
		}
	}
	return nil
}

// isValidCommand checks if a command is safe to execute in the lesson environment.
// It performs various security checks to prevent potentially dangerous commands.
//
// Security checks include:
// - Command must not be empty
// - Command must not be too long (over 1000 characters)
// - Command must not be a known dangerous command (e.g., rm -rf /)
// - Command must not contain shell escapes or other dangerous patterns
// - Command must not contain invalid control characters
//
// Parameters:
//   - cmd: The command to validate
//
// Returns:
//   - true if the command is valid, false otherwise
func isValidCommand(cmd string) bool {
	// Trim the command to remove leading/trailing whitespace
	cmd = strings.TrimSpace(cmd)

	// Check if command is empty
	if cmd == "" {
		return false
	}

	// Check for maximum length (prevent extremely long commands)
	if len(cmd) > 1000 {
		return false
	}

	// Check for potentially dangerous commands
	dangerousCommands := []string{
		"rm -rf /", "rm -rf /*", "rm -rf ~", "rm -rf .", "rm -rf ..",
		"mkfs", "dd if=/dev/zero", ":(){ :|:& };:", "> /dev/sda",
		"chmod -R 777 /", "wget", "curl", "nc", "telnet", "ssh",
		"sudo", "su", "passwd", "shutdown", "reboot", "halt", "poweroff",
		"init 0", "init 6",
	}

	for _, dangerous := range dangerousCommands {
		if strings.HasPrefix(cmd, dangerous) {
			return false
		}
	}

	// Check for shell escapes and other potentially dangerous patterns
	dangerousPatterns := []string{
		"`", "$(", "eval", "exec", "source", "bash -c", "sh -c",
		"python -c", "perl -e", "ruby -e", "php -r", "nc -e",
		"curl | bash", "wget | bash", "> /dev/null 2>&1",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmd, pattern) {
			return false
		}
	}

	// Check for invalid characters (control characters, etc.)
	for _, char := range cmd {
		if char < 32 && char != '\t' && char != '\n' && char != '\r' {
			return false
		}
	}

	// If all checks pass, the command is valid
	return true
}

// writeError logs an error and sends a standardized error response to the client.
// It formats the error message, logs it at the appropriate level based on the status code,
// and sends a JSON response with error details.
//
// Parameters:
//   - w: The HTTP response writer
//   - errType: The type of error (e.g., "ValidationError", "DatabaseError")
//   - code: The HTTP status code to return
//   - message: A human-readable error message
//   - err: The underlying error
func writeError(w http.ResponseWriter, errType string, code int, message string, err error) {
	// Log at appropriate level based on status code
	if code >= 500 {
		log.Printf("ERROR: [%s] Code: %d, Message: %s, Error: %v", errType, code, message, err)
	} else {
		log.Printf("WARN: [%s] Code: %d, Message: %s, Error: %v", errType, code, message, err)
	}

	// Send standardized error response to client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(middleware.ErrorResponse{
		Error:     errType,
		Code:      code,
		Message:   message,
		Details:   err.Error(),
		TimeStamp: time.Now(),
	})
}

func (h *LessonHandler) updateLesson(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if len(vars) == 0 {
		// For testing, try to get vars from context
		vars = GetURLVars(r)
	}
	id := vars["id"]

	var lesson lesson.Lesson
	if err := json.NewDecoder(r.Body).Decode(&lesson); err != nil {
		writeError(w, "InvalidRequest", http.StatusBadRequest, "Invalid lesson format", err)
		return
	}

	if err := validateLesson(&lesson); err != nil {
		writeError(w, "ValidationError", http.StatusBadRequest, "Lesson validation failed", err)
		return
	}

	if err := h.store.UpdateLesson(id, &lesson); err != nil {
		writeError(w, "DatabaseError", http.StatusInternalServerError, "Failed to update lesson", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lesson)
}

func (h *LessonHandler) deleteLesson(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if len(vars) == 0 {
		// For testing, try to get vars from context
		vars = GetURLVars(r)
	}
	id := vars["id"]

	// Check if lesson exists before deleting
	_, err := h.store.GetLesson(id)
	if err != nil {
		writeError(w, "NotFound", http.StatusNotFound, "Lesson not found", err)
		return
	}

	if err := h.store.DeleteLesson(id); err != nil {
		writeError(w, "DatabaseError", http.StatusInternalServerError, "Failed to delete lesson", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *LessonHandler) startLesson(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if len(vars) == 0 {
		// For testing, try to get vars from context
		vars = GetURLVars(r)
	}
	id := vars["id"]

	lesson, err := h.store.GetLesson(id)
	if err != nil {
		writeError(w, "NotFound", http.StatusNotFound, "Lesson not found", err)
		return
	}

	// Initialize lesson state
	lesson.CurrentStep = 0

	// Update the lesson in the store
	if err := h.store.UpdateLesson(id, lesson); err != nil {
		writeError(w, "DatabaseError", http.StatusInternalServerError, "Failed to update lesson state", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lesson)
}

func (h *LessonHandler) completeStep(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if len(vars) == 0 {
		// For testing, try to get vars from context
		vars = GetURLVars(r)
	}
	id := vars["id"]
	stepStr := vars["step"]

	// Get the lesson
	lesson, err := h.store.GetLesson(id)
	if err != nil {
		writeError(w, "NotFound", http.StatusNotFound, "Lesson not found", err)
		return
	}

	// Convert step string to integer
	stepIndex, err := strconv.Atoi(stepStr)
	if err != nil {
		writeError(w, "InvalidRequest", http.StatusBadRequest, "Invalid step index format", err)
		return
	}

	if stepIndex < 0 || stepIndex >= len(lesson.Steps) {
		writeError(w, "InvalidRequest", http.StatusBadRequest, "Step index out of range", fmt.Errorf("step index %d is out of range [0-%d]", stepIndex, len(lesson.Steps)-1))
		return
	}

	// Get the current step
	currentStep := lesson.Steps[stepIndex]

	// If the step has expected output, validate it
	if currentStep.Expected != "" {
		var output struct {
			Output string `json:"output"`
		}
		if err := json.NewDecoder(r.Body).Decode(&output); err != nil {
			writeError(w, "InvalidRequest", http.StatusBadRequest, "Invalid request body", err)
			return
		}

		// Normalize output and expected result
		normalizedOutput := strings.TrimSpace(output.Output)
		normalizedExpected := strings.TrimSpace(currentStep.Expected)

		if normalizedOutput != normalizedExpected {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"valid":    false,
				"message":  "Output does not match expected result",
				"expected": normalizedExpected,
				"received": normalizedOutput,
			})
			return
		}
	}

	// Update the current step in the lesson
	if stepIndex == lesson.CurrentStep {
		lesson.CurrentStep++
		// Update the lesson in the store
		if err := h.store.UpdateLesson(id, lesson); err != nil {
			writeError(w, "DatabaseError", http.StatusInternalServerError, "Failed to update lesson progress", err)
			return
		}
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":        true,
		"message":      "Step completed successfully",
		"current_step": lesson.CurrentStep,
	})
}

func (h *LessonHandler) validateStep(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if len(vars) == 0 {
		// For testing, try to get vars from context
		vars = GetURLVars(r)
	}
	id := vars["id"]
	stepIndex := vars["step"]

	lesson, err := h.store.GetLesson(id)
	if err != nil {
		writeError(w, "NotFound", http.StatusNotFound, "Lesson not found", err)
		return
	}

	step, err := strconv.Atoi(stepIndex)
	if err != nil {
		writeError(w, "InvalidRequest", http.StatusBadRequest, "Invalid step index format", err)
		return
	}

	if step < 0 || step >= len(lesson.Steps) {
		writeError(w, "InvalidRequest", http.StatusBadRequest, "Step index out of range", fmt.Errorf("step index %d is out of range [0-%d]", step, len(lesson.Steps)-1))
		return
	}

	var output struct {
		Output string `json:"output"`
	}
	if err := json.NewDecoder(r.Body).Decode(&output); err != nil {
		writeError(w, "InvalidRequest", http.StatusBadRequest, "Invalid request body", err)
		return
	}

	currentStep := lesson.Steps[step]
	if currentStep.Expected == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":   true,
			"message": "No expected output for this step",
		})
		return
	}

	// Normalize output and expected result
	normalizedOutput := strings.TrimSpace(output.Output)
	normalizedExpected := strings.TrimSpace(currentStep.Expected)

	w.Header().Set("Content-Type", "application/json")
	if normalizedOutput == normalizedExpected {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":   true,
			"message": "Step completed successfully",
		})
	} else {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":    false,
			"message":  "Output does not match expected result",
			"expected": normalizedExpected,
			"received": normalizedOutput,
		})
	}
}

// parseMarkdown parses markdown content into a lesson object.
// It accepts a JSON request with a 'markdown' field containing the markdown content.
// It returns a JSON response with the parsed lesson object.
//
// This endpoint is used by the lesson editor to convert markdown to a lesson object
// for preview and validation purposes.
func (h *LessonHandler) parseMarkdown(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Markdown string `json:"markdown"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, "InvalidRequest", http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if request.Markdown == "" {
		writeError(w, "InvalidRequest", http.StatusBadRequest, "Markdown content is required", fmt.Errorf("markdown content is empty"))
		return
	}

	// Parse the markdown content
	reader := strings.NewReader(request.Markdown)
	lesson, err := h.parser.Parse(reader)
	if err != nil {
		writeError(w, "ParsingError", http.StatusBadRequest, "Failed to parse markdown content", err)
		return
	}

	// Set default values for required fields if not present
	if lesson.ID == "" {
		lesson.ID = fmt.Sprintf("lesson-%s", time.Now().Format("20060102150405"))
	}

	if lesson.CreatedAt.IsZero() {
		lesson.CreatedAt = time.Now()
	}

	// Return the parsed lesson
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lesson)
}

// validateLesson validates a lesson object.
// It accepts a JSON request with a lesson object and validates it against the lesson validation rules.
// It returns a JSON response with a success message if the lesson is valid, or an error message if it's not.
//
// This endpoint is used by the lesson editor to validate lesson content before saving.
func (h *LessonHandler) validateLesson(w http.ResponseWriter, r *http.Request) {
	var lessonData lesson.Lesson

	if err := json.NewDecoder(r.Body).Decode(&lessonData); err != nil {
		writeError(w, "InvalidRequest", http.StatusBadRequest, "Invalid lesson format", err)
		return
	}

	// Validate the lesson
	if err := validateLesson(&lessonData); err != nil {
		writeError(w, "ValidationError", http.StatusBadRequest, "Lesson validation failed", err)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":   true,
		"message": "Lesson content is valid",
	})
}
