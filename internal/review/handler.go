package review

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/vsayfb/gig-platform-core-service/pkg/httputil"
	"github.com/vsayfb/gig-platform-core-service/pkg/jwt"
	"github.com/vsayfb/gig-platform-core-service/pkg/middleware"
)

type ReviewHandler struct {
	svc ReviewService
}

func NewReviewHandler(svc ReviewService) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

func (h *ReviewHandler) RegisterRoutes(r chi.Router, jwtManager *jwt.Manager) {
	r.Get("/users/{userID}/reviews", h.ListByUser)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtManager))
		r.Post("/contracts/{contractID}/reviews", h.Submit)
	})
}

// POST /contracts/:contractID/reviews
func (h *ReviewHandler) Submit(w http.ResponseWriter, r *http.Request) {
	reviewerID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	contractID, err := uuid.Parse(chi.URLParam(r, "contractID"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid contract id")
		return
	}

	var in CreateReviewInput
	if err := httputil.DecodeJSON(r, &in); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rev, err := h.svc.Submit(r.Context(), contractID, reviewerID, in)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httputil.WriteError(w, http.StatusNotFound, "contract not found")
		case errors.Is(err, ErrNotParty):
			httputil.WriteError(w, http.StatusForbidden, "not a party to this contract")
		case errors.Is(err, ErrContractNotDone):
			httputil.WriteError(w, http.StatusConflict, "contract is not completed")
		case errors.Is(err, ErrAlreadyReviewed):
			httputil.WriteError(w, http.StatusConflict, "already reviewed this contract")
		case errors.Is(err, ErrInvalidRating):
			httputil.WriteError(w, http.StatusUnprocessableEntity, "rating must be between 1 and 5")
		default:
			httputil.WriteError(w, http.StatusInternalServerError, "could not submit review")
		}
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, rev)
}

// GET /users/:userID/reviews
func (h *ReviewHandler) ListByUser(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())

	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	reviews, err := h.svc.ListByUser(r.Context(), userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, reviews)
}
