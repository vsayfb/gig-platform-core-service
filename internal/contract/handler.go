package contract

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/vsayfb/gig-platform-core-service/pkg/httputil"
	"github.com/vsayfb/gig-platform-core-service/pkg/middleware"
)

type Handler struct {
	svc ContractService
}

func NewHandler(svc ContractService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Get("/contracts/{id}", h.Get)
		r.Post("/applications/{applicationID}/hire", h.Hire)
		r.Put("/contracts/{id}/job-done", h.JobDone)
		r.Put("/contracts/{id}/approve", h.Approve)
		r.Put("/contracts/{id}/dispute", h.Dispute)
		r.Put("/contracts/{id}/cancel", h.Cancel)
	})
}

// GET /contracts/:id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	callerID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))

	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid contract id")
		return
	}

	c, err := h.svc.Get(r.Context(), id, callerID)

	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httputil.WriteError(w, http.StatusNotFound, "contract not found")
		case errors.Is(err, ErrNotParty):
			httputil.WriteError(w, http.StatusForbidden, "forbidden")
		default:
			httputil.WriteError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	httputil.WriteJSON(w, http.StatusOK, c)
}

// POST /applications/:applicationID/hire
func (h *Handler) Hire(w http.ResponseWriter, r *http.Request) {

	employerID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	applicationID, err := uuid.Parse(chi.URLParam(r, "applicationID"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid application id")
		return
	}

	c, err := h.svc.Hire(r.Context(), applicationID, employerID)

	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httputil.WriteError(w, http.StatusNotFound, "application not found")
		case errors.Is(err, ErrNotEmployer):
			httputil.WriteError(w, http.StatusForbidden, "not the employer")
		case errors.Is(err, ErrAlreadyHired):
			httputil.WriteError(w, http.StatusConflict, "gig already has an active contract")
		case errors.Is(err, ErrInvalidTransition):
			httputil.WriteError(w, http.StatusConflict, "application is not in a hireable state")
		default:
			httputil.WriteError(w, http.StatusInternalServerError, "could not hire")
		}
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, c)
}

// PUT /contracts/:id/job-done
func (h *Handler) JobDone(w http.ResponseWriter, r *http.Request) {

	employeeID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid contract id")
		return
	}

	c, err := h.svc.JobDone(r.Context(), id, employeeID)

	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httputil.WriteError(w, http.StatusNotFound, "contract not found")
		case errors.Is(err, ErrNotEmployee):
			httputil.WriteError(w, http.StatusForbidden, "not the employee")
		case errors.Is(err, ErrInvalidTransition):
			httputil.WriteError(w, http.StatusConflict, "contract is not active")
		default:
			httputil.WriteError(w, http.StatusInternalServerError, "could not update contract")
		}
		return
	}

	httputil.WriteJSON(w, http.StatusOK, c)
}

// PUT /contracts/:id/approve
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {

	employerID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid contract id")
		return
	}

	c, err := h.svc.Approve(r.Context(), id, employerID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httputil.WriteError(w, http.StatusNotFound, "contract not found")
		case errors.Is(err, ErrNotEmployer):
			httputil.WriteError(w, http.StatusForbidden, "not the employer")
		case errors.Is(err, ErrInvalidTransition):
			httputil.WriteError(w, http.StatusConflict, "contract is not awaiting approval")
		default:
			httputil.WriteError(w, http.StatusInternalServerError, "could not approve contract")
		}
		return
	}

	httputil.WriteJSON(w, http.StatusOK, c)
}

// PUT /contracts/:id/dispute
func (h *Handler) Dispute(w http.ResponseWriter, r *http.Request) {
	employerID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid contract id")
		return
	}

	c, err := h.svc.Dispute(r.Context(), id, employerID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httputil.WriteError(w, http.StatusNotFound, "contract not found")
		case errors.Is(err, ErrNotEmployer):
			httputil.WriteError(w, http.StatusForbidden, "not the employer")
		case errors.Is(err, ErrInvalidTransition):
			httputil.WriteError(w, http.StatusConflict, "contract is not awaiting approval")
		default:
			httputil.WriteError(w, http.StatusInternalServerError, "could not dispute contract")
		}
		return
	}

	httputil.WriteJSON(w, http.StatusOK, c)
}

// PUT /contracts/:id/cancel
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {

	callerID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid contract id")
		return
	}

	c, err := h.svc.Cancel(r.Context(), id, callerID)

	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httputil.WriteError(w, http.StatusNotFound, "contract not found")
		case errors.Is(err, ErrNotParty):
			httputil.WriteError(w, http.StatusForbidden, "not a party to this contract")
		case errors.Is(err, ErrInvalidTransition):
			httputil.WriteError(w, http.StatusConflict, "contract cannot be cancelled in its current status")
		default:
			httputil.WriteError(w, http.StatusInternalServerError, "could not cancel contract")
		}
		return
	}

	httputil.WriteJSON(w, http.StatusOK, c)
}
