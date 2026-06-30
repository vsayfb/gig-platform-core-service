package notification

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vsayfb/gig-platform-core-service/pkg/httputil"
	"github.com/vsayfb/gig-platform-core-service/pkg/middleware"
)

type NotificationHandler struct {
	service *NotificationService
}

func NewNotificationHandler(service *NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

type fcmTokenRequest struct {
	Token string `json:"token"`
}

func (h *NotificationHandler) RegisterRoutes(r chi.Router) {

	r.Post("/notifications/me/fcm-token", h.RegisterFCMToken)
	r.Delete("/notifications/me/fcm-token", h.DeleteFCMToken)
	r.Post("/notifications/me/notifications", h.ListNotifications)
}

func (h *NotificationHandler) RegisterFCMToken(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req fcmTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.RegisterFCMToken(r.Context(), userID, req.Token); err != nil {
		if err == ErrEmptyToken {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		slog.Error("failed to store token", "err", err)

		http.Error(w, "failed to store token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *NotificationHandler) DeleteFCMToken(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req fcmTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.RemoveFCMToken(r.Context(), userID, req.Token); err != nil {
		if err == ErrEmptyToken {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		slog.Error("failed to list notifications", "err", err)

		http.Error(w, "failed to delete token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var params ListNotificationsParams

	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
	}

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params.Limit = n
		}
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params.Offset = n
		}
	}

	notifications, err := h.service.ListNotifications(r.Context(), userID, params)

	if err != nil {
		slog.Error("failed to list notifications", "err", err)

		http.Error(w, "failed to list notifications", http.StatusInternalServerError)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, notifications)
}
