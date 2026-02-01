package http

import (
	"net/http"

	"service-timetable/internal/http/handlers"
)

type Router struct {
	mux *http.ServeMux
}

func NewRouter(adminHandler *handlers.AdminHandler) *Router {
	mux := http.NewServeMux()
	adminHandler.Register(mux)

	return &Router{mux: mux}
}

func (r *Router) Handler() http.Handler {
	return r.mux
}
