package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/config"
)

// PanicRecord 是一次被恢复的 panic 的追踪载荷，供错误追踪与告警消费者处理。
type PanicRecord struct {
	RequestID string    `json:"request_id"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Recovered string    `json:"recovered"`
	Stack     string    `json:"stack"`
	Time      time.Time `json:"time"`
}

// PanicReporter 消费被恢复的 panic，用于错误追踪与告警。
type PanicReporter interface {
	ReportPanic(ctx context.Context, record PanicRecord)
}

// LogPanicReporter 将 panic 写入结构化日志，是默认的错误追踪实现。
type LogPanicReporter struct{}

func (LogPanicReporter) ReportPanic(_ context.Context, record PanicRecord) {
	slog.Error("panic recovered",
		"request_id", record.RequestID,
		"method", record.Method,
		"path", record.Path,
		"recovered", record.Recovered,
		"stack", record.Stack,
	)
}

// WebhookPanicReporter 将 panic 追踪载荷 POST 到告警 webhook。
// 发送失败仅记录日志，不阻断请求；请求带超时，避免拖垮服务。
type WebhookPanicReporter struct {
	Endpoint string
	Client   *http.Client
	Timeout  time.Duration
}

func (r WebhookPanicReporter) ReportPanic(ctx context.Context, record PanicRecord) {
	client := r.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	body, err := json.Marshal(record)
	if err != nil {
		slog.Error("告警载荷序列化失败", "error", err)
		return
	}

	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, r.Endpoint, bytes.NewReader(body))
	if err != nil {
		slog.Error("构造告警请求失败", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("发送告警失败", "error", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		slog.Error("告警服务返回错误状态", "status", resp.StatusCode)
	}
}

// RecoveryMiddleware 捕获下游 panics，记录错误追踪上下文并返回 500 INTERNAL_ERROR，
// 避免单个请求的 panic 拖垮整个进程。reporter 为 nil 时退化为日志追踪（LogPanicReporter）。
func RecoveryMiddleware(next http.Handler, reporter PanicReporter) http.Handler {
	if reporter == nil {
		reporter = LogPanicReporter{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				record := PanicRecord{
					RequestID: RequestIDFromContext(r.Context()),
					Method:    r.Method,
					Path:      r.URL.Path,
					Recovered: fmt.Sprint(recovered),
					Stack:     string(debug.Stack()),
					Time:      time.Now().UTC(),
				}
				reporter.ReportPanic(r.Context(), record)
				writeJSON(w, http.StatusInternalServerError, api.NewErrorResponse(http.StatusInternalServerError, api.CodeInternalError, ""))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// panicReporterFor 依据启动配置选择 panics 的消费者：配置了 ALERT_WEBHOOK_URL 时发送 webhook 告警，
// 否则退化为结构化日志追踪。
func panicReporterFor(cfg *config.Config) PanicReporter {
	if cfg != nil && strings.TrimSpace(cfg.AlertWebhookURL) != "" {
		return WebhookPanicReporter{Endpoint: cfg.AlertWebhookURL}
	}
	return LogPanicReporter{}
}
