package api

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
	"lessoncraft/lesson"
)

type LessonHandler struct {
	parser *lesson.Parser
	store  LessonStore
}

type LessonStore interface {
	ListLessons() ([]lesson.Lesson, error)
	GetLesson(id string) (*lesson.Lesson, error)
	CreateLesson(l *lesson.Lesson) error
	UpdateLesson(id string, l *lesson.Lesson) error
	DeleteLesson(id string) error
}

func NewLessonHandler(store LessonStore) *LessonHandler {
	return &LessonHandler{
		parser: lesson.NewParser(),
		store:  store,
	}
}

func (h *LessonHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/lessons", h.listLessons).Methods("GET")
	r.HandleFunc("/api/lessons/{id}", h.getLesson).Methods("GET")
	r.HandleFunc("/api/lessons", h.createLesson).Methods("POST")
	r.HandleFunc("/api/lessons/{id}", h.updateLesson).Methods("PUT")
	r.HandleFunc("/api/lessons/{id}", h.deleteLesson).Methods("DELETE")
	r.HandleFunc("/api/lessons/{id}/start", h.startLesson).Methods("POST")
	r.HandleFunc("/api/lessons/{id}/steps/{step}/complete", h.completeStep).Methods("POST")
	r.HandleFunc("/api/lessons/{id}/validate", h.validateStep).Methods("POST")
}

func (h *LessonHandler) listLessons(w http.ResponseWriter, r *http.Request) {
	lessons, err := h.store.ListLessons()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(lessons)
}

func (h *LessonHandler) getLesson(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	lesson, err := h.store.GetLesson(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
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

func validateLesson(l *lesson.Lesson) error {
	if l.Title == "" {
		return fmt.Errorf("lesson title is required")
	}
	if len(l.Steps) == 0 {
		return fmt.Errorf("lesson must have at least one step")
	}
	for i, step := range l.Steps {
		if step.Content == "" {
			return fmt.Errorf("step %d content is required", i+1)
		}
		if step.Expected != "" && len(step.Commands) == 0 {
			return fmt.Errorf("step %d has expected output but no commands", i+1)
		}
	}
	return nil
}

func writeError(w http.ResponseWriter, errType string, code int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:     errType,
		Code:      code,
		Message:   message,
		Details:   err.Error(),
		TimeStamp: time.Now(),
	})
}

func (h *LessonHandler) updateLesson(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	var lesson lesson.Lesson
	if err := json.NewDecoder(r.Body).Decode(&lesson); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := h.store.UpdateLesson(id, &lesson); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(w).Encode(lesson)
}

func (h *LessonHandler) deleteLesson(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	if err := h.store.DeleteLesson(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

func (h *LessonHandler) startLesson(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	lesson, err := h.store.GetLesson(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	// Initialize lesson state
	lesson.CurrentStep = 0
	json.NewEncoder(w).Encode(lesson)
}

func (h *LessonHandler) completeStep(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	step := vars["step"]
	
	// Validate step completion
	// TODO: Implement validation logic
	
	w.WriteHeader(http.StatusOK)
}

func (h *LessonHandler) validateStep(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	stepIndex := vars["step"]
	
	lesson, err := h.store.GetLesson(id)
	if err != nil {
		http.Error(w, "Lesson not found", http.StatusNotFound)
		return
	}

	step, err := strconv.Atoi(stepIndex)
	if err != nil || step < 0 || step >= len(lesson.Steps) {
		http.Error(w, "Invalid step index", http.StatusBadRequest)
		return
	}

	var output struct {
		Output string `json:"output"`
	}
	if err := json.NewDecoder(r.Body).Decode(&output); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	currentStep := lesson.Steps[step]
	if currentStep.Expected == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Normalize output and expected result
	normalizedOutput := strings.TrimSpace(output.Output)
	normalizedExpected := strings.TrimSpace(currentStep.Expected)

	if normalizedOutput == normalizedExpected {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": true,
			"message": "Step completed successfully",
		})
	} else {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": false,
			"message": "Output does not match expected result",
			"expected": normalizedExpected,
			"received": normalizedOutput,
		})
	}
}
