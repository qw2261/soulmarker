package dto

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func assertJSONKeys(t *testing.T, value interface{}, expected ...string) map[string]interface{} {
	t.Helper()
	contents, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(contents, &decoded); err != nil {
		t.Fatal(err)
	}
	actual := make([]string, 0, len(decoded))
	for key := range decoded {
		actual = append(actual, key)
	}
	sort.Strings(actual)
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected JSON keys: actual=%v expected=%v json=%s", actual, expected, contents)
	}
	return decoded
}

func TestResponseEnvelopeContract(t *testing.T) {
	assertJSONKeys(t, Response{Code: 200, Message: "ok"}, "code", "message")
	total, page, pageSize := 1, 1, 20
	assertJSONKeys(t, Response{
		Code: 200, Message: "ok", Data: []string{}, Total: &total, Page: &page, PageSize: &pageSize,
	}, "code", "message", "data", "total", "page", "page_size")
}

func TestUserAndLoginResponseDoNotExposePassword(t *testing.T) {
	user := &model.User{
		ID: 1, Name: "用户", Contact: "user@example.com", PasswordHash: "secret",
		CreatedAt: time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	userResponse := User(user)
	assertJSONKeys(t, userResponse, "id", "name", "contact", "created_at")
	login := assertJSONKeys(t, LoginResponse{Token: "token", User: userResponse}, "token", "user")
	userJSON := login["user"].(map[string]interface{})
	if _, exists := userJSON["password_hash"]; exists {
		t.Fatal("password_hash must not be exposed")
	}
}

func TestEntityMappersFixPublicFieldSets(t *testing.T) {
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	userID := int64(7)
	ticketID := int64(8)

	assertJSONKeys(t, Event(&model.Event{
		ID: 1, OrganizerID: 2, OrganizerName: "门店", Title: "活动", Description: "说明",
		EventTime: now.Format(model.TimeFormat), Location: "线上", Capacity: 10, Price: 1,
		Status: "published", CreatedAt: now, UpdatedAt: now,
	}), "id", "organizer_id", "organizer_name", "title", "description", "event_time", "location", "capacity", "price", "status", "created_at", "updated_at")

	registration := assertJSONKeys(t, Registration(&model.Registration{
		ID: 1, EventID: 2, UserID: &userID, Name: "用户", Contact: "user@example.com",
		TicketID: &ticketID, TicketName: "票", IdentityStatus: model.IdentityStatusVerified, CreatedAt: now,
	}), "id", "event_id", "name", "contact", "ticket_id", "ticket_name", "identity_status", "created_at")
	if _, exists := registration["user_id"]; exists {
		t.Fatal("internal user_id must not be exposed by registration DTO")
	}

	post := assertJSONKeys(t, Post(&model.Post{
		ID: 1, EventID: 2, UserID: &userID, AuthorName: "作者", AuthorContact: "private@example.com",
		IdentityStatus: model.IdentityStatusVerified, Title: "标题", Content: "内容", ReplyCount: 3, CreatedAt: now,
	}), "id", "event_id", "author_name", "title", "content", "reply_count", "created_at")
	for _, forbidden := range []string{"user_id", "author_contact", "identity_status"} {
		if _, exists := post[forbidden]; exists {
			t.Fatalf("%s must not be exposed by post DTO", forbidden)
		}
	}
}

func TestPostDetailUsesStableEmptyArray(t *testing.T) {
	detail := PostDetail(&model.Post{ID: 1}, nil)
	decoded := assertJSONKeys(t, detail, "post", "replies")
	replies, ok := decoded["replies"].([]interface{})
	if !ok || len(replies) != 0 {
		t.Fatalf("expected empty replies array, got %#v", decoded["replies"])
	}
}
