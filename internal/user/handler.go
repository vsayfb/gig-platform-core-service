package user

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vsayfb/gig-platform-core-service/pkg/httputil"
	"github.com/vsayfb/gig-platform-core-service/pkg/middleware"
)

type UserHandler struct {
	service *UserService
}

func NewUserHandler(service *UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) RegisterRoutes(r chi.Router) {
	r.Get("/users/{id}", h.GetByID)
	r.Put("/users/me/profile", h.UpdateProfile)
	r.Post("/users/me/fcm-token", h.CreateFCMToken)
	r.Delete("/users/me/fcm-token", h.DeleteFCMToken)
	r.Put("/users/me/categories", h.UpdateCategories)

}

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))

	if err != nil {
		slog.WarnContext(r.Context(), "invalid request body", "err", err)
		httputil.WriteError(w, http.StatusBadRequest, "invalid user id")

		return
	}

	user, err := h.service.GetByID(r.Context(), id)

	if err != nil {
		slog.WarnContext(r.Context(), "user not found", "err", err)
		httputil.WriteError(w, http.StatusNotFound, "user not found")

		return
	}

	httputil.WriteJSON(w, http.StatusOK, user)
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name             string `json:"name"`
		Bio              string `json:"bio"`
		IsAvailableToday bool   `json:"is_available_today"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		slog.WarnContext(r.Context(), "invalid request body", "err", err)
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")

		return
	}

	userID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	updated, err := h.service.UpdateProfile(r.Context(), &User{
		ID:               userID,
		Name:             input.Name,
		Bio:              input.Bio,
		IsAvailableToday: input.IsAvailableToday,
	})

	if err != nil {

		slog.ErrorContext(r.Context(), "internal server error", "err", err)

		httputil.WriteError(w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, updated)
}

func (h *UserHandler) CreateFCMToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		slog.WarnContext(r.Context(), "invalid request body", "err", err)
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")

		return
	}

	userID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	err = h.service.AddFCMToken(r.Context(), userID, input.Token)

	if err != nil {
		slog.ErrorContext(r.Context(), "internal server error", "err", err)

		httputil.WriteError(w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) DeleteFCMToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		slog.WarnContext(r.Context(), "invalid request body", "err", err)
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")

		return
	}

	userID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	err = h.service.AddFCMToken(r.Context(), userID, input.Token)

	if err != nil {
		slog.ErrorContext(r.Context(), "internal server error", "err", err)

		httputil.WriteError(w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) UpdateCategories(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CategoryIDs []uuid.UUID `json:"category_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.WarnContext(r.Context(), "invalid request body", "err", err)
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	categoryIDs := req.CategoryIDs
	userID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	err = h.service.PutCategories(r.Context(), userID, categoryIDs)

	if err != nil {

		slog.ErrorContext(r.Context(), "internal server error", "err", err)

		httputil.WriteError(w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if err := h.service.Delete(r.Context(), userID); err != nil {
		slog.ErrorContext(r.Context(), "failed to delete user", "err", err)
		httputil.WriteError(w, http.StatusInternalServerError, "Internal server error.")

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
