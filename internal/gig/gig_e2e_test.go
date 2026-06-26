//go:build e2e

package gig_test

// E2E tests for the gig endpoints.
//
// Usage:
//
//   go test -tags=e2e -v ./internal/gig/... \
//     -e2e-base-url=http://localhost:8080 \
//     -e2e-jwt=eyJhbGci...
//
// The JWT must belong to an existing user (use the token from POST /auth/google).
// Public endpoints (GET /gigs, GET /gigs/{id}) run without a token.

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
	e2eBaseURL = flag.String("e2e-base-url", "http://localhost:8080", "Base URL of the running server")
	e2eJWT     = flag.String("e2e-jwt", "", "Valid JWT for an existing user")
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

func requireJWT(t *testing.T) string {
	t.Helper()
	if *e2eJWT == "" {
		t.Skip("skipping e2e test: -e2e-jwt not provided")
	}
	return *e2eJWT
}

func validGigBody() map[string]any {
	start := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	end := time.Now().UTC().Add(8 * 24 * time.Hour).Format(time.RFC3339)
	expires := time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339)
	return map[string]any{
		"title":             "E2E test gig",
		"description_raw":   "Looking for help with plumbing",
		"description_clean": "Looking for help with plumbing",
		"duration_type":     "DAILY",
		"start_date":        start,
		"end_date":          end,
		"slots":             1,
		"expires_at":        expires,
		"lat":               41.015137,
		"lng":               28.979530,
		"city":              "Istanbul",
		"district":          "Kadikoy",
		"category_ids":      []string{},
	}
}

// createGig is a helper that creates a gig and returns its ID.
func createGig(t *testing.T, token string) string {
	t.Helper()
	resp := do(t, http.MethodPost, "/gigs", token, validGigBody())
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	id, ok := body["id"].(string)
	require.True(t, ok, "response must contain id field")
	return id
}

// ---------------------------------------------------------------------------
// POST /gigs
// ---------------------------------------------------------------------------

func TestE2E_CreateGig_ValidInput_Returns201(t *testing.T) {
	token := requireJWT(t)

	resp := do(t, http.MethodPost, "/gigs", token, validGigBody())
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.NotEmpty(t, body["id"])
	assert.Equal(t, "OPEN", body["status"])
}

func TestE2E_CreateGig_NoAuth_Returns401(t *testing.T) {
	resp := do(t, http.MethodPost, "/gigs", "", validGigBody())
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_CreateGig_MissingTitle_Returns422(t *testing.T) {
	token := requireJWT(t)

	body := validGigBody()
	body["title"] = ""

	resp := do(t, http.MethodPost, "/gigs", token, body)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestE2E_CreateGig_InvalidDurationType_Returns422(t *testing.T) {
	token := requireJWT(t)

	body := validGigBody()
	body["duration_type"] = "HOURLY"

	resp := do(t, http.MethodPost, "/gigs", token, body)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestE2E_CreateGig_MalformedJSON_Returns400(t *testing.T) {
	token := requireJWT(t)

	req, err := http.NewRequest(http.MethodPost, *e2eBaseURL+"/gigs", strings.NewReader("{bad"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// GET /gigs/{id}
// ---------------------------------------------------------------------------

func TestE2E_GetGig_ExistingGig_Returns200(t *testing.T) {
	token := requireJWT(t)
	gigID := createGig(t, token)

	resp := do(t, http.MethodGet, "/gigs/"+gigID, "", nil)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, gigID, body["id"])
}

func TestE2E_GetGig_NonExistent_Returns404(t *testing.T) {
	resp := do(t, http.MethodGet, "/gigs/00000000-0000-0000-0000-000000000000", "", nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_GetGig_InvalidID_Returns400(t *testing.T) {
	resp := do(t, http.MethodGet, "/gigs/not-a-uuid", "", nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// GET /gigs (feed)
// ---------------------------------------------------------------------------

func TestE2E_Feed_WithLatLng_Returns200(t *testing.T) {
	resp := do(t, http.MethodGet, "/gigs?lat=41.015137&lng=28.979530", "", nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestE2E_Feed_MissingLat_Returns400(t *testing.T) {
	resp := do(t, http.MethodGet, "/gigs?lng=28.979530", "", nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_Feed_MissingLng_Returns400(t *testing.T) {
	resp := do(t, http.MethodGet, "/gigs?lat=41.015137", "", nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_Feed_CreatedGigAppearsInFeed(t *testing.T) {
	token := requireJWT(t)
	gigID := createGig(t, token)

	resp := do(t, http.MethodGet, "/gigs?lat=41.015137&lng=28.979530&radius=5000", "", nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var results []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&results))

	found := false
	for _, r := range results {
		if r["id"] == gigID {
			found = true
			break
		}
	}
	assert.True(t, found, "newly created gig must appear in feed")
}

// ---------------------------------------------------------------------------
// PUT /gigs/{id}
// ---------------------------------------------------------------------------

func TestE2E_EditGig_ValidInput_Returns200(t *testing.T) {
	token := requireJWT(t)
	gigID := createGig(t, token)

	resp := do(t, http.MethodPut, "/gigs/"+gigID, token, map[string]any{
		"title": "Updated title",
	})
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "Updated title", body["title"])
}

func TestE2E_EditGig_NoAuth_Returns401(t *testing.T) {
	token := requireJWT(t)
	gigID := createGig(t, token)

	resp := do(t, http.MethodPut, "/gigs/"+gigID, "", map[string]any{"title": "x"})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_EditGig_NonExistent_Returns404(t *testing.T) {
	token := requireJWT(t)

	resp := do(t, http.MethodPut, "/gigs/00000000-0000-0000-0000-000000000000", token, map[string]any{
		"title": "Ghost",
	})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// DELETE /gigs/{id}
// ---------------------------------------------------------------------------

func TestE2E_CancelGig_OpenGig_Returns204(t *testing.T) {
	token := requireJWT(t)
	gigID := createGig(t, token)

	resp := do(t, http.MethodDelete, "/gigs/"+gigID, token, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestE2E_CancelGig_NoAuth_Returns401(t *testing.T) {
	token := requireJWT(t)
	gigID := createGig(t, token)

	resp := do(t, http.MethodDelete, "/gigs/"+gigID, "", nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_CancelGig_AlreadyCancelled_Returns409(t *testing.T) {
	token := requireJWT(t)
	gigID := createGig(t, token)

	// First cancel
	resp := do(t, http.MethodDelete, "/gigs/"+gigID, token, nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Second cancel
	resp2 := do(t, http.MethodDelete, "/gigs/"+gigID, token, nil)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusConflict, resp2.StatusCode)
}

func TestE2E_CancelGig_NonExistent_Returns404(t *testing.T) {
	token := requireJWT(t)

	resp := do(t, http.MethodDelete, "/gigs/00000000-0000-0000-0000-000000000000", token, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_Feed_ResponseContentType_IsJSON(t *testing.T) {
	resp := do(t, http.MethodGet, "/gigs?lat=41.015137&lng=28.979530", "", nil)
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	assert.True(t,
		strings.HasPrefix(ct, "application/json"),
		"Content-Type must be application/json, got: %s", ct,
	)
}

func TestE2E_CreateGig_ResponseShape(t *testing.T) {
	token := requireJWT(t)

	resp := do(t, http.MethodPost, "/gigs", token, validGigBody())
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	// Verify top-level fields present
	for _, field := range []string{"id", "poster_id", "title", "status", "duration_type", "slots", "created_at"} {
		assert.Contains(t, body, field, "response must contain field: %s", field)
	}

	// Location must be nested
	loc, ok := body["location"].(map[string]any)
	require.True(t, ok, "location must be a nested object")
	assert.NotEmpty(t, loc["city"])

	// Categories must be present (even if empty)
	assert.Contains(t, body, "categories")
}

// ---------------------------------------------------------------------------
// e2e-only: verify feed excludes cancelled gigs
// ---------------------------------------------------------------------------

func TestE2E_Feed_CancelledGigNotInFeed(t *testing.T) {
	token := requireJWT(t)
	gigID := createGig(t, token)

	// Cancel it
	resp := do(t, http.MethodDelete, "/gigs/"+gigID, token, nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Should not appear in feed
	feedResp := do(t, http.MethodGet, fmt.Sprintf("/gigs?lat=41.015137&lng=28.979530&radius=5000"), "", nil)
	defer feedResp.Body.Close()
	require.Equal(t, http.StatusOK, feedResp.StatusCode)

	var results []map[string]any
	require.NoError(t, json.NewDecoder(feedResp.Body).Decode(&results))

	for _, r := range results {
		assert.NotEqual(t, gigID, r["id"], "cancelled gig must not appear in feed")
	}
}
