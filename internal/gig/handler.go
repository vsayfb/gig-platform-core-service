package gig

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/vsayfb/gig-platform-core-service/pkg/httputil"
	"github.com/vsayfb/gig-platform-core-service/pkg/jwt"
	"github.com/vsayfb/gig-platform-core-service/pkg/middleware"
	"github.com/vsayfb/gig-platform-core-service/pkg/squs"
)

type GigHandler struct {
	svc            *GigService
	eventPublisher *squs.SQSPublisher
}

func NewGigHandler(svc *GigService, eventPublisher *squs.SQSPublisher) *GigHandler {
	return &GigHandler{svc: svc, eventPublisher: eventPublisher}
}

func (h *GigHandler) RegisterRoutes(r chi.Router, jwtManager *jwt.Manager) {
	r.Get("/gigs", h.Feed)
	r.Get("/gigs/{id}", h.Get)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtManager))
		r.Post("/gigs", h.Create)
		r.Put("/gigs/{id}", h.Edit)
		r.Delete("/gigs/{id}", h.Cancel)
	})
}

// GET /gigs
func (h *GigHandler) Feed(w http.ResponseWriter, r *http.Request) {
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

	p := FeedParams{Lat: lat, Lng: lng, RadiusMeters: RADIUS_METERS}

	if v := q.Get("radius"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			p.RadiusMeters = f
		}
	}

	if v := q.Get("cursor"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			p.Cursor = &t
		}
	}

	feed, err := h.svc.Feed(r.Context(), p)

	if err != nil {
		slog.Error("internal server error", "err", err)

		httputil.WriteError(w, http.StatusInternalServerError, "feed unavailable")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, feed)
}

// GET /gigs/:id
func (h *GigHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))

	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid gig id")
		return
	}

	detail, err := h.svc.Get(r.Context(), id)

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

// POST /gigs
func (h *GigHandler) Create(w http.ResponseWriter, r *http.Request) {
	posterID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var in CreateGigInput

	slog.Warn("received body ", "body", r.Body)

	if err := httputil.DecodeJSON(r, &in); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	detail, err := h.svc.Create(r.Context(), posterID, in)

	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			httputil.WriteError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}

		slog.Error("internal server error", "err", err)

		httputil.WriteError(w, http.StatusInternalServerError, "could not create gig")
		return
	}

	go func(publishCtx context.Context) {
		err := h.eventPublisher.Publish(publishCtx, squs.GigCreatedEvent{
			GigID:       detail.ID,
			Title:       detail.Title,
			Description: detail.DescriptionRaw,
			Location: squs.GigLocation{
				Lat: detail.Location.Lat,
				Lng: detail.Location.Lng,
			},
		})
		if err != nil {
			slog.ErrorContext(publishCtx, "failed to publish event", "err", err)
		}
	}(trace.ContextWithSpan(context.Background(), trace.SpanFromContext(r.Context())))

	httputil.WriteJSON(w, http.StatusCreated, detail)
}

// PUT /gigs/:id
func (h *GigHandler) Edit(w http.ResponseWriter, r *http.Request) {
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

	detail, err := h.svc.Edit(r.Context(), gigID, posterID, in)
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

// DELETE /gigs/:id
func (h *GigHandler) Cancel(w http.ResponseWriter, r *http.Request) {
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

	if err := h.svc.Cancel(r.Context(), gigID, callerID); err != nil {
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
