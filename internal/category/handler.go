package category

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vsayfb/gig-platform-core-service/pkg/httputil"
	"github.com/vsayfb/gig-platform-core-service/pkg/jwt"
	"github.com/vsayfb/gig-platform-core-service/pkg/middleware"
)

type CategoryHandler struct {
	service CategoryService
}

func NewCategoryHandler(service CategoryService) *CategoryHandler {
	return &CategoryHandler{service: service}
}

func (h *CategoryHandler) RegisterRoutes(r chi.Router, jwtManager *jwt.Manager) {
	r.Get("/categories", h.FindActive)
	r.Get("/categories/search", h.FindBySlug)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtManager))
		r.Post("/categories/suggest", h.Suggest)
	})
}

func (h *CategoryHandler) FindActive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	var cursor uuid.UUID

	if value := r.URL.Query().Get("cursor"); value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil {
			http.Error(w, "invalid cursor", http.StatusBadRequest)
			return
		}

		cursor = parsed
	}

	categories, err := h.service.ListActive(
		ctx,
		cursor,
		limit,
	)

	if err != nil {
		slog.ErrorContext(r.Context(), "internal server error", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nextCursor := ""

	if len(categories) > 0 {
		nextCursor = categories[len(categories)-1].ID.String()
	}

	httputil.WriteJSON(w, http.StatusOK, struct {
		Data       []*Category `json:"data"`
		NextCursor string      `json:"next_cursor"`
	}{
		Data:       categories,
		NextCursor: nextCursor,
	})
}
func (h *CategoryHandler) FindBySlug(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := r.URL.Query().Get("q")

	categories, err := h.service.ListBySlug(
		ctx,
		query,
	)

	if err != nil {
		slog.ErrorContext(r.Context(), "internal server error", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
			slog.ErrorContext(r.Context(), "internal server error", "err", err)

			httputil.WriteError(w, http.StatusInternalServerError, "failed to suggest category")
		}

		return
	}

	httputil.WriteJSON(w, http.StatusCreated, category)
}
