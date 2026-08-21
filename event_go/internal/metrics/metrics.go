// Package metrics 提供基于 OpenMetrics/Prometheus 文本格式的 HTTP 运行时指标，
// 用于补齐 G6-R06 中「指标」这一可观测性能力，除此之外不引入任何外部依赖。
package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// latencyBuckets 定义 HTTP 请求延迟直方图的累计上界（秒），覆盖亚毫秒到 10 秒。
var latencyBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

const (
	metricRequestsTotal   = "soulmark_http_requests_total"
	metricRequestDuration = "soulmark_http_request_duration_seconds"
	metricInFlight        = "soulmark_http_requests_in_flight"
	metricUptime          = "soulmark_uptime_seconds"
	metricBuildInfo       = "soulmark_build_info"
)

// Registry 聚合进程级 HTTP 指标。计数与直方图按 method/status 维度聚合，
// 不使用原始路径作为标签，避免高基数导致无界内存增长。
type Registry struct {
	startTime time.Time
	version   string
	commit    string
	inFlight  atomic.Int64

	mu            sync.Mutex
	requestsTotal map[string]map[int]uint64 // method -> status -> 计数
	durations     map[string]*histogram     // method -> 延迟直方图
}

// histogram 记录一组固定上界桶的非累计计数，渲染时按需累加。
type histogram struct {
	bounds []float64
	counts []uint64
	sum    float64
	count  uint64
}

func newHistogram(bounds []float64) *histogram {
	return &histogram{bounds: bounds, counts: make([]uint64, len(bounds)+1)}
}

func (h *histogram) observe(value float64) {
	h.sum += value
	h.count++
	idx := len(h.bounds)
	for i, bound := range h.bounds {
		if value <= bound {
			idx = i
			break
		}
	}
	h.counts[idx]++
}

// NewRegistry 创建一个指标注册表，使用固定直方图桶。
func NewRegistry() *Registry {
	return &Registry{
		startTime:     time.Now().UTC(),
		requestsTotal: make(map[string]map[int]uint64),
		durations:     make(map[string]*histogram),
	}
}

// SetBuild 记录构建 provenance，以便 /metrics 暴露的制品信息与 /version、发布记录一致。
func (r *Registry) SetBuild(version, commit string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.version = version
	r.commit = commit
}

// IncInFlight 增加当前处理中的请求数。
func (r *Registry) IncInFlight() {
	r.inFlight.Add(1)
}

// DecInFlight 减少当前处理中的请求数。
func (r *Registry) DecInFlight() {
	r.inFlight.Add(-1)
}

// Observe 记录一个已完成的请求：方法与状态码计数、延迟直方图。
func (r *Registry) Observe(method string, status int, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	statuses := r.requestsTotal[method]
	if statuses == nil {
		statuses = make(map[int]uint64)
		r.requestsTotal[method] = statuses
	}
	statuses[status]++
	h := r.durations[method]
	if h == nil {
		h = newHistogram(latencyBuckets)
		r.durations[method] = h
	}
	h.observe(duration.Seconds())
}

// Handler 返回渲染 /metrics 的 HTTP handler，内容为 Prometheus/OpenMetrics 文本。
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		r.mu.Lock()
		defer r.mu.Unlock()
		writeMetrics(w, r)
	})
}

func writeMetrics(w http.ResponseWriter, r *Registry) {
	fmt.Fprintf(w, "# HELP %s Total number of HTTP requests processed.\n", metricRequestsTotal)
	fmt.Fprintf(w, "# TYPE %s counter\n", metricRequestsTotal)
	for _, method := range sortedMapKeys(r.requestsTotal) {
		statuses := r.requestsTotal[method]
		statusKeys := make([]int, 0, len(statuses))
		for status := range statuses {
			statusKeys = append(statusKeys, status)
		}
		sort.Ints(statusKeys)
		for _, status := range statusKeys {
			fmt.Fprintf(w, "%s{method=%q,status=%q} %d\n",
				metricRequestsTotal, method, fmt.Sprintf("%d", status), statuses[status])
		}
	}

	fmt.Fprintf(w, "# HELP %s Current number of in-flight HTTP requests.\n", metricInFlight)
	fmt.Fprintf(w, "# TYPE %s gauge\n", metricInFlight)
	fmt.Fprintf(w, "%s %d\n", metricInFlight, r.inFlight.Load())

	fmt.Fprintf(w, "# HELP %s Process uptime in seconds.\n", metricUptime)
	fmt.Fprintf(w, "# TYPE %s gauge\n", metricUptime)
	fmt.Fprintf(w, "%s %.2f\n", metricUptime, time.Since(r.startTime).Seconds())

	fmt.Fprintf(w, "# HELP %s Build provenance for the running process.\n", metricBuildInfo)
	fmt.Fprintf(w, "# TYPE %s gauge\n", metricBuildInfo)
	fmt.Fprintf(w, "%s{version=%q,commit=%q} 1\n", metricBuildInfo, r.version, r.commit)

	fmt.Fprintf(w, "# HELP %s HTTP request latency in seconds.\n", metricRequestDuration)
	fmt.Fprintf(w, "# TYPE %s histogram\n", metricRequestDuration)
	for _, method := range sortedMapKeys(r.durations) {
		h := r.durations[method]
		var cumulative uint64
		for i, bound := range h.bounds {
			cumulative += h.counts[i]
			fmt.Fprintf(w, "%s_bucket{method=%q,le=%q} %d\n",
				metricRequestDuration, method, fmt.Sprintf("%g", bound), cumulative)
		}
		cumulative += h.counts[len(h.bounds)]
		fmt.Fprintf(w, "%s_bucket{method=%q,le=\"+Inf\"} %d\n", metricRequestDuration, method, cumulative)
		fmt.Fprintf(w, "%s_sum{method=%q} %g\n", metricRequestDuration, method, h.sum)
		fmt.Fprintf(w, "%s_count{method=%q} %d\n", metricRequestDuration, method, h.count)
	}
}

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
