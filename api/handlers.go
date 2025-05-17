package api

import (
	"github.com/gorilla/mux"
)

type ApiHandler struct {
	router        *mux.Router
	lessonHandler *LessonHandler
}

func NewApiHandler(lessonStore LessonStore) *ApiHandler {
	return &ApiHandler{
		router:        mux.NewRouter(),
		lessonHandler: NewLessonHandler(lessonStore),
	}
}

func (h *ApiHandler) RegisterRoutes(*mux.Router) {
	h.lessonHandler.RegisterRoutes(h.router)
}
