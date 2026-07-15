package store

import (
	"fmt"
	"testing"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() {
		store.Close()
	})
	_ = store.CreateOrganizer(&model.Organizer{Name: "默认门店"})
	return store
}

func newTestEvent(title string) *model.Event {
	return &model.Event{
		OrganizerID: 1,
		Title:       title,
		Description: "测试描述",
		EventTime:   "2026-12-31T18:00:00+08:00",
		Location:    "线上",
		Capacity:    10,
		Price:       0,
	}
}

func TestCreateEvent(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("Go 入门讲座")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	if e.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if e.Status != "published" {
		t.Fatalf("expected status 'published', got %q", e.Status)
	}
	if e.CreatedAt.IsZero() {
		t.Fatal("expected non-zero CreatedAt")
	}
}

func TestCreateEventSetsTimestamps(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("时间测试")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	if e.CreatedAt.IsZero() {
		t.Fatal("expected non-zero CreatedAt")
	}
	if e.UpdatedAt.IsZero() {
		t.Fatal("expected non-zero UpdatedAt")
	}
}

func TestGetEvent(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("查找测试")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	got, err := store.GetEvent(e.ID)
	if err != nil {
		t.Fatalf("GetEvent failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected event, got nil")
	}
	if got.Title != e.Title {
		t.Fatalf("expected title %q, got %q", e.Title, got.Title)
	}
}

func TestCreateAndUpdateEventCoverURL(t *testing.T) {
	store := setupTestStore(t)
	event := newTestEvent("封面活动")
	event.CoverURL = "https://assets.example.com/cover-one.jpg"
	if err := store.CreateEvent(event); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	created, err := store.GetEvent(event.ID)
	if err != nil {
		t.Fatalf("GetEvent failed: %v", err)
	}
	if created.CoverURL != event.CoverURL {
		t.Fatalf("expected cover %q, got %q", event.CoverURL, created.CoverURL)
	}

	updatedCover := "https://assets.example.com/cover-two.jpg"
	updated, err := store.UpdateEvent(event.ID, model.UpdateEventReq{CoverURL: &updatedCover})
	if err != nil {
		t.Fatalf("UpdateEvent failed: %v", err)
	}
	if updated.CoverURL != updatedCover {
		t.Fatalf("expected updated cover %q, got %q", updatedCover, updated.CoverURL)
	}
}

func TestGetEventNotFound(t *testing.T) {
	store := setupTestStore(t)

	got, err := store.GetEvent(999)
	if err != nil {
		t.Fatalf("GetEvent failed: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for non-existent event")
	}
}

func TestListEventsEmpty(t *testing.T) {
	store := setupTestStore(t)

	events, _, err := store.ListEvents(model.ListEventsParams{})
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestListEventsMultiple(t *testing.T) {
	store := setupTestStore(t)

	titles := []string{"活动A", "活动B", "活动C"}
	for _, title := range titles {
		e := newTestEvent(title)
		if err := store.CreateEvent(e); err != nil {
			t.Fatalf("CreateEvent(%q) failed: %v", title, err)
		}
	}

	events, _, err := store.ListEvents(model.ListEventsParams{})
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	seen := make(map[string]bool)
	for _, e := range events {
		seen[e.Title] = true
	}
	for _, title := range titles {
		if !seen[title] {
			t.Fatalf("expected event %q in results", title)
		}
	}
}

func TestUpdateEvent(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("原始标题")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	newTitle := "更新后的标题"
	req := model.UpdateEventReq{Title: &newTitle}
	updated, err := store.UpdateEvent(e.ID, req)
	if err != nil {
		t.Fatalf("UpdateEvent failed: %v", err)
	}
	if updated.Title != newTitle {
		t.Fatalf("expected title %q, got %q", newTitle, updated.Title)
	}
}

func TestUpdateEventPartial(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("部分更新")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	newPrice := 99.9
	req := model.UpdateEventReq{Price: &newPrice}
	updated, err := store.UpdateEvent(e.ID, req)
	if err != nil {
		t.Fatalf("UpdateEvent failed: %v", err)
	}
	if updated.Price != 99.9 {
		t.Fatalf("expected price 99.9, got %f", updated.Price)
	}
	if updated.Title != "部分更新" {
		t.Fatalf("expected title unchanged '部分更新', got %q", updated.Title)
	}
}

func TestUpdateEventStatus(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("状态测试")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	status := "cancelled"
	req := model.UpdateEventReq{Status: &status}
	updated, err := store.UpdateEvent(e.ID, req)
	if err != nil {
		t.Fatalf("UpdateEvent failed: %v", err)
	}
	if updated.Status != "cancelled" {
		t.Fatalf("expected status 'cancelled', got %q", updated.Status)
	}
}

func TestUpdateEventNotFound(t *testing.T) {
	store := setupTestStore(t)

	title := "不存在"
	req := model.UpdateEventReq{Title: &title}
	_, err := store.UpdateEvent(999, req)
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteEvent(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("待删除")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	if err := store.DeleteEvent(e.ID); err != nil {
		t.Fatalf("DeleteEvent failed: %v", err)
	}

	got, _ := store.GetEvent(e.ID)
	if got != nil {
		t.Fatal("expected nil after deletion")
	}
}

func TestDeleteEventNotFound(t *testing.T) {
	store := setupTestStore(t)

	err := store.DeleteEvent(999)
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteEventCascadeRegistration(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("级联删除测试")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	reg := &model.Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}
	if err := store.Register(reg); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if err := store.DeleteEvent(e.ID); err != nil {
		t.Fatalf("DeleteEvent failed: %v", err)
	}

	regs, _, _ := store.ListRegistrations(e.ID, 0, 0)
	if len(regs) != 0 {
		t.Fatal("expected 0 registrations after cascade delete")
	}
}

func TestDeleteEventCascadePostsAndReplies(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("级联帖子")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	if err := store.Register(&model.Registration{EventID: e.ID, Name: "李四", Contact: "ls@email.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	post := &model.Post{EventID: e.ID, AuthorName: "李四", AuthorContact: "ls@email.com", Title: "好活动", Content: "赞"}
	if err := store.CreatePost(post); err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}
	if err := store.CreateReply(&model.Reply{PostID: post.ID, AuthorName: "李四", AuthorContact: "ls@email.com", Content: "回复"}); err != nil {
		t.Fatalf("CreateReply failed: %v", err)
	}

	if err := store.DeleteEvent(e.ID); err != nil {
		t.Fatalf("DeleteEvent failed: %v", err)
	}

	posts, _, _ := store.ListPosts(e.ID, 0, 0)
	if len(posts) != 0 {
		t.Fatal("expected 0 posts after cascade delete")
	}
	replies, _ := store.ListReplies(post.ID)
	if len(replies) != 0 {
		t.Fatal("expected 0 replies after cascade delete")
	}
}

func TestRegister(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("报名测试")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	reg := &model.Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}
	if err := store.Register(reg); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if reg.ID == 0 {
		t.Fatal("expected non-zero registration ID")
	}
}

func TestRegisterDuplicate(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("重复报名")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	reg := &model.Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}
	if err := store.Register(reg); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	err := store.Register(&model.Registration{EventID: e.ID, Name: "张三2", Contact: "zs@email.com"})
	if err != model.ErrDuplicate {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestRegisterFull(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("容量测试")
	e.Capacity = 2
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	for i, name := range []string{"张三", "李四"} {
		err := store.Register(&model.Registration{EventID: e.ID, Name: name, Contact: name + "@email.com"})
		if err != nil {
			t.Fatalf("registration %d failed: %v", i, err)
		}
	}

	err := store.Register(&model.Registration{EventID: e.ID, Name: "王五", Contact: "ww@email.com"})
	if err != model.ErrFull {
		t.Fatalf("expected ErrFull, got %v", err)
	}
}

func TestRegisterNotFound(t *testing.T) {
	store := setupTestStore(t)

	err := store.Register(&model.Registration{EventID: 999, Name: "张三", Contact: "zs@email.com"})
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRegisterWithTicket(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("门票报名")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	ticket := &model.Ticket{EventID: e.ID, Name: "普通票", Price: 0, Stock: 5}
	if err := store.CreateTicket(ticket); err != nil {
		t.Fatalf("CreateTicket failed: %v", err)
	}

	ticketID := ticket.ID
	reg := &model.Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com", TicketID: &ticketID}
	if err := store.Register(reg); err != nil {
		t.Fatalf("Register with ticket failed: %v", err)
	}
	if reg.TicketName != "普通票" {
		t.Fatalf("expected ticket_name '普通票', got %q", reg.TicketName)
	}

	updatedTicket, _ := store.GetTicket(ticketID)
	if updatedTicket.Stock != 4 {
		t.Fatalf("expected stock 4, got %d", updatedTicket.Stock)
	}
}

func TestRegisterTicketSoldOut(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("售罄测试")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	ticket := &model.Ticket{EventID: e.ID, Name: "VIP票", Price: 100, Stock: 1}
	if err := store.CreateTicket(ticket); err != nil {
		t.Fatalf("CreateTicket failed: %v", err)
	}

	ticketID := ticket.ID
	if err := store.Register(&model.Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com", TicketID: &ticketID}); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	err := store.Register(&model.Registration{EventID: e.ID, Name: "李四", Contact: "ls@email.com", TicketID: &ticketID})
	if err != model.ErrTicketSoldOut {
		t.Fatalf("expected ErrTicketSoldOut, got %v", err)
	}
}

func TestRegisterTicketNotFound(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("门票不存在")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	ticketID := int64(999)
	err := store.Register(&model.Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com", TicketID: &ticketID})
	if err != model.ErrTicketNotFound {
		t.Fatalf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestListRegistrations(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("报名列表")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	names := []string{"张三", "李四", "王五"}
	for _, name := range names {
		reg := &model.Registration{EventID: e.ID, Name: name, Contact: name + "@email.com"}
		if err := store.Register(reg); err != nil {
			t.Fatalf("Register(%q) failed: %v", name, err)
		}
	}

	regs, _, err := store.ListRegistrations(e.ID, 0, 0)
	if err != nil {
		t.Fatalf("ListRegistrations failed: %v", err)
	}
	if len(regs) != 3 {
		t.Fatalf("expected 3 registrations, got %d", len(regs))
	}
}

func TestListRegistrationsEmpty(t *testing.T) {
	store := setupTestStore(t)

	regs, _, err := store.ListRegistrations(999, 0, 0)
	if err != nil {
		t.Fatalf("ListRegistrations failed: %v", err)
	}
	if len(regs) != 0 {
		t.Fatalf("expected 0 registrations, got %d", len(regs))
	}
}

func TestIsRegistered(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("注册检查")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	contact := "test@email.com"
	reg := &model.Registration{EventID: e.ID, Name: "测试", Contact: contact}
	if err := store.Register(reg); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ok, err := store.IsRegistered(e.ID, contact)
	if err != nil {
		t.Fatalf("IsRegistered failed: %v", err)
	}
	if !ok {
		t.Fatal("expected registered to be true")
	}

	ok, _ = store.IsRegistered(e.ID, "other@email.com")
	if ok {
		t.Fatal("expected registered to be false for other contact")
	}
}

func TestCreatePost(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("发帖测试")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	if err := store.Register(&model.Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	post := &model.Post{EventID: e.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Title: "好活动", Content: "推荐给大家"}
	if err := store.CreatePost(post); err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}
	if post.ID == 0 {
		t.Fatal("expected non-zero post ID")
	}
	if post.ReplyCount != 0 {
		t.Fatalf("expected ReplyCount 0, got %d", post.ReplyCount)
	}
}

func TestListPosts(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("帖子列表")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	if err := store.Register(&model.Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	titles := []string{"帖子A", "帖子B", "帖子C"}
	for i, title := range titles {
		post := &model.Post{EventID: e.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Title: title, Content: "内容"}
		if err := store.CreatePost(post); err != nil {
			t.Fatalf("CreatePost %d failed: %v", i, err)
		}
	}

	posts, _, err := store.ListPosts(e.ID, 0, 0)
	if err != nil {
		t.Fatalf("ListPosts failed: %v", err)
	}
	if len(posts) != 3 {
		t.Fatalf("expected 3 posts, got %d", len(posts))
	}

	seen := make(map[string]bool)
	for _, p := range posts {
		seen[p.Title] = true
	}
	for _, title := range titles {
		if !seen[title] {
			t.Fatalf("expected post %q in results", title)
		}
	}
}

func TestListPostsEmpty(t *testing.T) {
	store := setupTestStore(t)

	posts, _, err := store.ListPosts(999, 0, 0)
	if err != nil {
		t.Fatalf("ListPosts failed: %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("expected 0 posts, got %d", len(posts))
	}
}

func TestGetPost(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("帖子详情")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	if err := store.Register(&model.Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	post := &model.Post{EventID: e.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Title: "详情测试", Content: "内容"}
	if err := store.CreatePost(post); err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}

	got, err := store.GetPost(post.ID)
	if err != nil {
		t.Fatalf("GetPost failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected post, got nil")
	}
	if got.Title != "详情测试" {
		t.Fatalf("expected title '详情测试', got %q", got.Title)
	}
}

func TestGetPostNotFound(t *testing.T) {
	store := setupTestStore(t)

	got, err := store.GetPost(999)
	if err != nil {
		t.Fatalf("GetPost failed: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for non-existent post")
	}
}

func TestCreateReply(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("回复测试")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	if err := store.Register(&model.Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	post := &model.Post{EventID: e.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Title: "帖子", Content: "内容"}
	if err := store.CreatePost(post); err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}

	reply := &model.Reply{PostID: post.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Content: "回复内容"}
	if err := store.CreateReply(reply); err != nil {
		t.Fatalf("CreateReply failed: %v", err)
	}
	if reply.ID == 0 {
		t.Fatal("expected non-zero reply ID")
	}
}

func TestListReplies(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("回复列表")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	if err := store.Register(&model.Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	post := &model.Post{EventID: e.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Title: "帖子", Content: "内容"}
	if err := store.CreatePost(post); err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}

	contents := []string{"回复1", "回复2", "回复3"}
	for _, c := range contents {
		reply := &model.Reply{PostID: post.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Content: c}
		if err := store.CreateReply(reply); err != nil {
			t.Fatalf("CreateReply(%q) failed: %v", c, err)
		}
	}

	replies, err := store.ListReplies(post.ID)
	if err != nil {
		t.Fatalf("ListReplies failed: %v", err)
	}
	if len(replies) != 3 {
		t.Fatalf("expected 3 replies, got %d", len(replies))
	}
}

func TestListRepliesEmpty(t *testing.T) {
	store := setupTestStore(t)

	replies, err := store.ListReplies(999)
	if err != nil {
		t.Fatalf("ListReplies failed: %v", err)
	}
	if len(replies) != 0 {
		t.Fatalf("expected 0 replies, got %d", len(replies))
	}
}

func TestGetPostIncludesReplyCount(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("回复数测试")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	if err := store.Register(&model.Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	post := &model.Post{EventID: e.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Title: "帖子", Content: "内容"}
	if err := store.CreatePost(post); err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		reply := &model.Reply{PostID: post.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Content: "回复"}
		if err := store.CreateReply(reply); err != nil {
			t.Fatalf("CreateReply %d failed: %v", i, err)
		}
	}

	got, _ := store.GetPost(post.ID)
	if got.ReplyCount != 3 {
		t.Fatalf("expected ReplyCount 3, got %d", got.ReplyCount)
	}

	posts, _, _ := store.ListPosts(e.ID, 0, 0)
	if posts[0].ReplyCount != 3 {
		t.Fatalf("expected ListPosts ReplyCount 3, got %d", posts[0].ReplyCount)
	}
}

func TestCreateTicket(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("门票测试")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	ticket := &model.Ticket{EventID: e.ID, Name: "普通票", Price: 0, Stock: 100}
	if err := store.CreateTicket(ticket); err != nil {
		t.Fatalf("CreateTicket failed: %v", err)
	}
	if ticket.ID == 0 {
		t.Fatal("expected non-zero ticket ID")
	}
}

func TestListTickets(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("门票列表")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		ticket := &model.Ticket{EventID: e.ID, Name: fmt.Sprintf("票%d", i), Price: float64(i * 10), Stock: 10}
		if err := store.CreateTicket(ticket); err != nil {
			t.Fatalf("CreateTicket %d failed: %v", i, err)
		}
	}

	tickets, _, err := store.ListTickets(e.ID, 0, 0)
	if err != nil {
		t.Fatalf("ListTickets failed: %v", err)
	}
	if len(tickets) != 3 {
		t.Fatalf("expected 3 tickets, got %d", len(tickets))
	}
}

func TestListTicketsEmpty(t *testing.T) {
	store := setupTestStore(t)

	tickets, _, err := store.ListTickets(999, 0, 0)
	if err != nil {
		t.Fatalf("ListTickets failed: %v", err)
	}
	if len(tickets) != 0 {
		t.Fatalf("expected 0 tickets, got %d", len(tickets))
	}
}

func TestGetTicket(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("门票详情")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	ticket := &model.Ticket{EventID: e.ID, Name: "VIP票", Price: 100, Stock: 5}
	if err := store.CreateTicket(ticket); err != nil {
		t.Fatalf("CreateTicket failed: %v", err)
	}

	got, err := store.GetTicket(ticket.ID)
	if err != nil {
		t.Fatalf("GetTicket failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected ticket, got nil")
	}
	if got.Name != "VIP票" {
		t.Fatalf("expected name 'VIP票', got %q", got.Name)
	}
}

func TestGetTicketNotFound(t *testing.T) {
	store := setupTestStore(t)

	got, err := store.GetTicket(999)
	if err != nil {
		t.Fatalf("GetTicket failed: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for non-existent ticket")
	}
}

func TestUpdateTicket(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("更新门票")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	ticket := &model.Ticket{EventID: e.ID, Name: "原始名称", Price: 0, Stock: 1}
	if err := store.CreateTicket(ticket); err != nil {
		t.Fatalf("CreateTicket failed: %v", err)
	}

	newName := "新名称"
	newPrice := 99.9
	newStock := 5
	req := model.UpdateTicketReq{Name: &newName, Price: &newPrice, Stock: &newStock}
	updated, err := store.UpdateTicket(ticket.ID, req)
	if err != nil {
		t.Fatalf("UpdateTicket failed: %v", err)
	}
	if updated.Name != "新名称" || updated.Price != 99.9 || updated.Stock != 5 {
		t.Fatalf("update mismatch: %+v", updated)
	}
}

func TestUpdateTicketNotFound(t *testing.T) {
	store := setupTestStore(t)

	name := "不存在"
	req := model.UpdateTicketReq{Name: &name}
	_, err := store.UpdateTicket(999, req)
	if err != model.ErrTicketNotFound {
		t.Fatalf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestDeleteTicket(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("删除门票")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	ticket := &model.Ticket{EventID: e.ID, Name: "待删除", Price: 0, Stock: 1}
	if err := store.CreateTicket(ticket); err != nil {
		t.Fatalf("CreateTicket failed: %v", err)
	}

	if err := store.DeleteTicket(ticket.ID); err != nil {
		t.Fatalf("DeleteTicket failed: %v", err)
	}

	got, _ := store.GetTicket(ticket.ID)
	if got != nil {
		t.Fatal("expected nil after ticket deletion")
	}
}

func TestDeleteTicketNotFound(t *testing.T) {
	store := setupTestStore(t)

	err := store.DeleteTicket(999)
	if err != model.ErrTicketNotFound {
		t.Fatalf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestListEventsFilterByStatus(t *testing.T) {
	store := setupTestStore(t)

	e1 := newTestEvent("已发布")
	if err := store.CreateEvent(e1); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	draft := "draft"
	store.UpdateEvent(e1.ID, model.UpdateEventReq{Status: &draft})

	e2 := newTestEvent("已发布2")
	if err := store.CreateEvent(e2); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	cancelled := "cancelled"
	store.UpdateEvent(e2.ID, model.UpdateEventReq{Status: &cancelled})

	e3 := newTestEvent("已发布3")
	if err := store.CreateEvent(e3); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	events, _, _ := store.ListEvents(model.ListEventsParams{Status: "published"})
	if len(events) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(events))
	}

	events, _, _ = store.ListEvents(model.ListEventsParams{Status: "draft"})
	if len(events) != 1 {
		t.Fatalf("expected 1 draft event, got %d", len(events))
	}
}

func TestListEventsFilterByPriceType(t *testing.T) {
	store := setupTestStore(t)

	freeEvent := newTestEvent("免费活动")
	if err := store.CreateEvent(freeEvent); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	paidEvent := newTestEvent("付费活动")
	paidEvent.Price = 99.9
	if err := store.CreateEvent(paidEvent); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	events, _, _ := store.ListEvents(model.ListEventsParams{PriceType: "free"})
	if len(events) != 1 {
		t.Fatalf("expected 1 free event, got %d", len(events))
	}

	events, _, _ = store.ListEvents(model.ListEventsParams{PriceType: "paid"})
	if len(events) != 1 {
		t.Fatalf("expected 1 paid event, got %d", len(events))
	}
}

func TestListEventsSearchByKeyword(t *testing.T) {
	store := setupTestStore(t)

	e1 := newTestEvent("Go 入门讲座")
	e1.Description = "从零开始学习 Go 语言"
	if err := store.CreateEvent(e1); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	e2 := newTestEvent("Docker 实战")
	e2.Description = "Docker 容器化部署"
	if err := store.CreateEvent(e2); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	events, _, _ := store.ListEvents(model.ListEventsParams{Keyword: "Go"})
	if len(events) != 1 {
		t.Fatalf("expected 1 event matching 'Go', got %d", len(events))
	}

	events, _, _ = store.ListEvents(model.ListEventsParams{Keyword: "Docker"})
	if len(events) != 1 {
		t.Fatalf("expected 1 event matching 'Docker', got %d", len(events))
	}

	events, _, _ = store.ListEvents(model.ListEventsParams{Keyword: "不存在的"})
	if len(events) != 0 {
		t.Fatalf("expected 0 events matching '不存在的', got %d", len(events))
	}
}

func TestListEventsCombinedFilter(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("Go 付费讲座")
	e.Price = 199
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	e2 := newTestEvent("Go 免费讲座")
	if err := store.CreateEvent(e2); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	events, _, _ := store.ListEvents(model.ListEventsParams{PriceType: "paid", Keyword: "Go"})
	if len(events) != 1 {
		t.Fatalf("expected 1 paid event matching 'Go', got %d", len(events))
	}
	if events[0].Price != 199 {
		t.Fatalf("expected price 199, got %f", events[0].Price)
	}
}

func TestDeleteEventCascadeTickets(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("级联门票")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	ticket := &model.Ticket{EventID: e.ID, Name: "票", Price: 0, Stock: 1}
	if err := store.CreateTicket(ticket); err != nil {
		t.Fatalf("CreateTicket failed: %v", err)
	}

	if err := store.DeleteEvent(e.ID); err != nil {
		t.Fatalf("DeleteEvent failed: %v", err)
	}

	tickets, _, _ := store.ListTickets(e.ID, 0, 0)
	if len(tickets) != 0 {
		t.Fatal("expected 0 tickets after cascade delete")
	}
}

func TestCreateOrganizer(t *testing.T) {
	store := setupTestStore(t)
	o := &model.Organizer{Name: "测试门店", Description: "描述", Contact: "1380000", Tags: "教育,讲座"}
	if err := store.CreateOrganizer(o); err != nil {
		t.Fatalf("CreateOrganizer failed: %v", err)
	}
	if o.ID != 2 {
		t.Fatalf("expected ID 2, got %d", o.ID)
	}
}

func TestGetOrganizer(t *testing.T) {
	store := setupTestStore(t)
	o, err := store.GetOrganizer(1)
	if err != nil {
		t.Fatalf("GetOrganizer failed: %v", err)
	}
	if o == nil {
		t.Fatal("expected organizer, got nil")
	}
	if o.Name != "默认门店" {
		t.Fatalf("expected 默认门店, got %s", o.Name)
	}
}

func TestGetOrganizerNotFound(t *testing.T) {
	store := setupTestStore(t)
	o, err := store.GetOrganizer(999)
	if err != nil {
		t.Fatalf("GetOrganizer err: %v", err)
	}
	if o != nil {
		t.Fatal("expected nil for not found")
	}
}

func TestListOrganizers(t *testing.T) {
	store := setupTestStore(t)
	organizers, total, err := store.ListOrganizers(0, 10)
	if err != nil {
		t.Fatalf("ListOrganizers failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 total, got %d", total)
	}
	if len(organizers) != 1 {
		t.Fatalf("expected 1 organizer, got %d", len(organizers))
	}
	if organizers[0].EventCount < 0 {
		t.Fatal("expected non-negative event_count")
	}
}

func TestUpdateOrganizer(t *testing.T) {
	store := setupTestStore(t)
	newName := "更新名称"
	o, err := store.UpdateOrganizer(1, model.UpdateOrganizerReq{Name: &newName})
	if err != nil {
		t.Fatalf("UpdateOrganizer failed: %v", err)
	}
	if o.Name != "更新名称" {
		t.Fatalf("expected 更新名称, got %s", o.Name)
	}
}

func TestUpdateOrganizerNotFound(t *testing.T) {
	store := setupTestStore(t)
	newName := "X"
	_, err := store.UpdateOrganizer(999, model.UpdateOrganizerReq{Name: &newName})
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestDeleteOrganizer(t *testing.T) {
	store := setupTestStore(t)
	if err := store.DeleteOrganizer(1); err != nil {
		t.Fatalf("DeleteOrganizer failed: %v", err)
	}
	o, err := store.GetOrganizer(1)
	if err != nil {
		t.Fatalf("GetOrganizer after delete: %v", err)
	}
	if o != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestDeleteOrganizerNotFound(t *testing.T) {
	store := setupTestStore(t)
	err := store.DeleteOrganizer(999)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestDeleteOrganizerUnlinkEvents(t *testing.T) {
	store := setupTestStore(t)
	e := newTestEvent("归属活动")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if err := store.DeleteOrganizer(1); err != nil {
		t.Fatalf("DeleteOrganizer: %v", err)
	}
	ev, err := store.GetEvent(e.ID)
	if err != nil {
		t.Fatalf("GetEvent: %v", err)
	}
	if ev.OrganizerID != 0 {
		t.Fatalf("expected organizer_id 0, got %d", ev.OrganizerID)
	}
}

func TestCreateUser(t *testing.T) {
	store := setupTestStore(t)
	u := &model.User{Name: "测试用户", Contact: "13800001111", PasswordHash: "hashed"}
	if err := store.CreateUser(u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if u.ID != 1 {
		t.Fatalf("expected ID 1, got %d", u.ID)
	}
}

func TestGetUserByContact(t *testing.T) {
	store := setupTestStore(t)
	u := &model.User{Name: "张三", Contact: "zhang@test.com", PasswordHash: "xxx"}
	store.CreateUser(u)

	found, err := store.GetUserByContact("zhang@test.com")
	if err != nil {
		t.Fatalf("GetUserByContact: %v", err)
	}
	if found == nil {
		t.Fatal("expected user, got nil")
	}
	if found.Name != "张三" {
		t.Fatalf("expected 张三, got %s", found.Name)
	}
}

func TestGetUserByContactNotFound(t *testing.T) {
	store := setupTestStore(t)
	u, err := store.GetUserByContact("no@exist.com")
	if err != nil {
		t.Fatalf("GetUserByContact: %v", err)
	}
	if u != nil {
		t.Fatal("expected nil")
	}
}

func TestCreateUserDuplicate(t *testing.T) {
	store := setupTestStore(t)
	u := &model.User{Name: "重复用户", Contact: "dup_user@test.com", PasswordHash: "aaa"}
	if err := store.CreateUser(u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	err := store.CreateUser(&model.User{Name: "重复用户2", Contact: "dup_user@test.com", PasswordHash: "bbb"})
	if err == nil {
		t.Fatal("expected error for duplicate contact")
	}
}

func TestPing(t *testing.T) {
	store := setupTestStore(t)
	if err := store.Ping(); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestPingNilDB(t *testing.T) {
	store := &Store{}
	err := store.Ping()
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewStoreReturnsInitializationError(t *testing.T) {
	if _, err := NewStore(t.TempDir()); err == nil {
		t.Fatal("expected NewStore to return an error for a directory path")
	}
}

func TestClose(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()
	if err := store.Ping(); err != nil {
		t.Fatalf("should work before close: %v", err)
	}
}

func TestCancelRegistrationStore(t *testing.T) {
	store := setupTestStore(t)
	e := newTestEvent("取消测试")
	e.EventTime = "2099-12-31T18:00:00+08:00"
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if err := store.Register(&model.Registration{EventID: e.ID, Name: "张三", Contact: "zs@test.com"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := store.CancelRegistration(e.ID, "zs@test.com"); err != nil {
		t.Fatalf("CancelRegistration: %v", err)
	}
	regs, _, _ := store.ListRegistrations(e.ID, 0, 0)
	if len(regs) != 0 {
		t.Fatal("expected 0 registrations after cancel")
	}
}

func TestCancelRegistrationStoreNotFound(t *testing.T) {
	store := setupTestStore(t)
	err := store.CancelRegistration(999, "no@exist.com")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestCancelRegistrationStoreWithTicket(t *testing.T) {
	store := setupTestStore(t)
	e := newTestEvent("取消退票")
	e.EventTime = "2099-12-31T18:00:00+08:00"
	store.CreateEvent(e)
	ticket := &model.Ticket{EventID: e.ID, Name: "VIP", Price: 99, Stock: 10}
	store.CreateTicket(ticket)
	tid := ticket.ID
	store.Register(&model.Registration{EventID: e.ID, Name: "李四", Contact: "ls@test.com", TicketID: &tid})

	if err := store.CancelRegistration(e.ID, "ls@test.com"); err != nil {
		t.Fatalf("CancelRegistration: %v", err)
	}
	updated, _ := store.GetTicket(tid)
	if updated.Stock != 10 {
		t.Fatalf("expected stock restored to 10, got %d", updated.Stock)
	}
}

func TestRegisterStoreFull(t *testing.T) {
	store := setupTestStore(t)
	e := newTestEvent("满员")
	e.Capacity = 1
	e.EventTime = "2099-12-31T18:00:00+08:00"
	store.CreateEvent(e)
	store.Register(&model.Registration{EventID: e.ID, Name: "用户1", Contact: "u1@test.com"})
	err := store.Register(&model.Registration{EventID: e.ID, Name: "用户2", Contact: "u2@test.com"})
	if err == nil {
		t.Fatal("expected ErrFull")
	}
}

func TestRegisterStoreDuplicate(t *testing.T) {
	store := setupTestStore(t)
	e := newTestEvent("重复")
	e.EventTime = "2099-12-31T18:00:00+08:00"
	store.CreateEvent(e)
	store.Register(&model.Registration{EventID: e.ID, Name: "赵六", Contact: "zhao@test.com"})
	err := store.Register(&model.Registration{EventID: e.ID, Name: "赵六", Contact: "zhao@test.com"})
	if err == nil {
		t.Fatal("expected ErrDuplicate")
	}
}

func TestListEventsOrgFilter(t *testing.T) {
	store := setupTestStore(t)

	o2 := &model.Organizer{Name: "门店2"}
	store.CreateOrganizer(o2)

	e1 := newTestEvent("活动A")
	e1.OrganizerID = 1
	store.CreateEvent(e1)
	e2 := newTestEvent("活动B")
	e2.OrganizerID = o2.ID
	store.CreateEvent(e2)

	events, _, _ := store.ListEvents(model.ListEventsParams{OrganizerID: 1})
	if len(events) != 1 {
		t.Fatalf("expected 1 event for org 1, got %d", len(events))
	}
	evs2, _, _ := store.ListEvents(model.ListEventsParams{OrganizerID: o2.ID})
	if len(evs2) != 1 {
		t.Fatalf("expected 1 event for org 2, got %d", len(evs2))
	}
}

func TestListRegistrationsStoreEmpty(t *testing.T) {
	store := setupTestStore(t)
	e := newTestEvent("空报名")
	store.CreateEvent(e)
	_, total, _ := store.ListRegistrations(e.ID, 0, 0)
	if total != 0 {
		t.Fatalf("expected 0, got %d", total)
	}
}

func TestListTicketsStoreEmpty(t *testing.T) {
	store := setupTestStore(t)
	e := newTestEvent("无票")
	store.CreateEvent(e)
	_, total, _ := store.ListTickets(e.ID, 0, 0)
	if total != 0 {
		t.Fatalf("expected 0, got %d", total)
	}
}

func TestListPostsStoreEmpty(t *testing.T) {
	store := setupTestStore(t)
	e := newTestEvent("无帖")
	store.CreateEvent(e)
	_, total, _ := store.ListPosts(e.ID, 0, 0)
	if total != 0 {
		t.Fatalf("expected 0, got %d", total)
	}
}

func TestCreateEventMissingOrganizer(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()
	_ = store.CreateOrganizer(&model.Organizer{Name: "测试"})

	e := newTestEvent("无门店活动")
	e.OrganizerID = 0
	err = store.CreateEvent(e)
	if err != nil {
		t.Fatalf("CreateEvent with organizer_id=0 should work: %v", err)
	}
	ev, _ := store.GetEvent(e.ID)
	if ev.OrganizerName != "" {
		t.Fatalf("expected empty organizer_name for id=0, got %s", ev.OrganizerName)
	}
}

func TestUpdateEventStatusChange(t *testing.T) {
	store := setupTestStore(t)
	e := newTestEvent("状态更新2")
	store.CreateEvent(e)

	status := "ended"
	ev, err := store.UpdateEvent(e.ID, model.UpdateEventReq{Status: &status})
	if err != nil {
		t.Fatalf("UpdateEvent: %v", err)
	}
	if ev.Status != "ended" {
		t.Fatalf("expected ended, got %s", ev.Status)
	}
}

func TestUpdateEventLocationField(t *testing.T) {
	store := setupTestStore(t)
	e := newTestEvent("地点更新")
	store.CreateEvent(e)

	loc := "新地址"
	ev, err := store.UpdateEvent(e.ID, model.UpdateEventReq{Location: &loc})
	if err != nil {
		t.Fatalf("UpdateEvent: %v", err)
	}
	if ev.Location != "新地址" {
		t.Fatalf("expected 新地址, got %s", ev.Location)
	}
}

func TestDeleteEventWithPosts(t *testing.T) {
	store := setupTestStore(t)
	e := newTestEvent("级联帖子")
	store.CreateEvent(e)

	p := &model.Post{EventID: e.ID, AuthorName: "作者", AuthorContact: "a@t.com", Title: "帖", Content: "内"}
	store.CreatePost(p)

	if err := store.DeleteEvent(e.ID); err != nil {
		t.Fatalf("DeleteEvent: %v", err)
	}

	got, _ := store.GetPost(p.ID)
	if got != nil {
		t.Fatal("expected nil post after cascade delete")
	}
}

func TestListOrganizersPageOne(t *testing.T) {
	store := setupTestStore(t)
	o2 := &model.Organizer{Name: "门店二"}
	store.CreateOrganizer(o2)

	orgs, total, _ := store.ListOrganizers(0, 1)
	if total != 2 {
		t.Fatalf("expected 2 total, got %d", total)
	}
	if len(orgs) != 1 {
		t.Fatalf("expected 1 page, got %d", len(orgs))
	}
}

func TestUpdateOrganizerMultipleFields(t *testing.T) {
	store := setupTestStore(t)
	desc := "新描述"
	addr := "新地址"
	o, err := store.UpdateOrganizer(1, model.UpdateOrganizerReq{Description: &desc, Address: &addr})
	if err != nil {
		t.Fatalf("UpdateOrganizer: %v", err)
	}
	if o.Description != "新描述" || o.Address != "新地址" {
		t.Fatal("expected updated fields")
	}
}

func TestDeleteOrganizerWithEvents(t *testing.T) {
	store := setupTestStore(t)
	e := newTestEvent("归属活动2")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if err := store.DeleteOrganizer(1); err != nil {
		t.Fatalf("DeleteOrganizer: %v", err)
	}
	ev, _ := store.GetEvent(e.ID)
	if ev.OrganizerID != 0 {
		t.Fatalf("expected organizer_id 0 after delete, got %d", ev.OrganizerID)
	}
}

func TestGetUserByID(t *testing.T) {
	store := setupTestStore(t)
	u := &model.User{Name: "李四", Contact: "lisi@test.com", PasswordHash: "yyy"}
	store.CreateUser(u)

	found, err := store.GetUserByID(u.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if found == nil {
		t.Fatal("expected user, got nil")
	}
	if found.Contact != "lisi@test.com" {
		t.Fatalf("expected lisi@test.com, got %s", found.Contact)
	}
}

func TestGetUserByIDNotFound(t *testing.T) {
	store := setupTestStore(t)
	u, err := store.GetUserByID(999)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if u != nil {
		t.Fatal("expected nil")
	}
}

func TestSQLiteForeignKeysEnabled(t *testing.T) {
	s := setupTestStore(t)
	var enabled int
	if err := s.db.QueryRow(`PRAGMA foreign_keys`).Scan(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled != 1 {
		t.Fatalf("expected foreign_keys=1, got %d", enabled)
	}
	_, err := s.db.Exec(`INSERT INTO events
		(organizer_id, title, event_time, location, capacity, created_at, updated_at)
		VALUES (999, '孤儿活动', '2099-12-31T18:00:00+08:00', '线上', 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`)
	if err == nil {
		t.Fatal("expected foreign key violation for missing organizer")
	}
}

func TestDeleteTicketPreservesRegistrationSnapshot(t *testing.T) {
	s := setupTestStore(t)
	event := newTestEvent("删除门票引用")
	if err := s.CreateEvent(event); err != nil {
		t.Fatal(err)
	}
	ticket := &model.Ticket{EventID: event.ID, Name: "历史票种", Stock: 1}
	if err := s.CreateTicket(ticket); err != nil {
		t.Fatal(err)
	}
	user := &model.User{Name: "报名用户", Contact: "snapshot@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	ticketID := ticket.ID
	if err := s.Register(&model.Registration{
		EventID: event.ID, UserID: &user.ID, Name: user.Name, Contact: user.Contact, TicketID: &ticketID,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteTicket(ticket.ID); err != nil {
		t.Fatal(err)
	}
	registrations, _, err := s.ListRegistrations(event.ID, 0, 0)
	if err != nil || len(registrations) != 1 {
		t.Fatalf("registrations=%+v err=%v", registrations, err)
	}
	if registrations[0].TicketID != nil || registrations[0].TicketName != "历史票种" {
		t.Fatalf("ticket snapshot not preserved: %+v", registrations[0])
	}
}
