package handler

import (
	"encoding/json"
	"net/http"
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
