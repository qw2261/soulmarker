package service

import (
	"fmt"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/clock"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

type OrganizationOperationsRepository interface {
	CreateEventForOrganization(organizationID int64, event *model.Event) error
	ListEventsForOrganization(organizationID int64, params model.ListEventsParams) ([]*model.Event, int, error)
	GetEventForOrganization(organizationID, eventID int64) (*model.Event, error)
	UpdateEventForOrganization(organizationID, eventID int64, req model.UpdateEventReq) (*model.Event, error)
	DeleteEventForOrganization(organizationID, eventID int64) error
	CreateTicketForOrganization(organizationID int64, ticket *model.Ticket) error
	ListTicketsForOrganization(organizationID, eventID int64, offset, limit int) ([]*model.Ticket, int, error)
	GetTicketForOrganization(organizationID, eventID, ticketID int64) (*model.Ticket, error)
	UpdateTicketForOrganization(organizationID, eventID, ticketID int64, req model.UpdateTicketReq) (*model.Ticket, error)
	DeleteTicketForOrganization(organizationID, eventID, ticketID int64) error
	ListRegistrationsForOrganization(organizationID, eventID int64, offset, limit int) ([]*model.Registration, int, error)
	CheckInForOrganization(organizationID, eventID int64, credential, actor string, checkedInAt time.Time) (*model.Checkin, bool, error)
	ListCheckinsForOrganization(organizationID, eventID int64, offset, limit int) ([]*model.Checkin, int, error)
}

type OrganizationOperationsService struct {
	repository OrganizationOperationsRepository
	clock      clock.Clock
}

func NewOrganizationOperationsService(
	repository OrganizationOperationsRepository,
	businessClock clock.Clock,
) *OrganizationOperationsService {
	return &OrganizationOperationsService{repository: repository, clock: businessClock}
}

func requireOrganizationScope(organizationID int64) error {
	if organizationID <= 0 {
		return fmt.Errorf("organization scope is required")
	}
	return nil
}

func (s *OrganizationOperationsService) CreateEvent(organizationID int64, event *model.Event) error {
	if err := requireOrganizationScope(organizationID); err != nil {
		return err
	}
	event.OrganizationID = organizationID
	return s.repository.CreateEventForOrganization(organizationID, event)
}

func (s *OrganizationOperationsService) ListEvents(
	organizationID int64,
	params model.ListEventsParams,
) ([]*model.Event, int, error) {
	if err := requireOrganizationScope(organizationID); err != nil {
		return nil, 0, err
	}
	return s.repository.ListEventsForOrganization(organizationID, params)
}

func (s *OrganizationOperationsService) GetEvent(organizationID, eventID int64) (*model.Event, error) {
	if err := requireOrganizationScope(organizationID); err != nil {
		return nil, err
	}
	return s.repository.GetEventForOrganization(organizationID, eventID)
}

func (s *OrganizationOperationsService) UpdateEvent(
	organizationID, eventID int64,
	req model.UpdateEventReq,
) (*model.Event, error) {
	if err := requireOrganizationScope(organizationID); err != nil {
		return nil, err
	}
	return s.repository.UpdateEventForOrganization(organizationID, eventID, req)
}

func (s *OrganizationOperationsService) DeleteEvent(organizationID, eventID int64) error {
	if err := requireOrganizationScope(organizationID); err != nil {
		return err
	}
	return s.repository.DeleteEventForOrganization(organizationID, eventID)
}

func (s *OrganizationOperationsService) CreateTicket(organizationID int64, ticket *model.Ticket) error {
	if err := requireOrganizationScope(organizationID); err != nil {
		return err
	}
	return s.repository.CreateTicketForOrganization(organizationID, ticket)
}

func (s *OrganizationOperationsService) ListTickets(
	organizationID, eventID int64,
	offset, limit int,
) ([]*model.Ticket, int, error) {
	if err := requireOrganizationScope(organizationID); err != nil {
		return nil, 0, err
	}
	return s.repository.ListTicketsForOrganization(organizationID, eventID, offset, limit)
}

func (s *OrganizationOperationsService) GetTicket(organizationID, eventID, ticketID int64) (*model.Ticket, error) {
	if err := requireOrganizationScope(organizationID); err != nil {
		return nil, err
	}
	return s.repository.GetTicketForOrganization(organizationID, eventID, ticketID)
}

func (s *OrganizationOperationsService) UpdateTicket(
	organizationID, eventID, ticketID int64,
	req model.UpdateTicketReq,
) (*model.Ticket, error) {
	if err := requireOrganizationScope(organizationID); err != nil {
		return nil, err
	}
	return s.repository.UpdateTicketForOrganization(organizationID, eventID, ticketID, req)
}

func (s *OrganizationOperationsService) DeleteTicket(organizationID, eventID, ticketID int64) error {
	if err := requireOrganizationScope(organizationID); err != nil {
		return err
	}
	return s.repository.DeleteTicketForOrganization(organizationID, eventID, ticketID)
}

func (s *OrganizationOperationsService) ListRegistrations(
	organizationID, eventID int64,
	offset, limit int,
) ([]*model.Registration, int, error) {
	if err := requireOrganizationScope(organizationID); err != nil {
		return nil, 0, err
	}
	return s.repository.ListRegistrationsForOrganization(organizationID, eventID, offset, limit)
}

func (s *OrganizationOperationsService) CheckIn(
	organizationID, eventID int64,
	credential, actor string,
) (*model.Checkin, bool, error) {
	if err := requireOrganizationScope(organizationID); err != nil {
		return nil, false, err
	}
	code, ok := normalizeCredential(credential)
	if !ok {
		return nil, false, model.ErrAdmissionNotFound
	}
	return s.repository.CheckInForOrganization(
		organizationID, eventID, code, actor, s.clock.Now(),
	)
}

func (s *OrganizationOperationsService) ListCheckins(
	organizationID, eventID int64,
	offset, limit int,
) ([]*model.Checkin, int, error) {
	if err := requireOrganizationScope(organizationID); err != nil {
		return nil, 0, err
	}
	return s.repository.ListCheckinsForOrganization(organizationID, eventID, offset, limit)
}
