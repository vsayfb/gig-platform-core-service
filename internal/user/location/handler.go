package location

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vsayfb/gig-platform-core-service/pkg/httputil"
	"github.com/vsayfb/gig-platform-core-service/pkg/middleware"
)

type UserLocationHandler struct {
	service *UserLocationService
}

func NewUserLocationHandler(service *UserLocationService) *UserLocationHandler {
	return &UserLocationHandler{service: service}
}

func (h *UserLocationHandler) RegisterRoutes(r chi.Router) {
	r.Put("/users/me/location", h.Upsert)
}

func (h *UserLocationHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	loc, err := h.service.Upsert(r.Context(), userID, input.Lat, input.Lng)

	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, loc)
}
