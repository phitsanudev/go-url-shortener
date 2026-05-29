package http

import (
	"net/http"

	"github.com/bell/go-url-shortener/backend/internal/application"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func NewRouter(service *application.URLService) http.Handler {
	handler := NewURLHandler(service)

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", handler.HealthCheck)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/shorten", handler.Shorten)
		r.Delete("/cleanup", handler.CleanupExpired)
	})

	r.Get("/{code}", handler.Redirect)

	return r
}
