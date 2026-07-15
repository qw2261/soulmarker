package service

import (
	"errors"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/clock"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

type fixedClock struct {
	now time.Time
}

type fixedCredentialGenerator struct {
	value string
	err   error
}

func (g fixedCredentialGenerator) NewCredential() (string, error) {
	return g.value, g.err
}

func newTestRegistrationService(repository RegistrationRepository, businessClock clock.Clock) *RegistrationService {
	return NewRegistrationService(repository, businessClock, 24*time.Hour, fixedCredentialGenerator{value: "00112233445566778899aabbccddeeff"})
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type fakeRegistrationRepository struct {
	event             *model.Event
	getEventErr       error
	registerErr       error
	cancelErr         error
	registered        *model.Registration
	cancelledEventID  int64
	cancelledUserID   int64
	registrationCalls int
	cancelCalls       int
	registeredStatus  bool
	registeredErr     error
}

func (r *fakeRegistrationRepository) GetEvent(int64) (*model.Event, error) {
	return r.event, r.getEventErr
}

func (r *fakeRegistrationRepository) Register(registration *model.Registration) error {
	r.registrationCalls++
	r.registered = registration
	return r.registerErr
}

func (r *fakeRegistrationRepository) CancelRegistrationByUserID(eventID, userID int64) error {
	r.cancelCalls++
	r.cancelledEventID = eventID
	r.cancelledUserID = userID
	return r.cancelErr
}

func (r *fakeRegistrationRepository) IsRegisteredByUserID(int64, int64) (bool, error) {
	return r.registeredStatus, r.registeredErr
}

func TestRegistrationServiceGetEvent(t *testing.T) {
	repository := &fakeRegistrationRepository{}
	service := newTestRegistrationService(repository, fixedClock{})
	if _, err := service.GetEvent(99); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	repository.getEventErr = errors.New("database unavailable")
	if _, err := service.GetEvent(99); err == nil || errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestRegistrationServiceRegisterCopiesTrustedIdentity(t *testing.T) {
	repository := &fakeRegistrationRepository{}
	issuedAt := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	service := newTestRegistrationService(repository, fixedClock{now: issuedAt})
	event := &model.Event{ID: 7, Status: "published", Capacity: 10}
	user := &model.User{ID: 8, Name: "可信用户", Contact: "trusted@example.com"}
	ticketID := int64(9)

	registration, err := service.Register(event, user, &ticketID)
	if err != nil {
		t.Fatal(err)
	}
	if repository.registrationCalls != 1 || repository.registered != registration {
		t.Fatalf("registration was not persisted: %+v", repository)
	}
	if registration.UserID == nil || *registration.UserID != user.ID || registration.Name != user.Name || registration.Contact != user.Contact {
		t.Fatalf("trusted identity was not copied: %+v", registration)
	}
	if registration.Admission == nil || registration.Admission.CredentialCode != "00112233445566778899aabbccddeeff" {
		t.Fatalf("admission credential was not generated: %+v", registration.Admission)
	}
	if !registration.Admission.IssuedAt.Equal(issuedAt) {
		t.Fatalf("injected clock was not used: %s", registration.Admission.IssuedAt)
	}
}

func TestRegistrationServiceStopsWhenCredentialGenerationFails(t *testing.T) {
	repository := &fakeRegistrationRepository{}
	service := NewRegistrationService(repository, fixedClock{}, 24*time.Hour, fixedCredentialGenerator{err: errors.New("entropy unavailable")})
	_, err := service.Register(&model.Event{ID: 1, Status: "published"}, &model.User{ID: 2}, nil)
	if err == nil || repository.registrationCalls != 0 {
		t.Fatalf("credential failure must stop registration: calls=%d err=%v", repository.registrationCalls, err)
	}
}

func TestRegistrationServiceRejectsUnpublishedAndWrapsCapacity(t *testing.T) {
	repository := &fakeRegistrationRepository{}
	service := newTestRegistrationService(repository, fixedClock{})
	user := &model.User{ID: 1}

	if _, err := service.Register(&model.Event{ID: 1, Status: "draft"}, user, nil); !errors.Is(err, ErrEventNotPublished) {
		t.Fatalf("expected unpublished error, got %v", err)
	}
	if repository.registrationCalls != 0 {
		t.Fatal("repository must not be called for unpublished event")
	}

	repository.registerErr = model.ErrFull
	_, err := service.Register(&model.Event{ID: 1, Status: "published", Capacity: 25}, user, nil)
	var fullError *RegistrationFullError
	if !errors.As(err, &fullError) || !errors.Is(err, model.ErrFull) || fullError.Capacity != 25 {
		t.Fatalf("unexpected full error: %v", err)
	}
}

func TestRegistrationServiceCancelDeadlineBoundary(t *testing.T) {
	eventTime := time.Date(2030, 1, 2, 12, 0, 0, 0, time.UTC)
	deadline := eventTime.Add(-24 * time.Hour)
	event := &model.Event{ID: 12, EventTime: eventTime.Format(model.TimeFormat)}

	repository := &fakeRegistrationRepository{}
	service := newTestRegistrationService(repository, fixedClock{now: deadline})
	if err := service.Cancel(event, 34); err != nil {
		t.Fatalf("deadline instant should still allow cancellation: %v", err)
	}
	if repository.cancelCalls != 1 || repository.cancelledEventID != event.ID || repository.cancelledUserID != 34 {
		t.Fatalf("unexpected cancellation call: %+v", repository)
	}

	repository = &fakeRegistrationRepository{}
	service = newTestRegistrationService(repository, fixedClock{now: deadline.Add(time.Nanosecond)})
	if err := service.Cancel(event, 34); !errors.Is(err, model.ErrCancelDeadlineExceeded) {
		t.Fatalf("expected deadline error, got %v", err)
	}
	if repository.cancelCalls != 0 {
		t.Fatal("repository must not be called after deadline")
	}
}

func TestRegistrationServiceCancelPropagatesFailures(t *testing.T) {
	repository := &fakeRegistrationRepository{cancelErr: model.ErrNotRegistered}
	service := newTestRegistrationService(repository, fixedClock{now: time.Now()})
	if err := service.Cancel(&model.Event{ID: 1, EventTime: "invalid"}, 2); err == nil {
		t.Fatal("expected invalid persisted event time to fail")
	}

	eventTime := time.Now().Add(72 * time.Hour).Format(model.TimeFormat)
	if err := service.Cancel(&model.Event{ID: 1, EventTime: eventTime}, 2); !errors.Is(err, model.ErrNotRegistered) {
		t.Fatalf("expected repository cancellation error, got %v", err)
	}
}

func TestRegistrationServiceIsRegistered(t *testing.T) {
	repository := &fakeRegistrationRepository{registeredStatus: true}
	service := newTestRegistrationService(repository, fixedClock{})
	registered, err := service.IsRegistered(1, 2)
	if err != nil || !registered {
		t.Fatalf("expected registered result, registered=%v err=%v", registered, err)
	}

	repository.registeredErr = errors.New("lookup failed")
	if _, err := service.IsRegistered(1, 2); err == nil {
		t.Fatal("expected registration lookup failure")
	}
}
