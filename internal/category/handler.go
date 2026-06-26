package category

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vsayfb/gig-platform-core-service/pkg/httputil"
	"github.com/vsayfb/gig-platform-core-service/pkg/jwt"
	"github.com/vsayfb/gig-platform-core-service/pkg/middleware"
)

type CategoryHandler struct {
	service *CategoryService
}

func NewCategoryHandler(service *CategoryService) *CategoryHandler {
	return &CategoryHandler{service: service}
}

func (h *CategoryHandler) RegisterRoutes(r chi.Router, jwtManager *jwt.Manager) {
	r.Get("/categories", h.ListActive)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtManager))
		r.Post("/categories/suggest", h.Suggest)
	})

}

func (h *CategoryHandler) ListActive(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.ListActive(r.Context())

	if err != nil {
		slog.Error("internal server error", "err", err)

		httputil.WriteError(w, http.StatusInternalServerError, "failed to fetch categories")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, categories)
}

func (h *CategoryHandler) Suggest(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Name == "" || input.Slug == "" {
		httputil.WriteError(w, http.StatusBadRequest, "name and slug are required")
		return
	}

	category, err := h.service.Suggest(r.Context(), input.Name, input.Slug)
	if err != nil {

		switch err {
		case ErrCategoryAlreadyExists:
			httputil.WriteError(w, http.StatusConflict, "category already exists")
		default:
			slog.Error("internal server error", "err", err)

			httputil.WriteError(w, http.StatusInternalServerError, "failed to suggest category")
		}

		return
	}

	httputil.WriteJSON(w, http.StatusCreated, category)
}
