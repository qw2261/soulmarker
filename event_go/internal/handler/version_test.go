package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qw2261/soulmarker/event_go/internal/config"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func TestVersionHandlerReturnsBuildInfo(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	h := newTestHandler(s, config.Load())

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()
	h.VersionHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", contentType)
	}

	var apiResp model.APIResp
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}

	data, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be object, got %T", apiResp.Data)
	}
	for _, field := range []string{"version", "commit", "build_time", "go_version"} {
		if _, exists := data[field]; !exists {
			t.Errorf("expected field %q in version response", field)
		}
	}
	if version, _ := data["version"].(string); version == "" {
		t.Errorf("expected non-empty version, got %q", version)
	}
	if commit, _ := data["commit"].(string); commit == "" {
		t.Errorf("expected non-empty commit, got %q", commit)
	}
}

func TestVersionRouteExposedByRouter(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	h := newTestHandler(s, config.Load())
	router := NewRouter(h, nil)

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 via router, got %d", resp.StatusCode)
	}

	var apiResp model.APIResp
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}
	data, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be object, got %T", apiResp.Data)
	}
	if _, exists := data["dependencies"]; !exists {
		t.Errorf("expected dependencies field in version response")
	}
}
