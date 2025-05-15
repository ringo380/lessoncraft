package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"lessoncraft/types"
)

type ApiHandler struct {
	router         *mux.Router
	lessonHandler *LessonHandler
}

func NewApiHandler(lessonStore LessonStore) *ApiHandler {
	return &ApiHandler{
		router:         mux.NewRouter(),
		lessonHandler: NewLessonHandler(lessonStore),
	}
}

func (h *ApiHandler) RegisterRoutes() {
	h.lessonHandler.RegisterRoutes(h.router)
}
