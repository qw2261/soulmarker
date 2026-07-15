package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/store"
)

type moderationFixture struct {
	store    *store.Store
	service  *ContentModerationService
	event    *model.Event
	author   *model.User
	reporter *model.User
	post     *model.Post
	reply    *model.Reply
}

func newModerationFixture(t *testing.T) moderationFixture {
	t.Helper()
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	organizer := &model.Organizer{Name: "治理测试门店"}
	if err := s.CreateOrganizer(organizer); err != nil {
		t.Fatal(err)
	}
	event := &model.Event{
		OrganizerID: organizer.ID, Title: "治理测试活动", EventTime: "2099-12-31T18:00:00+08:00",
		Location: "线上", Capacity: 20, Price: 1,
	}
	if err := s.CreateEvent(event); err != nil {
		t.Fatal(err)
	}
	author := &model.User{Name: "作者", Contact: "author@example.com", PasswordHash: "hash"}
	reporter := &model.User{Name: "举报者", Contact: "reporter@example.com", PasswordHash: "hash"}
	for _, user := range []*model.User{author, reporter} {
		if err := s.CreateUser(user); err != nil {
			t.Fatal(err)
		}
		userID := user.ID
		if err := s.Register(&model.Registration{
			EventID: event.ID, UserID: &userID, Name: user.Name, Contact: user.Contact,
		}); err != nil {
			t.Fatal(err)
		}
	}
	authorID := author.ID
	post := &model.Post{
		EventID: event.ID, UserID: &authorID, AuthorName: author.Name, AuthorContact: author.Contact,
		Title: "待治理帖子", Content: "帖子内容",
	}
	if err := s.CreatePost(post); err != nil {
		t.Fatal(err)
	}
	reply := &model.Reply{
		PostID: post.ID, UserID: &authorID, AuthorName: author.Name, AuthorContact: author.Contact,
		Content: "待治理回复",
	}
	if err := s.CreateReply(reply); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2031, 2, 3, 4, 5, 6, 0, time.UTC)
	return moderationFixture{
		store: s, service: NewContentModerationService(s, fixedClock{now: now}), event: event,
		author: author, reporter: reporter, post: post, reply: reply,
	}
}

func TestContentModerationReportIsValidatedAndIdempotent(t *testing.T) {
	fixture := newModerationFixture(t)
	report, created, err := fixture.service.ReportPost(
		fixture.event.ID, fixture.post.ID, fixture.reporter, model.ContentReportCategorySpam, "重复广告",
	)
	if err != nil || !created || report.ID == 0 {
		t.Fatalf("first report: report=%+v created=%v err=%v", report, created, err)
	}
	duplicate, created, err := fixture.service.ReportPost(
		fixture.event.ID, fixture.post.ID, fixture.reporter, model.ContentReportCategorySpam, "再次提交",
	)
	if err != nil || created || duplicate.ID != report.ID {
		t.Fatalf("duplicate report: report=%+v created=%v err=%v", duplicate, created, err)
	}
	if _, _, err := fixture.service.ReportPost(
		fixture.event.ID, fixture.post.ID, fixture.author, model.ContentReportCategoryAbuse, "自我举报",
	); !errors.Is(err, ErrContentReportOwnContent) {
		t.Fatalf("expected own content rejection, got %v", err)
	}
	if _, _, err := fixture.service.ReportReply(
		fixture.event.ID, fixture.post.ID, fixture.reply.ID, fixture.author, model.ContentReportCategoryAbuse, "自我举报",
	); !errors.Is(err, ErrContentReportOwnContent) {
		t.Fatalf("expected own reply rejection, got %v", err)
	}
	outsider := &model.User{ID: fixture.reporter.ID + 999, Name: "未报名用户"}
	if _, _, err := fixture.service.ReportPost(
		fixture.event.ID, fixture.post.ID, outsider, model.ContentReportCategorySpam, "未报名举报",
	); !errors.Is(err, model.ErrNotRegistered) {
		t.Fatalf("expected participation rejection, got %v", err)
	}
	if _, _, err := fixture.service.ReportPost(
		fixture.event.ID, fixture.post.ID, fixture.reporter, model.ContentReportCategoryOther, "",
	); !errors.Is(err, ErrContentReportDetailRequired) {
		t.Fatalf("expected detail requirement, got %v", err)
	}
	if _, _, err := fixture.service.ReportPost(
		fixture.event.ID, fixture.post.ID, fixture.reporter, model.ContentReportCategorySpam, strings.Repeat("长", 1001),
	); !errors.Is(err, ErrContentReportDetailTooLong) {
		t.Fatalf("expected length rejection, got %v", err)
	}
}

func TestContentModerationRemoveAndRestorePostKeepsAudit(t *testing.T) {
	fixture := newModerationFixture(t)
	report, _, err := fixture.service.ReportPost(
		fixture.event.ID, fixture.post.ID, fixture.reporter, model.ContentReportCategoryAbuse, "人身攻击",
	)
	if err != nil {
		t.Fatal(err)
	}
	secondReporter := &model.User{Name: "第二举报者", Contact: "second-reporter@example.com", PasswordHash: "hash"}
	if err := fixture.store.CreateUser(secondReporter); err != nil {
		t.Fatal(err)
	}
	if err := fixture.store.Register(&model.Registration{
		EventID: fixture.event.ID, UserID: &secondReporter.ID, Name: secondReporter.Name, Contact: secondReporter.Contact,
	}); err != nil {
		t.Fatal(err)
	}
	secondReport, created, err := fixture.service.ReportPost(
		fixture.event.ID, fixture.post.ID, secondReporter, model.ContentReportCategoryIllegal, "另一条待处理举报",
	)
	if err != nil || !created {
		t.Fatalf("second report: created=%v err=%v", created, err)
	}
	resolved, err := fixture.service.ResolveReport(report.ID, model.ContentResolutionRemove, "确认违规并移除", "platform_admin")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Status != model.ContentReportStatusResolved || resolved.TargetModerationStatus != model.ModerationStatusRemoved {
		t.Fatalf("unexpected resolution: %+v", resolved)
	}
	secondResolved, err := fixture.store.GetContentReport(secondReport.ID)
	if err != nil || secondResolved == nil || secondResolved.Status != model.ContentReportStatusResolved {
		t.Fatalf("related open report was not auto-resolved: report=%+v err=%v", secondResolved, err)
	}
	publicPost, err := fixture.store.GetPost(fixture.post.ID)
	if err != nil || publicPost != nil {
		t.Fatalf("removed post should be hidden: post=%+v err=%v", publicPost, err)
	}
	retained, err := fixture.store.GetPostForModeration(fixture.post.ID)
	if err != nil || retained == nil || retained.Content != fixture.post.Content {
		t.Fatalf("moderation evidence missing: post=%+v err=%v", retained, err)
	}
	if _, err := fixture.service.ResolveReport(report.ID, model.ContentResolutionRemove, "重复处理", "platform_admin"); !errors.Is(err, model.ErrContentReportResolved) {
		t.Fatalf("expected resolved conflict, got %v", err)
	}
	if err := fixture.service.RestorePost(fixture.event.ID, fixture.post.ID, "platform_admin", "复核后恢复"); err != nil {
		t.Fatal(err)
	}
	publicPost, err = fixture.store.GetPost(fixture.post.ID)
	if err != nil || publicPost == nil {
		t.Fatalf("restored post should be public: post=%+v err=%v", publicPost, err)
	}
	actions, total, err := fixture.service.ListActions(0, 20)
	if err != nil || total != 2 || len(actions) != 2 {
		t.Fatalf("unexpected audit actions: total=%d len=%d err=%v", total, len(actions), err)
	}
	if actions[0].Action != model.ContentModerationActionRestore || actions[1].Action != model.ContentModerationActionRemove {
		t.Fatalf("unexpected action order: %+v", actions)
	}
}

func TestContentModerationDismissReplyReportKeepsReplyVisible(t *testing.T) {
	fixture := newModerationFixture(t)
	report, created, err := fixture.service.ReportReply(
		fixture.event.ID, fixture.post.ID, fixture.reply.ID, fixture.reporter,
		model.ContentReportCategoryPrivacy, "疑似隐私信息",
	)
	if err != nil || !created {
		t.Fatalf("report reply: created=%v err=%v", created, err)
	}
	resolved, err := fixture.service.ResolveReport(report.ID, model.ContentResolutionDismiss, "未发现隐私信息", "platform_admin")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Status != model.ContentReportStatusDismissed || resolved.TargetModerationStatus != model.ModerationStatusVisible {
		t.Fatalf("unexpected dismissed report: %+v", resolved)
	}
	replies, err := fixture.store.ListReplies(fixture.post.ID)
	if err != nil || len(replies) != 1 {
		t.Fatalf("reply should remain visible: replies=%+v err=%v", replies, err)
	}
}

func TestContentModerationDirectReplyRemovalChecksScope(t *testing.T) {
	fixture := newModerationFixture(t)
	if err := fixture.service.RemoveReply(fixture.event.ID, fixture.post.ID+999, fixture.reply.ID, "platform_admin", "违规内容"); !errors.Is(err, model.ErrContentTargetNotFound) {
		t.Fatalf("expected scope rejection, got %v", err)
	}
	if err := fixture.service.RemoveReply(fixture.event.ID, fixture.post.ID, fixture.reply.ID, "platform_admin", "违规内容"); err != nil {
		t.Fatal(err)
	}
	replies, err := fixture.store.ListReplies(fixture.post.ID)
	if err != nil || len(replies) != 0 {
		t.Fatalf("removed reply should be hidden: replies=%+v err=%v", replies, err)
	}
	if err := fixture.service.RemoveReply(fixture.event.ID, fixture.post.ID, fixture.reply.ID, "platform_admin", "重复移除"); !errors.Is(err, model.ErrContentAlreadyRemoved) {
		t.Fatalf("expected already removed conflict, got %v", err)
	}
}
