package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/store"
)

func setupTestServer(t *testing.T) (*store.Store, *Handler, *httptest.Server) {
	t.Helper()
	s := store.NewStore(":memory:")
	h := NewHandler(s)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/events", AdminAuth(http.HandlerFunc(h.CreateEvent)).ServeHTTP)
	mux.HandleFunc("GET /api/events", h.ListEvents)
	mux.HandleFunc("GET /api/events/{id}", h.GetEvent)
	mux.HandleFunc("PUT /api/events/{id}", AdminAuth(http.HandlerFunc(h.UpdateEvent)).ServeHTTP)
	mux.HandleFunc("DELETE /api/events/{id}", AdminAuth(http.HandlerFunc(h.DeleteEvent)).ServeHTTP)
	mux.HandleFunc("POST /api/events/{id}/register", h.Register)
	mux.HandleFunc("GET /api/events/{id}/registrations", h.ListRegistrations)
	mux.HandleFunc("POST /api/events/{id}/posts", h.CreatePost)
	mux.HandleFunc("GET /api/events/{id}/posts", h.ListPosts)
	mux.HandleFunc("GET /api/events/{id}/posts/{postId}", h.GetPost)
	mux.HandleFunc("POST /api/events/{id}/posts/{postId}/replies", h.CreateReply)
	mux.HandleFunc("POST /api/events/{id}/tickets", AdminAuth(http.HandlerFunc(h.CreateTicket)).ServeHTTP)
	mux.HandleFunc("GET /api/events/{id}/tickets", h.ListTickets)
	mux.HandleFunc("GET /api/events/{id}/tickets/{ticketId}", h.GetTicket)
	mux.HandleFunc("PUT /api/events/{id}/tickets/{ticketId}", AdminAuth(http.HandlerFunc(h.UpdateTicket)).ServeHTTP)
	mux.HandleFunc("DELETE /api/events/{id}/tickets/{ticketId}", AdminAuth(http.HandlerFunc(h.DeleteTicket)).ServeHTTP)
	mux.HandleFunc("GET /health", h.HealthHandler)

	server := httptest.NewServer(LoggingMiddleware(CORS(mux)))
	t.Cleanup(func() {
		server.Close()
		s.Close()
	})
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

func TestCreateEventHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	body := `{"title":"Go 讲座","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":50,"price":0}`
	resp, err := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestCreateEventValidationErrors(t *testing.T) {
	_, _, srv := setupTestServer(t)

	tests := []struct {
		name string
		body string
	}{
		{"empty title", `{"event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":50}`},
		{"empty event_time", `{"title":"讲座","location":"线上","capacity":50}`},
		{"invalid time format", `{"title":"讲座","event_time":"invalid","location":"线上","capacity":50}`},
		{"empty location", `{"title":"讲座","event_time":"2026-12-31T18:00:00+08:00","capacity":50}`},
		{"zero capacity", `{"title":"讲座","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":0}`},
		{"negative price", `{"title":"讲座","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":-1}`},
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
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestListEventsHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	for i := 0; i < 2; i++ {
		body := `{"title":"活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
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

	createBody := `{"title":"详情测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
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
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUpdateEventHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"title":"原始标题","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	updateBody := `{"title":"新标题","price":99.9}`
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
}

func TestUpdateEventInvalidStatus(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"title":"测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
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

	createBody := `{"title":"待删除","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
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

	createBody := `{"title":"报名测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	regBody := `{"name":"张三","contact":"zs@email.com"}`
	resp, err := http.Post(srv.URL+"/api/events/"+itoa64(id)+"/register", "application/json", strings.NewReader(regBody))
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestRegisterHandlerDuplicate(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"title":"重复报名","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	regBody := `{"name":"张三","contact":"zs@email.com"}`
	http.Post(srv.URL+"/api/events/"+itoa64(id)+"/register", "application/json", strings.NewReader(regBody))

	resp, _ := http.Post(srv.URL+"/api/events/"+itoa64(id)+"/register", "application/json", strings.NewReader(regBody))
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestRegisterHandlerFull(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"title":"已满活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":1,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	http.Post(srv.URL+"/api/events/"+itoa64(id)+"/register", "application/json", strings.NewReader(`{"name":"张三","contact":"zs@email.com"}`))

	resp, _ := http.Post(srv.URL+"/api/events/"+itoa64(id)+"/register", "application/json", strings.NewReader(`{"name":"李四","contact":"ls@email.com"}`))
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestRegisterHandlerNotPublished(t *testing.T) {
	s, _, srv := setupTestServer(t)

	createBody := `{"title":"草稿活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	draft := "draft"
	s.UpdateEvent(id, model.UpdateEventReq{Status: &draft})

	resp, _ := http.Post(srv.URL+"/api/events/"+itoa64(id)+"/register", "application/json", strings.NewReader(`{"name":"张三","contact":"zs@email.com"}`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for draft event, got %d", resp.StatusCode)
	}
}

func TestRegisterHandlerNotFound(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, _ := http.Post(srv.URL+"/api/events/999/register", "application/json", strings.NewReader(`{"name":"张三","contact":"zs@email.com"}`))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestRegisterHandlerWithTicket(t *testing.T) {
	s, _, srv := setupTestServer(t)

	createBody := `{"title":"门票报名","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	ticket := &model.Ticket{EventID: id, Name: "普通票", Price: 0, Stock: 5}
	s.CreateTicket(ticket)

	regBody := `{"name":"张三","contact":"zs@email.com","ticket_id":` + itoa64(ticket.ID) + `}`
	resp, _ := http.Post(srv.URL+"/api/events/"+itoa64(id)+"/register", "application/json", strings.NewReader(regBody))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestListRegistrationsHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"title":"报名列表","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	http.Post(srv.URL+"/api/events/"+itoa64(id)+"/register", "application/json", strings.NewReader(`{"name":"张三","contact":"zs@email.com"}`))

	resp, _ := http.Get(srv.URL + "/api/events/" + itoa64(id) + "/registrations")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCreatePostHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"title":"发帖测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	http.Post(srv.URL+"/api/events/"+itoa64(id)+"/register", "application/json", strings.NewReader(`{"name":"张三","contact":"zs@email.com"}`))

	postBody := `{"author_name":"张三","author_contact":"zs@email.com","title":"好活动","content":"推荐给大家"}`
	resp, _ := http.Post(srv.URL+"/api/events/"+itoa64(id)+"/posts", "application/json", strings.NewReader(postBody))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestCreatePostHandlerNotRegistered(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"title":"权限测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	postBody := `{"author_name":"未报名","author_contact":"nobody@email.com","title":"无权限","content":"测试"}`
	resp, _ := http.Post(srv.URL+"/api/events/"+itoa64(id)+"/posts", "application/json", strings.NewReader(postBody))
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestListPostsHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"title":"帖子列表","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	http.Post(srv.URL+"/api/events/"+itoa64(id)+"/register", "application/json", strings.NewReader(`{"name":"张三","contact":"zs@email.com"}`))
	http.Post(srv.URL+"/api/events/"+itoa64(id)+"/posts", "application/json", strings.NewReader(`{"author_name":"张三","author_contact":"zs@email.com","title":"帖子","content":"内容"}`))

	resp, _ := http.Get(srv.URL + "/api/events/" + itoa64(id) + "/posts")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetPostHandlerWithReplies(t *testing.T) {
	s, _, srv := setupTestServer(t)

	createBody := `{"title":"帖子详情","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
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

func TestCreateReplyHandler(t *testing.T) {
	s, _, srv := setupTestServer(t)

	createBody := `{"title":"回复测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	s.Register(&model.Registration{EventID: id, Name: "张三", Contact: "zs@email.com"})
	post := &model.Post{EventID: id, AuthorName: "张三", AuthorContact: "zs@email.com", Title: "帖子", Content: "内容"}
	s.CreatePost(post)

	replyBody := `{"author_name":"张三","author_contact":"zs@email.com","content":"回复内容"}`
	resp, _ := http.Post(srv.URL+"/api/events/"+itoa64(id)+"/posts/"+itoa64(post.ID)+"/replies", "application/json", strings.NewReader(replyBody))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestCreateTicketHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"title":"门票测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
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

	createBody := `{"title":"门票校验","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
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

	createBody := `{"title":"门票列表","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
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

	createBody := `{"title":"更新门票","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
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

	createBody := `{"title":"删除门票","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
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

func TestAdminAuthMiddleware(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "test-token-123")
	defer func() { t.Setenv("ADMIN_TOKEN", "") }()

	_, _, srv := setupTestServer(t)

	t.Run("with valid token", func(t *testing.T) {
		req, _ := http.NewRequest("POST", srv.URL+"/api/events",
			strings.NewReader(`{"title":"测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token-123")
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
			strings.NewReader(`{"title":"测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`))
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

	createBody := `{"title":"已发布活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
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

	body := `{"title":"Go 入门讲座","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
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

	freeBody := `{"title":"免费活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(freeBody))

	paidBody := `{"title":"付费活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":99.9}`
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
	s := store.NewStore(":memory:")
	defer s.Close()
	h := NewHandler(s)

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
	s := store.NewStore(":memory:")
	h := NewHandler(s)
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
	if _, exists := data["db_error"]; !exists {
		t.Errorf("expected db_error field in response")
	}
}

func TestMiddlewareChainOrder(t *testing.T) {
	s := store.NewStore(":memory:")
	defer s.Close()
	handler := NewHandler(s)

	t.Setenv("ADMIN_TOKEN", "test-token")
	defer func() { t.Setenv("ADMIN_TOKEN", "") }()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/events", AdminAuth(http.HandlerFunc(handler.CreateEvent)).ServeHTTP)

	server := httptest.NewServer(LoggingMiddleware(CORS(mux)))
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL+"/api/events",
		strings.NewReader(`{"title":"测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

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
	if apiResp.Message == "" {
		t.Errorf("expected non-empty message, got %q", apiResp.Message)
	}
	if apiResp.Data != nil {
		t.Errorf("expected nil data for error response, got %v", apiResp.Data)
	}
}

func TestCORSOriginEnvConfig(t *testing.T) {
	originalOrigin := os.Getenv("CORS_ORIGIN")
	t.Setenv("CORS_ORIGIN", "https://example.com")
	defer func() {
		if originalOrigin != "" {
			t.Setenv("CORS_ORIGIN", originalOrigin)
		} else {
			os.Unsetenv("CORS_ORIGIN")
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(CORS(mux))
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

	body := `{"title":"Test Event","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
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
	_, _, srv := setupTestServer(t)

	createBody := `{"title":"获取门票测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created model.APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	eid := int64(eventData["id"].(float64))

	s := store.NewStore(":memory:")
	h := NewHandler(s)
	ticket := &model.Ticket{EventID: eid, Name: "测试票", Price: 10, Stock: 5}
	s.CreateTicket(ticket)

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
	s.Close()
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

func TestGetTicketInvalidID(t *testing.T) {
	s := store.NewStore(":memory:")
	defer s.Close()
	h := NewHandler(s)

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
	s := store.NewStore(":memory:")
	defer s.Close()
	h := NewHandler(s)
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
	s := store.NewStore(":memory:")
	defer s.Close()
	h := NewHandler(s)

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

	createBody := `{"title":"无门票活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
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

	req, _ := http.NewRequest("POST", srv.URL+"/api/events/999/posts/1/replies",
		strings.NewReader(`{"contact":"test@test.com","content":"回复"}`))
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

func TestCreateReplyInvalidBody(t *testing.T) {
	s := store.NewStore(":memory:")
	defer s.Close()
	h := NewHandler(s)
	e := &model.Event{Title: "test", EventTime: "2026-12-31T18:00:00+08:00", Location: "线上", Capacity: 10, Status: "published"}
	s.CreateEvent(e)
	p := &model.Post{EventID: e.ID, AuthorName: "testuser", Content: "帖子"}
	s.CreatePost(p)

	req := httptest.NewRequest("POST", "/api/events/"+itoa64(e.ID)+"/posts/"+itoa64(p.ID)+"/replies",
		strings.NewReader(`invalid`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateReply(w, req)

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

	createBody := `{"title":"无帖子活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
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

	createBody := `{"title":"无报名活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
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
	s := store.NewStore(":memory:")
	defer s.Close()
	h := NewHandler(s)
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
