package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAIChatDisabledRejectsRequests(t *testing.T) {
	handler := New(nil, nil, false)

	configRequest := httptest.NewRequest(http.MethodGet, "/v1/config", nil)
	configResponse := httptest.NewRecorder()
	handler.ServeHTTP(configResponse, configRequest)
	if configResponse.Code != http.StatusOK || !strings.Contains(configResponse.Body.String(), `"ai_enabled":false`) {
		t.Fatalf("unexpected public config response: %d %s", configResponse.Code, configResponse.Body.String())
	}
	if configResponse.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("public config must not be cached")
	}

	for _, route := range []string{"/v1/sessions", "/v1/agent/runs", "/v1/agent/runs/stream"} {
		request := httptest.NewRequest(http.MethodPost, route, strings.NewReader(`{}`))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Fatalf("expected %s to be forbidden, got %d", route, response.Code)
		}
	}
}

func TestAIChatEnabledIsPublished(t *testing.T) {
	handler := New(nil, nil, true)
	request := httptest.NewRequest(http.MethodGet, "/v1/config", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"ai_enabled":true`) {
		t.Fatalf("unexpected public config response: %d %s", response.Code, response.Body.String())
	}
}
