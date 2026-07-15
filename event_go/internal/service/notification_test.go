package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

type fakeNotificationRepository struct {
	listParams      model.ListNotificationsParams
	notifications   []*model.Notification
	total           int
	unread          int
	markedUserID    int64
	markedID        int64
	markedAt        time.Time
	markAllUpdated  int
	dispatchNow     time.Time
	dispatchWindow  time.Duration
	dispatchCreated int
	err             error
}

func (r *fakeNotificationRepository) ListNotifications(params model.ListNotificationsParams) ([]*model.Notification, int, error) {
	r.listParams = params
	return r.notifications, r.total, r.err
}

func (r *fakeNotificationRepository) CountUnreadNotifications(int64) (int, error) {
	return r.unread, r.err
}

func (r *fakeNotificationRepository) MarkNotificationRead(userID, notificationID int64, readAt time.Time) error {
	r.markedUserID, r.markedID, r.markedAt = userID, notificationID, readAt
	return r.err
}

func (r *fakeNotificationRepository) MarkAllNotificationsRead(userID int64, readAt time.Time) (int, error) {
	r.markedUserID, r.markedAt = userID, readAt
	return r.markAllUpdated, r.err
}

func (r *fakeNotificationRepository) CreateDueEventReminders(now time.Time, window time.Duration) (int, error) {
	r.dispatchNow, r.dispatchWindow = now, window
	return r.dispatchCreated, r.err
}

func TestNotificationServiceScopesQueriesAndUsesBusinessClock(t *testing.T) {
	now := time.Date(2034, 5, 6, 7, 8, 9, 0, time.UTC)
	repository := &fakeNotificationRepository{
		notifications: []*model.Notification{{ID: 1}}, total: 3, unread: 2, markAllUpdated: 2,
	}
	service := NewNotificationService(repository, fixedClock{now: now}, 24*time.Hour)

	notifications, total, err := service.List(7, true, 20, 10)
	if err != nil || total != 3 || len(notifications) != 1 {
		t.Fatalf("unexpected list result: notifications=%+v total=%d err=%v", notifications, total, err)
	}
	if repository.listParams.UserID != 7 || !repository.listParams.UnreadOnly || repository.listParams.Offset != 20 || repository.listParams.Limit != 10 {
		t.Fatalf("unexpected list params: %+v", repository.listParams)
	}
	if unread, err := service.UnreadCount(7); err != nil || unread != 2 {
		t.Fatalf("unexpected unread count: unread=%d err=%v", unread, err)
	}
	if err := service.MarkRead(7, 9); err != nil {
		t.Fatal(err)
	}
	if repository.markedUserID != 7 || repository.markedID != 9 || !repository.markedAt.Equal(now) {
		t.Fatalf("mark read did not use trusted scope and clock: %+v", repository)
	}
	updated, err := service.MarkAllRead(7)
	if err != nil || updated != 2 || !repository.markedAt.Equal(now) {
		t.Fatalf("mark all read mismatch: updated=%d repo=%+v err=%v", updated, repository, err)
	}
	created, err := service.DispatchDueReminders()
	if err != nil || created != 0 || !repository.dispatchNow.Equal(now) || repository.dispatchWindow != 24*time.Hour {
		t.Fatalf("dispatch mismatch: created=%d repo=%+v err=%v", created, repository, err)
	}
}

func TestNotificationDispatcherRunsImmediatelyAndReportsFailures(t *testing.T) {
	repository := &fakeNotificationRepository{err: errors.New("dispatch failed")}
	service := NewNotificationService(repository, fixedClock{now: time.Now()}, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	reported := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		service.RunReminderDispatcher(ctx, time.Hour, func(err error) {
			reported <- err
			cancel()
		})
		close(done)
	}()
	select {
	case err := <-reported:
		if err == nil || err.Error() != "dispatch failed" {
			t.Fatalf("unexpected reported error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("dispatcher did not run immediately")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("dispatcher did not stop after cancellation")
	}
}
