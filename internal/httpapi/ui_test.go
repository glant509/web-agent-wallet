package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRootServesMobileUI(t *testing.T) {
	handler := New(nil, nil)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("unexpected content type: %q", got)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "web3-service-agent") {
		t.Fatalf("unexpected body: %q", body)
	}
	if !strings.Contains(body, "tabbar") {
		t.Fatalf("expected mobile UI markup in body")
	}
	for _, marker := range []string{`data-tab="home"`, `data-tab="market"`, `data-tab="trade"`, `data-tab="pay"`, `data-tab="asset"`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected %s in body", marker)
		}
	}
}
