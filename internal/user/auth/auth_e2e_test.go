//go:build e2e

package auth_test

// E2E tests for POST /auth/google
//
// These tests require a real running server and a valid Google ID token.
// They are gated behind the "e2e" build tag so they never run in CI
// unless you explicitly opt in:
//
//   go test -tags=e2e -v ./internal/auth/... \
//     -e2e-base-url=http://localhost:8080 \
//     -e2e-google-token=eyJhbGci...
//
// Obtain a short-lived Google ID token for your CLIENT_ID by:
//   1. Using Google OAuth Playground: https://developers.google.com/oauthplayground
//   2. Or the gcloud CLI: gcloud auth print-identity-token
//
// The token expires after ~1 hour, so these tests are meant to be run
// on-demand (pre-release smoke test, local dev verification) not in
// automated CI pipelines.

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
	e2eBaseURL     = flag.String("e2e-base-url", "http://localhost:8080", "Base URL of the running server")
	e2eGoogleToken = flag.String("e2e-google-token", "", "Valid Google ID token for e2e tests")
)

type authResponse struct {
	Token string `json:"token"`
	User  struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"user"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func postGoogleLogin(t *testing.T, body map[string]any) *http.Response {
	t.Helper()

	b, err := json.Marshal(body)
	require.NoError(t, err)

	url := fmt.Sprintf("%s/auth/google", *e2eBaseURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

func requireGoogleToken(t *testing.T) string {
	t.Helper()
	if *e2eGoogleToken == "" {
		t.Skip("skipping e2e test: -e2e-google-token not provided")
	}
	return *e2eGoogleToken
}

func TestE2E_GoogleLogin_ValidToken_Returns200WithTokenAndUser(t *testing.T) {
	idToken := requireGoogleToken(t)

	resp := postGoogleLogin(t, map[string]any{"id_token": idToken})
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body authResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	assert.NotEmpty(t, body.Token, "JWT must be present")
	assert.NotEmpty(t, body.User.ID, "user.id must be a UUID")
	assert.NotEmpty(t, body.User.Email, "user.email must be set")
	assert.True(t,
		strings.Contains(body.User.Email, "@"),
		"user.email must look like an email, got: %s", body.User.Email,
	)

	// Basic JWT shape check — three base64 segments separated by dots.
	parts := strings.Split(body.Token, ".")
	assert.Len(t, parts, 3, "token must be a well-formed JWT (header.payload.sig)")
}

// TestE2E_GoogleLogin_SameToken_IdempotentLogin verifies that a second call
// with the same Google identity returns the same user ID (login, not re-register).
func TestE2E_GoogleLogin_SameToken_IdempotentLogin(t *testing.T) {
	idToken := requireGoogleToken(t)

	resp1 := postGoogleLogin(t, map[string]any{"id_token": idToken})
	defer resp1.Body.Close()
	require.Equal(t, http.StatusOK, resp1.StatusCode)

	var first authResponse
	require.NoError(t, json.NewDecoder(resp1.Body).Decode(&first))

	resp2 := postGoogleLogin(t, map[string]any{"id_token": idToken})
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	var second authResponse
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&second))

	assert.Equal(t, first.User.ID, second.User.ID,
		"repeated login must resolve to the same user, not create a new one")
}

// TestE2E_GoogleLogin_MissingToken_Returns400 covers the handler-level guard.
func TestE2E_GoogleLogin_MissingToken_Returns400(t *testing.T) {
	resp := postGoogleLogin(t, map[string]any{})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var body errorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.NotEmpty(t, body.Error)
}

// TestE2E_GoogleLogin_EmptyToken_Returns400 covers the explicit empty-string guard.
func TestE2E_GoogleLogin_EmptyToken_Returns400(t *testing.T) {
	resp := postGoogleLogin(t, map[string]any{"id_token": ""})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestE2E_GoogleLogin_InvalidToken_Returns401 sends a syntactically valid
// JWT-shaped string that won't pass Google's OIDC verification.
func TestE2E_GoogleLogin_InvalidToken_Returns401(t *testing.T) {
	fake := "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJmYWtlIn0.invalidsignature"
	resp := postGoogleLogin(t, map[string]any{"id_token": fake})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var body errorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.NotEmpty(t, body.Error)
}

// TestE2E_GoogleLogin_MalformedJSON_Returns400 exercises the JSON decode guard.
func TestE2E_GoogleLogin_MalformedJSON_Returns400(t *testing.T) {
	url := fmt.Sprintf("%s/auth/google", *e2eBaseURL)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader("{bad json"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestE2E_GoogleLogin_ResponseContentType_IsJSON verifies the server sets the
// correct Content-Type so clients can safely call json.Decode.
func TestE2E_GoogleLogin_ResponseContentType_IsJSON(t *testing.T) {
	idToken := requireGoogleToken(t)

	resp := postGoogleLogin(t, map[string]any{"id_token": idToken})
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	assert.True(t,
		strings.HasPrefix(ct, "application/json"),
		"Content-Type must be application/json, got: %s", ct,
	)
}
