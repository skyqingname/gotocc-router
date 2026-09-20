//go:build e2e

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// These tests require an isolated instance with registration enabled and email
// verification disabled. Once BASE_URL is supplied, API failures fail the test;
// missing routes, HTML fallbacks and login failures must never count as skips.
func requireUserFlowInstance(t *testing.T) {
	t.Helper()
	if strings.TrimSpace(os.Getenv("BASE_URL")) == "" {
		t.Skip("set BASE_URL to an isolated instance to run user lifecycle tests")
	}
}

type userFlowAuth struct {
	AccessToken string `json:"access_token"`
	User        struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

func createUserFlowUser(t *testing.T) (userFlowAuth, string) {
	t.Helper()
	email := fmt.Sprintf("e2e-%d@test.local", time.Now().UnixNano())
	password := "E2eTest@12345"
	data := userFlowAPI(t, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": email, "password": password,
	}, "", http.StatusOK)
	var auth userFlowAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		t.Fatalf("decode registration data: %v", err)
	}
	if auth.AccessToken == "" || auth.User.ID <= 0 || auth.User.Email != email {
		t.Fatal("registration must return a token and the registered user")
	}
	return auth, password
}

func TestUserRegistrationAndLogin(t *testing.T) {
	requireUserFlowInstance(t)
	registered, password := createUserFlowUser(t)
	data := userFlowAPI(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": registered.User.Email, "password": password,
	}, "", http.StatusOK)
	var loggedIn userFlowAuth
	if err := json.Unmarshal(data, &loggedIn); err != nil {
		t.Fatalf("decode login data: %v", err)
	}
	if loggedIn.AccessToken == "" || loggedIn.User.ID != registered.User.ID {
		t.Fatal("login must return a token for the same user")
	}
	data = userFlowAPI(t, http.MethodGet, "/api/v1/auth/me", nil, loggedIn.AccessToken, http.StatusOK)
	var profile struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(data, &profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	if profile.ID != registered.User.ID || profile.Email != registered.User.Email {
		t.Fatal("profile belongs to a different user")
	}
}

func TestAPIKeyLifecycle(t *testing.T) {
	requireUserFlowInstance(t)
	auth, _ := createUserFlowUser(t)
	data := userFlowAPI(t, http.MethodPost, "/api/v1/keys", map[string]any{
		"name": "e2e-smart-key", "routing_mode": "auto", "group_id": nil,
	}, auth.AccessToken, http.StatusOK)
	var key struct {
		ID          int64  `json:"id"`
		Key         string `json:"key"`
		RoutingMode string `json:"routing_mode"`
	}
	if err := json.Unmarshal(data, &key); err != nil {
		t.Fatalf("decode created key: %v", err)
	}
	if key.ID <= 0 || key.Key == "" || key.RoutingMode != "auto" {
		t.Fatal("key creation must return a persisted smart key")
	}
	path := fmt.Sprintf("/api/v1/keys/%d", key.ID)
	deleted := false
	t.Cleanup(func() {
		if !deleted {
			userFlowAPI(t, http.MethodDelete, path, nil, auth.AccessToken, http.StatusOK)
		}
	})
	data = userFlowAPI(t, http.MethodPut, path, map[string]string{"name": "e2e-renamed-key"}, auth.AccessToken, http.StatusOK)
	var updated struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &updated); err != nil {
		t.Fatalf("decode updated key: %v", err)
	}
	if updated.Name != "e2e-renamed-key" {
		t.Fatal("key name was not updated")
	}
	other, _ := createUserFlowUser(t)
	userFlowAPI(t, http.MethodGet, path, nil, other.AccessToken, http.StatusNotFound)

	// The public model catalog proves key authentication without issuing a paid
	// provider request. A fresh user's zero balance must not prevent listing it.
	resp, err := doRequest(t, http.MethodGet, "/v1/models", nil, key.Key)
	if err != nil {
		t.Fatalf("model catalog request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("model catalog status = %d, want 200", resp.StatusCode)
	}
	var catalog struct {
		Object string            `json:"object"`
		Data   []json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		t.Fatalf("decode model catalog: %v", err)
	}
	if catalog.Object != "list" {
		t.Fatal("gateway must return a model list")
	}
	userFlowAPI(t, http.MethodGet, "/api/v1/usage/dashboard/stats", nil, auth.AccessToken, http.StatusOK)
	userFlowAPI(t, http.MethodDelete, path, nil, auth.AccessToken, http.StatusOK)
	deleted = true
	userFlowAPI(t, http.MethodGet, path, nil, auth.AccessToken, http.StatusNotFound)
}

func userFlowAPI(t *testing.T, method, path string, payload any, token string, wantStatus int) json.RawMessage {
	t.Helper()
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode request: %v", err)
		}
	}
	resp, err := doRequest(t, method, path, body, token)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s: status %d, want %d", method, path, resp.StatusCode, wantStatus)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		t.Fatalf("%s %s returned non-JSON content", method, path)
	}
	var envelope struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode %s %s: %v", method, path, err)
	}
	if wantStatus == http.StatusOK && envelope.Code != 0 {
		t.Fatalf("%s %s returned application error %d", method, path, envelope.Code)
	}
	return envelope.Data
}

func doRequest(t *testing.T, method, path string, body []byte, token string) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequest(method, baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return (&http.Client{Timeout: 30 * time.Second}).Do(req)
}
