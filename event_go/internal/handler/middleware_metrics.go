package handler

import (
	"net/http"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/metrics"
)

// MetricsMiddleware 记录每个 HTTP 请求的方法、状态码与延迟，为 /metrics 提供数据。
// 它作为最外层中间件包裹整个路由，能统计所有请求（含探针），但跳过 /metrics 自身，
// 避免抓取请求自我污染 in-flight 仪表与计数器。
func MetricsMiddleware(reg *metrics.Registry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}
			start := time.Now()
			reg.IncInFlight()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			defer func() {
				reg.DecInFlight()
				reg.Observe(r.Method, rw.statusCode, time.Since(start))
			}()
			next.ServeHTTP(rw, r)
		})
	}
}
