package gig

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/vsayfb/gig-platform-core-service/pkg/httputil"
	"github.com/vsayfb/gig-platform-core-service/pkg/jwt"
	"github.com/vsayfb/gig-platform-core-service/pkg/middleware"
)

type Handler struct {
	service *GigService
}

func NewGigHandler(svc *GigService) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) RegisterRoutes(r chi.Router, jwtManager *jwt.Manager) {
	r.Get("/gigs", h.Feed)
	r.Get("/gigs/{id}", h.Get)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtManager))
		r.Post("/gigs", h.Create)
		r.Put("/gigs/{id}", h.Edit)
		r.Delete("/gigs/{id}", h.Cancel)
	})
}

func (h *Handler) Feed(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	lat, err := strconv.ParseFloat(q.Get("lat"), 64)

	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "lat is required")
		return
	}

	lng, err := strconv.ParseFloat(q.Get("lng"), 64)

	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "lng is required")
		return
	}

	p := FeedParams{Lat: lat, Lng: lng, RadiusMeters: 50000}

	if v := q.Get("radius"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			p.RadiusMeters = f
		}
	}
	if v := q.Get("duration_type"); v != "" {
		dt := DurationType(v)
		p.DurationType = &dt
	}

	if v := q.Get("category_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			p.CategoryID = &id
		}
	}
	if v := q.Get("cursor"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			p.Cursor = &t
		}
	}

	feed, err := h.service.Feed(r.Context(), p)

	if err != nil {
		slog.Error("internal server error", "err", err)

		httputil.WriteError(w, http.StatusInternalServerError, "feed unavailable")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, feed)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))

	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid gig id")
		return
	}

	detail, err := h.service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrGigNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "gig not found")
			return
		}
		slog.Error("internal server error", "err", err)

		httputil.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, detail)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	posterID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slog.Debug("new gig", "create", r.Body)

	var in CreateGigInput
	if err := httputil.DecodeJSON(r, &in); err != nil {

		slog.Warn("create gig - invalid body", "err", err)

		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	detail, err := h.service.Create(r.Context(), posterID, in)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {

			slog.Warn("create gig - invalid input", "err", err)

			httputil.WriteError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}

		slog.Error("could not create gig", "err", err)

		httputil.WriteError(w, http.StatusInternalServerError, "could not create gig")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, detail)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	posterID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	gigID, err := uuid.Parse(chi.URLParam(r, "id"))

	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid gig id")
		return
	}

	var in UpdateGigInput

	if err := httputil.DecodeJSON(r, &in); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	detail, err := h.service.Edit(r.Context(), gigID, posterID, in)

	if err != nil {
		switch {
		case errors.Is(err, ErrGigNotFound):
			httputil.WriteError(w, http.StatusNotFound, "gig not found")
		case errors.Is(err, ErrNotPoster):
			httputil.WriteError(w, http.StatusForbidden, "not the poster")
		case errors.Is(err, ErrGigNotEditable):
			httputil.WriteError(w, http.StatusConflict, "gig is not editable in its current status")
		default:
			slog.Error("internal server error", "err", err)

			httputil.WriteError(w, http.StatusInternalServerError, "could not edit gig")
		}
		return
	}

	httputil.WriteJSON(w, http.StatusOK, detail)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	callerID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	gigID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid gig id")
		return
	}

	if err := h.service.Cancel(r.Context(), gigID, callerID); err != nil {
		switch {
		case errors.Is(err, ErrGigNotFound):
			httputil.WriteError(w, http.StatusNotFound, "gig not found")
		case errors.Is(err, ErrNotPoster):
			httputil.WriteError(w, http.StatusForbidden, "not the poster")
		case errors.Is(err, ErrGigNotCancellable):
			httputil.WriteError(w, http.StatusConflict, "gig cannot be cancelled in its current status")
		default:
			slog.Error("internal server error", "err", err)

			httputil.WriteError(w, http.StatusInternalServerError, "could not cancel gig")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
