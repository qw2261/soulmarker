package handler

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/api"
)

// rateLimitBucket 记录一个固定窗口内的请求计数。
type rateLimitBucket struct {
	windowStart time.Time
	count       int
}

// rateLimiter 按固定窗口（每分钟）对客户端身份限流。
type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]*rateLimitBucket
	now     func() time.Time
}

func newRateLimiter(limit int) *rateLimiter {
	return &rateLimiter{
		limit:   limit,
		window:  time.Minute,
		buckets: make(map[string]*rateLimitBucket),
		now:     time.Now,
	}
}

// rateLimitPruneThreshold 控制过期桶的清理时机，避免无界内存增长。
const rateLimitPruneThreshold = 10000

// allow 判断指定身份是否可在当前窗口内继续请求。limit 非正数时表示不限制。
func (rl *rateLimiter) allow(key string) bool {
	if rl.limit <= 0 {
		return true
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := rl.now()
	bucket, ok := rl.buckets[key]
	if !ok || now.Sub(bucket.windowStart) >= rl.window {
		if len(rl.buckets) >= rateLimitPruneThreshold {
			rl.prune(now)
		}
		rl.buckets[key] = &rateLimitBucket{windowStart: now, count: 1}
		return true
	}
	if bucket.count >= rl.limit {
		return false
	}
	bucket.count++
	return true
}

// prune 清理已过期的窗口桶，配合限流阈值控制内存占用。
func (rl *rateLimiter) prune(now time.Time) {
	for key, bucket := range rl.buckets {
		if now.Sub(bucket.windowStart) >= rl.window {
			delete(rl.buckets, key)
		}
	}
}

// clientIP 提取用于限流的客户端 IP，优先取 X-Forwarded-For 首个地址，回退到 RemoteAddr。
func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if first := strings.TrimSpace(strings.Split(forwarded, ",")[0]); first != "" {
			return first
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// isHealthPath 识别探针和诊断端点，这些路径不应被限流以免探针被误伤或抓取被阻断。
func isHealthPath(path string) bool {
	switch path {
	case "/health", "/healthz", "/readyz", "/metrics", "/version":
		return true
	default:
		return false
	}
}

// RateLimitMiddleware 对 API 请求按每分钟固定窗口限流，超限返回 429 RATE_LIMITED。
// CORS 预检（OPTIONS）与健康探针直接放行。
func RateLimitMiddleware(next http.Handler, limitPerMinute int) http.Handler {
	limiter := newRateLimiter(limitPerMinute)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions || isHealthPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if !limiter.allow(clientIP(r)) {
			writeError(w, http.StatusTooManyRequests, api.CodeRateLimited, "")
			return
		}
		next.ServeHTTP(w, r)
	})
}
