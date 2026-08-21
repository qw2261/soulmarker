package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/config"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func TestRateLimiterWindowReset(t *testing.T) {
	now := time.Unix(1000, 0)
	rl := &rateLimiter{
		limit:   1,
		window:  time.Minute,
		buckets: make(map[string]*rateLimitBucket),
		now:     func() time.Time { return now },
	}
	if !rl.allow("a") {
		t.Fatal("first request in a fresh window should be allowed")
	}
	if rl.allow("a") {
		t.Fatal("second request within the same window should be blocked")
	}
	now = now.Add(time.Minute + time.Second)
	if !rl.allow("a") {
		t.Fatal("request after the window rolls over should be allowed again")
	}
}

func TestRateLimiterNoLimitWhenNonPositive(t *testing.T) {
	for _, limit := range []int{0, -1, -100} {
		rl := newRateLimiter(limit)
		for i := 0; i < 1000; i++ {
			if !rl.allow("a") {
				t.Fatalf("limit %d should not block any request", limit)
			}
		}
	}
}

func TestClientIPUsesForwardedForFirst(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.2")
	if got := clientIP(req); got != "203.0.113.9" {
		t.Fatalf("expected first forwarded IP, got %q", got)
	}
}

func TestClientIPFallsBackToRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	req.RemoteAddr = "10.0.0.7:8080"
	if got := clientIP(req); got != "10.0.0.7" {
		t.Fatalf("expected RemoteAddr host, got %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/events", nil)
	req.RemoteAddr = "no-port"
	if got := clientIP(req); got != "no-port" {
		t.Fatalf("expected raw RemoteAddr when no port, got %q", got)
	}
}

func TestIsHealthPath(t *testing.T) {
	for _, path := range []string{"/health", "/healthz", "/readyz"} {
		if !isHealthPath(path) {
			t.Errorf("expected %q to be a health path", path)
		}
	}
	for _, path := range []string{"/api/events", "/", "/healthz/extra"} {
		if isHealthPath(path) {
			t.Errorf("did not expect %q to be a health path", path)
		}
	}
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestRateLimitMiddlewareBlocksOverLimit(t *testing.T) {
	next := http.HandlerFunc(okHandler)
	mw := RateLimitMiddleware(next, 2)

	req := func() *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/api/events", nil)
		r.Header.Set("X-Forwarded-For", "203.0.113.10")
		return r
	}

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		mw.ServeHTTP(w, req())
		if w.Code != http.StatusOK {
			t.Fatalf("request %d within limit should be 200, got %d", i+1, w.Code)
		}
	}

	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req())
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("request over limit should be 429, got %d", w.Code)
	}
	var resp model.APIResp
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ErrorCode != string(api.CodeRateLimited) {
		t.Fatalf("expected error code %s, got %q", api.CodeRateLimited, resp.ErrorCode)
	}
}

func TestRateLimitMiddlewareSeparateIPsAreIndependent(t *testing.T) {
	next := http.HandlerFunc(okHandler)
	mw := RateLimitMiddleware(next, 2)

	do := func(ip string) int {
		r := httptest.NewRequest(http.MethodGet, "/api/events", nil)
		r.Header.Set("X-Forwarded-For", ip)
		w := httptest.NewRecorder()
		mw.ServeHTTP(w, r)
		return w.Code
	}

	if got := do("203.0.113.10"); got != http.StatusOK {
		t.Fatalf("IP A first request: got %d, want 200", got)
	}
	if got := do("203.0.113.10"); got != http.StatusOK {
		t.Fatalf("IP A second request: got %d, want 200", got)
	}
	if got := do("203.0.113.11"); got != http.StatusOK {
		t.Fatalf("IP B request should be independent: got %d, want 200", got)
	}
	if got := do("203.0.113.11"); got != http.StatusOK {
		t.Fatalf("IP B second request: got %d, want 200", got)
	}
	if got := do("203.0.113.10"); got != http.StatusTooManyRequests {
		t.Fatalf("IP A third request should be 429, got %d", got)
	}
}

func TestRateLimitMiddlewareNoLimitWhenNonPositive(t *testing.T) {
	for _, limit := range []int{0, -10} {
		mw := RateLimitMiddleware(http.HandlerFunc(okHandler), limit)
		for i := 0; i < 5; i++ {
			r := httptest.NewRequest(http.MethodGet, "/api/events", nil)
			r.Header.Set("X-Forwarded-For", "203.0.113.10")
			w := httptest.NewRecorder()
			mw.ServeHTTP(w, r)
			if w.Code != http.StatusOK {
				t.Fatalf("limit %d request %d should pass, got %d", limit, i+1, w.Code)
			}
		}
	}
}

func TestRateLimitMiddlewareExemptsHealthAndOptions(t *testing.T) {
	next := http.HandlerFunc(okHandler)
	mw := RateLimitMiddleware(next, 1)

	health := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, health)
	if w.Code != http.StatusOK {
		t.Fatalf("health probe should bypass rate limit, got %d", w.Code)
	}

	for i := 0; i < 3; i++ {
		opts := httptest.NewRequest(http.MethodOptions, "/api/events", nil)
		ow := httptest.NewRecorder()
		mw.ServeHTTP(ow, opts)
		if ow.Code != http.StatusOK {
			t.Fatalf("CORS preflight should bypass rate limit, got %d", ow.Code)
		}
	}
}

func TestRateLimitMiddlewareIntegratedWithRouter(t *testing.T) {
	s := mustNewStore(t)
	t.Cleanup(func() { s.Close() })
	cfg := config.Load()
	cfg.RateLimitPerMinute = 2
	h := newTestHandler(s, cfg)
	srv := httptest.NewServer(NewRouter(h, nil))
	t.Cleanup(srv.Close)

	req := func(path string, ip string) *http.Response {
		r, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		r.Header.Set("X-Forwarded-For", ip)
		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		return resp
	}
	closeResp := func(resp *http.Response) {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			var body model.APIResp
			_ = json.NewDecoder(resp.Body).Decode(&body)
			t.Fatalf("unexpected status %d, error_code=%q", resp.StatusCode, body.ErrorCode)
		}
	}

	// 同一 IP 前三/二次应正常，第三个超出窗口上限返回 429。
	closeResp(req("/api/events", "198.51.100.5"))
	closeResp(req("/api/events", "198.51.100.5"))
	third := req("/api/events", "198.51.100.5")
	defer third.Body.Close()
	if third.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("third request should be 429, got %d", third.StatusCode)
	}
	var limited model.APIResp
	if err := json.NewDecoder(third.Body).Decode(&limited); err != nil {
		t.Fatalf("decode third response: %v", err)
	}
	if limited.ErrorCode != string(api.CodeRateLimited) {
		t.Fatalf("expected error code %s, got %q", api.CodeRateLimited, limited.ErrorCode)
	}

	// 不同 IP 不受影响。
	closeResp(req("/api/events", "198.51.100.6"))
	closeResp(req("/api/events", "198.51.100.6"))

	// 探针豁免：即使被限流也返回 200。
	closeResp(req("/health", "198.51.100.5"))
	closeResp(req("/readyz", "198.51.100.5"))
	closeResp(req("/healthz", "198.51.100.5"))
}

func TestRateLimitMiddlewareDoesNotSetErrorCodeForClientWithoutBody(t *testing.T) {
	next := http.HandlerFunc(okHandler)
	mw := RateLimitMiddleware(next, 0)
	r := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.99")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
