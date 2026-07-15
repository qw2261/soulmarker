package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/authorization"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/store"
)

func organizationAuthRequest(t *testing.T, method, requestURL, userToken, adminToken string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, requestURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if userToken != "" {
		request.Header.Set("Authorization", "Bearer "+userToken)
	}
	if adminToken != "" {
		request.Header.Set("X-Admin-Token", adminToken)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func organizationJSONRequest(t *testing.T, method, requestURL, userToken, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, requestURL, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+userToken)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func createOrganizationAuthorizationFixture(t *testing.T, serverURL string) (*model.User, *model.User, *model.Organization, *model.OrganizerProfile) {
	t.Helper()
	value, ok := testServerStores.Load(serverURL)
	if !ok {
		t.Fatalf("test store not found for %s", serverURL)
	}
	s := value.(*store.Store)
	owner := &model.User{Name: "组织所有者", Contact: "tenant-owner@example.com", PasswordHash: "hash"}
	other := &model.User{Name: "其他用户", Contact: "other-tenant@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(owner); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateUser(other); err != nil {
		t.Fatal(err)
	}
	organization := &model.Organization{Name: "授权组织", Slug: "authorization-org"}
	profile := &model.OrganizerProfile{Name: "授权门店"}
	if err := s.CreateOrganizationWithOwner(organization, profile, owner.ID); err != nil {
		t.Fatal(err)
	}
	return owner, other, organization, profile
}

func TestOrganizationContextEndpointsRequireLiveTenantMembership(t *testing.T) {
	_, _, server := setupTestServer(t)
	owner, other, organization, profile := createOrganizationAuthorizationFixture(t, server.URL)
	ownerToken := makeTestJWT(t, owner.ID, owner.Name, owner.Contact)
	otherToken := makeTestJWT(t, other.ID, other.Name, other.Contact)

	unauthenticated := organizationAuthRequest(t, http.MethodGet, server.URL+"/api/v1/me/organizations", "", "")
	defer unauthenticated.Body.Close()
	if unauthenticated.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated list: expected 401, got %d", unauthenticated.StatusCode)
	}

	list := organizationAuthRequest(t, http.MethodGet, server.URL+"/api/v1/me/organizations", ownerToken, "")
	defer list.Body.Close()
	if list.StatusCode != http.StatusOK {
		t.Fatalf("organization list: expected 200, got %d", list.StatusCode)
	}
	var listPayload struct {
		Data []dto.OrganizationContextResponse `json:"data"`
	}
	if err := json.NewDecoder(list.Body).Decode(&listPayload); err != nil {
		t.Fatal(err)
	}
	if len(listPayload.Data) != 1 || listPayload.Data[0].OrganizationID != organization.ID ||
		listPayload.Data[0].Role != model.OrganizationRoleOwner ||
		listPayload.Data[0].PrincipalType != authorization.PrincipalTypeOrganizationMember {
		t.Fatalf("unexpected organization list: %+v", listPayload.Data)
	}
	if !containsString(listPayload.Data[0].Capabilities, string(authorization.CapabilityOwnerTransfer)) {
		t.Fatalf("owner capability missing: %v", listPayload.Data[0].Capabilities)
	}

	sessionURL := server.URL + "/api/v1/organizations/" + itoa64(organization.ID) + "/session"
	session := organizationAuthRequest(t, http.MethodGet, sessionURL, ownerToken, "")
	defer session.Body.Close()
	if session.StatusCode != http.StatusOK {
		t.Fatalf("owner session: expected 200, got %d", session.StatusCode)
	}
	var sessionPayload struct {
		Data dto.OrganizationContextResponse `json:"data"`
	}
	if err := json.NewDecoder(session.Body).Decode(&sessionPayload); err != nil {
		t.Fatal(err)
	}
	if sessionPayload.Data.OrganizationID != organization.ID || sessionPayload.Data.Role != model.OrganizationRoleOwner {
		t.Fatalf("unexpected session: %+v", sessionPayload.Data)
	}

	crossTenant := organizationAuthRequest(t, http.MethodGet, sessionURL, otherToken, "")
	defer crossTenant.Body.Close()
	assertOrganizationAccessDenied(t, crossTenant)

	invalidID := organizationAuthRequest(t, http.MethodGet, server.URL+"/api/v1/organizations/not-an-id/session", ownerToken, "")
	defer invalidID.Body.Close()
	if invalidID.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid organization ID: expected 400, got %d", invalidID.StatusCode)
	}

	value, _ := testServerStores.Load(server.URL)
	if err := value.(*store.Store).DeleteOrganizer(profile.ID); err != nil {
		t.Fatal(err)
	}
	suspended := organizationAuthRequest(t, http.MethodGet, sessionURL, ownerToken, "")
	defer suspended.Body.Close()
	assertOrganizationAccessDenied(t, suspended)

	suspendedList := organizationAuthRequest(t, http.MethodGet, server.URL+"/api/v1/me/organizations", ownerToken, "")
	defer suspendedList.Body.Close()
	if err := json.NewDecoder(suspendedList.Body).Decode(&listPayload); err != nil {
		t.Fatal(err)
	}
	if len(listPayload.Data) != 1 || listPayload.Data[0].OrganizationStatus != model.OrganizationStatusSuspended || len(listPayload.Data[0].Capabilities) != 0 {
		t.Fatalf("suspended organization context mismatch: %+v", listPayload.Data)
	}
}

func TestPlatformAdminAndTenantCredentialsCannotReplaceEachOther(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "platform-secret")
	_, _, server := setupTestServer(t)
	owner, _, organization, _ := createOrganizationAuthorizationFixture(t, server.URL)
	ownerToken := makeTestJWT(t, owner.ID, owner.Name, owner.Contact)
	tenantSessionURL := server.URL + "/api/v1/organizations/" + itoa64(organization.ID) + "/session"

	platformOnly := organizationAuthRequest(t, http.MethodGet, tenantSessionURL, "", "platform-secret")
	defer platformOnly.Body.Close()
	if platformOnly.StatusCode != http.StatusUnauthorized {
		t.Fatalf("platform token replaced user JWT: expected 401, got %d", platformOnly.StatusCode)
	}

	tenantOnly := organizationAuthRequest(t, http.MethodGet, server.URL+"/api/v1/admin/session", ownerToken, "")
	defer tenantOnly.Body.Close()
	if tenantOnly.StatusCode != http.StatusUnauthorized {
		t.Fatalf("tenant JWT replaced platform token: expected 401, got %d", tenantOnly.StatusCode)
	}

	platformSession := organizationAuthRequest(t, http.MethodGet, server.URL+"/api/v1/admin/session", "", "platform-secret")
	defer platformSession.Body.Close()
	if platformSession.StatusCode != http.StatusOK {
		t.Fatalf("platform session: expected 200, got %d", platformSession.StatusCode)
	}
	var payload struct {
		Data dto.AdminSessionResponse `json:"data"`
	}
	if err := json.NewDecoder(platformSession.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Data.Authenticated || payload.Data.PrincipalType != authorization.PrincipalTypePlatformAdmin {
		t.Fatalf("unexpected platform principal: %+v", payload.Data)
	}
}

func TestOrganizationAuthorizationFeatureFlagFailsClosed(t *testing.T) {
	t.Setenv("ORGANIZATION_AUTH_ENABLED", "false")
	_, _, server := setupTestServer(t)
	for _, path := range []string{"/api/v1/me/organizations", "/api/v1/organizations/1/session"} {
		response := organizationAuthRequest(t, http.MethodGet, server.URL+path, "", "")
		defer response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("disabled route %s: expected 404, got %d", path, response.StatusCode)
		}
		var payload struct {
			ErrorCode string `json:"error_code"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.ErrorCode != string(api.CodeAPIRouteNotFound) {
			t.Fatalf("disabled route %s returned %q", path, payload.ErrorCode)
		}
	}
}

func TestOrganizationResourceRoutesRejectCrossTenantIDs(t *testing.T) {
	s, _, server := setupTestServer(t)
	ownerA := &model.User{Name: "租户 A 所有者", Contact: "tenant-a-owner@example.com", PasswordHash: "hash"}
	ownerB := &model.User{Name: "租户 B 所有者", Contact: "tenant-b-owner@example.com", PasswordHash: "hash"}
	participant := &model.User{Name: "租户 B 参与者", Contact: "tenant-b-http-participant@example.com", PasswordHash: "hash"}
	for _, user := range []*model.User{ownerA, ownerB, participant} {
		if err := s.CreateUser(user); err != nil {
			t.Fatal(err)
		}
	}
	organizationA := &model.Organization{Name: "租户 A", Slug: "tenant-a-http"}
	organizationB := &model.Organization{Name: "租户 B", Slug: "tenant-b-http"}
	profileA := &model.OrganizerProfile{Name: "租户 A 门店"}
	profileB := &model.OrganizerProfile{Name: "租户 B 门店"}
	if err := s.CreateOrganizationWithOwner(organizationA, profileA, ownerA.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateOrganizationWithOwner(organizationB, profileB, ownerB.ID); err != nil {
		t.Fatal(err)
	}
	eventB := &model.Event{OrganizerID: profileB.ID, Title: "租户 B 活动", EventTime: "2099-01-01T00:00:00Z", Location: "B", Capacity: 10}
	if err := s.CreateEventForOrganization(organizationB.ID, eventB); err != nil {
		t.Fatal(err)
	}
	ticketB := &model.Ticket{EventID: eventB.ID, Name: "租户 B 票", Stock: 10}
	if err := s.CreateTicketForOrganization(organizationB.ID, ticketB); err != nil {
		t.Fatal(err)
	}
	credential := "abcdef0123456789abcdef0123456789"
	registration := &model.Registration{
		EventID: eventB.ID, UserID: &participant.ID, Name: participant.Name, Contact: participant.Contact,
		Admission: &model.Admission{CredentialCode: credential},
	}
	if err := s.Register(registration); err != nil {
		t.Fatal(err)
	}
	tokenA := makeTestJWT(t, ownerA.ID, ownerA.Name, ownerA.Contact)
	tokenB := makeTestJWT(t, ownerB.ID, ownerB.Name, ownerB.Contact)
	baseA := server.URL + "/api/v1/organizations/" + itoa64(organizationA.ID) + "/events/" + itoa64(eventB.ID)
	baseB := server.URL + "/api/v1/organizations/" + itoa64(organizationB.ID) + "/events/" + itoa64(eventB.ID)

	requests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, baseA, ""},
		{http.MethodPut, baseA, `{"title":"越权修改"}`},
		{http.MethodDelete, baseA, ""},
		{http.MethodPost, baseA + "/tickets", `{"name":"越权票","stock":1,"price":0}`},
		{http.MethodPut, baseA + "/tickets/" + itoa64(ticketB.ID), `{"name":"越权修改"}`},
		{http.MethodDelete, baseA + "/tickets/" + itoa64(ticketB.ID), ""},
		{http.MethodGet, baseA + "/registrations", ""},
		{http.MethodGet, baseA + "/registrations/export", ""},
		{http.MethodGet, baseA + "/checkins", ""},
		{http.MethodPost, baseA + "/checkins", `{"credential":"` + model.AdmissionCredentialPrefix + credential + `"}`},
	}
	for _, test := range requests {
		response := organizationJSONRequest(t, test.method, test.path, tokenA, test.body)
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("cross-tenant %s %s: expected 404, got %d", test.method, test.path, response.StatusCode)
		}
	}
	createMismatch := organizationJSONRequest(
		t, http.MethodPost,
		server.URL+"/api/v1/organizations/"+itoa64(organizationA.ID)+"/events",
		tokenA,
		`{"organizer_id":`+itoa64(profileB.ID)+`,"title":"错配活动","event_time":"2099-01-01T00:00:00Z","location":"X","capacity":1,"price":0}`,
	)
	createMismatch.Body.Close()
	if createMismatch.StatusCode != http.StatusBadRequest {
		t.Fatalf("cross-tenant organizer create: expected 400, got %d", createMismatch.StatusCode)
	}
	withoutMembership := organizationAuthRequest(t, http.MethodGet, baseB, tokenA, "")
	defer withoutMembership.Body.Close()
	assertOrganizationAccessDenied(t, withoutMembership)

	checkin := organizationJSONRequest(
		t, http.MethodPost, baseB+"/checkins", tokenB,
		`{"credential":"`+model.AdmissionCredentialPrefix+credential+`"}`,
	)
	defer checkin.Body.Close()
	if checkin.StatusCode != http.StatusCreated {
		t.Fatalf("tenant checkin: expected 201, got %d", checkin.StatusCode)
	}
	checkins, total, err := s.ListCheckinsForOrganization(organizationB.ID, eventB.ID, 0, 10)
	if err != nil || total != 1 || len(checkins) != 1 || checkins[0].CheckedInBy != "organization_member:"+itoa64(ownerB.ID) {
		t.Fatalf("tenant checkin actor mismatch: checkins=%+v total=%d err=%v", checkins, total, err)
	}
	if ticket, err := s.GetTicket(ticketB.ID); err != nil || ticket == nil || ticket.Name != ticketB.Name {
		t.Fatalf("cross-tenant HTTP attempts mutated ticket: ticket=%+v err=%v", ticket, err)
	}
}

func TestOrganizationBusinessRouteCapabilityMatrix(t *testing.T) {
	s, _, server := setupTestServer(t)
	owner := &model.User{Name: "矩阵 owner", Contact: "matrix-owner@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(owner); err != nil {
		t.Fatal(err)
	}
	organization := &model.Organization{Name: "权限矩阵组织", Slug: "route-capability-matrix"}
	profile := &model.OrganizerProfile{Name: "权限矩阵门店"}
	if err := s.CreateOrganizationWithOwner(organization, profile, owner.ID); err != nil {
		t.Fatal(err)
	}
	event := &model.Event{OrganizerID: profile.ID, Title: "权限矩阵活动", EventTime: "2099-01-01T00:00:00Z", Location: "线上", Capacity: 100}
	if err := s.CreateEventForOrganization(organization.ID, event); err != nil {
		t.Fatal(err)
	}

	type routeExpectation struct {
		method  string
		path    string
		body    string
		allowed map[string]bool
	}
	base := server.URL + "/api/v1/organizations/" + itoa64(organization.ID) + "/events/" + itoa64(event.ID)
	routes := []routeExpectation{
		{http.MethodPost, server.URL + "/api/v1/organizations/" + itoa64(organization.ID) + "/events", `{"organizer_id":` + itoa64(profile.ID) + `,"title":"矩阵新活动","event_time":"2099-01-01T00:00:00Z","location":"线上","capacity":1,"price":0}`, map[string]bool{model.OrganizationRoleOwner: true, model.OrganizationRoleAdmin: true, model.OrganizationRoleEditor: true}},
		{http.MethodPost, base + "/tickets", `{"name":"矩阵票","stock":10,"price":0}`, map[string]bool{model.OrganizationRoleOwner: true, model.OrganizationRoleAdmin: true, model.OrganizationRoleEditor: true}},
		{http.MethodGet, base + "/registrations", "", map[string]bool{model.OrganizationRoleOwner: true, model.OrganizationRoleAdmin: true, model.OrganizationRoleFinance: true}},
		{http.MethodGet, base + "/registrations/export", "", map[string]bool{model.OrganizationRoleOwner: true, model.OrganizationRoleFinance: true}},
		{http.MethodGet, base + "/checkins", "", map[string]bool{model.OrganizationRoleOwner: true, model.OrganizationRoleAdmin: true, model.OrganizationRoleChecker: true}},
	}
	roles := []string{model.OrganizationRoleOwner, model.OrganizationRoleAdmin, model.OrganizationRoleEditor, model.OrganizationRoleChecker, model.OrganizationRoleFinance}
	users := map[string]*model.User{model.OrganizationRoleOwner: owner}
	for _, role := range roles[1:] {
		user := &model.User{Name: "矩阵 " + role, Contact: "matrix-" + role + "@example.com", PasswordHash: "hash"}
		if err := s.CreateUser(user); err != nil {
			t.Fatal(err)
		}
		if err := s.AddOrganizationMember(&model.OrganizationMember{
			OrganizationID: organization.ID, UserID: user.ID, Role: role, Status: model.OrganizationMemberStatusActive,
		}); err != nil {
			t.Fatal(err)
		}
		users[role] = user
	}
	for _, route := range routes {
		for _, role := range roles {
			user := users[role]
			token := makeTestJWT(t, user.ID, user.Name, user.Contact)
			response := organizationJSONRequest(t, route.method, route.path, token, route.body)
			response.Body.Close()
			if route.allowed[role] {
				if response.StatusCode < 200 || response.StatusCode >= 300 {
					t.Fatalf("role %s should access %s %s, got %d", role, route.method, route.path, response.StatusCode)
				}
			} else if response.StatusCode != http.StatusForbidden {
				t.Fatalf("role %s should be denied %s %s, got %d", role, route.method, route.path, response.StatusCode)
			}
		}
	}
}

func assertOrganizationAccessDenied(t *testing.T, response *http.Response) {
	t.Helper()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.StatusCode)
	}
	var payload struct {
		ErrorCode string `json:"error_code"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.ErrorCode != string(api.CodeOrganizationAccessDenied) {
		t.Fatalf("unexpected error code: %q", payload.ErrorCode)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
