package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"lessoncraft/types"
)

type ApiHandler struct {
	router *mux.Router
}

func NewApiHandler() *ApiHandler {
	return &ApiHandler{
		router: mux.NewRouter(),
	}
}

func (h *ApiHandler) RegisterRoutes() {
	h.router.HandleFunc("/api/lessons", h.listLessons).Methods("GET")
	h.router.HandleFunc("/api/lessons/{id}", h.getLesson).Methods("GET")
	h.router.HandleFunc("/api/lessons", h.createLesson).Methods("POST")
	h.router.HandleFunc("/api/lessons/{id}", h.updateLesson).Methods("PUT")
	h.router.HandleFunc("/api/lessons/{id}", h.deleteLesson).Methods("DELETE")
	h.router.HandleFunc("/api/lessons/{id}/start", h.startLesson).Methods("POST")
	h.router.HandleFunc("/api/lessons/{id}/steps/{step}/complete", h.completeStep).Methods("POST")
	h.router.HandleFunc("/api/lessons/{id}/validate", h.validateStep).Methods("POST")
}

func (h *ApiHandler) listLessons(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement lesson listing
}

func (h *ApiHandler) getLesson(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement lesson retrieval
}

func (h *ApiHandler) createLesson(w http.ResponseWriter, r *http.Request) {
	var lesson types.Lesson
	if err := json.NewDecoder(r.Body).Decode(&lesson); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	lesson.CreatedAt = time.Now()
	// TODO: Save lesson to storage
}

func (h *ApiHandler) updateLesson(w http.ResponseWriter, r *http.Request) {
	var lesson types.Lesson
	if err := json.NewDecoder(r.Body).Decode(&lesson); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// TODO: Update lesson in storage
}

func (h *ApiHandler) deleteLesson(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	// TODO: Delete lesson from storage
}

func (h *ApiHandler) startLesson(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	// TODO: Start lesson session
}

func (h *ApiHandler) completeStep(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	step := vars["step"]
	// TODO: Mark step as complete
}

func (h *ApiHandler) validateStep(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	// TODO: Validate current step
}
