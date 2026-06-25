package application

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/vsayfb/gig-platform-core-service/internal/gig"
	"github.com/vsayfb/gig-platform-core-service/pkg/httputil"
	"github.com/vsayfb/gig-platform-core-service/pkg/jwt"
	"github.com/vsayfb/gig-platform-core-service/pkg/middleware"
)

type ApplicationHandler struct {
	svc ApplicationService
}

func NewApplicationHandler(svc ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{svc: svc}
}

func (h *ApplicationHandler) RegisterRoutes(r chi.Router, jwtManager *jwt.Manager) {

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtManager))
		r.Get("/gigs/{gigID}/applications", h.ListByGig)
		r.Post("/gigs/{gigID}/applications", h.Apply)
		r.Get("/applications/{id}", h.Get)
		r.Delete("/applications/{id}", h.Withdraw)
	})

}

// GET /gigs/:gigID/applications  (poster only)
func (h *ApplicationHandler) ListByGig(w http.ResponseWriter, r *http.Request) {
	callerID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	gigID, err := uuid.Parse(chi.URLParam(r, "gigID"))

	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid gig id")
		return
	}

	apps, err := h.svc.ListByGig(r.Context(), gigID, callerID)

	if err != nil {
		switch {
		case errors.Is(err, gig.ErrGigNotFound):
			httputil.WriteError(w, http.StatusNotFound, "gig not found")
		case errors.Is(err, gig.ErrNotPoster):
			httputil.WriteError(w, http.StatusForbidden, "not the poster")
		default:
			httputil.WriteError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	httputil.WriteJSON(w, http.StatusOK, apps)
}

// GET /applications/:id
func (h *ApplicationHandler) Get(w http.ResponseWriter, r *http.Request) {
	callerID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid application id")
		return
	}

	a, err := h.svc.Get(r.Context(), id, callerID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httputil.WriteError(w, http.StatusNotFound, "application not found")
		case errors.Is(err, ErrNotApplicant):
			httputil.WriteError(w, http.StatusForbidden, "forbidden")
		default:
			httputil.WriteError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	httputil.WriteJSON(w, http.StatusOK, a)
}

// POST /gigs/:gigID/applications
func (h *ApplicationHandler) Apply(w http.ResponseWriter, r *http.Request) {
	applicantID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	gigID, err := uuid.Parse(chi.URLParam(r, "gigID"))

	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid gig id")
		return
	}

	a, err := h.svc.Apply(r.Context(), gigID, applicantID)

	if err != nil {
		switch {
		case errors.Is(err, gig.ErrGigNotFound):
			httputil.WriteError(w, http.StatusNotFound, "gig not found")
		case errors.Is(err, ErrGigNotOpen):
			httputil.WriteError(w, http.StatusConflict, "gig is not open")
		case errors.Is(err, ErrCannotApplyOwn):
			httputil.WriteError(w, http.StatusUnprocessableEntity, "cannot apply to your own gig")
		case errors.Is(err, ErrAlreadyApplied):
			httputil.WriteError(w, http.StatusConflict, "already applied to this gig")
		default:
			httputil.WriteError(w, http.StatusInternalServerError, "could not apply")
		}
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, a)
}

// DELETE /applications/:id
func (h *ApplicationHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	applicantID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))

	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid application id")
		return
	}

	if err := h.svc.Withdraw(r.Context(), id, applicantID); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httputil.WriteError(w, http.StatusNotFound, "application not found")
		case errors.Is(err, ErrNotApplicant):
			httputil.WriteError(w, http.StatusForbidden, "not the applicant")
		case errors.Is(err, ErrNotWithdrawable):
			httputil.WriteError(w, http.StatusConflict, "only pending applications can be withdrawn")
		default:
			slog.Error("internal error during withdraw", "err", err)
			httputil.WriteError(w, http.StatusInternalServerError, "could not withdraw")
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
