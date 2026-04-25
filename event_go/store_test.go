package main

import (
	"fmt"
	"testing"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	store := NewStore(":memory:")
	t.Cleanup(func() {
		store.Close()
	})
	return store
}

func newTestEvent(title string) *Event {
	return &Event{
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

	events, err := store.ListEvents("", "", "")
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

	events, err := store.ListEvents("", "", "")
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
	req := UpdateEventReq{Title: &newTitle}
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
	req := UpdateEventReq{Price: &newPrice}
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
	req := UpdateEventReq{Status: &status}
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
	req := UpdateEventReq{Title: &title}
	_, err := store.UpdateEvent(999, req)
	if err != ErrNotFound {
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
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteEventCascadeRegistration(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("级联删除测试")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	reg := &Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}
	if err := store.Register(reg); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if err := store.DeleteEvent(e.ID); err != nil {
		t.Fatalf("DeleteEvent failed: %v", err)
	}

	regs, _ := store.ListRegistrations(e.ID)
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
	if err := store.Register(&Registration{EventID: e.ID, Name: "李四", Contact: "ls@email.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	post := &Post{EventID: e.ID, AuthorName: "李四", AuthorContact: "ls@email.com", Title: "好活动", Content: "赞"}
	if err := store.CreatePost(post); err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}
	if err := store.CreateReply(&Reply{PostID: post.ID, AuthorName: "李四", AuthorContact: "ls@email.com", Content: "回复"}); err != nil {
		t.Fatalf("CreateReply failed: %v", err)
	}

	if err := store.DeleteEvent(e.ID); err != nil {
		t.Fatalf("DeleteEvent failed: %v", err)
	}

	posts, _ := store.ListPosts(e.ID)
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

	reg := &Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}
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

	reg := &Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}
	if err := store.Register(reg); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	err := store.Register(&Registration{EventID: e.ID, Name: "张三2", Contact: "zs@email.com"})
	if err != ErrDuplicate {
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
		err := store.Register(&Registration{EventID: e.ID, Name: name, Contact: name + "@email.com"})
		if err != nil {
			t.Fatalf("registration %d failed: %v", i, err)
		}
	}

	err := store.Register(&Registration{EventID: e.ID, Name: "王五", Contact: "ww@email.com"})
	if err != ErrFull {
		t.Fatalf("expected ErrFull, got %v", err)
	}
}

func TestRegisterNotFound(t *testing.T) {
	store := setupTestStore(t)

	err := store.Register(&Registration{EventID: 999, Name: "张三", Contact: "zs@email.com"})
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRegisterWithTicket(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("门票报名")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	ticket := &Ticket{EventID: e.ID, Name: "普通票", Price: 0, Stock: 5}
	if err := store.CreateTicket(ticket); err != nil {
		t.Fatalf("CreateTicket failed: %v", err)
	}

	ticketID := ticket.ID
	reg := &Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com", TicketID: &ticketID}
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
	ticket := &Ticket{EventID: e.ID, Name: "VIP票", Price: 100, Stock: 1}
	if err := store.CreateTicket(ticket); err != nil {
		t.Fatalf("CreateTicket failed: %v", err)
	}

	ticketID := ticket.ID
	if err := store.Register(&Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com", TicketID: &ticketID}); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	err := store.Register(&Registration{EventID: e.ID, Name: "李四", Contact: "ls@email.com", TicketID: &ticketID})
	if err != ErrTicketSoldOut {
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
	err := store.Register(&Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com", TicketID: &ticketID})
	if err != ErrTicketNotFound {
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
		reg := &Registration{EventID: e.ID, Name: name, Contact: name + "@email.com"}
		if err := store.Register(reg); err != nil {
			t.Fatalf("Register(%q) failed: %v", name, err)
		}
	}

	regs, err := store.ListRegistrations(e.ID)
	if err != nil {
		t.Fatalf("ListRegistrations failed: %v", err)
	}
	if len(regs) != 3 {
		t.Fatalf("expected 3 registrations, got %d", len(regs))
	}
}

func TestListRegistrationsEmpty(t *testing.T) {
	store := setupTestStore(t)

	regs, err := store.ListRegistrations(999)
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
	reg := &Registration{EventID: e.ID, Name: "测试", Contact: contact}
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
	if err := store.Register(&Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	post := &Post{EventID: e.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Title: "好活动", Content: "推荐给大家"}
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
	if err := store.Register(&Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	titles := []string{"帖子A", "帖子B", "帖子C"}
	for i, title := range titles {
		post := &Post{EventID: e.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Title: title, Content: "内容"}
		if err := store.CreatePost(post); err != nil {
			t.Fatalf("CreatePost %d failed: %v", i, err)
		}
	}

	posts, err := store.ListPosts(e.ID)
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

	posts, err := store.ListPosts(999)
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
	if err := store.Register(&Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	post := &Post{EventID: e.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Title: "详情测试", Content: "内容"}
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
	if err := store.Register(&Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	post := &Post{EventID: e.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Title: "帖子", Content: "内容"}
	if err := store.CreatePost(post); err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}

	reply := &Reply{PostID: post.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Content: "回复内容"}
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
	if err := store.Register(&Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	post := &Post{EventID: e.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Title: "帖子", Content: "内容"}
	if err := store.CreatePost(post); err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}

	contents := []string{"回复1", "回复2", "回复3"}
	for _, c := range contents {
		reply := &Reply{PostID: post.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Content: c}
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
	if err := store.Register(&Registration{EventID: e.ID, Name: "张三", Contact: "zs@email.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	post := &Post{EventID: e.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Title: "帖子", Content: "内容"}
	if err := store.CreatePost(post); err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		reply := &Reply{PostID: post.ID, AuthorName: "张三", AuthorContact: "zs@email.com", Content: "回复"}
		if err := store.CreateReply(reply); err != nil {
			t.Fatalf("CreateReply %d failed: %v", i, err)
		}
	}

	got, _ := store.GetPost(post.ID)
	if got.ReplyCount != 3 {
		t.Fatalf("expected ReplyCount 3, got %d", got.ReplyCount)
	}

	posts, _ := store.ListPosts(e.ID)
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

	ticket := &Ticket{EventID: e.ID, Name: "普通票", Price: 0, Stock: 100}
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
		ticket := &Ticket{EventID: e.ID, Name: fmt.Sprintf("票%d", i), Price: float64(i * 10), Stock: 10}
		if err := store.CreateTicket(ticket); err != nil {
			t.Fatalf("CreateTicket %d failed: %v", i, err)
		}
	}

	tickets, err := store.ListTickets(e.ID)
	if err != nil {
		t.Fatalf("ListTickets failed: %v", err)
	}
	if len(tickets) != 3 {
		t.Fatalf("expected 3 tickets, got %d", len(tickets))
	}
}

func TestListTicketsEmpty(t *testing.T) {
	store := setupTestStore(t)

	tickets, err := store.ListTickets(999)
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
	ticket := &Ticket{EventID: e.ID, Name: "VIP票", Price: 100, Stock: 5}
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

	e := newTestEvent("门票更新")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	ticket := &Ticket{EventID: e.ID, Name: "原名称", Price: 50, Stock: 10}
	if err := store.CreateTicket(ticket); err != nil {
		t.Fatalf("CreateTicket failed: %v", err)
	}

	newName := "新名称"
	newPrice := 99.9
	newStock := 5
	req := UpdateTicketReq{Name: &newName, Price: &newPrice, Stock: &newStock}
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
	req := UpdateTicketReq{Name: &name}
	_, err := store.UpdateTicket(999, req)
	if err != ErrTicketNotFound {
		t.Fatalf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestDeleteTicket(t *testing.T) {
	store := setupTestStore(t)

	e := newTestEvent("删除门票")
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	ticket := &Ticket{EventID: e.ID, Name: "待删除", Price: 0, Stock: 1}
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
	if err != ErrTicketNotFound {
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
	store.UpdateEvent(e1.ID, UpdateEventReq{Status: &draft})

	e2 := newTestEvent("已发布2")
	if err := store.CreateEvent(e2); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	cancelled := "cancelled"
	store.UpdateEvent(e2.ID, UpdateEventReq{Status: &cancelled})

	e3 := newTestEvent("已发布3")
	if err := store.CreateEvent(e3); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	events, _ := store.ListEvents("published", "", "")
	if len(events) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(events))
	}

	events, _ = store.ListEvents("draft", "", "")
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

	events, _ := store.ListEvents("", "free", "")
	if len(events) != 1 {
		t.Fatalf("expected 1 free event, got %d", len(events))
	}

	events, _ = store.ListEvents("", "paid", "")
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

	events, _ := store.ListEvents("", "", "Go")
	if len(events) != 1 {
		t.Fatalf("expected 1 event matching 'Go', got %d", len(events))
	}

	events, _ = store.ListEvents("", "", "Docker")
	if len(events) != 1 {
		t.Fatalf("expected 1 event matching 'Docker', got %d", len(events))
	}

	events, _ = store.ListEvents("", "", "不存在的")
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

	events, _ := store.ListEvents("", "paid", "Go")
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
	ticket := &Ticket{EventID: e.ID, Name: "票", Price: 0, Stock: 1}
	if err := store.CreateTicket(ticket); err != nil {
		t.Fatalf("CreateTicket failed: %v", err)
	}

	if err := store.DeleteEvent(e.ID); err != nil {
		t.Fatalf("DeleteEvent failed: %v", err)
	}

	tickets, _ := store.ListTickets(e.ID)
	if len(tickets) != 0 {
		t.Fatal("expected 0 tickets after cascade delete")
	}
}
