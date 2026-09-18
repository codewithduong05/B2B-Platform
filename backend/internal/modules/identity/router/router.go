package router

import (
	"github.com/go-chi/chi/v5"
)

type Router struct {
	router chi.Router
}

func New() *Router {
	r := chi.NewRouter()
	return &Router{router: r}
}

func (r *Router) ChiRouter() chi.Router {
	return r.router
}

func (r *Router) RegisterRoutes(authMiddleware interface{}, adminMiddleware interface{}) {
	// TODO: Implement routes
}
