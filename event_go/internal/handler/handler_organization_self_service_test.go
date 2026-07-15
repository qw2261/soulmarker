package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/clock"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/notification"
	"github.com/qw2261/soulmarker/event_go/internal/service"
)

type fixedOrganizationInvitationToken struct{ token string }

func (g fixedOrganizationInvitationToken) NewResetToken() (string, error) { return g.token, nil }

func TestOrganizationSelfServiceHTTPJourneyNeedsNoPlatformToken(t *testing.T) {
	s, h, server := setupTestServer(t)
	h.selfService = service.NewOrganizationSelfService(
		s, clock.System{}, fixedOrganizationInvitationToken{token: "known-http-invitation"},
		notification.DiscardPasswordResetSender{}, server.URL, 72*time.Hour,
	)
	owner := &model.User{Name: "HTTP 自助所有者", Contact: "http-self-owner@example.com", PasswordHash: "hash"}
	invitee := &model.User{Name: "HTTP 自助成员", Contact: "http-self-member@example.com", PasswordHash: "hash"}
	for _, user := range []*model.User{owner, invitee} {
		if err := s.CreateUser(user); err != nil {
			t.Fatal(err)
		}
	}
	ownerToken := makeTestJWT(t, owner.ID, owner.Name, owner.Contact)
	inviteeToken := makeTestJWT(t, invitee.ID, invitee.Name, invitee.Contact)
	create := organizationJSONRequest(
		t, http.MethodPost, server.URL+"/api/v1/organizations", ownerToken,
		`{"name":"HTTP 自助组织","slug":"http-self-service","profile_name":"HTTP 自助门店","profile_contact":"public@example.com"}`,
	)
	defer create.Body.Close()
	if create.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(create.Body)
		t.Fatalf("create organization: expected 201, got %d: %s", create.StatusCode, body)
	}
	var created struct {
		Data dto.OrganizationWorkspaceResponse `json:"data"`
	}
	if err := json.NewDecoder(create.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Data.ID <= 0 || created.Data.Profile.ID <= 0 || created.Data.Status != model.OrganizationStatusActive {
		t.Fatalf("organization response mismatch: %+v", created.Data)
	}
	organizationID := created.Data.ID
	profileID := created.Data.Profile.ID
	base := server.URL + "/api/v1/organizations/" + itoa64(organizationID)

	createEvent := organizationJSONRequest(
		t, http.MethodPost, base+"/events", ownerToken,
		`{"organizer_id":`+itoa64(profileID)+`,"title":"无需平台令牌的活动","event_time":"2099-01-01T00:00:00Z","location":"线上","capacity":10,"price":0}`,
	)
	defer createEvent.Body.Close()
	if createEvent.StatusCode != http.StatusCreated {
		t.Fatalf("owner could not create tenant event without platform token: %d", createEvent.StatusCode)
	}

	invite := organizationJSONRequest(
		t, http.MethodPost, base+"/invitations", ownerToken,
		`{"email":"HTTP-SELF-MEMBER@example.com","role":"editor"}`,
	)
	defer invite.Body.Close()
	if invite.StatusCode != http.StatusCreated {
		t.Fatalf("create invitation: expected 201, got %d", invite.StatusCode)
	}
	invitePayload, err := io.ReadAll(invite.Body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(invitePayload), "known-http-invitation") || strings.Contains(string(invitePayload), "token_hash") {
		t.Fatalf("invitation response leaked credential: %s", invitePayload)
	}

	accept := organizationJSONRequest(
		t, http.MethodPost, server.URL+"/api/v1/organization-invitations/accept", inviteeToken,
		`{"token":"known-http-invitation"}`,
	)
	defer accept.Body.Close()
	if accept.StatusCode != http.StatusOK {
		t.Fatalf("accept invitation: expected 200, got %d", accept.StatusCode)
	}

	members := organizationJSONRequest(t, http.MethodGet, base+"/members", ownerToken, "")
	defer members.Body.Close()
	var membersPayload struct {
		Data []dto.OrganizationMemberResponse `json:"data"`
	}
	if err := json.NewDecoder(members.Body).Decode(&membersPayload); err != nil {
		t.Fatal(err)
	}
	if members.StatusCode != http.StatusOK || len(membersPayload.Data) != 2 {
		t.Fatalf("member list mismatch: status=%d data=%+v", members.StatusCode, membersPayload.Data)
	}
	var inviteeMemberID int64
	for _, member := range membersPayload.Data {
		if member.UserID == invitee.ID {
			inviteeMemberID = member.ID
		}
	}
	if inviteeMemberID == 0 {
		t.Fatal("accepted invitee missing from member list")
	}

	update := organizationJSONRequest(
		t, http.MethodPut, base+"/members/"+itoa64(inviteeMemberID), ownerToken, `{"role":"checker"}`,
	)
	update.Body.Close()
	if update.StatusCode != http.StatusOK {
		t.Fatalf("update member role: expected 200, got %d", update.StatusCode)
	}
	checkerSession := organizationAuthRequest(t, http.MethodGet, base+"/session", inviteeToken, "")
	defer checkerSession.Body.Close()
	if checkerSession.StatusCode != http.StatusOK {
		t.Fatalf("updated checker session: expected 200, got %d", checkerSession.StatusCode)
	}

	revoke := organizationJSONRequest(
		t, http.MethodDelete, base+"/members/"+itoa64(inviteeMemberID), ownerToken, "",
	)
	revoke.Body.Close()
	if revoke.StatusCode != http.StatusOK {
		t.Fatalf("revoke member: expected 200, got %d", revoke.StatusCode)
	}
	revokedAccess := organizationAuthRequest(t, http.MethodGet, base+"/session", inviteeToken, "")
	defer revokedAccess.Body.Close()
	assertOrganizationAccessDenied(t, revokedAccess)

	ownerMember, err := s.GetOrganizationMember(organizationID, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	selfRevoke := organizationJSONRequest(
		t, http.MethodDelete, base+"/members/"+itoa64(ownerMember.ID), ownerToken, "",
	)
	defer selfRevoke.Body.Close()
	if selfRevoke.StatusCode != http.StatusForbidden {
		t.Fatalf("owner self revoke: expected 403, got %d", selfRevoke.StatusCode)
	}
	var errorPayload struct {
		ErrorCode string `json:"error_code"`
	}
	if err := json.NewDecoder(selfRevoke.Body).Decode(&errorPayload); err != nil {
		t.Fatal(err)
	}
	if errorPayload.ErrorCode != string(api.CodeOrganizationMemberChangeDenied) {
		t.Fatalf("unexpected owner protection error: %q", errorPayload.ErrorCode)
	}
}

func TestOrganizationSelfServiceFeatureFlagFailsClosed(t *testing.T) {
	t.Setenv("ORGANIZATION_AUTH_ENABLED", "false")
	s, _, server := setupTestServer(t)
	user := &model.User{Name: "关闭自助用户", Contact: "disabled-self@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	token := makeTestJWT(t, user.ID, user.Name, user.Contact)
	for _, test := range []struct {
		path, body string
	}{
		{"/api/v1/organizations", `{"name":"关闭组织","slug":"disabled-org","profile_name":"关闭门店"}`},
		{"/api/v1/organization-invitations/accept", `{"token":"disabled"}`},
	} {
		response := organizationJSONRequest(t, http.MethodPost, server.URL+test.path, token, test.body)
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("disabled self-service route %s: expected 404, got %d", test.path, response.StatusCode)
		}
	}
}
