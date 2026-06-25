//go:build e2e

package location_test

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

var (
	e2eBaseURL = flag.String("e2e-base-url", "http://localhost:8080", "Base URL of the running server")
	e2eJWT     = flag.String("e2e-jwt", "", "Valid JWT for an existing user")
	e2eUserID  = flag.String("e2e-user-id", "", "UUID of the user the JWT belongs to")
)

type locationResponse struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	IsFlagged bool    `json:"is_flagged"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func putLocation(t *testing.T, token string, body map[string]any) *http.Response {
	t.Helper()

	b, err := json.Marshal(body)
	require.NoError(t, err)

	url := fmt.Sprintf("%s/users/me/location", *e2eBaseURL)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
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

func TestE2E_UpsertLocation_ValidRequest_Returns200(t *testing.T) {
	token := requireJWT(t)

	resp := putLocation(t, token, map[string]any{
		"lat": 41.015137,
		"lng": 28.979530,
	})
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body locationResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	assert.NotEmpty(t, body.ID)
	assert.NotEmpty(t, body.UserID)
	assert.InDelta(t, 41.015137, body.Lat, 0.0001)
	assert.InDelta(t, 28.979530, body.Lng, 0.0001)
}

func TestE2E_UpsertLocation_CalledTwice_SecondCallUpdates(t *testing.T) {
	token := requireJWT(t)
	ctx := map[string]any{"lat": 41.015137, "lng": 28.979530}

	resp1 := putLocation(t, token, ctx)
	defer resp1.Body.Close()
	require.Equal(t, http.StatusOK, resp1.StatusCode)

	resp2 := putLocation(t, token, map[string]any{
		"lat": 39.925533,
		"lng": 32.866287, // Ankara
	})
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	var body locationResponse
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&body))

	assert.InDelta(t, 39.925533, body.Lat, 0.0001)
	assert.InDelta(t, 32.866287, body.Lng, 0.0001)
}

func TestE2E_UpsertLocation_MissingAuthHeader_Returns401(t *testing.T) {
	resp := putLocation(t, "", map[string]any{"lat": 41.0, "lng": 28.0})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_UpsertLocation_InvalidToken_Returns401(t *testing.T) {
	resp := putLocation(t, "not.a.real.token", map[string]any{"lat": 41.0, "lng": 28.0})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_UpsertLocation_InvalidLatitude_Returns400(t *testing.T) {
	token := requireJWT(t)

	resp := putLocation(t, token, map[string]any{"lat": 91.0, "lng": 28.0})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var body errorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.NotEmpty(t, body.Error)
}

func TestE2E_UpsertLocation_InvalidLongitude_Returns400(t *testing.T) {
	token := requireJWT(t)

	resp := putLocation(t, token, map[string]any{"lat": 41.0, "lng": 181.0})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_UpsertLocation_MalformedJSON_Returns400(t *testing.T) {
	token := requireJWT(t)

	url := fmt.Sprintf("%s/users/me/location", *e2eBaseURL)
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader("{bad json"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_UpsertLocation_ResponseContentType_IsJSON(t *testing.T) {
	token := requireJWT(t)

	resp := putLocation(t, token, map[string]any{"lat": 41.0, "lng": 28.0})
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	assert.True(t,
		strings.HasPrefix(ct, "application/json"),
		"Content-Type must be application/json, got: %s", ct,
	)
}
