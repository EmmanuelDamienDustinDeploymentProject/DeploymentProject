package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// helper to set env (silences any linter complaining about unhandled error even though Setenv returns no error)
func setEnv(k, v string) { _ = os.Setenv(k, v) }
func unsetEnv(k string)  { _ = os.Unsetenv(k) }

func TestValidateBearerTokenWithValidToken(t *testing.T) {
	setEnv("GITHUB_CLIENT_ID", "test_client_id")
	setEnv("GITHUB_CLIENT_SECRET", "test_client_secret")
	InitOAuth()

	token := "valid_test_token"
	validTokens.Add(token, &TokenInfo{
		AccessToken: token,
		Username:    "testuser",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	})

	err := ValidateBearerToken(token)
	if err != nil {
		t.Errorf("Expected valid token to pass validation, got error: %v", err)
	}
}

func TestValidateBearerTokenWithExpiredToken(t *testing.T) {
	setEnv("GITHUB_CLIENT_ID", "test_client_id")
	setEnv("GITHUB_CLIENT_SECRET", "test_client_secret")
	InitOAuth()

	token := "expired_test_token"
	validTokens.Add(token, &TokenInfo{
		AccessToken: token,
		Username:    "testuser",
		ExpiresAt:   time.Now().Add(-1 * time.Hour),
	})

	err := ValidateBearerToken(token)
	if err == nil {
		t.Error("Expected expired token to fail validation")
	}
}

func TestValidateBearerTokenWithInvalidToken(t *testing.T) {
	setEnv("GITHUB_CLIENT_ID", "test_client_id")
	setEnv("GITHUB_CLIENT_SECRET", "test_client_secret")
	InitOAuth()

	err := ValidateBearerToken("nonexistent_token")
	if err == nil {
		t.Error("Expected invalid token to fail validation")
	}
}

func TestValidateBearerTokenWithEmptyToken(t *testing.T) {
	setEnv("GITHUB_CLIENT_ID", "test_client_id")
	setEnv("GITHUB_CLIENT_SECRET", "test_client_secret")
	InitOAuth()

	err := ValidateBearerToken("")
	if err == nil {
		t.Error("Expected empty token to fail validation")
	}
}

func TestValidateBearerTokenWithoutOAuthConfigured(t *testing.T) {
	unsetEnv("GITHUB_CLIENT_ID")
	unsetEnv("GITHUB_CLIENT_SECRET")
	oauthConfig = nil

	err := ValidateBearerToken("any_token")
	if err != nil {
		t.Errorf("Expected validation to pass when OAuth not configured (development mode), got error: %v", err)
	}
}

func TestInitOAuthWithAllEnvironmentVariables(t *testing.T) {
	setEnv("GITHUB_CLIENT_ID", "test_client_id")
	setEnv("GITHUB_CLIENT_SECRET", "test_client_secret")
	setEnv("OAUTH_REDIRECT_URL", "http://example.com/callback")

	InitOAuth()

	if oauthConfig == nil {
		TFatal(t, "Expected oauthConfig to be initialized")
	}
	if oauthConfig.ClientID != "test_client_id" {
		TErrorf(t, "Expected ClientID to be 'test_client_id', got '%s'", oauthConfig.ClientID)
	}
	if oauthConfig.ClientSecret != "test_client_secret" {
		TErrorf(t, "Expected ClientSecret to be 'test_client_secret', got '%s'", oauthConfig.ClientSecret)
	}
	if oauthConfig.RedirectURL != "http://example.com/callback" {
		TErrorf(t, "Expected RedirectURL to be 'http://example.com/callback', got '%s'", oauthConfig.RedirectURL)
	}
}

func TestInitOAuthWithDefaultRedirectURL(t *testing.T) {
	setEnv("GITHUB_CLIENT_ID", "test_client_id")
	setEnv("GITHUB_CLIENT_SECRET", "test_client_secret")
	unsetEnv("OAUTH_REDIRECT_URL")

	InitOAuth()

	if oauthConfig == nil {
		TFatal(t, "Expected oauthConfig to be initialized")
	}
	if oauthConfig.RedirectURL != "http://localhost:8080/oauth/callback" {
		TErrorf(t, "Expected default RedirectURL, got '%s'", oauthConfig.RedirectURL)
	}
}

func TestInitOAuthWithoutCredentials(t *testing.T) {
	unsetEnv("GITHUB_CLIENT_ID")
	unsetEnv("GITHUB_CLIENT_SECRET")

	oauthConfig = nil
	InitOAuth()

	if oauthConfig != nil {
		TError(t, "Expected oauthConfig to remain nil when credentials not provided")
	}
}

func TestOAuthLoginHandlerRedirectsToGitHub(t *testing.T) {
	setEnv("GITHUB_CLIENT_ID", "test_client_id")
	setEnv("GITHUB_CLIENT_SECRET", "test_client_secret")
	InitOAuth()

	req := httptest.NewRequest("GET", "/oauth/login", nil)
	w := httptest.NewRecorder()

	oauthLoginHandler(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		TErrorf(t, "Expected status %d, got %d", http.StatusTemporaryRedirect, w.Code)
	}

	location := w.Header().Get("Location")
	if !strings.Contains(location, "github.com/login/oauth/authorize") {
		TErrorf(t, "Expected redirect to GitHub, got: %s", location)
	}
	if !strings.Contains(location, "client_id=test_client_id") {
		TError(t, "Expected client_id in redirect URL")
	}
}

func TestOAuthLoginHandlerWithoutConfiguration(t *testing.T) {
	oauthConfig = nil

	req := httptest.NewRequest("GET", "/oauth/login", nil)
	w := httptest.NewRecorder()

	oauthLoginHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		TErrorf(t, "Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestOAuthCallbackHandlerWithoutCode(t *testing.T) {
	setEnv("GITHUB_CLIENT_ID", "test_client_id")
	setEnv("GITHUB_CLIENT_SECRET", "test_client_secret")
	InitOAuth()

	req := httptest.NewRequest("GET", "/oauth/callback", nil)
	w := httptest.NewRecorder()

	oauthCallbackHandler(w, req)

	if w.Code != http.StatusBadRequest {
		TErrorf(t, "Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestOAuthCallbackHandlerWithoutConfiguration(t *testing.T) {
	oauthConfig = nil

	req := httptest.NewRequest("GET", "/oauth/callback?code=test", nil)
	w := httptest.NewRecorder()

	oauthCallbackHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		TErrorf(t, "Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestGetGitHubUserWithUnauthorizedResponse(t *testing.T) {
	// direct invalid token call should error
	ctx := context.Background()
	_, err := getGitHubUser(ctx, "invalid_token")
	if err == nil {
		TError(t, "Expected error for unauthorized response or invalid token")
	}
}

func TestTokenStoreAddAndGet(t *testing.T) {
	store := &TokenStore{tokens: make(map[string]*TokenInfo)}

	token := "test_token_123"
	info := &TokenInfo{AccessToken: token, Username: "testuser", ExpiresAt: time.Now().Add(1 * time.Hour)}
	store.Add(token, info)

	retrieved, exists := store.Get(token)
	if !exists {
		TError(t, "Expected token to exist in store")
		return
	}
	if retrieved != nil && retrieved.Username != "testuser" {
		TErrorf(t, "Expected username 'testuser', got '%s'", retrieved.Username)
	}
}

func TestTokenStoreGetNonexistentToken(t *testing.T) {
	store := &TokenStore{tokens: make(map[string]*TokenInfo)}
	_, exists := store.Get("nonexistent")
	if exists {
		TError(t, "Expected token to not exist")
	}
}

func TestTokenStoreGetExpiredToken(t *testing.T) {
	store := &TokenStore{tokens: make(map[string]*TokenInfo)}
	token := "expired_token"
	info := &TokenInfo{AccessToken: token, Username: "testuser", ExpiresAt: time.Now().Add(-1 * time.Hour)}
	store.Add(token, info)
	_, exists := store.Get(token)
	if exists {
		TError(t, "Expected expired token to not be returned")
	}
}

func TestTokenStoreDelete(t *testing.T) {
	store := &TokenStore{tokens: make(map[string]*TokenInfo)}
	token := "test_token"
	info := &TokenInfo{AccessToken: token, Username: "testuser", ExpiresAt: time.Now().Add(1 * time.Hour)}
	store.Add(token, info)
	store.Delete(token)
	_, exists := store.Get(token)
	if exists {
		TError(t, "Expected token to be deleted")
	}
}

func TestCreateOAuthHTTPClientAddsAuthorizationHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test_token_123" {
			TErrorf(t, "Expected 'Bearer test_token_123', got '%s'", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := CreateOAuthHTTPClient("test_token_123")
	resp, err := client.Get(server.URL)
	if err != nil {
		TFatal(t, "Expected no error")
	}
	if resp == nil {
		TFatal(t, "Expected non-nil response")
		return
	}
	body := resp.Body
	T := t
	T.Cleanup(func() { _ = body.Close() })
	if resp.StatusCode != http.StatusOK {
		TErrorf(t, "Expected status 200, got %d", resp.StatusCode)
	}
}

func TestBearerTokenMiddlewareWithValidToken(t *testing.T) {
	setEnv("GITHUB_CLIENT_ID", "test_client_id")
	setEnv("GITHUB_CLIENT_SECRET", "test_client_secret")
	InitOAuth()

	token := "valid_middleware_token"
	validTokens.Add(token, &TokenInfo{AccessToken: token, Username: "testuser", ExpiresAt: time.Now().Add(1 * time.Hour)})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	handler := bearerTokenMiddleware(nextHandler)
	req := httptest.NewRequest("GET", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		TErrorf(t, "Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "success" {
		TErrorf(t, "Expected 'success', got '%s'", w.Body.String())
	}
}

func TestBearerTokenMiddlewareWithoutAuthorizationHeader(t *testing.T) {
	setEnv("GITHUB_CLIENT_ID", "test_client_id")
	setEnv("GITHUB_CLIENT_SECRET", "test_client_secret")
	InitOAuth()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { TError(t, "Should not reach next handler") })
	handler := bearerTokenMiddleware(nextHandler)
	req := httptest.NewRequest("GET", "/mcp", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		TErrorf(t, "Expected status 401, got %d", w.Code)
	}
}

func TestBearerTokenMiddlewareWithInvalidFormat(t *testing.T) {
	setEnv("GITHUB_CLIENT_ID", "test_client_id")
	setEnv("GITHUB_CLIENT_SECRET", "test_client_secret")
	InitOAuth()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { TError(t, "Should not reach next handler") })
	handler := bearerTokenMiddleware(nextHandler)
	req := httptest.NewRequest("GET", "/mcp", nil)
	req.Header.Set("Authorization", "InvalidFormat token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		TErrorf(t, "Expected status 401, got %d", w.Code)
	}
}

func TestBearerTokenMiddlewareWithInvalidToken(t *testing.T) {
	setEnv("GITHUB_CLIENT_ID", "test_client_id")
	setEnv("GITHUB_CLIENT_SECRET", "test_client_secret")
	InitOAuth()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { TError(t, "Should not reach next handler") })
	handler := bearerTokenMiddleware(nextHandler)
	req := httptest.NewRequest("GET", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer invalid_token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		TErrorf(t, "Expected status 401, got %d", w.Code)
	}
}

func TestHealthCheckHandlerReturnsOK(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	healthCheckHandler(w, req)
	if w.Code != http.StatusOK {
		TErrorf(t, "Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "OK" {
		TErrorf(t, "Expected 'OK', got '%s'", w.Body.String())
	}
}

func TestGetTimeWithNYC(t *testing.T) {
	params := &GetTimeParams{City: "nyc"}
	result, _, err := getTime(context.Background(), nil, params)
	if err != nil {
		TFatal(t, "Expected no error")
		return
	}
	if result == nil {
		TFatal(t, "Expected non-nil result")
		return
	}
	content := result.Content
	if len(content) == 0 {
		TFatal(t, "Expected content in result")
		return
	}
	textContent, ok := content[0].(*mcp.TextContent)
	if !ok {
		TFatal(t, "Expected TextContent")
		return
	}
	if !strings.Contains(textContent.Text, "New York City") {
		TErrorf(t, "Expected 'New York City' in response, got: %s", textContent.Text)
	}
}

func TestGetTimeWithSF(t *testing.T) {
	params := &GetTimeParams{City: "sf"}
	result, _, err := getTime(context.Background(), nil, params)
	if err != nil {
		TFatal(t, "Expected no error")
		return
	}
	if result == nil {
		TFatal(t, "Expected non-nil result")
		return
	}
	content := result.Content
	if len(content) == 0 {
		TFatal(t, "Expected content in result")
		return
	}
	textContent, ok := content[0].(*mcp.TextContent)
	if !ok {
		TFatal(t, "Expected TextContent")
		return
	}
	if !strings.Contains(textContent.Text, "San Francisco") {
		TErrorf(t, "Expected 'San Francisco' in response, got: %s", textContent.Text)
	}
}

func TestGetTimeWithBoston(t *testing.T) {
	params := &GetTimeParams{City: "boston"}
	result, _, err := getTime(context.Background(), nil, params)
	if err != nil {
		TFatal(t, "Expected no error")
		return
	}
	if result == nil {
		TFatal(t, "Expected non-nil result")
		return
	}
	content := result.Content
	if len(content) == 0 {
		TFatal(t, "Expected content in result")
		return
	}
	textContent, ok := content[0].(*mcp.TextContent)
	if !ok {
		TFatal(t, "Expected TextContent")
		return
	}
	if !strings.Contains(textContent.Text, "Boston") {
		TErrorf(t, "Expected 'Boston' in response, got: %s", textContent.Text)
	}
}

func TestGetTimeWithEmptyCity(t *testing.T) {
	params := &GetTimeParams{City: ""}
	result, _, err := getTime(context.Background(), nil, params)
	if err != nil {
		TFatal(t, "Expected no error")
		return
	}
	if result == nil {
		TFatal(t, "Expected non-nil result")
		return
	}
	content := result.Content
	if len(content) == 0 {
		TFatal(t, "Expected content in result")
		return
	}
	textContent, ok := content[0].(*mcp.TextContent)
	if !ok {
		TFatal(t, "Expected TextContent")
		return
	}
	if !strings.Contains(textContent.Text, "New York City") {
		TError(t, "Expected empty city to default to NYC")
	}
}

func TestGetTimeWithUnknownCity(t *testing.T) {
	params := &GetTimeParams{City: "unknown"}
	_, _, err := getTime(context.Background(), nil, params)
	if err == nil {
		TError(t, "Expected error for unknown city")
	} else if !strings.Contains(err.Error(), "unknown city") {
		TErrorf(t, "Expected 'unknown city' error, got: %v", err)
	}
}

func TestTokenTransportRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my_test_token" {
			TErrorf(t, "Expected 'Bearer my_test_token', got '%s'", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := &tokenTransport{token: "my_test_token", base: http.DefaultTransport}
	client := &http.Client{Transport: transport}
	resp, err := client.Get(server.URL)
	if err != nil {
		TFatal(t, "Expected no error")
	}
	if resp == nil {
		TFatal(t, "Expected non-nil response")
		return
	}
	body := resp.Body
	T := t
	T.Cleanup(func() { _ = body.Close() })
	if resp.StatusCode != http.StatusOK {
		TErrorf(t, "Expected status 200, got %d", resp.StatusCode)
	}
}

func TestGenerateStateTokenReturnsNonEmptyString(t *testing.T) {
	state := generateStateToken()
	if state == "" {
		TError(t, "Expected non-empty state token")
	}
	if len(state) < 20 {
		TError(t, "Expected state token to be reasonably long")
	}
}

func TestGenerateStateTokenReturnsUniqueTokens(t *testing.T) {
	state1 := generateStateToken()
	state2 := generateStateToken()
	if state1 == state2 {
		TError(t, "Expected unique state tokens")
	}
}

// minimal wrappers for consistency
func TError(t *testing.T, msg string)          { t.Error(msg) }
func TErrorf(t *testing.T, f string, a ...any) { t.Errorf(f, a...) }
func TFatal(t *testing.T, msg string)          { t.Fatal(msg) }
