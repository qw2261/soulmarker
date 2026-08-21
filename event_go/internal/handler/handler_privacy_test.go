package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

// rawUserRequest 发起一个不带用户令牌的请求，用于验证必须登录的防护。
func rawUserRequest(t *testing.T, method, requestURL string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, requestURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func decodeResponse(t *testing.T, body io.Reader, out interface{}) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(out); err != nil {
		t.Fatal(err)
	}
}

func TestListMyPrivacyRequestsHandler(t *testing.T) {
	s, _, server := setupTestServer(t)

	// 未登录访问必须被拒。
	unauth := rawUserRequest(t, http.MethodGet, server.URL+"/api/v1/me/privacy/requests")
	defer unauth.Body.Close()
	if unauth.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(unauth.Body)
		t.Fatalf("unauth list: expected 401, got %d body=%s", unauth.StatusCode, body)
	}

	// 已登录但未产生请求时返回空列表。
	resp := doUserJSON(t, http.MethodGet, server.URL+"/api/v1/me/privacy/requests", "", 0, "隐私清单用户", "privacy-list@example.com")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("list: expected 200, got %d body=%s", resp.StatusCode, body)
	}
	var empty struct {
		Data []dto.DataSubjectRequestResponse `json:"data"`
	}
	decodeResponse(t, resp.Body, &empty)
	if len(empty.Data) != 0 {
		t.Fatalf("expected empty privacy request list, got %+v", empty.Data)
	}

	// 产生一次数据导出请求后，列表中应出现且状态为 pending。
	user, err := s.GetUserByContact("privacy-list@example.com")
	if err != nil || user == nil {
		t.Fatalf("load user: %v", err)
	}
	if _, err := s.CreateDataSubjectRequest(user.ID, model.DataSubjectRequestDataExport); err != nil {
		t.Fatalf("seed data export request: %v", err)
	}
	resp2 := doUserJSON(t, http.MethodGet, server.URL+"/api/v1/me/privacy/requests", "", 0, "隐私清单用户", "privacy-list@example.com")
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("list after seed: expected 200, got %d body=%s", resp2.StatusCode, body)
	}
	var listed struct {
		Data []dto.DataSubjectRequestResponse `json:"data"`
	}
	decodeResponse(t, resp2.Body, &listed)
	if len(listed.Data) != 1 {
		t.Fatalf("expected 1 privacy request, got %+v", listed.Data)
	}
	first := listed.Data[0]
	if first.UserID != user.ID || first.RequestType != model.DataSubjectRequestDataExport || first.Status != model.DataSubjectStatusPending {
		t.Fatalf("unexpected privacy request: %+v", first)
	}
}

func TestExportMyDataHandler(t *testing.T) {
	s, _, server := setupTestServer(t)

	unauth := rawUserRequest(t, http.MethodPost, server.URL+"/api/v1/me/privacy/data-export")
	defer unauth.Body.Close()
	if unauth.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(unauth.Body)
		t.Fatalf("unauth export: expected 401, got %d body=%s", unauth.StatusCode, body)
	}

	resp := doUserJSON(t, http.MethodPost, server.URL+"/api/v1/me/privacy/data-export", "", 0, "导出用户", "privacy-export@example.com")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("export: expected 200, got %d body=%s", resp.StatusCode, body)
	}
	var payload struct {
		Code    int                  `json:"code"`
		Message string               `json:"message"`
		Data    model.UserDataExport `json:"data"`
	}
	decodeResponse(t, resp.Body, &payload)
	if payload.Data.UserID == 0 {
		t.Fatalf("export missing user_id: %+v", payload.Data)
	}
	if payload.Data.Profile == nil || payload.Data.Profile.Name != "导出用户" || payload.Data.Profile.Contact != "privacy-export@example.com" {
		t.Fatalf("export profile incomplete: %+v", payload.Data.Profile)
	}
	if payload.Data.ExportedAt.IsZero() {
		t.Fatalf("export exported_at missing: %+v", payload.Data)
	}

	// 导出请求应被记录为 pending，供数据主体追踪。
	user, err := s.GetUserByContact("privacy-export@example.com")
	if err != nil || user == nil {
		t.Fatalf("load user: %v", err)
	}
	requests, err := s.ListDataSubjectRequests(user.ID)
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}
	if len(requests) != 1 || requests[0].RequestType != model.DataSubjectRequestDataExport || requests[0].Status != model.DataSubjectStatusPending {
		t.Fatalf("expected 1 pending data export request, got %+v", requests)
	}
}

func TestRequestAccountErasureHandler(t *testing.T) {
	s, _, server := setupTestServer(t)

	unauth := rawUserRequest(t, http.MethodPost, server.URL+"/api/v1/me/privacy/account-erasure")
	defer unauth.Body.Close()
	if unauth.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(unauth.Body)
		t.Fatalf("unauth erasure: expected 401, got %d body=%s", unauth.StatusCode, body)
	}

	resp := doUserJSON(t, http.MethodPost, server.URL+"/api/v1/me/privacy/account-erasure", "", 0, "注销用户", "privacy-erasure@example.com")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("erasure: expected 200, got %d body=%s", resp.StatusCode, body)
	}
	var payload struct {
		Code    int                            `json:"code"`
		Message string                         `json:"message"`
		Data    dto.DataSubjectRequestResponse `json:"data"`
	}
	decodeResponse(t, resp.Body, &payload)
	if payload.Data.RequestType != model.DataSubjectRequestAccountErasure || payload.Data.Status != model.DataSubjectStatusCompleted {
		t.Fatalf("erasure returned non-completed request: %+v", payload.Data)
	}
	if payload.Data.ProcessedAt == nil {
		t.Fatalf("erasure request missing processed_at: %+v", payload.Data)
	}

	user, err := s.GetUserByContact("privacy-erasure@example.com")
	if err != nil || user == nil {
		t.Fatalf("load erased user: %v", err)
	}
	if user.DeletedAt == nil {
		t.Fatalf("expected DeletedAt to be set after erasure")
	}
	if user.AuthVersion != 2 {
		t.Fatalf("expected auth_version 2 after erasure, got %d", user.AuthVersion)
	}

	// 注销后，旧 token 访问任何需登录接口都应被拒绝（会话已失效）。
	stale := doUserJSON(t, http.MethodGet, server.URL+"/api/v1/me/privacy/requests", "", 0, "注销用户", "privacy-erasure@example.com")
	defer stale.Body.Close()
	if stale.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(stale.Body)
		t.Fatalf("stale token: expected 401, got %d body=%s", stale.StatusCode, body)
	}
	var denied struct {
		ErrorCode string `json:"error_code"`
	}
	decodeResponse(t, stale.Body, &denied)
	if denied.ErrorCode != string(api.CodeUserTokenInvalid) {
		t.Fatalf("stale token: expected %q, got %q", api.CodeUserTokenInvalid, denied.ErrorCode)
	}
}
