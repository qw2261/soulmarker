package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRegistryObserveAggregatesByMethodAndStatus(t *testing.T) {
	reg := NewRegistry()
	reg.Observe("GET", 200, 10*time.Millisecond)
	reg.Observe("GET", 200, 20*time.Millisecond)
	reg.Observe("GET", 404, 5*time.Millisecond)
	reg.Observe("POST", 201, 30*time.Millisecond)

	reg.mu.Lock()
	if got := reg.requestsTotal["GET"][200]; got != 2 {
		t.Errorf("expected GET/200 count=2, got %d", got)
	}
	if got := reg.requestsTotal["GET"][404]; got != 1 {
		t.Errorf("expected GET/404 count=1, got %d", got)
	}
	if got := reg.requestsTotal["POST"][201]; got != 1 {
		t.Errorf("expected POST/201 count=1, got %d", got)
	}
	if got := reg.durations["GET"].count; got != 3 {
		t.Errorf("expected GET histogram count=3, got %d", got)
	}
	if got := reg.durations["POST"].count; got != 1 {
		t.Errorf("expected POST histogram count=1, got %d", got)
	}
	reg.mu.Unlock()
}

func TestRegistryObserveBucketAccumulation(t *testing.T) {
	reg := NewRegistry()
	reg.Observe("GET", 200, 5*time.Millisecond)   // <= 0.005
	reg.Observe("GET", 200, 100*time.Millisecond) // <= 0.1
	reg.Observe("GET", 200, 20*time.Second)       // overflow

	reg.mu.Lock()
	h := reg.durations["GET"]
	// non-cumulative counts: bucket0 (<=0.005)=1, bucket for 0.1=1, overflow=1
	var cumulative uint64
	cumulative += h.counts[0]
	if cumulative != 1 {
		t.Errorf("expected le=0.005 cumulative=1, got %d", cumulative)
	}
	// le=0.1 should include 0.005 and 0.1 observations
	for i, bound := range h.bounds {
		if bound == 0.1 {
			cumulative += h.counts[i]
			break
		}
	}
	if cumulative != 2 {
		t.Errorf("expected le=0.1 cumulative=2, got %d", cumulative)
	}
	reg.mu.Unlock()
}

func TestRegistryHandlerRendersPrometheusText(t *testing.T) {
	reg := NewRegistry()
	reg.SetBuild("v6.2.0", "8e226fb")
	reg.IncInFlight()
	reg.Observe("GET", 200, 10*time.Millisecond)
	reg.Observe("POST", 201, 50*time.Millisecond)
	reg.DecInFlight()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	reg.Handler().ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/plain; version=0.0.4; charset=utf-8" {
		t.Errorf("unexpected content type: %s", ct)
	}

	body := w.Body.String()
	for _, expected := range []string{
		"# TYPE soulmark_http_requests_total counter",
		`soulmark_http_requests_total{method="GET",status="200"} 1`,
		`soulmark_http_requests_total{method="POST",status="201"} 1`,
		"# TYPE soulmark_http_requests_in_flight gauge",
		"soulmark_http_requests_in_flight 0",
		"# TYPE soulmark_http_request_duration_seconds histogram",
		`soulmark_http_request_duration_seconds_bucket{method="GET",le="+Inf"} 1`,
		`soulmark_http_request_duration_seconds_bucket{method="POST",le="+Inf"} 1`,
		`soulmark_http_request_duration_seconds_count{method="GET"} 1`,
		`soulmark_build_info{version="v6.2.0",commit="8e226fb"} 1`,
		"# TYPE soulmark_uptime_seconds gauge",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("metrics output missing %q\n---got---\n%s", expected, body)
		}
	}
}

func TestRegistryHandlerEmptyRegistry(t *testing.T) {
	reg := NewRegistry()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	reg.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "soulmark_http_requests_total") {
		t.Errorf("expected metric family present, got:\n%s", body)
	}
	if strings.Contains(body, "soulmark_http_request_duration_seconds_count{method=") {
		t.Errorf("empty registry should not render any histogram count, got:\n%s", body)
	}
}
