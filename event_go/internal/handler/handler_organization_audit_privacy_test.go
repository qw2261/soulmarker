package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func organizationRequestWithID(t *testing.T, method, requestURL, userToken, body, requestID string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, requestURL, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+userToken)
	request.Header.Set("Content-Type", "application/json")
	if requestID != "" {
		request.Header.Set("X-Request-ID", requestID)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func TestRequestIDIsSafeAndCorrelatedWithLogs(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
	var logs bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))

	var contextualID string
	handler := RequestIDMiddleware(LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextualID = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})))

	valid := httptest.NewRequest(http.MethodGet, "/health", nil)
	valid.Header.Set("X-Request-ID", "client-request-001")
	validRecorder := httptest.NewRecorder()
	handler.ServeHTTP(validRecorder, valid)
	if contextualID != "client-request-001" || validRecorder.Header().Get("X-Request-ID") != contextualID ||
		!strings.Contains(logs.String(), `"request_id":"client-request-001"`) {
		t.Fatalf("request ID was not correlated: context=%q header=%q log=%s", contextualID, validRecorder.Header().Get("X-Request-ID"), logs.String())
	}

	unsafe := httptest.NewRequest(http.MethodGet, "/health", nil)
	unsafe.Header.Set("X-Request-ID", "bad id\nforged")
	unsafeRecorder := httptest.NewRecorder()
	handler.ServeHTTP(unsafeRecorder, unsafe)
	generated := unsafeRecorder.Header().Get("X-Request-ID")
	if generated == "bad id\nforged" || !requestIDPattern.MatchString(generated) || len(generated) != 32 {
		t.Fatalf("unsafe request ID was not replaced with 128-bit ID: %q", generated)
	}
}

func TestOwnershipTransferIsImmediateAndAudited(t *testing.T) {
	s, _, server := setupTestServer(t)
	owner := &model.User{Name: "原所有者", Contact: "audit-old-owner@example.com", PasswordHash: "hash"}
	newOwner := &model.User{Name: "新所有者", Contact: "audit-new-owner@example.com", PasswordHash: "hash"}
	outsider := &model.User{Name: "外部用户", Contact: "audit-outsider@example.com", PasswordHash: "hash"}
	for _, user := range []*model.User{owner, newOwner, outsider} {
		if err := s.CreateUser(user); err != nil {
			t.Fatal(err)
		}
	}
	organization := &model.Organization{Name: "可审计组织", Slug: "audited-owner-transfer"}
	profile := &model.OrganizerProfile{Name: "可审计门店"}
	if err := s.CreateOrganizationWithOwner(organization, profile, owner.ID); err != nil {
		t.Fatal(err)
	}
	member := &model.OrganizationMember{
		OrganizationID: organization.ID, UserID: newOwner.ID,
		Role: model.OrganizationRoleAdmin, Status: model.OrganizationMemberStatusActive,
	}
	if err := s.AddOrganizationMember(member); err != nil {
		t.Fatal(err)
	}
	ownerToken := makeTestJWT(t, owner.ID, owner.Name, owner.Contact)
	newOwnerToken := makeTestJWT(t, newOwner.ID, newOwner.Name, newOwner.Contact)
	outsiderToken := makeTestJWT(t, outsider.ID, outsider.Name, outsider.Contact)
	base := server.URL + "/api/v1/organizations/" + itoa64(organization.ID)

	transfer := organizationRequestWithID(t, http.MethodPost, base+"/owner-transfer", ownerToken, `{"member_id":`+itoa64(member.ID)+`}`, "owner-transfer-001")
	defer transfer.Body.Close()
	if transfer.StatusCode != http.StatusOK || transfer.Header.Get("X-Request-ID") != "owner-transfer-001" {
		body, _ := io.ReadAll(transfer.Body)
		t.Fatalf("owner transfer failed: status=%d request_id=%q body=%s", transfer.StatusCode, transfer.Header.Get("X-Request-ID"), body)
	}

	oldSession := organizationAuthRequest(t, http.MethodGet, base+"/session", ownerToken, "")
	defer oldSession.Body.Close()
	newSession := organizationAuthRequest(t, http.MethodGet, base+"/session", newOwnerToken, "")
	defer newSession.Body.Close()
	var oldPayload, newPayload struct {
		Data dto.OrganizationContextResponse `json:"data"`
	}
	if err := json.NewDecoder(oldSession.Body).Decode(&oldPayload); err != nil {
		t.Fatal(err)
	}
	if err := json.NewDecoder(newSession.Body).Decode(&newPayload); err != nil {
		t.Fatal(err)
	}
	if oldPayload.Data.Role != model.OrganizationRoleAdmin || containsString(oldPayload.Data.Capabilities, "owner.transfer") {
		t.Fatalf("former owner retained owner capability: %+v", oldPayload.Data)
	}
	if newPayload.Data.Role != model.OrganizationRoleOwner || !containsString(newPayload.Data.Capabilities, "owner.transfer") {
		t.Fatalf("new owner did not gain owner capability: %+v", newPayload.Data)
	}

	denied := organizationRequestWithID(t, http.MethodPost, base+"/owner-transfer", ownerToken, `{"member_id":`+itoa64(member.ID)+`}`, "former-owner-denied-001")
	defer denied.Body.Close()
	if denied.StatusCode != http.StatusForbidden {
		t.Fatalf("former owner transfer: expected 403, got %d", denied.StatusCode)
	}
	crossTenant := organizationRequestWithID(t, http.MethodGet, base+"/audits", outsiderToken, "", "cross-tenant-audit-001")
	defer crossTenant.Body.Close()
	assertOrganizationAccessDenied(t, crossTenant)

	audits := organizationRequestWithID(t, http.MethodGet, base+"/audits?page_size=100", newOwnerToken, "", "audit-read-001")
	defer audits.Body.Close()
	if audits.StatusCode != http.StatusOK {
		t.Fatalf("audit list: expected 200, got %d", audits.StatusCode)
	}
	var payload struct {
		Data []dto.OrganizationAuditResponse `json:"data"`
	}
	if err := json.NewDecoder(audits.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"owner-transfer-001":      model.AuditOutcomeSuccess,
		"former-owner-denied-001": model.AuditOutcomeDenied,
		"cross-tenant-audit-001":  model.AuditOutcomeDenied,
	}
	for _, entry := range payload.Data {
		if outcome, ok := want[entry.RequestID]; ok {
			if entry.Outcome != outcome || entry.ActorID == nil || entry.ResourceType == "" || entry.Action == "" || entry.HTTPStatus == 0 {
				t.Fatalf("incomplete audit entry: %+v", entry)
			}
			delete(want, entry.RequestID)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing audit request IDs: %+v; entries=%+v", want, payload.Data)
	}
}

func TestOrganizationPIIUsesRoleBasedLeastPrivilege(t *testing.T) {
	s, _, server := setupTestServer(t)
	users := map[string]*model.User{
		model.OrganizationRoleOwner:   {Name: "完整所有者", Contact: "pii-owner@example.com", PasswordHash: "hash"},
		model.OrganizationRoleAdmin:   {Name: "管理成员", Contact: "pii-admin@example.com", PasswordHash: "hash"},
		model.OrganizationRoleChecker: {Name: "核销成员", Contact: "pii-checker@example.com", PasswordHash: "hash"},
		model.OrganizationRoleFinance: {Name: "财务成员", Contact: "pii-finance@example.com", PasswordHash: "hash"},
	}
	participant := &model.User{Name: "报名参与者", Contact: "participant@example.com", PasswordHash: "hash"}
	for _, user := range users {
		if err := s.CreateUser(user); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.CreateUser(participant); err != nil {
		t.Fatal(err)
	}
	organization := &model.Organization{Name: "隐私组织", Slug: "privacy-role-policy"}
	profile := &model.OrganizerProfile{Name: "隐私门店", Contact: "organizer@example.com"}
	if err := s.CreateOrganizationWithOwner(organization, profile, users[model.OrganizationRoleOwner].ID); err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{model.OrganizationRoleAdmin, model.OrganizationRoleChecker, model.OrganizationRoleFinance} {
		if err := s.AddOrganizationMember(&model.OrganizationMember{
			OrganizationID: organization.ID, UserID: users[role].ID, Role: role, Status: model.OrganizationMemberStatusActive,
		}); err != nil {
			t.Fatal(err)
		}
	}
	event := &model.Event{OrganizerID: profile.ID, Title: "隐私活动", EventTime: "2099-01-01T00:00:00Z", Location: "现场", Capacity: 10}
	if err := s.CreateEventForOrganization(organization.ID, event); err != nil {
		t.Fatal(err)
	}
	ticket := &model.Ticket{EventID: event.ID, Name: "隐私票", Stock: 10}
	if err := s.CreateTicketForOrganization(organization.ID, ticket); err != nil {
		t.Fatal(err)
	}
	credentialCode := "abcdef0123456789abcdef0123456789"
	registration := &model.Registration{
		EventID: event.ID, UserID: &participant.ID, Name: participant.Name, Contact: participant.Contact,
		TicketID: &ticket.ID, Admission: &model.Admission{CredentialCode: credentialCode},
	}
	if err := s.Register(registration); err != nil {
		t.Fatal(err)
	}
	tokens := make(map[string]string, len(users))
	for role, user := range users {
		tokens[role] = makeTestJWT(t, user.ID, user.Name, user.Contact)
	}
	base := server.URL + "/api/v1/organizations/" + itoa64(organization.ID)
	eventBase := base + "/events/" + itoa64(event.ID)
	checkin := organizationJSONRequest(t, http.MethodPost, eventBase+"/checkins", tokens[model.OrganizationRoleOwner], `{"credential":"`+model.AdmissionCredentialPrefix+credentialCode+`"}`)
	checkin.Body.Close()
	if checkin.StatusCode != http.StatusCreated {
		t.Fatalf("fixture checkin: expected 201, got %d", checkin.StatusCode)
	}

	ownerMembers := organizationAuthRequest(t, http.MethodGet, base+"/members", tokens[model.OrganizationRoleOwner], "")
	defer ownerMembers.Body.Close()
	adminMembers := organizationAuthRequest(t, http.MethodGet, base+"/members", tokens[model.OrganizationRoleAdmin], "")
	defer adminMembers.Body.Close()
	var ownerMembersPayload, adminMembersPayload struct {
		Data []dto.OrganizationMemberResponse `json:"data"`
	}
	if err := json.NewDecoder(ownerMembers.Body).Decode(&ownerMembersPayload); err != nil {
		t.Fatal(err)
	}
	if err := json.NewDecoder(adminMembers.Body).Decode(&adminMembersPayload); err != nil {
		t.Fatal(err)
	}
	if ownerMembersPayload.Data[0].Contact == "" || strings.Contains(ownerMembersPayload.Data[0].Contact, "***") {
		t.Fatalf("owner did not receive full member PII: %+v", ownerMembersPayload.Data)
	}
	for _, member := range adminMembersPayload.Data {
		if member.Contact != "" && !strings.Contains(member.Contact, "***") {
			t.Fatalf("admin received unmasked member PII: %+v", member)
		}
	}

	adminRegistrations := organizationAuthRequest(t, http.MethodGet, eventBase+"/registrations", tokens[model.OrganizationRoleAdmin], "")
	defer adminRegistrations.Body.Close()
	financeRegistrations := organizationAuthRequest(t, http.MethodGet, eventBase+"/registrations", tokens[model.OrganizationRoleFinance], "")
	defer financeRegistrations.Body.Close()
	var adminRegistrationPayload, financeRegistrationPayload struct {
		Data []dto.RegistrationResponse `json:"data"`
	}
	if err := json.NewDecoder(adminRegistrations.Body).Decode(&adminRegistrationPayload); err != nil {
		t.Fatal(err)
	}
	if err := json.NewDecoder(financeRegistrations.Body).Decode(&financeRegistrationPayload); err != nil {
		t.Fatal(err)
	}
	if got := adminRegistrationPayload.Data[0]; !strings.Contains(got.Contact, "***") || got.Admission != nil {
		t.Fatalf("admin registration was not minimized: %+v", got)
	}
	if got := financeRegistrationPayload.Data[0]; got.Name != participant.Name || got.Contact != participant.Contact {
		t.Fatalf("finance registration did not include authorized PII: %+v", got)
	}

	checkerCheckins := organizationAuthRequest(t, http.MethodGet, eventBase+"/checkins", tokens[model.OrganizationRoleChecker], "")
	defer checkerCheckins.Body.Close()
	var checkerPayload struct {
		Data []dto.CheckinResponse `json:"data"`
	}
	if err := json.NewDecoder(checkerCheckins.Body).Decode(&checkerPayload); err != nil {
		t.Fatal(err)
	}
	if got := checkerPayload.Data[0]; got.CredentialCode != "" || !strings.Contains(got.UserContact, "***") {
		t.Fatalf("checker response exposed credentials or contact: %+v", got)
	}

	financeExport := organizationAuthRequest(t, http.MethodGet, eventBase+"/registrations/export", tokens[model.OrganizationRoleFinance], "")
	defer financeExport.Body.Close()
	exportBody, _ := io.ReadAll(financeExport.Body)
	if financeExport.StatusCode != http.StatusOK || !bytes.Contains(exportBody, []byte(participant.Contact)) {
		t.Fatalf("finance export omitted authorized PII: status=%d body=%q", financeExport.StatusCode, exportBody)
	}
	adminExport := organizationAuthRequest(t, http.MethodGet, eventBase+"/registrations/export", tokens[model.OrganizationRoleAdmin], "")
	defer adminExport.Body.Close()
	if adminExport.StatusCode != http.StatusForbidden {
		t.Fatalf("admin export: expected 403, got %d", adminExport.StatusCode)
	}

	publicOrganizer, err := http.Get(server.URL + "/api/v1/organizers/" + itoa64(profile.ID))
	if err != nil {
		t.Fatal(err)
	}
	defer publicOrganizer.Body.Close()
	var publicPayload struct {
		Data dto.OrganizerResponse `json:"data"`
	}
	if err := json.NewDecoder(publicOrganizer.Body).Decode(&publicPayload); err != nil {
		t.Fatal(err)
	}
	if publicPayload.Data.Contact != "o***@example.com" {
		t.Fatalf("public organizer contact not masked: %q", publicPayload.Data.Contact)
	}

	var deniedPayload struct {
		ErrorCode string `json:"error_code"`
	}
	if err := json.NewDecoder(adminExport.Body).Decode(&deniedPayload); err == nil && deniedPayload.ErrorCode != string(api.CodeOrganizationAccessDenied) {
		t.Fatalf("unexpected admin export denial: %+v", deniedPayload)
	}
}
