package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/config"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/openapi"
)

type openAPIDocument struct {
	OpenAPI    string                                `json:"openapi"`
	Servers    []struct{ URL string }                `json:"servers"`
	Paths      map[string]map[string]json.RawMessage `json:"paths"`
	Components struct {
		Schemas map[string]struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"schemas"`
		RequestBodies   map[string]json.RawMessage `json:"requestBodies"`
		SecuritySchemes map[string]json.RawMessage `json:"securitySchemes"`
	} `json:"components"`
}

type openAPIOperation struct {
	OperationID string `json:"operationId"`
	RequestBody *struct {
		Ref string `json:"$ref"`
	} `json:"requestBody"`
}

func parseOpenAPIDocument(t *testing.T) openAPIDocument {
	t.Helper()
	var document openAPIDocument
	if err := json.Unmarshal(openapi.V1, &document); err != nil {
		t.Fatalf("parse embedded OpenAPI: %v", err)
	}
	return document
}

func TestOpenAPIRoutesMatchRouterCatalog(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	h := newTestHandler(s, config.Load())
	document := parseOpenAPIDocument(t)
	if document.OpenAPI != "3.1.0" {
		t.Fatalf("expected OpenAPI 3.1.0, got %q", document.OpenAPI)
	}
	if len(document.Servers) != 1 || document.Servers[0].URL != "/api/v1" {
		t.Fatalf("unexpected servers: %+v", document.Servers)
	}

	operationIDs := make(map[string]bool)
	documentedOperations := 0
	for path, pathItem := range document.Paths {
		if path == "/openapi.json" {
			continue
		}
		for method := range pathItem {
			if method != "parameters" {
				documentedOperations++
			}
		}
	}
	if documentedOperations != len(h.apiRoutes()) {
		t.Errorf("documented operation count mismatch: OpenAPI=%d router=%d", documentedOperations, len(h.apiRoutes()))
	}
	for _, scheme := range []string{"BearerAuth", "AdminToken"} {
		if _, exists := document.Components.SecuritySchemes[scheme]; !exists {
			t.Errorf("security scheme missing: %s", scheme)
		}
	}

	for _, route := range h.apiRoutes() {
		pathItem, exists := document.Paths[route.Path]
		if !exists {
			t.Errorf("route missing from OpenAPI: %s %s", route.Method, route.Path)
			continue
		}
		rawOperation, exists := pathItem[strings.ToLower(route.Method)]
		if !exists {
			t.Errorf("method missing from OpenAPI: %s %s", route.Method, route.Path)
			continue
		}
		var operation openAPIOperation
		if err := json.Unmarshal(rawOperation, &operation); err != nil {
			t.Fatalf("parse operation %s %s: %v", route.Method, route.Path, err)
		}
		if operation.OperationID != route.OperationID {
			t.Errorf("operationId mismatch for %s %s: got %q want %q", route.Method, route.Path, operation.OperationID, route.OperationID)
		}
		if operationIDs[operation.OperationID] {
			t.Errorf("duplicate operationId: %s", operation.OperationID)
		}
		operationIDs[operation.OperationID] = true

		if route.RequestSchema == "" {
			if operation.RequestBody != nil {
				t.Errorf("unexpected requestBody for %s %s", route.Method, route.Path)
			}
			continue
		}
		expectedRef := "#/components/requestBodies/" + route.RequestSchema
		if operation.RequestBody == nil || operation.RequestBody.Ref != expectedRef {
			t.Errorf("requestBody mismatch for %s %s: got %+v want %s", route.Method, route.Path, operation.RequestBody, expectedRef)
		}
		if _, exists := document.Components.RequestBodies[route.RequestSchema]; !exists {
			t.Errorf("requestBody component missing: %s", route.RequestSchema)
		}
	}
}

func TestOpenAPISchemasMatchDTOJSONFields(t *testing.T) {
	document := parseOpenAPIDocument(t)
	schemaTypes := map[string]reflect.Type{
		"APIResponse":                         dtoType(dto.Response{}),
		"RegisterUserRequest":                 dtoType(dto.RegisterUserRequest{}),
		"LoginRequest":                        dtoType(dto.LoginRequest{}),
		"PasswordResetRequest":                dtoType(dto.PasswordResetRequest{}),
		"PasswordResetConfirmRequest":         dtoType(dto.PasswordResetConfirmRequest{}),
		"RecoveryEmailRequest":                dtoType(dto.RecoveryEmailRequest{}),
		"RecoveryEmailConfirmRequest":         dtoType(dto.RecoveryEmailConfirmRequest{}),
		"CreateOrganizationRequest":           dtoType(dto.CreateOrganizationRequest{}),
		"CreateOrganizationInvitationRequest": dtoType(dto.CreateOrganizationInvitationRequest{}),
		"AcceptOrganizationInvitationRequest": dtoType(dto.AcceptOrganizationInvitationRequest{}),
		"UpdateOrganizationMemberRequest":     dtoType(dto.UpdateOrganizationMemberRequest{}),
		"TransferOrganizationOwnerRequest":    dtoType(dto.TransferOrganizationOwnerRequest{}),
		"CreateOrganizerRequest":              dtoType(dto.CreateOrganizerRequest{}),
		"UpdateOrganizerRequest":              dtoType(dto.UpdateOrganizerRequest{}),
		"CreateEventRequest":                  dtoType(dto.CreateEventRequest{}),
		"UpdateEventRequest":                  dtoType(dto.UpdateEventRequest{}),
		"CreateTicketRequest":                 dtoType(dto.CreateTicketRequest{}),
		"UpdateTicketRequest":                 dtoType(dto.UpdateTicketRequest{}),
		"RegisterEventRequest":                dtoType(dto.RegisterEventRequest{}),
		"CheckinRequest":                      dtoType(dto.CheckinRequest{}),
		"CreatePostRequest":                   dtoType(dto.CreatePostRequest{}),
		"CreateReplyRequest":                  dtoType(dto.CreateReplyRequest{}),
		"CreateContentReportRequest":          dtoType(dto.CreateContentReportRequest{}),
		"ResolveContentReportRequest":         dtoType(dto.ResolveContentReportRequest{}),
		"ModerateContentRequest":              dtoType(dto.ModerateContentRequest{}),
		"UserResponse":                        dtoType(dto.UserResponse{}),
		"LoginResponse":                       dtoType(dto.LoginResponse{}),
		"AdminSessionResponse":                dtoType(dto.AdminSessionResponse{}),
		"OrganizationContextResponse":         dtoType(dto.OrganizationContextResponse{}),
		"OrganizationWorkspaceResponse":       dtoType(dto.OrganizationWorkspaceResponse{}),
		"OrganizationMemberResponse":          dtoType(dto.OrganizationMemberResponse{}),
		"OrganizationInvitationResponse":      dtoType(dto.OrganizationInvitationResponse{}),
		"OrganizationAuditResponse":           dtoType(dto.OrganizationAuditResponse{}),
		"OrganizerResponse":                   dtoType(dto.OrganizerResponse{}),
		"EventResponse":                       dtoType(dto.EventResponse{}),
		"TicketResponse":                      dtoType(dto.TicketResponse{}),
		"RegistrationResponse":                dtoType(dto.RegistrationResponse{}),
		"AdmissionResponse":                   dtoType(dto.AdmissionResponse{}),
		"MyAdmissionResponse":                 dtoType(dto.MyAdmissionResponse{}),
		"MyActivityResponse":                  dtoType(dto.MyActivityResponse{}),
		"CheckinResponse":                     dtoType(dto.CheckinResponse{}),
		"CheckinResultResponse":               dtoType(dto.CheckinResultResponse{}),
		"RegistrationStatusResponse":          dtoType(dto.RegistrationStatusResponse{}),
		"MyRegistrationResponse":              dtoType(dto.MyRegistrationResponse{}),
		"NotificationResponse":                dtoType(dto.NotificationResponse{}),
		"NotificationUnreadCountResponse":     dtoType(dto.NotificationUnreadCountResponse{}),
		"NotificationsMarkedReadResponse":     dtoType(dto.NotificationsMarkedReadResponse{}),
		"PostResponse":                        dtoType(dto.PostResponse{}),
		"ReplyResponse":                       dtoType(dto.ReplyResponse{}),
		"PostDetailResponse":                  dtoType(dto.PostDetailResponse{}),
		"ContentReportReceiptResponse":        dtoType(dto.ContentReportReceiptResponse{}),
		"ContentReportResponse":               dtoType(dto.ContentReportResponse{}),
		"ContentModerationActionResponse":     dtoType(dto.ContentModerationActionResponse{}),
		"IdentityEntityStatsResponse":         dtoType(dto.IdentityEntityStatsResponse{}),
		"LegacyIdentityRecordResponse":        dtoType(dto.LegacyIdentityRecordResponse{}),
		"IdentityMigrationReportResponse":     dtoType(dto.IdentityMigrationReportResponse{}),
		"HealthResponse":                      dtoType(dto.HealthResponse{}),
	}

	for schemaName, goType := range schemaTypes {
		schema, exists := document.Components.Schemas[schemaName]
		if !exists {
			t.Errorf("OpenAPI schema missing: %s", schemaName)
			continue
		}
		actual := make([]string, 0, len(schema.Properties))
		for field := range schema.Properties {
			actual = append(actual, field)
		}
		expected := jsonFieldNames(goType)
		sort.Strings(actual)
		sort.Strings(expected)
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("schema fields mismatch for %s: OpenAPI=%v Go=%v", schemaName, actual, expected)
		}
	}
}

func TestOpenAPIErrorCodesMatchCatalog(t *testing.T) {
	document := parseOpenAPIDocument(t)
	schema, exists := document.Components.Schemas["ErrorResponse"]
	if !exists {
		t.Fatal("OpenAPI ErrorResponse schema missing")
	}
	rawProperty, exists := schema.Properties["error_code"]
	if !exists {
		t.Fatal("OpenAPI ErrorResponse.error_code missing")
	}
	var property struct {
		Enum []string `json:"enum"`
	}
	if err := json.Unmarshal(rawProperty, &property); err != nil {
		t.Fatalf("parse error_code schema: %v", err)
	}
	want := make([]string, 0, len(api.ErrorCodes()))
	for _, code := range api.ErrorCodes() {
		want = append(want, string(code))
	}
	sort.Strings(property.Enum)
	sort.Strings(want)
	if !reflect.DeepEqual(property.Enum, want) {
		t.Fatalf("OpenAPI error codes differ from catalog: OpenAPI=%v Go=%v", property.Enum, want)
	}
}

func dtoType(value interface{}) reflect.Type {
	return reflect.TypeOf(value)
}

func jsonFieldNames(valueType reflect.Type) []string {
	fields := make([]string, 0, valueType.NumField())
	for i := 0; i < valueType.NumField(); i++ {
		if valueType.Field(i).Anonymous && valueType.Field(i).Type.Kind() == reflect.Struct {
			fields = append(fields, jsonFieldNames(valueType.Field(i).Type)...)
			continue
		}
		tag := valueType.Field(i).Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name != "" && name != "-" {
			fields = append(fields, name)
		}
	}
	return fields
}

func TestVersionedAndLegacyRoutesAreEquivalent(t *testing.T) {
	_, _, server := setupTestServer(t)
	legacy, err := http.Get(server.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer legacy.Body.Close()
	versioned, err := http.Get(server.URL + "/api/v1/events")
	if err != nil {
		t.Fatal(err)
	}
	defer versioned.Body.Close()

	legacyBody, _ := io.ReadAll(legacy.Body)
	versionedBody, _ := io.ReadAll(versioned.Body)
	if legacy.StatusCode != versioned.StatusCode || string(legacyBody) != string(versionedBody) {
		t.Fatalf("legacy/v1 mismatch: legacy=%d %s versioned=%d %s", legacy.StatusCode, legacyBody, versioned.StatusCode, versionedBody)
	}
}

func TestOpenAPISpecIsServedFromVersionedAPI(t *testing.T) {
	_, _, server := setupTestServer(t)
	response, err := http.Get(server.URL + "/api/v1/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.Contains(contentType, "application/vnd.oai.openapi+json") {
		t.Fatalf("unexpected content type: %s", contentType)
	}
	contents, _ := io.ReadAll(response.Body)
	if !json.Valid(contents) {
		t.Fatal("served OpenAPI document is not valid JSON")
	}
}
