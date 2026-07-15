package store

import (
	"errors"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func TestPreV13EventWritesStayTenantCompatible(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	first := &model.Organizer{Name: "旧应用门店一"}
	second := &model.Organizer{Name: "旧应用门店二"}
	if err := s.CreateOrganizer(first); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateOrganizer(second); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(model.TimeFormat)
	result, err := s.db.Exec(
		`INSERT INTO events
		 (organizer_id, title, description, cover_url, event_time, location, capacity, price, status, created_at, updated_at)
		 VALUES (?, '旧应用活动', '', '', '2099-01-01T00:00:00Z', '线上', 10, 0, 'published', ?, ?)`,
		first.ID, now, now,
	)
	if err != nil {
		t.Fatalf("pre-v13 event insert failed: %v", err)
	}
	eventID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	event, err := s.GetEvent(eventID)
	if err != nil || event == nil || event.OrganizationID != first.OrganizationID {
		t.Fatalf("insert compatibility tenant mismatch: event=%+v err=%v", event, err)
	}
	if _, err := s.db.Exec(`UPDATE events SET organizer_id = ? WHERE id = ?`, second.ID, eventID); err != nil {
		t.Fatalf("pre-v13 organizer update failed: %v", err)
	}
	event, err = s.GetEvent(eventID)
	if err != nil || event == nil || event.OrganizationID != second.OrganizationID {
		t.Fatalf("update compatibility tenant mismatch: event=%+v err=%v", event, err)
	}
	if err := s.DeleteOrganizer(second.ID); err != nil {
		t.Fatal(err)
	}
	event, err = s.GetEvent(eventID)
	if err != nil || event == nil || event.OrganizerID != 0 || event.OrganizationID != second.OrganizationID {
		t.Fatalf("organizer deletion lost stable tenant: event=%+v err=%v", event, err)
	}
}

func TestTenantScopedStoreOperationsCannotCrossOrganization(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	first := &model.Organizer{Name: "租户 A 门店"}
	second := &model.Organizer{Name: "租户 B 门店"}
	if err := s.CreateOrganizer(first); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateOrganizer(second); err != nil {
		t.Fatal(err)
	}
	firstEvent := &model.Event{OrganizerID: first.ID, Title: "租户 A 活动", EventTime: "2099-01-01T00:00:00Z", Location: "A", Capacity: 10}
	secondEvent := &model.Event{OrganizerID: second.ID, Title: "租户 B 活动", EventTime: "2099-01-01T00:00:00Z", Location: "B", Capacity: 10}
	if err := s.CreateEventForOrganization(first.OrganizationID, firstEvent); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateEventForOrganization(second.OrganizationID, secondEvent); err != nil {
		t.Fatal(err)
	}

	if event, err := s.GetEventForOrganization(first.OrganizationID, secondEvent.ID); err != nil || event != nil {
		t.Fatalf("cross-tenant event read escaped scope: event=%+v err=%v", event, err)
	}
	events, total, err := s.ListEventsForOrganization(first.OrganizationID, model.ListEventsParams{})
	if err != nil || total != 1 || len(events) != 1 || events[0].ID != firstEvent.ID {
		t.Fatalf("tenant event list mismatch: events=%+v total=%d err=%v", events, total, err)
	}
	changed := "越权修改"
	if _, err := s.UpdateEventForOrganization(first.OrganizationID, secondEvent.ID, model.UpdateEventReq{Title: &changed}); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("cross-tenant event update returned %v", err)
	}
	if err := s.DeleteEventForOrganization(first.OrganizationID, secondEvent.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("cross-tenant event delete returned %v", err)
	}
	if _, err := s.UpdateEventForOrganization(first.OrganizationID, firstEvent.ID, model.UpdateEventReq{OrganizerID: &second.ID}); !errors.Is(err, model.ErrOrganizerNotFound) {
		t.Fatalf("cross-tenant organizer reassignment returned %v", err)
	}
	if _, err := s.UpdateEvent(firstEvent.ID, model.UpdateEventReq{OrganizerID: &second.ID}); !errors.Is(err, model.ErrOrganizerNotFound) {
		t.Fatalf("generic event update moved tenant: %v", err)
	}
	if err := s.CreateEventForOrganization(first.OrganizationID, &model.Event{
		OrganizerID: second.ID, Title: "错配活动", EventTime: "2099-01-01T00:00:00Z", Location: "X", Capacity: 1,
	}); !errors.Is(err, model.ErrOrganizerNotFound) {
		t.Fatalf("cross-tenant event creation returned %v", err)
	}

	secondTicket := &model.Ticket{EventID: secondEvent.ID, Name: "租户 B 票", Stock: 10}
	if err := s.CreateTicketForOrganization(second.OrganizationID, secondTicket); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateTicketForOrganization(first.OrganizationID, &model.Ticket{EventID: secondEvent.ID, Name: "越权票", Stock: 1}); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("cross-tenant ticket creation returned %v", err)
	}
	if ticket, err := s.GetTicketForOrganization(first.OrganizationID, secondEvent.ID, secondTicket.ID); err != nil || ticket != nil {
		t.Fatalf("cross-tenant ticket read escaped scope: ticket=%+v err=%v", ticket, err)
	}
	if _, err := s.UpdateTicketForOrganization(first.OrganizationID, secondEvent.ID, secondTicket.ID, model.UpdateTicketReq{Name: &changed}); !errors.Is(err, model.ErrTicketNotFound) {
		t.Fatalf("cross-tenant ticket update returned %v", err)
	}
	if err := s.DeleteTicketForOrganization(first.OrganizationID, secondEvent.ID, secondTicket.ID); !errors.Is(err, model.ErrTicketNotFound) {
		t.Fatalf("cross-tenant ticket delete returned %v", err)
	}

	participant := &model.User{Name: "租户 B 参与者", Contact: "tenant-b-participant@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(participant); err != nil {
		t.Fatal(err)
	}
	credential := "0123456789abcdef0123456789abcdef"
	registration := &model.Registration{
		EventID: secondEvent.ID, UserID: &participant.ID, Name: participant.Name, Contact: participant.Contact,
		Admission: &model.Admission{CredentialCode: credential, IssuedAt: time.Now().UTC()},
	}
	if err := s.Register(registration); err != nil {
		t.Fatal(err)
	}
	registrations, total, err := s.ListRegistrationsForOrganization(first.OrganizationID, secondEvent.ID, 0, 20)
	if err != nil || total != 0 || len(registrations) != 0 {
		t.Fatalf("cross-tenant registrations escaped scope: registrations=%+v total=%d err=%v", registrations, total, err)
	}
	if _, _, err := s.CheckInForOrganization(first.OrganizationID, secondEvent.ID, credential, "organization_member:1", time.Now()); !errors.Is(err, model.ErrAdmissionNotFound) {
		t.Fatalf("cross-tenant checkin returned %v", err)
	}
	if _, _, err := s.CheckInForOrganization(second.OrganizationID, secondEvent.ID, credential, "organization_member:2", time.Now()); err != nil {
		t.Fatal(err)
	}
	checkins, total, err := s.ListCheckinsForOrganization(first.OrganizationID, secondEvent.ID, 0, 20)
	if err != nil || total != 0 || len(checkins) != 0 {
		t.Fatalf("cross-tenant checkins escaped scope: checkins=%+v total=%d err=%v", checkins, total, err)
	}
	if ticket, err := s.GetTicket(secondTicket.ID); err != nil || ticket == nil || ticket.Name != secondTicket.Name {
		t.Fatalf("cross-tenant attempts mutated ticket: ticket=%+v err=%v", ticket, err)
	}
}

func TestEventTenantMismatchIsRejectedByDatabase(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	first := &model.Organizer{Name: "边界门店一"}
	second := &model.Organizer{Name: "边界门店二"}
	if err := s.CreateOrganizer(first); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateOrganizer(second); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(model.TimeFormat)
	if _, err := s.db.Exec(
		`INSERT INTO events
		 (organization_id, organizer_id, title, event_time, location, capacity, created_at, updated_at)
		 VALUES (?, ?, '错配活动', '2099-01-01T00:00:00Z', '线上', 10, ?, ?)`,
		first.OrganizationID, second.ID, now, now,
	); err == nil {
		t.Fatal("event with mismatched organization and organizer was created")
	}
}
