package store

import (
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func TestCreateDataSubjectRequest(t *testing.T) {
	s := setupTestStore(t)
	user := &model.User{Name: "隐私用户", Contact: "privacy@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	request, err := s.CreateDataSubjectRequest(user.ID, model.DataSubjectRequestDataExport)
	if err != nil {
		t.Fatalf("CreateDataSubjectRequest: %v", err)
	}
	if request.ID == 0 {
		t.Fatal("expected non-zero request ID")
	}
	if request.Status != model.DataSubjectStatusPending {
		t.Fatalf("expected pending, got %q", request.Status)
	}
	if request.RequestType != model.DataSubjectRequestDataExport {
		t.Fatalf("expected data_export, got %q", request.RequestType)
	}
	if request.RequestedAt.IsZero() {
		t.Fatal("expected non-zero RequestedAt")
	}
}

func TestCreateDataSubjectRequestInvalidType(t *testing.T) {
	s := setupTestStore(t)
	user := &model.User{Name: "隐私用户", Contact: "invalid@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if _, err := s.CreateDataSubjectRequest(user.ID, "unknown_type"); err == nil {
		t.Fatal("expected error for invalid request type")
	}
}

func TestDataSubjectPendingDoesNotExportProcessedFields(t *testing.T) {
	s := setupTestStore(t)
	user := &model.User{Name: "隐私用户", Contact: "pending@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	request, err := s.CreateDataSubjectRequest(user.ID, model.DataSubjectRequestAccountErasure)
	if err != nil {
		t.Fatalf("CreateDataSubjectRequest: %v", err)
	}
	if request.ProcessedAt != nil {
		t.Fatal("expected ProcessedAt nil for pending request")
	}
	if request.ProcessedBy != nil {
		t.Fatal("expected ProcessedBy nil for pending request")
	}
}

func TestListDataSubjectRequests(t *testing.T) {
	s := setupTestStore(t)
	user := &model.User{Name: "隐私用户", Contact: "list@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	other := &model.User{Name: "其他用户", Contact: "other@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(other); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if _, err := s.CreateDataSubjectRequest(user.ID, model.DataSubjectRequestDataExport); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := s.CreateDataSubjectRequest(user.ID, model.DataSubjectRequestAccountErasure); err != nil {
		t.Fatalf("second create: %v", err)
	}
	if _, err := s.CreateDataSubjectRequest(other.ID, model.DataSubjectRequestDataExport); err != nil {
		t.Fatalf("other create: %v", err)
	}

	requests, err := s.ListDataSubjectRequests(user.ID)
	if err != nil {
		t.Fatalf("ListDataSubjectRequests: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 requests for user, got %d", len(requests))
	}
	// 按 requested_at 倒序，最新（account_erasure）排在最前。
	if requests[0].RequestType != model.DataSubjectRequestAccountErasure {
		t.Fatalf("expected newest request first, got %q", requests[0].RequestType)
	}
}

func TestCompleteDataSubjectRequest(t *testing.T) {
	s := setupTestStore(t)
	user := &model.User{Name: "隐私用户", Contact: "complete@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	request, err := s.CreateDataSubjectRequest(user.ID, model.DataSubjectRequestDataExport)
	if err != nil {
		t.Fatalf("CreateDataSubjectRequest: %v", err)
	}

	if err := s.CompleteDataSubjectRequest(request.ID, user.ID, model.DataSubjectStatusCompleted, "已导出", 1); err != nil {
		t.Fatalf("CompleteDataSubjectRequest: %v", err)
	}

	requests, _ := s.ListDataSubjectRequests(user.ID)
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].Status != model.DataSubjectStatusCompleted {
		t.Fatalf("expected completed, got %q", requests[0].Status)
	}
	if requests[0].Resolution != "已导出" {
		t.Fatalf("expected resolution '已导出', got %q", requests[0].Resolution)
	}
	if requests[0].ProcessedAt == nil {
		t.Fatal("expected non-nil ProcessedAt after completion")
	}
}

func TestCompleteDataSubjectRequestNotFound(t *testing.T) {
	s := setupTestStore(t)
	if err := s.CompleteDataSubjectRequest(999, 1, model.DataSubjectStatusCompleted, "x", 1); err != model.ErrDataSubjectRequestNotFound {
		t.Fatalf("expected ErrDataSubjectRequestNotFound, got %v", err)
	}
}

func TestCompleteDataSubjectRequestAlreadyProcessed(t *testing.T) {
	s := setupTestStore(t)
	user := &model.User{Name: "隐私用户", Contact: "processed@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	request, _ := s.CreateDataSubjectRequest(user.ID, model.DataSubjectRequestDataExport)
	if err := s.CompleteDataSubjectRequest(request.ID, user.ID, model.DataSubjectStatusCompleted, "done", 1); err != nil {
		t.Fatalf("first complete: %v", err)
	}
	if err := s.CompleteDataSubjectRequest(request.ID, user.ID, model.DataSubjectStatusRejected, "re", 1); err != model.ErrDataSubjectRequestNotPending {
		t.Fatalf("expected ErrDataSubjectRequestNotPending, got %v", err)
	}
}

func TestCompleteDataSubjectRequestInvalidTerminalState(t *testing.T) {
	s := setupTestStore(t)
	user := &model.User{Name: "隐私用户", Contact: "invalid-state@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	request, _ := s.CreateDataSubjectRequest(user.ID, model.DataSubjectRequestDataExport)
	if err := s.CompleteDataSubjectRequest(request.ID, user.ID, model.DataSubjectStatusPending, "x", 1); err == nil {
		t.Fatal("expected error for pending terminal state")
	}
	if err := s.CompleteDataSubjectRequest(request.ID, user.ID, "bogus", "x", 1); err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestExportUserData(t *testing.T) {
	s := setupTestStore(t)
	user := &model.User{Name: "导出用户", Contact: "export@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	org := &model.Organization{Name: "导出组织", Slug: "export-org"}
	profile := &model.OrganizerProfile{Name: "导出组织", Contact: "export@example.com"}
	if err := s.CreateOrganizationWithOwner(org, profile, user.ID); err != nil {
		t.Fatalf("CreateOrganizationWithOwner: %v", err)
	}

	event := newTestEvent("导出活动")
	if err := s.CreateEvent(event); err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if err := s.Register(&model.Registration{EventID: event.ID, UserID: &user.ID, Name: user.Name, Contact: user.Contact}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if _, err := s.CreateDataSubjectRequest(user.ID, model.DataSubjectRequestDataExport); err != nil {
		t.Fatalf("CreateDataSubjectRequest: %v", err)
	}

	export, err := s.ExportUserData(user.ID)
	if err != nil {
		t.Fatalf("ExportUserData: %v", err)
	}
	if export.UserID != user.ID {
		t.Fatalf("expected user_id %d, got %d", user.ID, export.UserID)
	}
	if export.Profile == nil || export.Profile.Name != "导出用户" {
		t.Fatalf("expected profile name '导出用户', got %+v", export.Profile)
	}
	if len(export.Memberships) != 1 {
		t.Fatalf("expected 1 membership, got %d", len(export.Memberships))
	}
	if export.Memberships[0].OrganizationName != "导出组织" {
		t.Fatalf("expected organization '导出组织', got %q", export.Memberships[0].OrganizationName)
	}
	if len(export.Registrations) != 1 {
		t.Fatalf("expected 1 registration, got %d", len(export.Registrations))
	}
	if export.Registrations[0].EventTitle != "导出活动" {
		t.Fatalf("expected event title '导出活动', got %q", export.Registrations[0].EventTitle)
	}
	// 报名会触发一条报名成功通知，纳入导出。
	if len(export.Notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(export.Notifications))
	}
	if len(export.PrivacyRequests) != 1 {
		t.Fatalf("expected 1 privacy request, got %d", len(export.PrivacyRequests))
	}
}

func TestExportUserDataMissingUser(t *testing.T) {
	s := setupTestStore(t)
	if _, err := s.ExportUserData(999); err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestErasureUser(t *testing.T) {
	s := setupTestStore(t)
	user := &model.User{Name: "注销用户", Contact: "erasure@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	request, err := s.ErasureUser(user.ID)
	if err != nil {
		t.Fatalf("ErasureUser: %v", err)
	}
	if request.ID == 0 {
		t.Fatal("expected non-zero request ID")
	}
	// 返回对象应反映已完成的终态。
	if request.Status != model.DataSubjectStatusCompleted {
		t.Fatalf("expected completed, got %q", request.Status)
	}
	if request.ProcessedAt == nil {
		t.Fatal("expected non-nil ProcessedAt")
	}

	// 用户标记为已注销。
	persisted, err := s.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if persisted.DeletedAt == nil {
		t.Fatal("expected DeletedAt to be set after erasure")
	}

	// 会话版本递增，使已有 token 失效。
	if persisted.AuthVersion != user.AuthVersion+1 {
		t.Fatalf("expected auth_version %d, got %d", user.AuthVersion+1, persisted.AuthVersion)
	}

	// 请求在库中呈现为 completed。
	requests, _ := s.ListDataSubjectRequests(user.ID)
	if len(requests) != 1 || requests[0].Status != model.DataSubjectStatusCompleted {
		t.Fatalf("expected 1 completed request, got %+v", requests)
	}

	// 重复注销返回 ErrUserAlreadyDeleted。
	if _, err := s.ErasureUser(user.ID); err != model.ErrUserAlreadyDeleted {
		t.Fatalf("expected ErrUserAlreadyDeleted, got %v", err)
	}
}

func TestSweepExpiredRequests(t *testing.T) {
	s := setupTestStore(t)
	user := &model.User{Name: "隐私用户", Contact: "sweep@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	processed, _ := s.CreateDataSubjectRequest(user.ID, model.DataSubjectRequestDataExport)
	if err := s.CompleteDataSubjectRequest(processed.ID, user.ID, model.DataSubjectStatusCompleted, "done", 1); err != nil {
		t.Fatalf("CompleteDataSubjectRequest: %v", err)
	}
	if _, err := s.CreateDataSubjectRequest(user.ID, model.DataSubjectRequestAccountErasure); err != nil {
		t.Fatalf("CreateDataSubjectRequest: %v", err)
	}

	// 以未来时间作为留存边界，确保已处理请求过期；pending 请求始终保留。
	retainedBefore := time.Now().Add(time.Hour)
	cleaned, err := s.SweepExpiredRequests(retainedBefore)
	if err != nil {
		t.Fatalf("SweepExpiredRequests: %v", err)
	}
	if cleaned != 1 {
		t.Fatalf("expected 1 cleaned request, got %d", cleaned)
	}

	requests, _ := s.ListDataSubjectRequests(user.ID)
	if len(requests) != 1 {
		t.Fatalf("expected 1 remaining request, got %d", len(requests))
	}
	if requests[0].RequestType != model.DataSubjectRequestAccountErasure {
		t.Fatalf("expected pending request to survive, got %q", requests[0].RequestType)
	}
}
