package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/clock"
	"github.com/qw2261/soulmarker/event_go/internal/identifier"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

var ErrEventNotPublished = errors.New("活动未发布，暂无法报名")

type RegistrationFullError struct {
	Capacity int
}

func (e *RegistrationFullError) Error() string {
	return fmt.Sprintf("活动报名已满（上限 %d 人）", e.Capacity)
}

func (e *RegistrationFullError) Unwrap() error {
	return model.ErrFull
}

// RegistrationRepository 只描述报名用例实际需要的数据能力。
type RegistrationRepository interface {
	GetEvent(id int64) (*model.Event, error)
	Register(registration *model.Registration) error
	CancelRegistrationByUserID(eventID, userID int64) error
	IsRegisteredByUserID(eventID, userID int64) (bool, error)
}

type RegistrationService struct {
	repository     RegistrationRepository
	clock          clock.Clock
	cancelDeadline time.Duration
	credentials    identifier.CredentialGenerator
}

func NewRegistrationService(repository RegistrationRepository, businessClock clock.Clock, cancelDeadline time.Duration, credentials identifier.CredentialGenerator) *RegistrationService {
	return &RegistrationService{
		repository:     repository,
		clock:          businessClock,
		cancelDeadline: cancelDeadline,
		credentials:    credentials,
	}
}

func (s *RegistrationService) GetEvent(eventID int64) (*model.Event, error) {
	event, err := s.repository.GetEvent(eventID)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, model.ErrNotFound
	}
	return event, nil
}

func (s *RegistrationService) Register(event *model.Event, user *model.User, ticketID *int64) (*model.Registration, error) {
	if event.Status != "published" {
		return nil, ErrEventNotPublished
	}
	credentialCode, err := s.credentials.NewCredential()
	if err != nil {
		return nil, fmt.Errorf("generate admission credential: %w", err)
	}

	userID := user.ID
	registration := &model.Registration{
		EventID:  event.ID,
		UserID:   &userID,
		Name:     user.Name,
		Contact:  user.Contact,
		TicketID: ticketID,
		Admission: &model.Admission{
			CredentialCode: credentialCode,
			IssuedAt:       s.clock.Now().UTC(),
		},
	}
	if err := s.repository.Register(registration); err != nil {
		if errors.Is(err, model.ErrFull) {
			return nil, &RegistrationFullError{Capacity: event.Capacity}
		}
		return nil, err
	}
	return registration, nil
}

func (s *RegistrationService) Cancel(event *model.Event, userID int64) error {
	eventTime, err := time.Parse(model.TimeFormat, event.EventTime)
	if err != nil {
		return fmt.Errorf("parse event time: %w", err)
	}
	deadline := eventTime.Add(-s.cancelDeadline)
	if s.clock.Now().After(deadline) {
		return model.ErrCancelDeadlineExceeded
	}
	return s.repository.CancelRegistrationByUserID(event.ID, userID)
}

func (s *RegistrationService) IsRegistered(eventID, userID int64) (bool, error) {
	return s.repository.IsRegisteredByUserID(eventID, userID)
}
