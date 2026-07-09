package bootstrap

import (
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/vsayfb/gig-platform-core-service/config"
	"github.com/vsayfb/gig-platform-core-service/pkg/jwt"
	"github.com/vsayfb/gig-platform-core-service/pkg/middleware"
)

func newRouter(cfg *config.Config, h *handlers, jwtManager *jwt.Manager) *chi.Mux {
	r := chi.NewRouter()

	if cfg.Env != "production" {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
		}))
	}

	r.Use(chimiddleware.RequestID)
	r.Use(middleware.TracingMiddleware)
	r.Use(middleware.StructuredLogger)
	r.Use(middleware.MetricsMiddleware)
	r.Use(chimiddleware.Recoverer)

	r.Group(func(r chi.Router) {
		h.auth.RegisterRoutes(r)
		h.category.RegisterRoutes(r, jwtManager)
		h.gig.RegisterRoutes(r, jwtManager)
		h.application.RegisterRoutes(r, jwtManager)
		h.review.RegisterRoutes(r, jwtManager)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtManager))
		h.user.RegisterRoutes(r)
		h.location.RegisterRoutes(r)
		h.contract.RegisterRoutes(r)
		h.notification.RegisterRoutes(r)
	})

	return r
}
