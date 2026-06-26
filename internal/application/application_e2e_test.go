//go:build e2e

package application_test

// E2E tests for application endpoints.
//
// These tests require two valid JWTs — one for a poster and one for an
// applicant — because most flows need two distinct users.
//
// Usage:
//
//   go test -tags=e2e -v ./internal/application/... \
//     -e2e-base-url=http://localhost:8080 \
//     -e2e-poster-jwt=eyJhbGci...   (JWT of the gig poster)
//     -e2e-applicant-jwt=eyJhbGci... (JWT of a different user)
//
// Get both tokens by calling POST /auth/google with two different Google
// accounts, or reuse the token from a previous session as the poster and
// mint a second one for the applicant.

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Flags
// ---------------------------------------------------------------------------

var (
	e2eBaseURL      = flag.String("e2e-base-url", "http://localhost:8080", "Base URL of the running server")
	e2ePosterJWT    = flag.String("e2e-poster-jwt", "", "JWT for the gig poster user")
	e2eApplicantJWT = flag.String("e2e-applicant-jwt", "", "JWT for the applicant user")
)

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

func do(t *testing.T, method, path, token string, body any) *http.Response {
	t.Helper()

	var req *http.Request
	var err error

	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		req, err = http.NewRequest(method, *e2eBaseURL+path, bytes.NewReader(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, *e2eBaseURL+path, nil)
		require.NoError(t, err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

func requirePosterJWT(t *testing.T) string {
	t.Helper()
	if *e2ePosterJWT == "" {
		t.Skip("skipping e2e test: -e2e-poster-jwt not provided")
	}
	return *e2ePosterJWT
}

func requireApplicantJWT(t *testing.T) (string, string) {
	t.Helper()
	if *e2ePosterJWT == "" || *e2eApplicantJWT == "" {
		t.Skip("skipping e2e test: -e2e-poster-jwt and -e2e-applicant-jwt both required")
	}
	return *e2ePosterJWT, *e2eApplicantJWT
}

// createGig creates a gig as the poster and returns its ID.
func createGig(t *testing.T, posterToken string) string {
	t.Helper()
	start := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	end := time.Now().UTC().Add(8 * 24 * time.Hour).Format(time.RFC3339)
	expires := time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339)

	resp := do(t, http.MethodPost, "/gigs", posterToken, map[string]any{
		"title":             "E2E application test gig",
		"description_raw":   "Looking for help",
		"description_clean": "Looking for help",
		"duration_type":     "DAILY",
		"start_date":        start,
		"end_date":          end,
		"slots":             2,
		"expires_at":        expires,
		"lat":               41.015137,
		"lng":               28.979530,
		"city":              "Istanbul",
		"district":          "Kadikoy",
		"category_ids":      []string{},
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	id, ok := body["id"].(string)
	require.True(t, ok)
	return id
}

// applyToGig applies to a gig as the applicant and returns the application ID.
func applyToGig(t *testing.T, applicantToken, gigID string) string {
	t.Helper()
	resp := do(t, http.MethodPost, fmt.Sprintf("/gigs/%s/applications", gigID), applicantToken, nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	id, ok := body["id"].(string)
	require.True(t, ok)
	return id
}

// ---------------------------------------------------------------------------
// POST /gigs/{gigID}/applications
// ---------------------------------------------------------------------------

func TestE2E_Apply_ValidRequest_Returns201(t *testing.T) {
	posterToken, applicantToken := requireApplicantJWT(t)
	gigID := createGig(t, posterToken)

	resp := do(t, http.MethodPost, fmt.Sprintf("/gigs/%s/applications", gigID), applicantToken, nil)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.NotEmpty(t, body["id"])
	assert.Equal(t, gigID, body["gig_id"])
	assert.Equal(t, "PENDING", body["status"])
}

func TestE2E_Apply_NoAuth_Returns401(t *testing.T) {
	posterToken := requirePosterJWT(t)
	gigID := createGig(t, posterToken)

	resp := do(t, http.MethodPost, fmt.Sprintf("/gigs/%s/applications", gigID), "", nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_Apply_NonExistentGig_Returns404(t *testing.T) {
	_, applicantToken := requireApplicantJWT(t)

	resp := do(t, http.MethodPost, "/gigs/00000000-0000-0000-0000-000000000000/applications", applicantToken, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_Apply_InvalidGigID_Returns400(t *testing.T) {
	_, applicantToken := requireApplicantJWT(t)

	resp := do(t, http.MethodPost, "/gigs/not-a-uuid/applications", applicantToken, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_Apply_DuplicateApplication_Returns409(t *testing.T) {
	posterToken, applicantToken := requireApplicantJWT(t)
	gigID := createGig(t, posterToken)

	// First application
	resp1 := do(t, http.MethodPost, fmt.Sprintf("/gigs/%s/applications", gigID), applicantToken, nil)
	defer resp1.Body.Close()
	require.Equal(t, http.StatusCreated, resp1.StatusCode)

	// Duplicate
	resp2 := do(t, http.MethodPost, fmt.Sprintf("/gigs/%s/applications", gigID), applicantToken, nil)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusConflict, resp2.StatusCode)
}

func TestE2E_Apply_OwnGig_Returns422(t *testing.T) {
	posterToken, _ := requireApplicantJWT(t)
	gigID := createGig(t, posterToken)

	// Poster tries to apply to their own gig
	resp := do(t, http.MethodPost, fmt.Sprintf("/gigs/%s/applications", gigID), posterToken, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// GET /applications/{id}
// ---------------------------------------------------------------------------

func TestE2E_GetApplication_AsApplicant_Returns200(t *testing.T) {
	posterToken, applicantToken := requireApplicantJWT(t)
	gigID := createGig(t, posterToken)
	appID := applyToGig(t, applicantToken, gigID)

	resp := do(t, http.MethodGet, fmt.Sprintf("/applications/%s", appID), applicantToken, nil)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, appID, body["id"])
}

func TestE2E_GetApplication_AsPoster_Returns200(t *testing.T) {
	posterToken, applicantToken := requireApplicantJWT(t)
	gigID := createGig(t, posterToken)
	appID := applyToGig(t, applicantToken, gigID)

	// Poster can view the application too
	resp := do(t, http.MethodGet, fmt.Sprintf("/applications/%s", appID), posterToken, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestE2E_GetApplication_NoAuth_Returns401(t *testing.T) {
	posterToken, applicantToken := requireApplicantJWT(t)
	gigID := createGig(t, posterToken)
	appID := applyToGig(t, applicantToken, gigID)

	resp := do(t, http.MethodGet, fmt.Sprintf("/applications/%s", appID), "", nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_GetApplication_NotFound_Returns404(t *testing.T) {
	_, applicantToken := requireApplicantJWT(t)

	resp := do(t, http.MethodGet, "/applications/00000000-0000-0000-0000-000000000000", applicantToken, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_GetApplication_InvalidID_Returns400(t *testing.T) {
	_, applicantToken := requireApplicantJWT(t)

	resp := do(t, http.MethodGet, "/applications/not-a-uuid", applicantToken, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// GET /gigs/{gigID}/applications
// ---------------------------------------------------------------------------

func TestE2E_ListByGig_AsPoster_Returns200(t *testing.T) {
	posterToken, applicantToken := requireApplicantJWT(t)
	gigID := createGig(t, posterToken)
	applyToGig(t, applicantToken, gigID)

	resp := do(t, http.MethodGet, fmt.Sprintf("/gigs/%s/applications", gigID), posterToken, nil)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var apps []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&apps))
	assert.Len(t, apps, 1)
	assert.Equal(t, gigID, apps[0]["gig_id"])
}

func TestE2E_ListByGig_AsNonPoster_Returns403(t *testing.T) {
	posterToken, applicantToken := requireApplicantJWT(t)
	gigID := createGig(t, posterToken)

	// Applicant tries to list applications — should be forbidden
	resp := do(t, http.MethodGet, fmt.Sprintf("/gigs/%s/applications", gigID), applicantToken, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestE2E_ListByGig_NoAuth_Returns401(t *testing.T) {
	posterToken := requirePosterJWT(t)
	gigID := createGig(t, posterToken)

	resp := do(t, http.MethodGet, fmt.Sprintf("/gigs/%s/applications", gigID), "", nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_ListByGig_GigNotFound_Returns404(t *testing.T) {
	posterToken := requirePosterJWT(t)

	resp := do(t, http.MethodGet, "/gigs/00000000-0000-0000-0000-000000000000/applications", posterToken, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// DELETE /applications/{id}
// ---------------------------------------------------------------------------

func TestE2E_Withdraw_PendingApplication_Returns204(t *testing.T) {
	posterToken, applicantToken := requireApplicantJWT(t)
	gigID := createGig(t, posterToken)
	appID := applyToGig(t, applicantToken, gigID)

	resp := do(t, http.MethodDelete, fmt.Sprintf("/applications/%s", appID), applicantToken, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestE2E_Withdraw_NoAuth_Returns401(t *testing.T) {
	posterToken, applicantToken := requireApplicantJWT(t)
	gigID := createGig(t, posterToken)
	appID := applyToGig(t, applicantToken, gigID)

	resp := do(t, http.MethodDelete, fmt.Sprintf("/applications/%s", appID), "", nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_Withdraw_AlreadyWithdrawn_Returns409(t *testing.T) {
	posterToken, applicantToken := requireApplicantJWT(t)
	gigID := createGig(t, posterToken)
	appID := applyToGig(t, applicantToken, gigID)

	// First withdraw
	resp1 := do(t, http.MethodDelete, fmt.Sprintf("/applications/%s", appID), applicantToken, nil)
	defer resp1.Body.Close()
	require.Equal(t, http.StatusNoContent, resp1.StatusCode)

	// Second withdraw — already withdrawn, not PENDING
	resp2 := do(t, http.MethodDelete, fmt.Sprintf("/applications/%s", appID), applicantToken, nil)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusConflict, resp2.StatusCode)
}

func TestE2E_Withdraw_NotFound_Returns404(t *testing.T) {
	_, applicantToken := requireApplicantJWT(t)

	resp := do(t, http.MethodDelete, "/applications/00000000-0000-0000-0000-000000000000", applicantToken, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_Withdraw_InvalidID_Returns400(t *testing.T) {
	_, applicantToken := requireApplicantJWT(t)

	resp := do(t, http.MethodDelete, "/applications/not-a-uuid", applicantToken, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_Apply_ResponseShape(t *testing.T) {
	posterToken, applicantToken := requireApplicantJWT(t)
	gigID := createGig(t, posterToken)

	resp := do(t, http.MethodPost, fmt.Sprintf("/gigs/%s/applications", gigID), applicantToken, nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	for _, field := range []string{"id", "gig_id", "applicant_id", "status", "created_at"} {
		assert.Contains(t, body, field, "response must contain field: %s", field)
	}

	ct := resp.Header.Get("Content-Type")
	assert.True(t, strings.HasPrefix(ct, "application/json"),
		"Content-Type must be application/json, got: %s", ct)
}
