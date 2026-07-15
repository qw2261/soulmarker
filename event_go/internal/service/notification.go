package service

import (
	"context"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/clock"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

type NotificationRepository interface {
	ListNotifications(params model.ListNotificationsParams) ([]*model.Notification, int, error)
	CountUnreadNotifications(userID int64) (int, error)
	MarkNotificationRead(userID, notificationID int64, readAt time.Time) error
	MarkAllNotificationsRead(userID int64, readAt time.Time) (int, error)
	CreateDueEventReminders(now time.Time, window time.Duration) (int, error)
}

type NotificationService struct {
	repository     NotificationRepository
	clock          clock.Clock
	reminderWindow time.Duration
}

func NewNotificationService(repository NotificationRepository, businessClock clock.Clock, reminderWindow time.Duration) *NotificationService {
	return &NotificationService{repository: repository, clock: businessClock, reminderWindow: reminderWindow}
}

func (s *NotificationService) List(userID int64, unreadOnly bool, offset, limit int) ([]*model.Notification, int, error) {
	return s.repository.ListNotifications(model.ListNotificationsParams{
		UserID: userID, UnreadOnly: unreadOnly, Offset: offset, Limit: limit,
	})
}

func (s *NotificationService) UnreadCount(userID int64) (int, error) {
	return s.repository.CountUnreadNotifications(userID)
}

func (s *NotificationService) MarkRead(userID, notificationID int64) error {
	return s.repository.MarkNotificationRead(userID, notificationID, s.clock.Now())
}

func (s *NotificationService) MarkAllRead(userID int64) (int, error) {
	return s.repository.MarkAllNotificationsRead(userID, s.clock.Now())
}

func (s *NotificationService) DispatchDueReminders() (int, error) {
	return s.repository.CreateDueEventReminders(s.clock.Now(), s.reminderWindow)
}

func (s *NotificationService) RunReminderDispatcher(ctx context.Context, interval time.Duration, reportError func(error)) {
	dispatch := func() {
		if _, err := s.DispatchDueReminders(); err != nil && reportError != nil {
			reportError(err)
		}
	}
	dispatch()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dispatch()
		}
	}
}
