package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSAllowsPackagedClientsForLoopbackDevelopment(t *testing.T) {
	testCases := []string{
		"chrome-extension://abcdefghijklmnopabcdefghijklmnop",
		"capacitor://localhost",
		"https://localhost",
		"http://localhost:4312",
		"http://127.0.0.1:5173",
	}
	for _, origin := range testCases {
		t.Run(origin, func(t *testing.T) {
			handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			request := httptest.NewRequest(http.MethodOptions, "http://localhost:8080/v1/asset/portfolio", nil)
			request.Header.Set("Origin", origin)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusNoContent {
				t.Fatalf("expected preflight 204, got %d", recorder.Code)
			}
			if recorder.Header().Get("Access-Control-Allow-Origin") != origin {
				t.Fatalf("expected origin %q, got %q", origin, recorder.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestCORSAllowsAndroidEmulatorDevelopmentOrigin(t *testing.T) {
	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	request := httptest.NewRequest(http.MethodOptions, "http://10.0.2.2:8080/v1/asset/portfolio", nil)
	request.Host = "10.0.2.2:8080"
	request.Header.Set("Origin", "https://localhost")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
}

func TestCORSAllowsLocalFileOriginForLoopbackDevelopment(t *testing.T) {
	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	request := httptest.NewRequest(http.MethodOptions, "http://localhost:8081/v1/wallet/evm/history", nil)
	request.Header.Set("Origin", "null")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected file-origin preflight 204, got %d", recorder.Code)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "null" {
		t.Fatalf("expected null origin, got %q", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSRejectsUnconfiguredOriginForNonLoopbackAPI(t *testing.T) {
	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	request := httptest.NewRequest(http.MethodPost, "https://api.example.com/v1/asset/portfolio", nil)
	request.Host = "api.example.com"
	request.Header.Set("Origin", "chrome-extension://abcdefghijklmnopabcdefghijklmnop")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestCORSAllowsExplicitOrigin(t *testing.T) {
	t.Setenv(allowedOriginsEnvironment, "https://wallet.example.com, chrome-extension://walletid")
	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	request := httptest.NewRequest(http.MethodPost, "https://api.example.com/v1/agent/runs", nil)
	request.Host = "api.example.com"
	request.Header.Set("Origin", "chrome-extension://walletid")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}
