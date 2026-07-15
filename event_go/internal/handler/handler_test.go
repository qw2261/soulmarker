package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/qw2261/soulmarker/event_go/internal/api"
	appauth "github.com/qw2261/soulmarker/event_go/internal/auth"
	"github.com/qw2261/soulmarker/event_go/internal/clock"
	"github.com/qw2261/soulmarker/event_go/internal/config"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/identifier"
	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/notification"
	"github.com/qw2261/soulmarker/event_go/internal/service"
	"github.com/qw2261/soulmarker/event_go/internal/store"
	"golang.org/x/crypto/bcrypt"
)

var testServerStores sync.Map

type testClock struct {
	now time.Time
}

func (c testClock) Now() time.Time {
	return c.now
}

type recordingTokenManager struct {
	token    string
	user     *model.User
	issuedAt time.Time
	ttl      time.Duration
}

type sequenceResetTokenGenerator struct {
	tokens []string
	index  int
}

func (g *sequenceResetTokenGenerator) NewResetToken() (string, error) {
	if g.index >= len(g.tokens) {
		return fmt.Sprintf("reset-token-%d", g.index+1), nil
	}
	token := g.tokens[g.index]
	g.index++
	return token, nil
}

type recordingPasswordResetSender struct {
	contact           string
	resetURL          string
	expiresAt         time.Time
	verificationEmail string
	verificationURL   string
	verificationCalls int
	calls             int
}

func (s *recordingPasswordResetSender) SendPasswordReset(_ context.Context, contact, resetURL string, expiresAt time.Time) error {
	s.contact, s.resetURL, s.expiresAt = contact, resetURL, expiresAt
	s.calls++
	return nil
}

func (s *recordingPasswordResetSender) SendRecoveryEmailVerification(_ context.Context, email, verificationURL string, _ time.Time) error {
	s.verificationEmail, s.verificationURL = email, verificationURL
	s.verificationCalls++
	return nil
}

func (m *recordingTokenManager) SignUser(user *model.User, issuedAt time.Time, ttl time.Duration) (string, error) {
	m.user = user
	m.issuedAt = issuedAt
	m.ttl = ttl
	return m.token, nil
}

func (m *recordingTokenManager) VerifyUser(string) (*model.UserClaims, error) {
	return nil, errors.New("not implemented in recording token manager")
}

func mustNewStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

func newTestHandler(s *store.Store, cfg *config.Config) *Handler {
	businessClock := clock.System{}
	authentication := service.NewAuthenticationService(
		s, businessClock, identifier.CryptoResetTokenGenerator{},
		notification.DiscardPasswordResetSender{}, cfg.PublicBaseURL,
		time.Duration(cfg.PasswordResetTTLMin)*time.Minute,
		time.Duration(cfg.RecoveryEmailTTLMin)*time.Minute,
	)
	return NewHandler(s, cfg, Dependencies{
		Clock:          businessClock,
		Tokens:         appauth.NewJWTManager(cfg.JWTSecret),
		Registrations:  service.NewRegistrationService(s, businessClock, time.Duration(cfg.CancelDeadlineHours)*time.Hour, identifier.CryptoCredentialGenerator{}),
		Discussions:    service.NewDiscussionService(s),
		Admissions:     service.NewAdmissionService(s, businessClock),
		Authentication: authentication,
		Moderation:     service.NewContentModerationService(s, businessClock),
		Notifications:  service.NewNotificationService(s, businessClock, 24*time.Hour),
		Organizations:  service.NewOrganizationAuthorizationService(s),
		Operations:     service.NewOrganizationOperationsService(s, businessClock),
	})
}

func makeTestJWT(t *testing.T, userID int64, name, contact string) string {
	t.Helper()
	claims := &model.UserClaims{
		UserID:  userID,
		Name:    name,
		Contact: contact,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("event-go-dev-secret-change-in-production"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func doUserJSON(t *testing.T, method, requestURL, body string, _ int64, name, contact string) *http.Response {
	t.Helper()
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		t.Fatalf("parse request URL: %v", err)
	}
	baseURL := parsedURL.Scheme + "://" + parsedURL.Host
	value, ok := testServerStores.Load(baseURL)
	if !ok {
		t.Fatalf("test store not found for %s", baseURL)
	}
	s := value.(*store.Store)
	user, err := s.GetUserByContact(contact)
	if err != nil {
		t.Fatalf("load test user: %v", err)
	}
	if user == nil {
		user = &model.User{Name: name, Contact: contact, PasswordHash: "test-only"}
		if err := s.CreateUser(user); err != nil {
			t.Fatalf("create test user: %v", err)
		}
	}
	token := makeTestJWT(t, user.ID, user.Name, user.Contact)

	req, err := http.NewRequest(method, requestURL, strings.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func setupTestServer(t *testing.T) (*store.Store, *Handler, *httptest.Server) {
	t.Helper()
	s := mustNewStore(t)
	h := newTestHandler(s, config.Load())

	server := httptest.NewServer(NewRouter(h, nil))
	testServerStores.Store(server.URL, s)
	t.Cleanup(func() {
		testServerStores.Delete(server.URL)
		server.Close()
		s.Close()
	})

	_ = s.CreateOrganizer(&model.Organizer{Name: "默认门店"})

	return s, h, server
}

func parseResp(t *testing.T, body []byte) model.APIResp {
	t.Helper()
	var resp model.APIResp
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("json unmarshal failed: %v, body: %s", err, string(body))
	}
	return resp
}

func itoa64(i int64) string {
	return fmt.Sprintf("%d", i)
}

func createTestOrganizer(t *testing.T, srv *httptest.Server) int64 {
	t.Helper()
	body := `{"name":"测试门店","description":"测试用"}`
	resp, err := http.Post(srv.URL+"/api/organizers", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create organizer failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	r := parseResp(t, respBody)
	orgMap, ok := r.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("organizer data not a map")
	}
	return int64(orgMap["id"].(float64))
}

func makeEventBody(organizerID int64, extra ...string) string {
	base := fmt.Sprintf(`{"organizer_id":%d,"title":"Go 讲座","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":50,"price":0`, organizerID)
	for _, e := range extra {
		base += "," + e
	}
	base += "}"
	return base
}

func createStoreEvent(t *testing.T, s *store.Store, title string) *model.Event {
	t.Helper()
	e := &model.Event{
		OrganizerID: 1,
		Title:       title,
		EventTime:   "2099-12-31T18:00:00+08:00",
		Location:    "线上",
		Capacity:    100,
		Price:       0,
	}
	if err := s.CreateEvent(e); err != nil {
		t.Fatalf("create store event: %v", err)
	}
	return e
}

func TestCreateEventHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	body := `{"organizer_id":1,"title":"Go 讲座","cover_url":" https://assets.example.com/go.jpg ","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":50,"price":0}`
	resp, err := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var apiResp model.APIResp
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if coverURL := apiResp.Data.(map[string]interface{})["cover_url"]; coverURL != "https://assets.example.com/go.jpg" {
		t.Fatalf("unexpected cover_url: %v", coverURL)
	}
}

func TestCreateEventValidationErrors(t *testing.T) {
	_, _, srv := setupTestServer(t)

	tests := []struct {
		name string
		body string
	}{
		{"empty title", `{"event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":50}`},
		{"empty event_time", `{"organizer_id":1,"title":"讲座","location":"线上","capacity":50}`},
		{"invalid time format", `{"organizer_id":1,"title":"讲座","event_time":"invalid","location":"线上","capacity":50}`},
		{"empty location", `{"organizer_id":1,"title":"讲座","event_time":"2026-12-31T18:00:00+08:00","capacity":50}`},
		{"zero capacity", `{"organizer_id":1,"title":"讲座","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":0}`},
		{"negative price", `{"organizer_id":1,"title":"讲座","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":-1}`},
		{"unsafe cover url", `{"organizer_id":1,"title":"讲座","cover_url":"javascript:alert(1)","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(tt.body))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d for case %q", resp.StatusCode, tt.name)
			}
		})
	}
}

func TestCreateEventInvalidJSON(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader("{invalid}"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

func TestListEventsHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	for i := 0; i < 2; i++ {
		body := `{"organizer_id":1,"title":"活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
		resp, err := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("create event failed: %v", err)
		}
		resp.Body.Close()
	}

	resp, err := http.Get(srv.URL + "/api/events")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var apiResp model.APIResp
	json.NewDecoder(resp.Body).Decode(&apiResp)
	resp.Body.Close()

	events, ok := apiResp.Data.([]interface{})
	if !ok {
		t.Fatal("expected array data")
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestGetEventHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"详情测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, err := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	resp, err := http.Get(srv.URL + "/api/events/" + itoa64(id))
	if err != nil {
		t.Fatalf("get event failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetEventHandlerNotFound(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/events/999")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetEventHandlerInvalidID(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/events/abc")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid ID, got %d", resp.StatusCode)
	}
}

func TestUpdateEventHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"原始标题","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	updateBody := `{"organizer_id":1,"title":"新标题","cover_url":"https://assets.example.com/new.jpg","price":99.9}`
	req, _ := http.NewRequest("PUT", srv.URL+"/api/events/"+itoa64(id), strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var updateResp model.APIResp
	json.NewDecoder(resp.Body).Decode(&updateResp)
	resp.Body.Close()
	updated := updateResp.Data.(map[string]interface{})
	if updated["title"] != "新标题" {
		t.Fatalf("expected title '新标题', got %v", updated["title"])
	}
	if updated["price"].(float64) != 99.9 {
		t.Fatalf("expected price 99.9, got %v", updated["price"])
	}
	if updated["cover_url"] != "https://assets.example.com/new.jpg" {
		t.Fatalf("unexpected cover_url: %v", updated["cover_url"])
	}
}

func TestUpdateEventInvalidStatus(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	updateBody := `{"status":"invalid_status"}`
	req, _ := http.NewRequest("PUT", srv.URL+"/api/events/"+itoa64(id), strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUpdateEventHandlerNotFound(t *testing.T) {
	_, _, srv := setupTestServer(t)

	updateBody := `{"title":"新标题"}`
	req, _ := http.NewRequest("PUT", srv.URL+"/api/events/999", strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteEventHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"待删除","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	req, _ := http.NewRequest("DELETE", srv.URL+"/api/events/"+itoa64(id), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	getResp, _ := http.Get(srv.URL + "/api/events/" + itoa64(id))
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", getResp.StatusCode)
	}
	getResp.Body.Close()
}

func TestDeleteEventHandlerNotFound(t *testing.T) {
	_, _, srv := setupTestServer(t)

	req, _ := http.NewRequest("DELETE", srv.URL+"/api/events/999", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestRegisterHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"报名测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	resp := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(id)+"/register", `{}`, 1, "张三", "zs@email.com")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestRegisterHandlerDuplicate(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"重复报名","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	first := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(id)+"/register", `{}`, 1, "张三", "zs@email.com")
	first.Body.Close()
	resp := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(id)+"/register", `{}`, 1, "张三", "zs@email.com")
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestRegisterHandlerFull(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"已满活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":1,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	first := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(id)+"/register", `{}`, 1, "张三", "zs@email.com")
	first.Body.Close()
	resp := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(id)+"/register", `{}`, 2, "李四", "ls@email.com")
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestRegisterHandlerNotPublished(t *testing.T) {
	s, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"草稿活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	draft := "draft"
	s.UpdateEvent(id, model.UpdateEventReq{Status: &draft})

	resp := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(id)+"/register", `{}`, 1, "张三", "zs@email.com")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for draft event, got %d", resp.StatusCode)
	}
}

func TestRegisterHandlerNotFound(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/999/register", `{}`, 1, "张三", "zs@email.com")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestRegisterHandlerWithTicket(t *testing.T) {
	s, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"门票报名","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	ticket := &model.Ticket{EventID: id, Name: "普通票", Price: 0, Stock: 5}
	s.CreateTicket(ticket)

	regBody := `{"ticket_id":` + itoa64(ticket.ID) + `}`
	resp := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(id)+"/register", regBody, 1, "张三", "zs@email.com")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestListRegistrationsHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"报名列表","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	registration := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(id)+"/register", `{}`, 1, "张三", "zs@email.com")
	registration.Body.Close()

	resp, _ := http.Get(srv.URL + "/api/events/" + itoa64(id) + "/registrations")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCreatePostHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"发帖测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	registration := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(id)+"/register", `{}`, 1, "张三", "zs@email.com")
	registration.Body.Close()

	postBody := `{"title":"好活动","content":"推荐给大家"}`
	resp := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(id)+"/posts", postBody, 1, "张三", "zs@email.com")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestCreatePostHandlerNotRegistered(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"权限测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	postBody := `{"title":"无权限","content":"测试"}`
	resp := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(id)+"/posts", postBody, 9, "未报名", "nobody@email.com")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestListPostsHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"帖子列表","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	registration := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(id)+"/register", `{}`, 1, "张三", "zs@email.com")
	registration.Body.Close()
	postResponse := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(id)+"/posts", `{"title":"帖子","content":"内容"}`, 1, "张三", "zs@email.com")
	postResponse.Body.Close()

	resp, _ := http.Get(srv.URL + "/api/events/" + itoa64(id) + "/posts")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetPostHandlerWithReplies(t *testing.T) {
	s, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"帖子详情","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	s.Register(&model.Registration{EventID: id, Name: "张三", Contact: "zs@email.com"})
	post := &model.Post{EventID: id, AuthorName: "张三", AuthorContact: "zs@email.com", Title: "帖子", Content: "内容"}
	s.CreatePost(post)

	resp, _ := http.Get(srv.URL + "/api/events/" + itoa64(id) + "/posts/" + itoa64(post.ID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var apiResp model.APIResp
	json.NewDecoder(resp.Body).Decode(&apiResp)
	resp.Body.Close()
	data := apiResp.Data.(map[string]interface{})
	if _, ok := data["post"]; !ok {
		t.Fatal("expected 'post' in response")
	}
	if _, ok := data["replies"]; !ok {
		t.Fatal("expected 'replies' in response")
	}
}

func TestPostRoutesRequireEventOwnership(t *testing.T) {
	s, _, srv := setupTestServer(t)
	eventA := createStoreEvent(t, s, "帖子所属活动")
	eventB := createStoreEvent(t, s, "错误路径活动")

	if err := s.Register(&model.Registration{EventID: eventA.ID, Name: "张三", Contact: "post-scope@test.com"}); err != nil {
		t.Fatal(err)
	}
	post := &model.Post{
		EventID:       eventA.ID,
		AuthorName:    "张三",
		AuthorContact: "post-scope@test.com",
		Title:         "仅属于活动 A",
		Content:       "内容",
	}
	if err := s.CreatePost(post); err != nil {
		t.Fatal(err)
	}

	resp, err := http.Get(srv.URL + "/api/events/" + itoa64(eventB.ID) + "/posts/" + itoa64(post.ID))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("wrong event get post: expected 404, got %d", resp.StatusCode)
	}

	resp = doUserJSON(t, http.MethodPost,
		srv.URL+"/api/events/"+itoa64(eventB.ID)+"/posts/"+itoa64(post.ID)+"/replies",
		`{"content":"越权回复"}`, 1, "张三", "post-scope@test.com")
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("wrong event create reply: expected 404, got %d", resp.StatusCode)
	}
}

func TestCreateReplyHandler(t *testing.T) {
	s, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"回复测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	registration := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(id)+"/register", `{}`, 1, "张三", "zs@email.com")
	registration.Body.Close()
	post := &model.Post{EventID: id, AuthorName: "张三", AuthorContact: "zs@email.com", Title: "帖子", Content: "内容"}
	s.CreatePost(post)

	replyBody := `{"content":"回复内容"}`
	resp := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(id)+"/posts/"+itoa64(post.ID)+"/replies", replyBody, 1, "张三", "zs@email.com")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestContentModerationHTTPJourney(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "moderation-admin-token")
	s, _, srv := setupTestServer(t)
	event := createStoreEvent(t, s, "内容治理活动")
	author := &model.User{Name: "违规作者", Contact: "moderation-author@example.com", PasswordHash: "hash"}
	reporter := &model.User{Name: "举报用户", Contact: "moderation-reporter@example.com", PasswordHash: "hash"}
	for _, user := range []*model.User{author, reporter} {
		if err := s.CreateUser(user); err != nil {
			t.Fatal(err)
		}
		if err := s.Register(&model.Registration{
			EventID: event.ID, UserID: &user.ID, Name: user.Name, Contact: user.Contact,
		}); err != nil {
			t.Fatal(err)
		}
	}
	post := &model.Post{
		EventID: event.ID, UserID: &author.ID, AuthorName: author.Name, AuthorContact: author.Contact,
		Title: "治理帖子", Content: "帖子内容",
	}
	if err := s.CreatePost(post); err != nil {
		t.Fatal(err)
	}
	reply := &model.Reply{
		PostID: post.ID, UserID: &author.ID, AuthorName: author.Name,
		AuthorContact: author.Contact, Content: "需要移除的回复",
	}
	if err := s.CreateReply(reply); err != nil {
		t.Fatal(err)
	}

	reportURL := srv.URL + "/api/v1/events/" + itoa64(event.ID) + "/posts/" + itoa64(post.ID) + "/replies/" + itoa64(reply.ID) + "/reports"
	resp := doUserJSON(t, http.MethodPost, reportURL,
		`{"category":"abuse","detail":"包含攻击性内容"}`, reporter.ID, reporter.Name, reporter.Contact)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create report: expected 201, got %d", resp.StatusCode)
	}
	var reportResp model.APIResp
	if err := json.NewDecoder(resp.Body).Decode(&reportResp); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	reportID := int64(reportResp.Data.(map[string]interface{})["id"].(float64))

	resp = doUserJSON(t, http.MethodPost, reportURL,
		`{"category":"abuse","detail":"重复举报"}`, reporter.ID, reporter.Name, reporter.Contact)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("duplicate report: expected 200, got %d", resp.StatusCode)
	}

	listURL := srv.URL + "/api/v1/admin/content-reports?status=open"
	resp, err := http.Get(listURL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous moderation queue: expected 401, got %d", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodGet, listURL, nil)
	req.Header.Set("X-Admin-Token", "moderation-admin-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var listResp model.APIResp
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	reports := listResp.Data.([]interface{})
	if len(reports) != 1 || reports[0].(map[string]interface{})["target_content"] != reply.Content {
		t.Fatalf("unexpected moderation queue: %#v", reports)
	}

	resolveURL := srv.URL + "/api/v1/admin/content-reports/" + itoa64(reportID)
	req, _ = http.NewRequest(http.MethodPut, resolveURL, strings.NewReader(`{"resolution":"remove","note":"确认违规"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", "moderation-admin-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("resolve report: expected 200, got %d", resp.StatusCode)
	}

	postURL := srv.URL + "/api/v1/events/" + itoa64(event.ID) + "/posts/" + itoa64(post.ID)
	resp, err = http.Get(postURL)
	if err != nil {
		t.Fatal(err)
	}
	var postResp model.APIResp
	if err := json.NewDecoder(resp.Body).Decode(&postResp); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	replies := postResp.Data.(map[string]interface{})["replies"].([]interface{})
	if len(replies) != 0 {
		t.Fatalf("removed reply still public: %#v", replies)
	}

	actionsURL := srv.URL + "/api/v1/admin/content-actions"
	req, _ = http.NewRequest(http.MethodGet, actionsURL, nil)
	req.Header.Set("X-Admin-Token", "moderation-admin-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var actionsResp model.APIResp
	if err := json.NewDecoder(resp.Body).Decode(&actionsResp); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if actions := actionsResp.Data.([]interface{}); len(actions) != 1 || actions[0].(map[string]interface{})["action"] != "remove" {
		t.Fatalf("unexpected actions: %#v", actions)
	}

	restoreURL := postURL + "/replies/" + itoa64(reply.ID) + "/restore"
	req, _ = http.NewRequest(http.MethodPut, restoreURL, strings.NewReader(`{"reason":"复核后恢复"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", "moderation-admin-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("restore reply: expected 200, got %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodDelete, postURL, strings.NewReader(`{"reason":"管理员直接移除"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", "moderation-admin-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("remove post: expected 200, got %d", resp.StatusCode)
	}
	resp, err = http.Get(postURL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("removed post should be hidden: got %d", resp.StatusCode)
	}
}

func TestCreateTicketHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"门票测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	ticketBody := `{"name":"普通票","price":0,"stock":100}`
	resp, _ := http.Post(srv.URL+"/api/events/"+itoa64(id)+"/tickets", "application/json", strings.NewReader(ticketBody))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestCreateTicketValidationErrors(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"门票校验","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	tests := []struct {
		name string
		body string
	}{
		{"empty name", `{"price":0,"stock":10}`},
		{"negative price", `{"name":"票","price":-1,"stock":10}`},
		{"zero stock", `{"name":"票","price":0,"stock":0}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, _ := http.Post(srv.URL+"/api/events/"+itoa64(id)+"/tickets", "application/json", strings.NewReader(tt.body))
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d for case %q", resp.StatusCode, tt.name)
			}
		})
	}
}

func TestListTicketsHandler(t *testing.T) {
	s, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"门票列表","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	eid := int64(eventData["id"].(float64))

	s.CreateTicket(&model.Ticket{EventID: eid, Name: "票1", Price: 0, Stock: 10})

	resp, _ := http.Get(srv.URL + "/api/events/" + itoa64(eid) + "/tickets")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestUpdateTicketHandler(t *testing.T) {
	s, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"更新门票","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	eid := int64(eventData["id"].(float64))

	ticket := &model.Ticket{EventID: eid, Name: "原始名称", Price: 0, Stock: 1}
	s.CreateTicket(ticket)

	updateBody := `{"name":"新名称","price":99.9,"stock":5}`
	req, _ := http.NewRequest("PUT", srv.URL+"/api/events/"+itoa64(eid)+"/tickets/"+itoa64(ticket.ID), strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("update ticket failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDeleteTicketHandler(t *testing.T) {
	s, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"删除门票","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	eid := int64(eventData["id"].(float64))

	ticket := &model.Ticket{EventID: eid, Name: "待删除", Price: 0, Stock: 1}
	s.CreateTicket(ticket)

	req, _ := http.NewRequest("DELETE", srv.URL+"/api/events/"+itoa64(eid)+"/tickets/"+itoa64(ticket.ID), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete ticket failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestTicketRoutesRequireEventOwnership(t *testing.T) {
	s, _, srv := setupTestServer(t)
	eventA := createStoreEvent(t, s, "门票所属活动")
	eventB := createStoreEvent(t, s, "错误路径活动")
	ticket := &model.Ticket{EventID: eventA.ID, Name: "活动 A 门票", Price: 10, Stock: 5}
	if err := s.CreateTicket(ticket); err != nil {
		t.Fatal(err)
	}

	resp, err := http.Get(srv.URL + "/api/events/" + itoa64(eventB.ID) + "/tickets/" + itoa64(ticket.ID))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("wrong event get ticket: expected 404, got %d", resp.StatusCode)
	}

	req, _ := http.NewRequest(
		"PUT",
		srv.URL+"/api/events/"+itoa64(eventB.ID)+"/tickets/"+itoa64(ticket.ID),
		strings.NewReader("{\"name\":\"越权修改\"}"),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("wrong event update ticket: expected 404, got %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(
		"DELETE",
		srv.URL+"/api/events/"+itoa64(eventB.ID)+"/tickets/"+itoa64(ticket.ID),
		nil,
	)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("wrong event delete ticket: expected 404, got %d", resp.StatusCode)
	}

	stored, err := s.GetTicket(ticket.ID)
	if err != nil || stored == nil || stored.Name != "活动 A 门票" {
		t.Fatalf("ticket changed through wrong event path: ticket=%+v err=%v", stored, err)
	}
}

func TestUpdateValidationRejectsInvalidValues(t *testing.T) {
	s, _, srv := setupTestServer(t)
	event := createStoreEvent(t, s, "更新校验")
	ticket := &model.Ticket{EventID: event.ID, Name: "有效门票", Price: 10, Stock: 5}
	if err := s.CreateTicket(ticket); err != nil {
		t.Fatal(err)
	}

	eventCases := []string{
		"{\"organizer_id\":0}",
		"{\"organizer_id\":999}",
		"{\"title\":\" \"}",
		"{\"event_time\":\"invalid\"}",
		"{\"location\":\" \"}",
		"{\"capacity\":0}",
		"{\"price\":-1}",
	}
	for _, body := range eventCases {
		req, _ := http.NewRequest("PUT", srv.URL+"/api/events/"+itoa64(event.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("event body %s: expected 400, got %d", body, resp.StatusCode)
		}
	}

	ticketCases := []string{
		"{\"name\":\" \"}",
		"{\"price\":-1}",
		"{\"stock\":-1}",
	}
	for _, body := range ticketCases {
		req, _ := http.NewRequest(
			"PUT",
			srv.URL+"/api/events/"+itoa64(event.ID)+"/tickets/"+itoa64(ticket.ID),
			strings.NewReader(body),
		)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("ticket body %s: expected 400, got %d", body, resp.StatusCode)
		}
	}

	req, _ := http.NewRequest("PUT", srv.URL+"/api/organizers/1", strings.NewReader("{\"name\":\" \"}"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty organizer name: expected 400, got %d", resp.StatusCode)
	}
}

func TestAdminAuthMiddleware(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "test-token-123")
	defer func() { t.Setenv("ADMIN_TOKEN", "") }()

	_, _, srv := setupTestServer(t)

	t.Run("session validates configured token", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/admin/session", nil)
		req.Header.Set("X-Admin-Token", "test-token-123")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var payload struct {
			Data dto.AdminSessionResponse `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if !payload.Data.Authenticated {
			t.Fatal("valid admin token was not confirmed")
		}
	})

	t.Run("session rejects invalid token", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/admin/session", nil)
		req.Header.Set("X-Admin-Token", "wrong-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("with valid token", func(t *testing.T) {
		req, _ := http.NewRequest("POST", srv.URL+"/api/events",
			strings.NewReader(`{"organizer_id":1,"title":"测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Admin-Token", "test-token-123")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.StatusCode)
		}
	})

	t.Run("without token", func(t *testing.T) {
		req, _ := http.NewRequest("POST", srv.URL+"/api/events",
			strings.NewReader(`{"organizer_id":1,"title":"测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})
}

func TestCORSHeaders(t *testing.T) {
	_, _, srv := setupTestServer(t)
	resp, err := http.Get(srv.URL + "/api/events")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected CORS origin *, got %q", resp.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestResponseFormat(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/events")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Fatalf("expected JSON content type, got %q", contentType)
	}

	var apiResp model.APIResp
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("expected valid JSON response: %v", err)
	}
	if apiResp.Code != 200 {
		t.Fatalf("expected code 200, got %d", apiResp.Code)
	}
}

func TestFilterEventsByStatusHandler(t *testing.T) {
	store, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"已发布活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	draft := "draft"
	store.UpdateEvent(id, model.UpdateEventReq{Status: &draft})

	resp, _ := http.Get(srv.URL + "/api/events?status=published")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestSearchEventsByKeywordHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	body := `{"organizer_id":1,"title":"Go 入门讲座","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(body))

	resp, _ := http.Get(srv.URL + "/api/events?q=Go")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var apiResp model.APIResp
	json.NewDecoder(resp.Body).Decode(&apiResp)
	resp.Body.Close()
	events := apiResp.Data.([]interface{})
	if len(events) != 1 {
		t.Fatalf("expected 1 event matching 'Go', got %d", len(events))
	}
}

func TestFilterEventsByPriceTypeHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	freeBody := `{"organizer_id":1,"title":"免费活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(freeBody))

	paidBody := `{"organizer_id":1,"title":"付费活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":99.9}`
	http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(paidBody))

	resp, _ := http.Get(srv.URL + "/api/events?price_type=paid")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var apiResp model.APIResp
	json.NewDecoder(resp.Body).Decode(&apiResp)
	resp.Body.Close()
	events := apiResp.Data.([]interface{})
	if len(events) != 1 {
		t.Fatalf("expected 1 paid event, got %d", len(events))
	}
}

func TestHealthHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var apiResp model.APIResp
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}
	resp.Body.Close()

	if apiResp.Code != 200 {
		t.Fatalf("expected code 200, got %d", apiResp.Code)
	}

	data, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be object, got %T", apiResp.Data)
	}

	if data["status"] != "ok" {
		t.Errorf("expected status ok, got %v", data["status"])
	}
	if data["version"] != "dev" {
		t.Errorf("expected version dev, got %v", data["version"])
	}
	if data["db"] != "connected" {
		t.Errorf("expected db connected, got %v", data["db"])
	}
	uptime, ok := data["uptime_seconds"].(float64)
	if !ok {
		t.Errorf("expected uptime_seconds to be number, got %T", data["uptime_seconds"])
	} else if uptime < 0 {
		t.Errorf("expected uptime_seconds >= 0, got %v", uptime)
	}
}

func TestHealthHandlerResponseStructure(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	h := newTestHandler(s, config.Load())

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	h.HealthHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var apiResp model.APIResp
	json.NewDecoder(resp.Body).Decode(&apiResp)

	data, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be object, got %T", apiResp.Data)
	}

	requiredFields := []string{"status", "version", "uptime_seconds", "db"}
	for _, field := range requiredFields {
		if _, exists := data[field]; !exists {
			t.Errorf("expected field %q in response", field)
		}
	}
}

func TestHandlerUsesStartupConfigSnapshot(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	cfg := config.Load()
	cfg.Version = "injected-version"
	cfg.CORSOrigin = "https://startup.example.com"
	h := newTestHandler(s, cfg)
	router := NewRouter(h, nil)

	t.Setenv("VERSION", "changed-after-startup")
	t.Setenv("CORS_ORIGIN", "https://changed.example.com")

	healthReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthResp := httptest.NewRecorder()
	router.ServeHTTP(healthResp, healthReq)
	var apiResp model.APIResp
	if err := json.NewDecoder(healthResp.Body).Decode(&apiResp); err != nil {
		t.Fatal(err)
	}
	data := apiResp.Data.(map[string]interface{})
	if data["version"] != "injected-version" {
		t.Fatalf("expected injected version, got %v", data["version"])
	}
	if origin := healthResp.Header().Get("Access-Control-Allow-Origin"); origin != "https://startup.example.com" {
		t.Fatalf("expected injected CORS origin, got %q", origin)
	}
}

func TestGenerateTokenUsesInjectedClockAndSigner(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	cfg := config.Load()
	cfg.JWTExpireHours = 12
	fixedTime := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	businessClock := testClock{now: fixedTime}
	tokens := &recordingTokenManager{token: "signed-by-test-manager"}
	h := NewHandler(s, cfg, Dependencies{
		Clock:          businessClock,
		Tokens:         tokens,
		Registrations:  service.NewRegistrationService(s, businessClock, 24*time.Hour, identifier.CryptoCredentialGenerator{}),
		Discussions:    service.NewDiscussionService(s),
		Admissions:     service.NewAdmissionService(s, businessClock),
		Authentication: service.NewAuthenticationService(s, businessClock, identifier.CryptoResetTokenGenerator{}, notification.DiscardPasswordResetSender{}, cfg.PublicBaseURL, 30*time.Minute, 30*time.Minute),
		Moderation:     service.NewContentModerationService(s, businessClock),
		Notifications:  service.NewNotificationService(s, businessClock, 24*time.Hour),
		Organizations:  service.NewOrganizationAuthorizationService(s),
		Operations:     service.NewOrganizationOperationsService(s, businessClock),
	})
	user := &model.User{ID: 7, Name: "注入用户", Contact: "injected@example.com"}

	token, err := h.generateToken(user)
	if err != nil {
		t.Fatal(err)
	}
	if token != tokens.token || tokens.user != user || !tokens.issuedAt.Equal(fixedTime) || tokens.ttl != 12*time.Hour {
		t.Fatalf("injected dependencies were not used: %+v", tokens)
	}
}

func TestResponseWriterCapturesStatusCode(t *testing.T) {
	rw := &responseWriter{ResponseWriter: httptest.NewRecorder(), statusCode: http.StatusOK}

	if rw.statusCode != http.StatusOK {
		t.Fatalf("expected initial status %d, got %d", http.StatusOK, rw.statusCode)
	}

	rw.WriteHeader(http.StatusNotFound)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rw.statusCode)
	}

	rw.WriteHeader(http.StatusInternalServerError)
	if rw.statusCode != http.StatusInternalServerError {
		t.Errorf("expected status %d after overwrite, got %d", http.StatusInternalServerError, rw.statusCode)
	}
}

func TestLoggingMiddlewarePassesThrough(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"code":201}`))
	})

	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"data":"test"}`))
	req.Header.Set("Content-Type", "application/json")

	var buf bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	defer slog.SetDefault(originalLogger)

	w := httptest.NewRecorder()
	LoggingMiddleware(next).ServeHTTP(w, req)

	result := w.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", result.StatusCode)
	}

	if !bytes.Contains(buf.Bytes(), []byte(`"status":201`)) {
		t.Errorf("expected log to contain status 201, got: %s", buf.String())
	}
}

func TestHealthHandlerDBDisconnected(t *testing.T) {
	s := mustNewStore(t)
	h := newTestHandler(s, config.Load())
	s.Close()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	h.HealthHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var apiResp model.APIResp
	json.NewDecoder(resp.Body).Decode(&apiResp)

	data := apiResp.Data.(map[string]interface{})
	if data["status"] != "degraded" {
		t.Errorf("expected status degraded, got %v", data["status"])
	}
	if data["db"] != "disconnected" {
		t.Errorf("expected db disconnected, got %v", data["db"])
	}
	if _, exists := data["db_error"]; exists {
		t.Errorf("database error details must not be exposed")
	}
}

func TestStrictJSONRequestBoundary(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	h := newTestHandler(s, config.Load())

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "unknown field",
			body:       `{"name":"测试","contact":"test@example.com","password":"123456","admin":true}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_JSON",
		},
		{
			name:       "trailing object",
			body:       `{"name":"测试","contact":"test@example.com","password":"123456"}{}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_JSON",
		},
		{
			name:       "body too large",
			body:       `{"name":"` + strings.Repeat("a", int(maxRequestBodyBytes)) + `","contact":"test@example.com","password":"123456"}`,
			wantStatus: http.StatusRequestEntityTooLarge,
			wantCode:   "REQUEST_TOO_LARGE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			h.RegisterUser(w, req)

			resp := w.Result()
			defer resp.Body.Close()
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			var apiResp model.APIResp
			if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if apiResp.ErrorCode != tt.wantCode {
				t.Fatalf("expected error code %s, got %s", tt.wantCode, apiResp.ErrorCode)
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	SecurityHeaders(next).ServeHTTP(w, req)

	for header, expected := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Permissions-Policy":     "camera=(), microphone=(), geolocation=()",
	} {
		if got := w.Header().Get(header); got != expected {
			t.Errorf("expected %s=%q, got %q", header, expected, got)
		}
	}
	if got := w.Header().Get("Content-Security-Policy"); !strings.Contains(got, "frame-ancestors 'none'") {
		t.Errorf("unexpected Content-Security-Policy: %q", got)
	}
}

func TestInternalErrorsDoNotLeak(t *testing.T) {
	s := mustNewStore(t)
	h := newTestHandler(s, config.Load())
	_ = s.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	h.ListEvents(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	var apiResp model.APIResp
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if apiResp.ErrorCode != "INTERNAL_ERROR" || apiResp.Message != "服务器内部错误" {
		t.Fatalf("unexpected public error: %+v", apiResp)
	}
	if strings.Contains(strings.ToLower(apiResp.Message), "sql") || strings.Contains(strings.ToLower(apiResp.Message), "database") {
		t.Fatalf("internal details leaked: %q", apiResp.Message)
	}
}

func TestMiddlewareChainOrder(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	handler := newTestHandler(s, config.Load())

	_ = s.CreateOrganizer(&model.Organizer{Name: "测试门店"})

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/events", AdminAuth(http.HandlerFunc(handler.CreateEvent), "test-token").ServeHTTP)

	server := httptest.NewServer(LoggingMiddleware(CORS(mux, "*")))
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL+"/api/events",
		strings.NewReader(`{"organizer_id":1,"title":"测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", "test-token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected CORS header *, got %q", resp.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestErrorResponseFormat(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, _ := http.Get(srv.URL + "/api/events/999")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	var apiResp model.APIResp
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}

	if apiResp.Code != 404 {
		t.Errorf("expected code 404, got %d", apiResp.Code)
	}
	if apiResp.ErrorCode != "EVENT_NOT_FOUND" {
		t.Errorf("expected EVENT_NOT_FOUND, got %q", apiResp.ErrorCode)
	}
	if apiResp.Message == "" {
		t.Errorf("expected non-empty message, got %q", apiResp.Message)
	}
	if apiResp.Data != nil {
		t.Errorf("expected nil data for error response, got %v", apiResp.Data)
	}
}

func TestSameHTTPStatusUsesDistinctBusinessErrorCodes(t *testing.T) {
	invalidUserToken := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	request.Header.Set("Authorization", "Basic invalid")
	UserAuth(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("invalid user token reached handler")
	}), appauth.NewJWTManager("test-secret")).ServeHTTP(invalidUserToken, request)

	invalidAdminToken := httptest.NewRecorder()
	AdminAuth(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("invalid admin token reached handler")
	}), "admin-secret").ServeHTTP(invalidAdminToken, httptest.NewRequest(http.MethodGet, "/api/events", nil))

	responses := []struct {
		name     string
		recorder *httptest.ResponseRecorder
		wantCode string
	}{
		{"user token", invalidUserToken, "USER_TOKEN_INVALID"},
		{"admin token", invalidAdminToken, "ADMIN_AUTH_INVALID"},
	}
	for _, tt := range responses {
		t.Run(tt.name, func(t *testing.T) {
			if tt.recorder.Code != http.StatusUnauthorized {
				t.Fatalf("expected HTTP 401, got %d", tt.recorder.Code)
			}
			var response model.APIResp
			if err := json.NewDecoder(tt.recorder.Body).Decode(&response); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("legacy numeric code changed: got %d", response.Code)
			}
			if response.ErrorCode != tt.wantCode {
				t.Fatalf("expected %s, got %s", tt.wantCode, response.ErrorCode)
			}
		})
	}
}

func TestAPIFallbackReturnsStructuredErrors(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	router := NewRouter(newTestHandler(s, config.Load()), nil)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantCode   string
	}{
		{"unknown route", http.MethodGet, "/api/v1/unknown", http.StatusNotFound, "API_ROUTE_NOT_FOUND"},
		{"unsupported method", http.MethodPatch, "/api/v1/events/1", http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED"},
		{"OpenAPI unsupported method", http.MethodPost, "/api/v1/openapi.json", http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED"},
		{"legacy unknown route", http.MethodGet, "/api/unknown", http.StatusNotFound, "API_ROUTE_NOT_FOUND"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(tt.method, tt.path, nil))
			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, recorder.Code)
			}
			var response model.APIResp
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("expected JSON error response: %v", err)
			}
			if response.Code != tt.wantStatus || response.ErrorCode != tt.wantCode {
				t.Fatalf("unexpected error response: %+v", response)
			}
		})
	}
}

func TestCORSOriginEnvConfig(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(CORS(mux, "https://example.com"))
	defer server.Close()

	resp, err := http.Get(server.URL + "/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("expected CORS origin https://example.com, got %q", resp.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestSearchKeywordBoundary(t *testing.T) {
	_, _, srv := setupTestServer(t)

	body := `{"organizer_id":1,"title":"Test Event","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(body))

	tests := []struct {
		name  string
		query string
	}{
		{"empty query", srv.URL + "/api/events?q="},
		{"no query param", srv.URL + "/api/events"},
		{"special characters", srv.URL + "/api/events?q=" + url.QueryEscape("!@#$%")},
		{"long keyword", srv.URL + "/api/events?q=" + strings.Repeat("a", 100)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(tt.query)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200, got %d for case %q", resp.StatusCode, tt.name)
			}
		})
	}
}

func TestCORSPreflightRequest(t *testing.T) {
	_, _, srv := setupTestServer(t)

	req, _ := http.NewRequest("OPTIONS", srv.URL+"/api/events", nil)
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("preflight request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204 for preflight, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Access-Control-Allow-Methods") == "" {
		t.Errorf("expected Access-Control-Allow-Methods header")
	}
}

func TestLoggingMiddlewareIPExtraction(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	var buf bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	defer slog.SetDefault(originalLogger)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")

	w := httptest.NewRecorder()
	LoggingMiddleware(next).ServeHTTP(w, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, `"ip":"192.168.1.100"`) {
		t.Errorf("expected IP 192.168.1.100 in log, got: %s", logOutput)
	}
}

func TestGetTicketHandler(t *testing.T) {
	s, h, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"获取门票测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	eid := int64(eventData["id"].(float64))

	ticket := &model.Ticket{EventID: eid, Name: "测试票", Price: 10, Stock: 5}
	if err := s.CreateTicket(ticket); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/events/"+itoa64(eid)+"/tickets/"+itoa64(ticket.ID), nil)
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/events/{id}/tickets/{ticketId}", h.GetTicket)
	mux.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var apiResp model.APIResp
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Code != 200 {
		t.Errorf("expected code 200, got %d", apiResp.Code)
	}
}

func TestGetTicketNotFound(t *testing.T) {
	_, _, srv := setupTestServer(t)

	req, _ := http.NewRequest("GET", srv.URL+"/api/events/999/tickets/999", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestCancelRegistrationHandlerWithJWT(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"取消JWT","event_time":"2099-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	resp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	ed := created.Data.(map[string]interface{})
	eid := int64(ed["id"].(float64))

	registration := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(eid)+"/register", `{}`, 1, "测试人", "cancel_jwt@test.com")
	registration.Body.Close()

	resp = doUserJSON(t, http.MethodDelete, srv.URL+"/api/events/"+itoa64(eid)+"/register", `{}`, 1, "测试人", "cancel_jwt@test.com")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRegistrationStatusUsesAuthenticatedUser(t *testing.T) {
	s, _, srv := setupTestServer(t)
	event := createStoreEvent(t, s, "报名状态")
	user := &model.User{Name: "已报名用户", Contact: "registered-status@test.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	if err := s.Register(&model.Registration{
		EventID: event.ID,
		UserID:  &user.ID,
		Name:    "已报名用户",
		Contact: "registered-status@test.com",
	}); err != nil {
		t.Fatal(err)
	}

	url := srv.URL + "/api/events/" + itoa64(event.ID) + "/registration"

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous status: expected 401, got %d", resp.StatusCode)
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+makeTestJWT(t, user.ID, "已报名用户", "registered-status@test.com"))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var apiResp model.APIResp
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("registered status: expected 200, got %d", resp.StatusCode)
	}
	data := apiResp.Data.(map[string]interface{})
	if registered, _ := data["registered"].(bool); !registered {
		t.Fatal("expected registered=true")
	}

	req, _ = http.NewRequest("GET", url, nil)
	other := &model.User{Name: "其他用户", Contact: "other-status@test.com", PasswordHash: "hash"}
	if err := s.CreateUser(other); err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+makeTestJWT(t, other.ID, "其他用户", "other-status@test.com"))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	apiResp = model.APIResp{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	data = apiResp.Data.(map[string]interface{})
	if registered, _ := data["registered"].(bool); registered {
		t.Fatal("expected registered=false for another user")
	}
}

func TestRegistrationListRequiresAdminToken(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "registration-admin-token")
	s, _, srv := setupTestServer(t)
	event := createStoreEvent(t, s, "报名名单权限")
	if err := s.Register(&model.Registration{
		EventID: event.ID,
		Name:    "隐私用户",
		Contact: "private-registration@test.com",
	}); err != nil {
		t.Fatal(err)
	}

	url := srv.URL + "/api/events/" + itoa64(event.ID) + "/registrations"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous registration list: expected 401, got %d", resp.StatusCode)
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Admin-Token", "registration-admin-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin registration list: expected 200, got %d", resp.StatusCode)
	}
}

func TestCancelRegistrationRejectsContactBody(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"取消Body","event_time":"2099-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	resp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	ed := created.Data.(map[string]interface{})
	eid := int64(ed["id"].(float64))

	registration := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/"+itoa64(eid)+"/register", `{}`, 1, "张三", "cancel_body@test.com")
	registration.Body.Close()

	resp = doUserJSON(t, http.MethodDelete, srv.URL+"/api/events/"+itoa64(eid)+"/register",
		`{"contact":"cancel_body@test.com"}`, 1, "张三", "cancel_body@test.com")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCancelRegistrationHandlerNoContact(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"取消NoContact","event_time":"2099-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	resp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	ed := created.Data.(map[string]interface{})
	eid := int64(ed["id"].(float64))

	req, _ := http.NewRequest("DELETE", srv.URL+"/api/events/"+itoa64(eid)+"/register",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestUserAuthNoToken(t *testing.T) {
	var captured bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = true
		if _, ok := model.UserFromContext(r.Context()); ok {
			t.Error("expected no user claims without token")
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	UserAuth(next, appauth.NewJWTManager(config.DefaultJWTSecret)).ServeHTTP(w, req)

	if !captured {
		t.Error("next handler not called")
	}
}

func TestUserAuthWithValidToken(t *testing.T) {
	token := makeTestJWT(t, 1, "测试", "test@auth.com")

	var captured bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = true
		claims, ok := model.UserFromContext(r.Context())
		if !ok {
			t.Fatal("expected user claims")
		}
		if claims.Name != "测试" {
			t.Errorf("expected name 测试, got %s", claims.Name)
		}
		if claims.Contact != "test@auth.com" {
			t.Errorf("expected contact test@auth.com, got %s", claims.Contact)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	UserAuth(next, appauth.NewJWTManager(config.DefaultJWTSecret)).ServeHTTP(w, req)

	if !captured {
		t.Error("next handler not called with valid token")
	}
}

func TestUserAuthWithInvalidToken(t *testing.T) {
	var captured bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()
	UserAuth(next, appauth.NewJWTManager(config.DefaultJWTSecret)).ServeHTTP(w, req)

	if captured {
		t.Error("next handler must not be called with invalid token")
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUpdateOrganizerHandlerNotFound(t *testing.T) {
	_, _, srv := setupTestServer(t)

	body := `{"name":"不存在的"}`
	req, _ := http.NewRequest("PUT", srv.URL+"/api/organizers/999", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteOrganizerHandlerNotFound(t *testing.T) {
	_, _, srv := setupTestServer(t)

	req, _ := http.NewRequest("DELETE", srv.URL+"/api/organizers/999", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestRegisterUserEmptyBody(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Post(srv.URL+"/api/auth/register", "application/json", strings.NewReader(`{"name":"","contact":"","password":""}`))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLoginHandlerBadJSON(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Post(srv.URL+"/api/auth/login", "application/json", strings.NewReader(`not json`))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateEventWithAuth(t *testing.T) {
	_, _, srv := setupTestServer(t)

	body := `{"organizer_id":1,"title":"管理员创建","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	req, _ := http.NewRequest("POST", srv.URL+"/api/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestGetEventNotFoundHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/events/999")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetEventInvalidIDHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/events/abc")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestListEventsInvalidPage(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/events?page=-1")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRegisterEmptyBody(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/1/register", `{}`, 1, "测试", "test@example.com")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent event, got %d", resp.StatusCode)
	}
}

func TestRegisterBadJSON(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/1/register", `bad`, 1, "测试", "test@example.com")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent event, got %d", resp.StatusCode)
	}
}

func TestListRegistrationsInvalidID(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/events/abc/registrations")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreatePostHandlerBadJSON(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/1/posts", `bad json`, 1, "测试", "test@example.com")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent event, got %d", resp.StatusCode)
	}
}

func TestCreateReplyHandlerBadJSON(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/1/posts/1/replies", `bad`, 1, "测试", "test@example.com")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent event, got %d", resp.StatusCode)
	}
}

func TestListPostsInvalidID(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/events/abc/posts")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetPostInvalidID(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/events/1/posts/abc")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateTicketInvalidBody(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Post(srv.URL+"/api/events/1/tickets", "application/json", strings.NewReader(`bad`))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent event, got %d", resp.StatusCode)
	}
}

func TestListTicketsInvalidID(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/events/abc/tickets")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateOrganizerHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	body := `{"name":"新门店","description":"测试","tags":"教育,咖啡"}`
	resp, err := http.Post(srv.URL+"/api/organizers", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestCreateOrganizerEmptyName(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Post(srv.URL+"/api/organizers", "application/json", strings.NewReader(`{"name":""}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestListOrganizersHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/organizers")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var apiResp model.APIResp
	json.NewDecoder(resp.Body).Decode(&apiResp)
	data, _ := json.Marshal(apiResp.Data)
	var orgs []model.Organizer
	json.Unmarshal(data, &orgs)
	if len(orgs) != 1 {
		t.Fatalf("expected 1 organizer, got %d", len(orgs))
	}
}

func TestGetOrganizerHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/organizers/1")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetOrganizerHandlerNotFound(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/organizers/999")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestRegisterUserHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	body := `{"name":"测试用户","contact":"reg@test.com","password":"12345678"}`
	resp, err := http.Post(srv.URL+"/api/auth/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var apiResp model.APIResp
	json.NewDecoder(resp.Body).Decode(&apiResp)
	data, _ := json.Marshal(apiResp.Data)
	var loginResp dto.LoginResponse
	json.Unmarshal(data, &loginResp)
	if loginResp.Token == "" {
		t.Fatal("expected token in response")
	}
}

func TestRegisterUserDuplicate(t *testing.T) {
	_, _, srv := setupTestServer(t)

	body := `{"name":"重复","contact":"dup@test.com","password":"12345678"}`
	http.Post(srv.URL+"/api/auth/register", "application/json", strings.NewReader(body))
	resp, err := http.Post(srv.URL+"/api/auth/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestRegisterUserShortPassword(t *testing.T) {
	_, _, srv := setupTestServer(t)

	body := `{"name":"短密码","contact":"short@test.com","password":"12"}`
	resp, err := http.Post(srv.URL+"/api/auth/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLoginHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	http.Post(srv.URL+"/api/auth/register", "application/json",
		strings.NewReader(`{"name":"登录测试","contact":"login@test.com","password":"abcdefgh"}`))

	body := `{"contact":"login@test.com","password":"abcdefgh"}`
	resp, err := http.Post(srv.URL+"/api/auth/login", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var apiResp model.APIResp
	json.NewDecoder(resp.Body).Decode(&apiResp)
	data, _ := json.Marshal(apiResp.Data)
	var loginResp dto.LoginResponse
	json.Unmarshal(data, &loginResp)
	if loginResp.Token == "" {
		t.Fatal("expected token in response")
	}
}

func TestLoginHandlerWrongPassword(t *testing.T) {
	_, _, srv := setupTestServer(t)

	http.Post(srv.URL+"/api/auth/register", "application/json",
		strings.NewReader(`{"name":"错误","contact":"wrong@test.com","password":"pass1pass"}`))

	body := `{"contact":"wrong@test.com","password":"wrongpass"}`
	resp, err := http.Post(srv.URL+"/api/auth/login", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestLoginHandlerNotFound(t *testing.T) {
	_, _, srv := setupTestServer(t)

	body := `{"contact":"noexist@test.com","password":"anything"}`
	resp, err := http.Post(srv.URL+"/api/auth/login", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestPasswordResetAndLogoutHTTPJourney(t *testing.T) {
	s, h, srv := setupTestServer(t)
	sender := &recordingPasswordResetSender{}
	generator := &sequenceResetTokenGenerator{tokens: []string{"known-reset-token", "throttled-reset-token", "unknown-reset-token"}}
	h.authentication = service.NewAuthenticationService(
		s, clock.System{}, generator, sender, "https://events.example.com", 30*time.Minute, 30*time.Minute,
	)

	registerResponse, err := http.Post(srv.URL+"/api/v1/auth/register", "application/json", strings.NewReader(
		`{"name":"重置用户","contact":"reset-http@example.com","password":"old-password"}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	var registerPayload model.APIResp
	if err := json.NewDecoder(registerResponse.Body).Decode(&registerPayload); err != nil {
		t.Fatal(err)
	}
	registerResponse.Body.Close()
	if registerResponse.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", registerResponse.StatusCode)
	}
	registerData := registerPayload.Data.(map[string]interface{})
	oldToken := registerData["token"].(string)

	requestReset := func(contact string) model.APIResp {
		t.Helper()
		response, err := http.Post(srv.URL+"/api/v1/auth/password-reset/request", "application/json", strings.NewReader(
			`{"contact":"`+contact+`"}`,
		))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusAccepted {
			t.Fatalf("reset request: expected 202, got %d", response.StatusCode)
		}
		var payload model.APIResp
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		return payload
	}
	knownResponse := requestReset("reset-http@example.com")
	throttledResponse := requestReset("reset-http@example.com")
	unknownResponse := requestReset("missing-http@example.com")
	if knownResponse.Message != throttledResponse.Message || knownResponse.Message != unknownResponse.Message || sender.calls != 1 || sender.contact != "reset-http@example.com" {
		t.Fatalf("reset request leaked account state: known=%+v throttled=%+v unknown=%+v sender=%+v", knownResponse, throttledResponse, unknownResponse, sender)
	}
	resetURL, err := url.Parse(sender.resetURL)
	if err != nil || resetURL.Query().Get("token") != "known-reset-token" {
		t.Fatalf("unexpected reset URL %q: %v", sender.resetURL, err)
	}

	confirmBody := `{"token":"known-reset-token","password":"new-password"}`
	confirmResponse, err := http.Post(srv.URL+"/api/v1/auth/password-reset/confirm", "application/json", strings.NewReader(confirmBody))
	if err != nil {
		t.Fatal(err)
	}
	confirmResponse.Body.Close()
	if confirmResponse.StatusCode != http.StatusOK {
		t.Fatalf("confirm reset: expected 200, got %d", confirmResponse.StatusCode)
	}

	reuseResponse, err := http.Post(srv.URL+"/api/v1/auth/password-reset/confirm", "application/json", strings.NewReader(confirmBody))
	if err != nil {
		t.Fatal(err)
	}
	var reusePayload model.APIResp
	if err := json.NewDecoder(reuseResponse.Body).Decode(&reusePayload); err != nil {
		t.Fatal(err)
	}
	reuseResponse.Body.Close()
	if reuseResponse.StatusCode != http.StatusBadRequest || reusePayload.ErrorCode != string(api.CodePasswordResetInvalid) {
		t.Fatalf("consumed reset token accepted: status=%d payload=%+v", reuseResponse.StatusCode, reusePayload)
	}

	oldSessionRequest, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/me/activities", nil)
	oldSessionRequest.Header.Set("Authorization", "Bearer "+oldToken)
	oldSessionResponse, err := http.DefaultClient.Do(oldSessionRequest)
	if err != nil {
		t.Fatal(err)
	}
	oldSessionResponse.Body.Close()
	if oldSessionResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("password reset did not revoke old JWT: %d", oldSessionResponse.StatusCode)
	}

	login := func(password string) (*http.Response, model.APIResp) {
		t.Helper()
		response, err := http.Post(srv.URL+"/api/v1/auth/login", "application/json", strings.NewReader(
			`{"contact":"reset-http@example.com","password":"`+password+`"}`,
		))
		if err != nil {
			t.Fatal(err)
		}
		var payload model.APIResp
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		return response, payload
	}
	oldLogin, _ := login("old-password")
	if oldLogin.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old password accepted: %d", oldLogin.StatusCode)
	}
	newLogin, newLoginPayload := login("new-password")
	if newLogin.StatusCode != http.StatusOK {
		t.Fatalf("new password rejected: %d", newLogin.StatusCode)
	}
	newToken := newLoginPayload.Data.(map[string]interface{})["token"].(string)

	logoutRequest, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/auth/logout", nil)
	logoutRequest.Header.Set("Authorization", "Bearer "+newToken)
	logoutResponse, err := http.DefaultClient.Do(logoutRequest)
	if err != nil {
		t.Fatal(err)
	}
	logoutResponse.Body.Close()
	if logoutResponse.StatusCode != http.StatusOK {
		t.Fatalf("logout: expected 200, got %d", logoutResponse.StatusCode)
	}

	revokedRequest, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/me/activities", nil)
	revokedRequest.Header.Set("Authorization", "Bearer "+newToken)
	revokedResponse, err := http.DefaultClient.Do(revokedRequest)
	if err != nil {
		t.Fatal(err)
	}
	revokedResponse.Body.Close()
	if revokedResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("logout did not revoke JWT: %d", revokedResponse.StatusCode)
	}
}

func TestRecoveryEmailBindingHTTPJourney(t *testing.T) {
	s, h, srv := setupTestServer(t)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("legacy-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := &model.User{Name: "历史手机用户", Contact: "13800138000", PasswordHash: string(passwordHash)}
	if err := s.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	sender := &recordingPasswordResetSender{}
	generator := &sequenceResetTokenGenerator{tokens: []string{
		"known-recovery-token", "pre-confirm-reset-token", "post-confirm-reset-token",
	}}
	h.authentication = service.NewAuthenticationService(
		s, clock.System{}, generator, sender, "https://events.example.com", 30*time.Minute, 20*time.Minute,
	)
	oldToken := makeTestJWT(t, user.ID, user.Name, user.Contact)
	bindRequest, _ := http.NewRequest(
		http.MethodPost, srv.URL+"/api/v1/me/recovery-email/request",
		strings.NewReader(`{"email":"Recovery@Example.COM","password":"legacy-password"}`),
	)
	bindRequest.Header.Set("Content-Type", "application/json")
	bindRequest.Header.Set("Authorization", "Bearer "+oldToken)
	bindResponse, err := http.DefaultClient.Do(bindRequest)
	if err != nil {
		t.Fatal(err)
	}
	bindResponse.Body.Close()
	if bindResponse.StatusCode != http.StatusAccepted || sender.verificationCalls != 1 || sender.verificationEmail != "recovery@example.com" {
		t.Fatalf("request recovery email: status=%d sender=%+v", bindResponse.StatusCode, sender)
	}
	verificationURL, err := url.Parse(sender.verificationURL)
	if err != nil || verificationURL.Query().Get("token") != "known-recovery-token" {
		t.Fatalf("unexpected verification URL %q: %v", sender.verificationURL, err)
	}
	preConfirmReset, err := http.Post(
		srv.URL+"/api/v1/auth/password-reset/request", "application/json",
		strings.NewReader(`{"contact":"recovery@example.com"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	preConfirmReset.Body.Close()
	if preConfirmReset.StatusCode != http.StatusAccepted || sender.calls != 0 {
		t.Fatalf("unverified recovery email received reset: status=%d sender=%+v", preConfirmReset.StatusCode, sender)
	}
	confirmResponse, err := http.Post(
		srv.URL+"/api/v1/auth/recovery-email/confirm", "application/json",
		strings.NewReader(`{"token":"known-recovery-token"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmResponse.Body.Close()
	if confirmResponse.StatusCode != http.StatusOK {
		t.Fatalf("confirm recovery email: expected 200, got %d", confirmResponse.StatusCode)
	}
	oldSessionRequest, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/me/activities", nil)
	oldSessionRequest.Header.Set("Authorization", "Bearer "+oldToken)
	oldSessionResponse, err := http.DefaultClient.Do(oldSessionRequest)
	if err != nil {
		t.Fatal(err)
	}
	oldSessionResponse.Body.Close()
	if oldSessionResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("recovery email confirmation did not revoke old JWT: %d", oldSessionResponse.StatusCode)
	}
	postConfirmReset, err := http.Post(
		srv.URL+"/api/v1/auth/password-reset/request", "application/json",
		strings.NewReader(`{"contact":"recovery@example.com"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	postConfirmReset.Body.Close()
	if postConfirmReset.StatusCode != http.StatusAccepted || sender.calls != 1 || sender.contact != "recovery@example.com" {
		t.Fatalf("verified recovery email did not receive reset: status=%d sender=%+v", postConfirmReset.StatusCode, sender)
	}
	loginResponse, err := http.Post(
		srv.URL+"/api/v1/auth/login", "application/json",
		strings.NewReader(`{"contact":"13800138000","password":"legacy-password"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	var loginPayload model.APIResp
	if err := json.NewDecoder(loginResponse.Body).Decode(&loginPayload); err != nil {
		t.Fatal(err)
	}
	loginResponse.Body.Close()
	if loginResponse.StatusCode != http.StatusOK {
		t.Fatalf("legacy phone login changed after binding: %d", loginResponse.StatusCode)
	}
	userData := loginPayload.Data.(map[string]interface{})["user"].(map[string]interface{})
	if userData["contact"] != "13800138000" || userData["recovery_email"] != "recovery@example.com" {
		t.Fatalf("unexpected bound login identity: %#v", userData)
	}
}

func TestGetTicketInvalidID(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	h := newTestHandler(s, config.Load())

	req := httptest.NewRequest("GET", "/api/events/abc/tickets/1", nil)
	w := httptest.NewRecorder()
	h.GetTicket(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUpdateTicketNotFound(t *testing.T) {
	_, _, srv := setupTestServer(t)

	req, _ := http.NewRequest("PUT", srv.URL+"/api/events/999/tickets/999",
		strings.NewReader(`{"name":"新名称"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateTicketInvalidBody(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	h := newTestHandler(s, config.Load())
	ticket := &model.Ticket{EventID: 1, Name: "原始", Price: 10, Stock: 5}
	s.CreateTicket(ticket)

	req := httptest.NewRequest("PUT", "/api/events/1/tickets/"+itoa64(ticket.ID),
		strings.NewReader(`invalid json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.UpdateTicket(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestDeleteTicketNotFound(t *testing.T) {
	_, _, srv := setupTestServer(t)

	req, _ := http.NewRequest("DELETE", srv.URL+"/api/events/999/tickets/999", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteTicketInvalidID(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	h := newTestHandler(s, config.Load())

	req := httptest.NewRequest("DELETE", "/api/events/abc/tickets/1", nil)
	w := httptest.NewRecorder()
	h.DeleteTicket(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestListTicketsEmpty(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"无门票活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	eid := int64(eventData["id"].(float64))

	resp, _ := http.Get(srv.URL + "/api/events/" + itoa64(eid) + "/tickets")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	var apiResp model.APIResp
	json.NewDecoder(resp.Body).Decode(&apiResp)
	data, ok := apiResp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected array data, got %T", apiResp.Data)
	}
	if len(data) != 0 {
		t.Errorf("expected 0 tickets, got %d", len(data))
	}
}

func TestCreateReplyNotFound(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp := doUserJSON(t, http.MethodPost, srv.URL+"/api/events/999/posts/1/replies", `{"content":"回复"}`, 1, "测试", "test@test.com")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestCreateReplyInvalidBody(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	h := newTestHandler(s, config.Load())
	user := &model.User{Name: "testuser", Contact: "test@test.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	e := &model.Event{Title: "test", EventTime: "2026-12-31T18:00:00+08:00", Location: "线上", Capacity: 10, Status: "published"}
	s.CreateEvent(e)
	p := &model.Post{EventID: e.ID, AuthorName: "testuser", Content: "帖子"}
	s.CreatePost(p)

	req := httptest.NewRequest("POST", "/api/events/"+itoa64(e.ID)+"/posts/"+itoa64(p.ID)+"/replies",
		strings.NewReader(`invalid`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+makeTestJWT(t, user.ID, "testuser", "test@test.com"))
	w := httptest.NewRecorder()
	UserAuth(http.HandlerFunc(h.CreateReply), appauth.NewJWTManager(config.DefaultJWTSecret)).ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetPostNotFound(t *testing.T) {
	_, _, srv := setupTestServer(t)

	req, _ := http.NewRequest("GET", srv.URL+"/api/events/999/posts/999", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestListPostsEmpty(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"无帖子活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	eid := int64(eventData["id"].(float64))

	resp, _ := http.Get(srv.URL + "/api/events/" + itoa64(eid) + "/posts")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	var apiResp model.APIResp
	json.NewDecoder(resp.Body).Decode(&apiResp)
	data, ok := apiResp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected array data, got %T", apiResp.Data)
	}
	if len(data) != 0 {
		t.Errorf("expected 0 posts, got %d", len(data))
	}
}

func TestListRegistrationsEmpty(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"organizer_id":1,"title":"无报名活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	eid := int64(eventData["id"].(float64))

	resp, _ := http.Get(srv.URL + "/api/events/" + itoa64(eid) + "/registrations")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	var apiResp model.APIResp
	json.NewDecoder(resp.Body).Decode(&apiResp)
	data, ok := apiResp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected array data, got %T", apiResp.Data)
	}
	if len(data) != 0 {
		t.Errorf("expected 0 registrations, got %d", len(data))
	}
}

func TestUpdateEventInvalidBody(t *testing.T) {
	s := mustNewStore(t)
	defer s.Close()
	h := newTestHandler(s, config.Load())
	e := &model.Event{Title: "test", EventTime: "2026-12-31T18:00:00+08:00", Location: "线上", Capacity: 10, Status: "draft"}
	s.CreateEvent(e)

	req := httptest.NewRequest("PUT", "/api/events/"+itoa64(e.ID), strings.NewReader(`invalid`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.UpdateEvent(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestDeleteEventNotFound(t *testing.T) {
	_, _, srv := setupTestServer(t)

	req, _ := http.NewRequest("DELETE", srv.URL+"/api/events/999", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestTrustedUserIDAuthorizationFlow(t *testing.T) {
	s, _, srv := setupTestServer(t)
	event := createStoreEvent(t, s, "可信身份流程")
	registerURL := srv.URL + "/api/events/" + itoa64(event.ID) + "/register"

	resp, err := http.Post(registerURL, "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous registration: expected 401, got %d", resp.StatusCode)
	}

	resp = doUserJSON(t, http.MethodPost, registerURL, `{}`, 101, "可信用户", "trusted@example.com")
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("authenticated registration: expected 201, got %d", resp.StatusCode)
	}

	registrations, _, err := s.ListRegistrations(event.ID, 0, 0)
	if err != nil || len(registrations) != 1 {
		t.Fatalf("list registrations: registrations=%+v err=%v", registrations, err)
	}
	persistedUser, err := s.GetUserByContact("trusted@example.com")
	if err != nil || persistedUser == nil {
		t.Fatalf("load persisted user: user=%+v err=%v", persistedUser, err)
	}
	if registrations[0].UserID == nil || *registrations[0].UserID != persistedUser.ID {
		t.Fatalf("registration did not persist trusted user ID: %+v", registrations[0])
	}
	if registrations[0].Name != "可信用户" || registrations[0].Contact != "trusted@example.com" || registrations[0].IdentityStatus != model.IdentityStatusVerified {
		t.Fatalf("registration identity was not sourced from JWT: %+v", registrations[0])
	}

	postURL := srv.URL + "/api/events/" + itoa64(event.ID) + "/posts"
	resp = doUserJSON(t, http.MethodPost, postURL, `{"title":"伪造","content":"无权","author_contact":"trusted@example.com"}`, 202, "其他用户", "other@example.com")
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("spoofed author field: expected 400, got %d", resp.StatusCode)
	}
	otherUser, err := s.GetUserByContact("other@example.com")
	if err != nil || otherUser == nil {
		t.Fatalf("load other user: user=%+v err=%v", otherUser, err)
	}
	spoofReq, _ := http.NewRequest(http.MethodPost, postURL, strings.NewReader(`{"title":"仍无权","content":"联系方式不能授权"}`))
	spoofReq.Header.Set("Content-Type", "application/json")
	spoofReq.Header.Set("Authorization", "Bearer "+makeTestJWT(t, otherUser.ID, "伪造名称", "trusted@example.com"))
	resp, err = http.DefaultClient.Do(spoofReq)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("contact fallback must not authorize: expected 403, got %d", resp.StatusCode)
	}

	resp, err = http.Get(srv.URL + "/api/me/registrations")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous my registrations: expected 401, got %d", resp.StatusCode)
	}
	resp = doUserJSON(t, http.MethodGet, srv.URL+"/api/me/registrations", "", 101, "可信用户", "trusted@example.com")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("my registrations: expected 200, got %d", resp.StatusCode)
	}
	var apiResp model.APIResp
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatal(err)
	}
	items, ok := apiResp.Data.([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("expected one my-registration item, got %#v", apiResp.Data)
	}
}

func TestIdentityMigrationReportRequiresAdmin(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "identity-admin-token")
	s, _, srv := setupTestServer(t)
	event := createStoreEvent(t, s, "身份报告")
	user := &model.User{Name: "已验证", Contact: "verified@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	if err := s.Register(&model.Registration{
		EventID: event.ID, UserID: &user.ID, Name: user.Name, Contact: user.Contact,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Register(&model.Registration{
		EventID: event.ID, Name: "待处理", Contact: "legacy@example.com",
	}); err != nil {
		t.Fatal(err)
	}

	url := srv.URL + "/api/admin/identity-migration?legacy_limit=10"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous report: expected 401, got %d", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-Admin-Token", "identity-admin-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin report: expected 200, got %d", resp.StatusCode)
	}
	var apiResp model.APIResp
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatal(err)
	}
	data := apiResp.Data.(map[string]interface{})
	registrations := data["registrations"].(map[string]interface{})
	if registrations["verified"].(float64) != 1 || registrations["legacy"].(float64) != 1 {
		t.Fatalf("unexpected identity report: %#v", registrations)
	}
}

func TestAdmissionAndCheckinHTTPJourney(t *testing.T) {
	_, _, server := setupTestServer(t)

	createResponse, err := http.Post(server.URL+"/api/events", "application/json", strings.NewReader(
		`{"organizer_id":1,"title":"免费核销活动","event_time":"2099-12-31T18:00:00+08:00","location":"现场","capacity":10,"price":0}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	var eventResponse model.APIResp
	if err := json.NewDecoder(createResponse.Body).Decode(&eventResponse); err != nil {
		t.Fatal(err)
	}
	createResponse.Body.Close()
	eventID := int64(eventResponse.Data.(map[string]interface{})["id"].(float64))

	registerResponse := doUserJSON(t, http.MethodPost, server.URL+"/api/events/"+itoa64(eventID)+"/register", `{}`, 0, "凭证用户", "admission-http@example.com")
	if registerResponse.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", registerResponse.StatusCode)
	}
	var registrationResponse model.APIResp
	if err := json.NewDecoder(registerResponse.Body).Decode(&registrationResponse); err != nil {
		t.Fatal(err)
	}
	registerResponse.Body.Close()
	admission := registrationResponse.Data.(map[string]interface{})["admission"].(map[string]interface{})
	credential := admission["credential"].(string)
	if !strings.HasPrefix(credential, service.AdmissionCredentialPrefix) {
		t.Fatalf("unexpected credential payload %q", credential)
	}

	getResponse := doUserJSON(t, http.MethodGet, server.URL+"/api/events/"+itoa64(eventID)+"/admission", "", 0, "凭证用户", "admission-http@example.com")
	if getResponse.StatusCode != http.StatusOK {
		t.Fatalf("get admission: expected 200, got %d", getResponse.StatusCode)
	}
	getResponse.Body.Close()

	listResponse := doUserJSON(t, http.MethodGet, server.URL+"/api/me/admissions", "", 0, "凭证用户", "admission-http@example.com")
	var listPayload model.APIResp
	if err := json.NewDecoder(listResponse.Body).Decode(&listPayload); err != nil {
		t.Fatal(err)
	}
	listResponse.Body.Close()
	if listResponse.StatusCode != http.StatusOK || listPayload.Total == nil || *listPayload.Total != 1 {
		t.Fatalf("unexpected admission list: status=%d payload=%+v", listResponse.StatusCode, listPayload)
	}

	activityResponse := doUserJSON(t, http.MethodGet, server.URL+"/api/v1/me/activities", "", 0, "凭证用户", "admission-http@example.com")
	var activityPayload model.APIResp
	if err := json.NewDecoder(activityResponse.Body).Decode(&activityPayload); err != nil {
		t.Fatal(err)
	}
	activityResponse.Body.Close()
	activities := activityPayload.Data.([]interface{})
	activity := activities[0].(map[string]interface{})
	if activityResponse.StatusCode != http.StatusOK || activityPayload.Total == nil || *activityPayload.Total != 1 || activity["kind"] != model.ActivityKindAdmission || activity["admission"] == nil {
		t.Fatalf("unexpected unified activity list: status=%d payload=%+v", activityResponse.StatusCode, activityPayload)
	}

	checkinBody := `{"credential":"` + credential + `"}`
	first, err := http.Post(server.URL+"/api/events/"+itoa64(eventID)+"/checkins", "application/json", strings.NewReader(checkinBody))
	if err != nil {
		t.Fatal(err)
	}
	var firstPayload model.APIResp
	if err := json.NewDecoder(first.Body).Decode(&firstPayload); err != nil {
		t.Fatal(err)
	}
	first.Body.Close()
	if first.StatusCode != http.StatusCreated || firstPayload.Data.(map[string]interface{})["already_checked_in"].(bool) {
		t.Fatalf("unexpected first checkin: status=%d payload=%+v", first.StatusCode, firstPayload)
	}

	second, err := http.Post(server.URL+"/api/events/"+itoa64(eventID)+"/checkins", "application/json", strings.NewReader(checkinBody))
	if err != nil {
		t.Fatal(err)
	}
	var secondPayload model.APIResp
	if err := json.NewDecoder(second.Body).Decode(&secondPayload); err != nil {
		t.Fatal(err)
	}
	second.Body.Close()
	if second.StatusCode != http.StatusOK || !secondPayload.Data.(map[string]interface{})["already_checked_in"].(bool) {
		t.Fatalf("unexpected duplicate checkin: status=%d payload=%+v", second.StatusCode, secondPayload)
	}

	auditResponse, err := http.Get(server.URL + "/api/events/" + itoa64(eventID) + "/checkins")
	if err != nil {
		t.Fatal(err)
	}
	var auditPayload model.APIResp
	if err := json.NewDecoder(auditResponse.Body).Decode(&auditPayload); err != nil {
		t.Fatal(err)
	}
	auditResponse.Body.Close()
	if auditResponse.StatusCode != http.StatusOK || auditPayload.Total == nil || *auditPayload.Total != 1 {
		t.Fatalf("unexpected audit list: status=%d payload=%+v", auditResponse.StatusCode, auditPayload)
	}

	cancelResponse := doUserJSON(t, http.MethodDelete, server.URL+"/api/events/"+itoa64(eventID)+"/register", `{}`, 0, "凭证用户", "admission-http@example.com")
	var cancelPayload model.APIResp
	if err := json.NewDecoder(cancelResponse.Body).Decode(&cancelPayload); err != nil {
		t.Fatal(err)
	}
	cancelResponse.Body.Close()
	if cancelResponse.StatusCode != http.StatusConflict || cancelPayload.ErrorCode != "ADMISSION_ALREADY_CHECKED_IN" {
		t.Fatalf("checked-in cancellation was not rejected: status=%d payload=%+v", cancelResponse.StatusCode, cancelPayload)
	}

	deleteRequest, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/events/"+itoa64(eventID), nil)
	deleteResponse, err := http.DefaultClient.Do(deleteRequest)
	if err != nil {
		t.Fatal(err)
	}
	var deletePayload model.APIResp
	if err := json.NewDecoder(deleteResponse.Body).Decode(&deletePayload); err != nil {
		t.Fatal(err)
	}
	deleteResponse.Body.Close()
	if deleteResponse.StatusCode != http.StatusConflict || deletePayload.ErrorCode != "EVENT_HAS_ADMISSIONS" {
		t.Fatalf("event with admissions was deleted: status=%d payload=%+v", deleteResponse.StatusCode, deletePayload)
	}
}

func TestNotificationHTTPJourneyIsUserScoped(t *testing.T) {
	s, _, server := setupTestServer(t)
	event := createStoreEvent(t, s, "通知 HTTP 活动")
	userName, userContact := "通知用户", "notification-http@example.com"

	registerResponse := doUserJSON(
		t, http.MethodPost, server.URL+"/api/v1/events/"+itoa64(event.ID)+"/register",
		`{}`, 0, userName, userContact,
	)
	if registerResponse.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(registerResponse.Body)
		registerResponse.Body.Close()
		t.Fatalf("register: expected 201, got %d: %s", registerResponse.StatusCode, body)
	}
	registerResponse.Body.Close()

	type notificationListResponse struct {
		Data  []dto.NotificationResponse `json:"data"`
		Total int                        `json:"total"`
	}
	listResponse := doUserJSON(
		t, http.MethodGet, server.URL+"/api/v1/me/notifications?unread_only=true&page=1&page_size=10",
		"", 0, userName, userContact,
	)
	defer listResponse.Body.Close()
	if listResponse.StatusCode != http.StatusOK {
		t.Fatalf("list notifications: expected 200, got %d", listResponse.StatusCode)
	}
	var listed notificationListResponse
	if err := json.NewDecoder(listResponse.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if listed.Total != 1 || len(listed.Data) != 1 || listed.Data[0].Type != model.NotificationRegistrationConfirmed {
		t.Fatalf("registration notification missing: %+v", listed)
	}
	registrationNotificationID := listed.Data[0].ID

	otherResponse := doUserJSON(
		t, http.MethodPut,
		server.URL+"/api/v1/me/notifications/"+itoa64(registrationNotificationID)+"/read",
		"", 0, "其他通知用户", "other-notification-http@example.com",
	)
	defer otherResponse.Body.Close()
	if otherResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-user mark read: expected 404, got %d", otherResponse.StatusCode)
	}
	var otherPayload struct {
		ErrorCode string `json:"error_code"`
	}
	if err := json.NewDecoder(otherResponse.Body).Decode(&otherPayload); err != nil {
		t.Fatal(err)
	}
	if otherPayload.ErrorCode != string(api.CodeNotificationNotFound) {
		t.Fatalf("unexpected cross-user error code: %q", otherPayload.ErrorCode)
	}

	markResponse := doUserJSON(
		t, http.MethodPut,
		server.URL+"/api/v1/me/notifications/"+itoa64(registrationNotificationID)+"/read",
		"", 0, userName, userContact,
	)
	markResponse.Body.Close()
	if markResponse.StatusCode != http.StatusOK {
		t.Fatalf("mark own notification: expected 200, got %d", markResponse.StatusCode)
	}

	newLocation := "通知新场地"
	if _, err := s.UpdateEvent(event.ID, model.UpdateEventReq{Location: &newLocation}); err != nil {
		t.Fatal(err)
	}
	cancelResponse := doUserJSON(
		t, http.MethodDelete, server.URL+"/api/v1/events/"+itoa64(event.ID)+"/register",
		"", 0, userName, userContact,
	)
	cancelResponse.Body.Close()
	if cancelResponse.StatusCode != http.StatusOK {
		t.Fatalf("cancel registration: expected 200, got %d", cancelResponse.StatusCode)
	}

	unreadResponse := doUserJSON(
		t, http.MethodGet, server.URL+"/api/v1/me/notifications/unread-count",
		"", 0, userName, userContact,
	)
	defer unreadResponse.Body.Close()
	if unreadResponse.StatusCode != http.StatusOK {
		t.Fatalf("unread count: expected 200, got %d", unreadResponse.StatusCode)
	}
	var unreadPayload struct {
		Data dto.NotificationUnreadCountResponse `json:"data"`
	}
	if err := json.NewDecoder(unreadResponse.Body).Decode(&unreadPayload); err != nil {
		t.Fatal(err)
	}
	if unreadPayload.Data.Unread != 2 {
		t.Fatalf("expected event update and cancellation unread, got %d", unreadPayload.Data.Unread)
	}

	markAllResponse := doUserJSON(
		t, http.MethodPut, server.URL+"/api/v1/me/notifications/read-all",
		"", 0, userName, userContact,
	)
	defer markAllResponse.Body.Close()
	if markAllResponse.StatusCode != http.StatusOK {
		t.Fatalf("mark all read: expected 200, got %d", markAllResponse.StatusCode)
	}
	var markAllPayload struct {
		Data dto.NotificationsMarkedReadResponse `json:"data"`
	}
	if err := json.NewDecoder(markAllResponse.Body).Decode(&markAllPayload); err != nil {
		t.Fatal(err)
	}
	if markAllPayload.Data.Updated != 2 {
		t.Fatalf("expected two notifications marked read, got %d", markAllPayload.Data.Updated)
	}
}

func TestRevokedAdmissionCannotBeCheckedIn(t *testing.T) {
	_, _, server := setupTestServer(t)
	createResponse, _ := http.Post(server.URL+"/api/events", "application/json", strings.NewReader(
		`{"organizer_id":1,"title":"取消凭证活动","event_time":"2099-12-31T18:00:00+08:00","location":"现场","capacity":10,"price":0}`,
	))
	var eventResponse model.APIResp
	_ = json.NewDecoder(createResponse.Body).Decode(&eventResponse)
	createResponse.Body.Close()
	eventID := int64(eventResponse.Data.(map[string]interface{})["id"].(float64))
	registerResponse := doUserJSON(t, http.MethodPost, server.URL+"/api/events/"+itoa64(eventID)+"/register", `{}`, 0, "取消用户", "revoked-http@example.com")
	var registrationResponse model.APIResp
	_ = json.NewDecoder(registerResponse.Body).Decode(&registrationResponse)
	registerResponse.Body.Close()
	credential := registrationResponse.Data.(map[string]interface{})["admission"].(map[string]interface{})["credential"].(string)
	cancelResponse := doUserJSON(t, http.MethodDelete, server.URL+"/api/events/"+itoa64(eventID)+"/register", `{}`, 0, "取消用户", "revoked-http@example.com")
	cancelResponse.Body.Close()
	if cancelResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected cancellation success, got %d", cancelResponse.StatusCode)
	}

	response, err := http.Post(server.URL+"/api/events/"+itoa64(eventID)+"/checkins", "application/json", strings.NewReader(`{"credential":"`+credential+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	var payload model.APIResp
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusConflict || payload.ErrorCode != "ADMISSION_REVOKED" {
		t.Fatalf("revoked credential was accepted: status=%d payload=%+v", response.StatusCode, payload)
	}
}
