package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/config"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

type recordingPanicReporter struct {
	records []PanicRecord
}

func (r *recordingPanicReporter) ReportPanic(_ context.Context, record PanicRecord) {
	r.records = append(r.records, record)
}

func TestRecoveryMiddlewareReturns500OnPanic(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()
	RecoveryMiddleware(next, nil).ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}

	var apiResp model.APIResp
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if apiResp.Code != http.StatusInternalServerError {
		t.Errorf("expected code 500, got %d", apiResp.Code)
	}
	if apiResp.ErrorCode != string(api.CodeInternalError) {
		t.Errorf("expected INTERNAL_ERROR, got %s", apiResp.ErrorCode)
	}
}

func TestRecoveryMiddlewarePassesThroughHealthyHandler(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()
	RecoveryMiddleware(next, nil).ServeHTTP(w, req)

	if got := w.Result().StatusCode; got != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", got)
	}
}

func TestRecoveryMiddlewareReportsPanicRecord(t *testing.T) {
	reporter := &recordingPanicReporter{}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("爆炸")
	})

	req := httptest.NewRequest(http.MethodPost, "/api/explode", nil)
	req.Header.Set("X-Request-ID", "test-req-12345")
	w := httptest.NewRecorder()
	RequestIDMiddleware(RecoveryMiddleware(next, reporter)).ServeHTTP(w, req)

	if len(reporter.records) != 1 {
		t.Fatalf("expected 1 panic record, got %d", len(reporter.records))
	}
	rec := reporter.records[0]
	if rec.RequestID != "test-req-12345" {
		t.Errorf("expected request id, got %q", rec.RequestID)
	}
	if rec.Method != http.MethodPost {
		t.Errorf("expected method POST, got %q", rec.Method)
	}
	if rec.Path != "/api/explode" {
		t.Errorf("expected path /api/explode, got %q", rec.Path)
	}
	if !strings.Contains(rec.Recovered, "爆炸") {
		t.Errorf("expected recovered to contain panic value, got %q", rec.Recovered)
	}
	if rec.Stack == "" {
		t.Error("expected non-empty stack trace")
	}
}

func TestWebhookPanicReporterPostsToEndpoint(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = body
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	reporter := WebhookPanicReporter{Endpoint: server.URL}
	record := PanicRecord{
		RequestID: "req-abc-123",
		Method:    http.MethodGet,
		Path:      "/api/health",
		Recovered: "panic value",
		Stack:     "goroutine stack",
		Time:      time.Now().UTC(),
	}
	reporter.ReportPanic(context.Background(), record)

	if len(gotBody) == 0 {
		t.Fatal("expected webhook to receive a body")
	}
	var decoded PanicRecord
	if err := json.Unmarshal(gotBody, &decoded); err != nil {
		t.Fatalf("decode received payload: %v", err)
	}
	if decoded.RequestID != record.RequestID || decoded.Path != record.Path || decoded.Recovered != record.Recovered {
		t.Fatalf("payload mismatch: %+v", decoded)
	}
}

func TestWebhookPanicReporterToleratesEndpointFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	url := server.URL
	server.Close()

	reporter := WebhookPanicReporter{Endpoint: url, Client: &http.Client{Timeout: 200 * time.Millisecond}}
	// 目标已不可达：不应 panic，也不应阻塞长于超时。
	done := make(chan struct{})
	go func() {
		reporter.ReportPanic(context.Background(), PanicRecord{RequestID: "r", Method: "GET", Path: "/", Recovered: "x", Time: time.Now()})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ReportPanic blocked on unreachable endpoint")
	}
}

func TestPanicReporterForSelectsReporter(t *testing.T) {
	if _, ok := panicReporterFor(nil).(LogPanicReporter); !ok {
		t.Fatal("nil config should default to LogPanicReporter")
	}
	if _, ok := panicReporterFor(&config.Config{}).(LogPanicReporter); !ok {
		t.Fatal("empty config should default to LogPanicReporter")
	}
	reporter := panicReporterFor(&config.Config{AlertWebhookURL: "https://hooks.example.com/x"})
	webhook, ok := reporter.(WebhookPanicReporter)
	if !ok {
		t.Fatalf("expected WebhookPanicReporter, got %T", reporter)
	}
	if webhook.Endpoint != "https://hooks.example.com/x" {
		t.Fatalf("expected endpoint, got %q", webhook.Endpoint)
	}
}
