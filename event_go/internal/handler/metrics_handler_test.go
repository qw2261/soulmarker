package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qw2261/soulmarker/event_go/internal/config"
)

func TestMetricsRouteExposedByRouter(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	h := newTestHandler(s, config.Load())
	router := NewRouter(h, nil)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 via router, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/plain; version=0.0.4; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "soulmark_http_requests_total") {
		t.Errorf("expected metrics family in response, got:\n%s", body)
	}
}

func TestMetricsMiddlewareRecordsRequests(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	h := newTestHandler(s, config.Load())
	router := NewRouter(h, nil)

	// 一次正常探针请求应经过 MetricsMiddleware 被计数。
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	reqBody := httptest.NewRecorder()
	router.ServeHTTP(reqBody, req)

	// 抓取 /metrics，应能看到 GET/200 计数与延迟直方图。
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, `soulmark_http_requests_total{method="GET",status="200"}`) {
		t.Errorf("expected recorded GET/200 request in metrics, got:\n%s", body)
	}
	if !strings.Contains(body, `soulmark_http_request_duration_seconds_count{method="GET"}`) {
		t.Errorf("expected latency histogram for GET, got:\n%s", body)
	}
}

func TestMetricsMiddlewareSkipsMetricsScrape(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	h := newTestHandler(s, config.Load())
	router := NewRouter(h, nil)

	// 只抓取 /metrics 本身（不产生其他业务请求），in-flight 应为 0。
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "soulmark_http_requests_in_flight 0") {
		t.Errorf("expected in-flight 0 when only /metrics is scraped, got:\n%s", body)
	}
}
