package store

import (
	"errors"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func TestNotificationLifecycleIsTransactionalAndUserScoped(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	organizer := &model.Organizer{Name: "通知门店"}
	if err := s.CreateOrganizer(organizer); err != nil {
		t.Fatal(err)
	}
	event := &model.Event{
		OrganizerID: organizer.ID, Title: "通知活动", Description: "初始说明",
		EventTime: "2099-12-31T18:00:00+08:00", Location: "A 场地", Capacity: 10, Price: 10,
	}
	if err := s.CreateEvent(event); err != nil {
		t.Fatal(err)
	}
	user := &model.User{Name: "通知用户", Contact: "notify@example.com", PasswordHash: "hash"}
	other := &model.User{Name: "其他用户", Contact: "other-notify@example.com", PasswordHash: "hash"}
	for _, candidate := range []*model.User{user, other} {
		if err := s.CreateUser(candidate); err != nil {
			t.Fatal(err)
		}
	}
	registration := &model.Registration{
		EventID: event.ID, UserID: &user.ID, Name: user.Name, Contact: user.Contact,
	}
	if err := s.Register(registration); err != nil {
		t.Fatal(err)
	}
	notifications, total, err := s.ListNotifications(model.ListNotificationsParams{UserID: user.ID, Limit: 20})
	if err != nil || total != 1 || notifications[0].Type != model.NotificationRegistrationConfirmed {
		t.Fatalf("registration notification missing: total=%d notifications=%+v err=%v", total, notifications, err)
	}
	registrationNotificationID := notifications[0].ID
	if err := s.MarkNotificationRead(other.ID, registrationNotificationID, time.Now()); !errors.Is(err, model.ErrNotificationNotFound) {
		t.Fatalf("cross-user notification read allowed: %v", err)
	}
	newLocation := "B 场地"
	updated, err := s.UpdateEvent(event.ID, model.UpdateEventReq{Location: &newLocation})
	if err != nil || updated.Location != newLocation {
		t.Fatalf("update event: event=%+v err=%v", updated, err)
	}
	if err := s.CancelRegistrationByUserID(event.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	notifications, total, err = s.ListNotifications(model.ListNotificationsParams{UserID: user.ID, Limit: 20})
	if err != nil || total != 3 {
		t.Fatalf("unexpected notification lifecycle: total=%d notifications=%+v err=%v", total, notifications, err)
	}
	wantTypes := map[string]bool{
		model.NotificationRegistrationConfirmed: true,
		model.NotificationRegistrationCancelled: true,
		model.NotificationEventUpdated:          true,
	}
	for _, notification := range notifications {
		delete(wantTypes, notification.Type)
	}
	if len(wantTypes) != 0 {
		t.Fatalf("missing notification types: %+v", wantTypes)
	}
	if unread, err := s.CountUnreadNotifications(user.ID); err != nil || unread != 3 {
		t.Fatalf("unexpected unread count: unread=%d err=%v", unread, err)
	}
	updatedCount, err := s.MarkAllNotificationsRead(user.ID, time.Now())
	if err != nil || updatedCount != 3 {
		t.Fatalf("mark all read: updated=%d err=%v", updatedCount, err)
	}
	if err := s.DeleteEvent(event.ID); err != nil {
		t.Fatal(err)
	}
	notifications, _, err = s.ListNotifications(model.ListNotificationsParams{UserID: user.ID, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	for _, notification := range notifications {
		if notification.EventID != nil {
			t.Fatalf("deleted event reference retained: %+v", notification)
		}
	}
}

func TestDueEventRemindersAreIdempotentAndFollowEventTime(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	organizer := &model.Organizer{Name: "提醒门店"}
	if err := s.CreateOrganizer(organizer); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2033, 4, 5, 6, 0, 0, 0, time.UTC)
	eventTime := now.Add(2 * time.Hour).Format(model.TimeFormat)
	event := &model.Event{
		OrganizerID: organizer.ID, Title: "临近活动", EventTime: eventTime,
		Location: "提醒场地", Capacity: 10, Price: 10,
	}
	if err := s.CreateEvent(event); err != nil {
		t.Fatal(err)
	}
	user := &model.User{Name: "提醒用户", Contact: "reminder@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	if err := s.Register(&model.Registration{
		EventID: event.ID, UserID: &user.ID, Name: user.Name, Contact: user.Contact,
	}); err != nil {
		t.Fatal(err)
	}
	created, err := s.CreateDueEventReminders(now, 24*time.Hour)
	if err != nil || created != 1 {
		t.Fatalf("first reminder dispatch: created=%d err=%v", created, err)
	}
	created, err = s.CreateDueEventReminders(now.Add(time.Minute), 24*time.Hour)
	if err != nil || created != 0 {
		t.Fatalf("duplicate reminder dispatch: created=%d err=%v", created, err)
	}
	newTime := now.Add(3 * time.Hour).Format(model.TimeFormat)
	if _, err := s.UpdateEvent(event.ID, model.UpdateEventReq{EventTime: &newTime}); err != nil {
		t.Fatal(err)
	}
	created, err = s.CreateDueEventReminders(now.Add(2*time.Minute), 24*time.Hour)
	if err != nil || created != 1 {
		t.Fatalf("updated event reminder: created=%d err=%v", created, err)
	}
	notifications, _, err := s.ListNotifications(model.ListNotificationsParams{UserID: user.ID, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	reminders := 0
	for _, notification := range notifications {
		if notification.Type == model.NotificationEventReminder24H {
			reminders++
		}
	}
	if reminders != 2 {
		t.Fatalf("expected reminders for both event schedules, got %d", reminders)
	}
}
