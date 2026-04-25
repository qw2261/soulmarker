package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func setupTestServer(t *testing.T) (*Store, *Handler, *httptest.Server) {
	t.Helper()
	store := NewStore(":memory:")
	h := NewHandler(store)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/events", adminAuth(http.HandlerFunc(h.CreateEvent)).ServeHTTP)
	mux.HandleFunc("GET /api/events", h.ListEvents)
	mux.HandleFunc("GET /api/events/{id}", h.GetEvent)
	mux.HandleFunc("PUT /api/events/{id}", adminAuth(http.HandlerFunc(h.UpdateEvent)).ServeHTTP)
	mux.HandleFunc("DELETE /api/events/{id}", adminAuth(http.HandlerFunc(h.DeleteEvent)).ServeHTTP)
	mux.HandleFunc("POST /api/events/{id}/register", h.Register)
	mux.HandleFunc("GET /api/events/{id}/registrations", h.ListRegistrations)
	mux.HandleFunc("POST /api/events/{id}/posts", h.CreatePost)
	mux.HandleFunc("GET /api/events/{id}/posts", h.ListPosts)
	mux.HandleFunc("GET /api/events/{id}/posts/{postId}", h.GetPost)
	mux.HandleFunc("POST /api/events/{id}/posts/{postId}/replies", h.CreateReply)
	mux.HandleFunc("POST /api/events/{id}/tickets", adminAuth(http.HandlerFunc(h.CreateTicket)).ServeHTTP)
	mux.HandleFunc("GET /api/events/{id}/tickets", h.ListTickets)
	mux.HandleFunc("GET /api/events/{id}/tickets/{ticketId}", h.GetTicket)
	mux.HandleFunc("PUT /api/events/{id}/tickets/{ticketId}", adminAuth(http.HandlerFunc(h.UpdateTicket)).ServeHTTP)
	mux.HandleFunc("DELETE /api/events/{id}/tickets/{ticketId}", adminAuth(http.HandlerFunc(h.DeleteTicket)).ServeHTTP)

	server := httptest.NewServer(cors(mux))
	t.Cleanup(func() {
		server.Close()
		store.Close()
	})
	return store, h, server
}

func parseResp(t *testing.T, body []byte) APIResp {
	t.Helper()
	var resp APIResp
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("json unmarshal failed: %v, body: %s", err, string(body))
	}
	return resp
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

	// 先创建两个活动
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

	var apiResp APIResp
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
	var created APIResp
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
	var created APIResp
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

	// 验证字段已更新
	var updateResp APIResp
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
	var created APIResp
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
	var created APIResp
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

	// 确认已删除
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
	var created APIResp
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
	var created APIResp
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
	var created APIResp
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
	store, _, srv := setupTestServer(t)

	createBody := `{"title":"草稿活动","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	// 设置为 draft
	draft := "draft"
	store.UpdateEvent(id, UpdateEventReq{Status: &draft})

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
	store, _, srv := setupTestServer(t)

	createBody := `{"title":"门票报名","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	ticket := &Ticket{EventID: id, Name: "普通票", Price: 0, Stock: 5}
	store.CreateTicket(ticket)

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
	var created APIResp
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
	var created APIResp
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
	var created APIResp
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
	var created APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	// 正常发帖
	http.Post(srv.URL+"/api/events/"+itoa64(id)+"/register", "application/json", strings.NewReader(`{"name":"张三","contact":"zs@email.com"}`))
	http.Post(srv.URL+"/api/events/"+itoa64(id)+"/posts", "application/json", strings.NewReader(`{"author_name":"张三","author_contact":"zs@email.com","title":"帖子","content":"内容"}`))

	resp, _ := http.Get(srv.URL + "/api/events/" + itoa64(id) + "/posts")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetPostHandlerWithReplies(t *testing.T) {
	store, _, srv := setupTestServer(t)

	createBody := `{"title":"帖子详情","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	store.Register(&Registration{EventID: id, Name: "张三", Contact: "zs@email.com"})
	post := &Post{EventID: id, AuthorName: "张三", AuthorContact: "zs@email.com", Title: "帖子", Content: "内容"}
	store.CreatePost(post)

	resp, _ := http.Get(srv.URL + "/api/events/" + itoa64(id) + "/posts/" + itoa64(post.ID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var apiResp APIResp
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
	store, _, srv := setupTestServer(t)

	createBody := `{"title":"回复测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created APIResp
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	eventData := created.Data.(map[string]interface{})
	id := int64(eventData["id"].(float64))

	store.Register(&Registration{EventID: id, Name: "张三", Contact: "zs@email.com"})
	post := &Post{EventID: id, AuthorName: "张三", AuthorContact: "zs@email.com", Title: "帖子", Content: "内容"}
	store.CreatePost(post)

	replyBody := `{"author_name":"张三","author_contact":"zs@email.com","content":"回复内容"}`
	resp, _ := http.Post(srv.URL+"/api/events/"+itoa64(id)+"/posts/"+itoa64(post.ID)+"/replies", "application/json", strings.NewReader(replyBody))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestCreateTicketHandler(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"title":"门票创建","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created APIResp
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

func TestCreateTicketHandlerValidation(t *testing.T) {
	_, _, srv := setupTestServer(t)

	createBody := `{"title":"门票校验","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	createResp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	var created APIResp
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
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

func TestListTicketsHandler(t *testing.T) {
	store, _, srv := setupTestServer(t)

	e := &Event{Title: "门票列表", EventTime: "2026-12-31T18:00:00+08:00", Location: "线上", Capacity: 10, Price: 0}
	store.CreateEvent(e)

	resp, _ := http.Get(srv.URL + "/api/events/" + itoa64(e.ID) + "/tickets")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestUpdateTicketHandler(t *testing.T) {
	store, _, srv := setupTestServer(t)

	e := &Event{Title: "门票更新", EventTime: "2026-12-31T18:00:00+08:00", Location: "线上", Capacity: 10, Price: 0}
	store.CreateEvent(e)
	ticket := &Ticket{EventID: e.ID, Name: "原名称", Price: 50, Stock: 10}
	store.CreateTicket(ticket)

	updateBody := `{"name":"新名称","price":99.9,"stock":5}`
	req, _ := http.NewRequest("PUT", srv.URL+"/api/events/"+itoa64(e.ID)+"/tickets/"+itoa64(ticket.ID), strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDeleteTicketHandler(t *testing.T) {
	store, _, srv := setupTestServer(t)

	e := &Event{Title: "删除门票", EventTime: "2026-12-31T18:00:00+08:00", Location: "线上", Capacity: 10, Price: 0}
	store.CreateEvent(e)
	ticket := &Ticket{EventID: e.ID, Name: "待删除", Price: 0, Stock: 1}
	store.CreateTicket(ticket)

	req, _ := http.NewRequest("DELETE", srv.URL+"/api/events/"+itoa64(e.ID)+"/tickets/"+itoa64(ticket.ID), nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCORSMiddleware(t *testing.T) {
	_, _, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/events")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	origin := resp.Header.Get("Access-Control-Allow-Origin")
	if origin == "" {
		t.Fatal("expected CORS header Access-Control-Allow-Origin")
	}

	methods := resp.Header.Get("Access-Control-Allow-Methods")
	if methods == "" {
		t.Fatal("expected CORS header Access-Control-Allow-Methods")
	}
}

func TestAdminAuthMiddlewareWithToken(t *testing.T) {
	os.Setenv("ADMIN_TOKEN", "test-token-123")
	defer os.Unsetenv("ADMIN_TOKEN")

	// 重新设置以使用新的环境变量
	store := NewStore(":memory:")
	h := NewHandler(store)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/events", adminAuth(http.HandlerFunc(h.CreateEvent)).ServeHTTP)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	defer store.Close()

	body := `{"title":"认证测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`

	// 无 Token
	resp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(body))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 错误 Token
	req, _ := http.NewRequest("POST", srv.URL+"/api/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-token")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong token, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 正确 Token
	req, _ = http.NewRequest("POST", srv.URL+"/api/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token-123")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 with correct token, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestAdminAuthMiddlewareWithoutToken(t *testing.T) {
	os.Unsetenv("ADMIN_TOKEN")

	_, _, srv := setupTestServer(t)

	// 未设置 ADMIN_TOKEN 时，管理接口应可直接访问
	body := `{"title":"无需认证","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	resp, _ := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(body))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 when ADMIN_TOKEN is not set, got %d", resp.StatusCode)
	}
}

func TestResponseFormat(t *testing.T) {
	_, _, srv := setupTestServer(t)

	// 创建一个活动确保返回正确格式
	createBody := `{"title":"格式测试","event_time":"2026-12-31T18:00:00+08:00","location":"线上","capacity":10,"price":0}`
	resp, err := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	defer resp.Body.Close()

	var apiResp APIResp
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}
	if apiResp.Code != 201 {
		t.Fatalf("expected code 201, got %d", apiResp.Code)
	}
	if apiResp.Message == "" {
		t.Fatal("expected non-empty message")
	}
	if apiResp.Data == nil {
		t.Fatal("expected non-nil data")
	}
}

func TestListEventsFilterByStatusHandler(t *testing.T) {
	store, _, srv := setupTestServer(t)

	e := &Event{Title: "已发布", EventTime: "2026-12-31T18:00:00+08:00", Location: "线上", Capacity: 10, Price: 0}
	store.CreateEvent(e)
	draft := "draft"
	store.UpdateEvent(e.ID, UpdateEventReq{Status: &draft})

	e2 := &Event{Title: "已发布2", EventTime: "2026-12-31T18:00:00+08:00", Location: "线上", Capacity: 10, Price: 0}
	store.CreateEvent(e2)

	resp, _ := http.Get(srv.URL + "/api/events?status=published")
	var apiResp APIResp
	json.NewDecoder(resp.Body).Decode(&apiResp)
	resp.Body.Close()
	events := apiResp.Data.([]interface{})
	if len(events) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(events))
	}
}

func TestListEventsFilterByKeywordHandler(t *testing.T) {
	store, _, srv := setupTestServer(t)

	e := &Event{Title: "Go 讲座", EventTime: "2026-12-31T18:00:00+08:00", Location: "线上", Capacity: 10, Price: 0}
	e.Description = "学习 Go 语言"
	store.CreateEvent(e)

	e2 := &Event{Title: "Docker 实战", EventTime: "2026-12-31T18:00:00+08:00", Location: "线上", Capacity: 10, Price: 0}
	store.CreateEvent(e2)

	resp, _ := http.Get(srv.URL + "/api/events?q=Go")
	var apiResp APIResp
	json.NewDecoder(resp.Body).Decode(&apiResp)
	resp.Body.Close()
	events := apiResp.Data.([]interface{})
	if len(events) != 1 {
		t.Fatalf("expected 1 event matching 'Go', got %d", len(events))
	}
}

func itoa64(i int64) string {
	return fmt.Sprintf("%d", i)
}
