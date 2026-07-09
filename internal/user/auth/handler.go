package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	util "github.com/vsayfb/gig-platform-core-service/pkg/httputil"
)

type UserAuthHandler struct {
	service UserAuthService
}

func NewUserAuthHandler(service UserAuthService) *UserAuthHandler {
	return &UserAuthHandler{service: service}
}

func (h *UserAuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/auth/google", h.GoogleLogin)
}

func (h *UserAuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var input struct {
		IDToken string `json:"id_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.IDToken == "" {
		util.WriteError(w, http.StatusBadRequest, "id_token is required")
		return
	}

	result, err := h.service.GoogleLogin(r.Context(), input.IDToken)

	if err != nil {
		slog.Warn("authentication failed", "err", err)

		util.WriteError(w, http.StatusUnauthorized, "authentication failed")
		return
	}

	util.WriteJSON(w, http.StatusOK, map[string]any{
		"token": result.Token,
		"user":  result.User,
	})
}
