package store

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func createAdmissionFixture(t *testing.T, ticketPrice *float64) (*Store, *model.Event, *model.User, *model.Registration) {
	t.Helper()
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	organizer := &model.Organizer{Name: "核销测试门店"}
	if err := store.CreateOrganizer(organizer); err != nil {
		t.Fatal(err)
	}
	event := &model.Event{
		OrganizerID: organizer.ID, Title: "核销测试活动",
		EventTime: time.Now().Add(72 * time.Hour).Format(model.TimeFormat),
		Location:  "测试场地", Capacity: 20,
	}
	if err := store.CreateEvent(event); err != nil {
		t.Fatal(err)
	}
	user := &model.User{Name: "核销用户", Contact: "checkin@example.com", PasswordHash: "hash"}
	if err := store.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	registration := &model.Registration{
		EventID: event.ID, UserID: &user.ID, Name: user.Name, Contact: user.Contact,
		Admission: &model.Admission{CredentialCode: "00112233445566778899aabbccddeeff"},
	}
	if ticketPrice != nil {
		ticket := &model.Ticket{EventID: event.ID, Name: "测试票", Price: *ticketPrice, Stock: 2}
		if err := store.CreateTicket(ticket); err != nil {
			t.Fatal(err)
		}
		registration.TicketID = &ticket.ID
	}
	if err := store.Register(registration); err != nil {
		t.Fatal(err)
	}
	return store, event, user, registration
}

func TestFreeRegistrationCreatesAdmissionAtomically(t *testing.T) {
	store, event, user, registration := createAdmissionFixture(t, nil)
	if registration.Admission == nil || registration.Admission.ID == 0 {
		t.Fatalf("free registration has no admission: %+v", registration)
	}
	admission, err := store.GetAdmissionByUser(event.ID, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if admission.CredentialCode != "00112233445566778899aabbccddeeff" || admission.Status != model.AdmissionStatusActive {
		t.Fatalf("unexpected admission: %+v", admission)
	}
	if admission.RegistrationID == nil || *admission.RegistrationID != registration.ID {
		t.Fatalf("admission is not linked to registration: %+v", admission)
	}
}

func TestPaidTicketDoesNotCreateAdmission(t *testing.T) {
	price := 10.0
	store, event, user, registration := createAdmissionFixture(t, &price)
	if registration.Admission != nil {
		t.Fatalf("paid ticket must not create admission: %+v", registration.Admission)
	}
	if _, err := store.GetAdmissionByUser(event.ID, user.ID); !errors.Is(err, model.ErrAdmissionNotFound) {
		t.Fatalf("expected no admission, got %v", err)
	}
}

func TestCheckInIsIdempotentAndImmutable(t *testing.T) {
	store, event, _, registration := createAdmissionFixture(t, nil)
	checkedAt := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	first, duplicate, err := store.CheckIn(event.ID, registration.Admission.CredentialCode, "test-admin", checkedAt)
	if err != nil || duplicate {
		t.Fatalf("first checkin failed: checkin=%+v duplicate=%v err=%v", first, duplicate, err)
	}
	if first.CredentialCode != registration.Admission.CredentialCode || first.UserName != "核销用户" || first.UserContact != "checkin@example.com" {
		t.Fatalf("first checkin response lacks admission context: %+v", first)
	}
	second, duplicate, err := store.CheckIn(event.ID, registration.Admission.CredentialCode, "test-admin", checkedAt.Add(time.Hour))
	if err != nil || !duplicate || second.ID != first.ID || !second.CheckedInAt.Equal(first.CheckedInAt) || second.CredentialCode != first.CredentialCode || second.UserName != first.UserName || second.UserContact != first.UserContact {
		t.Fatalf("duplicate checkin changed result: first=%+v second=%+v duplicate=%v err=%v", first, second, duplicate, err)
	}
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM checkins`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("expected one checkin, count=%d err=%v", count, err)
	}
	if _, err := store.db.Exec(`UPDATE checkins SET checked_in_by = 'other' WHERE id = ?`, first.ID); err == nil {
		t.Fatal("immutable checkin accepted update")
	}
	if _, err := store.db.Exec(`DELETE FROM checkins WHERE id = ?`, first.ID); err == nil {
		t.Fatal("immutable checkin accepted delete")
	}
	if err := store.CancelRegistrationByUserID(event.ID, *registration.UserID); !errors.Is(err, model.ErrAdmissionCheckedIn) {
		t.Fatalf("checked-in registration must not be cancelled: %v", err)
	}
}

func TestConcurrentDuplicateCheckInCreatesOneRecord(t *testing.T) {
	store, event, _, registration := createAdmissionFixture(t, nil)
	const workers = 16
	start := make(chan struct{})
	var wait sync.WaitGroup
	var mu sync.Mutex
	ids := make(map[int64]int)
	duplicates := 0
	errorsSeen := make([]error, 0)
	for i := 0; i < workers; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			checkin, duplicate, err := store.CheckIn(event.ID, registration.Admission.CredentialCode, "test-admin", time.Now())
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errorsSeen = append(errorsSeen, err)
				return
			}
			ids[checkin.ID]++
			if duplicate {
				duplicates++
			}
		}()
	}
	close(start)
	wait.Wait()
	if len(errorsSeen) != 0 || len(ids) != 1 || duplicates != workers-1 {
		t.Fatalf("non-idempotent concurrent result: ids=%v duplicates=%d errors=%v", ids, duplicates, errorsSeen)
	}
}

func TestCancellationRevokesAdmissionAndRestoresStock(t *testing.T) {
	price := 0.0
	store, event, user, registration := createAdmissionFixture(t, &price)
	if err := store.CancelRegistrationByUserID(event.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	admission, err := store.GetAdmissionByUser(event.ID, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if admission.Status != model.AdmissionStatusRevoked || admission.RevokedAt == nil || admission.RegistrationID != nil {
		t.Fatalf("admission was not revoked: %+v", admission)
	}
	if _, _, err := store.CheckIn(event.ID, registration.Admission.CredentialCode, "test-admin", time.Now()); !errors.Is(err, model.ErrAdmissionRevoked) {
		t.Fatalf("revoked admission was accepted: %v", err)
	}
	ticket, err := store.GetTicket(*registration.TicketID)
	if err != nil || ticket.Stock != 2 {
		t.Fatalf("ticket stock was not restored: ticket=%+v err=%v", ticket, err)
	}
}

func TestCheckInRejectsCredentialFromAnotherEvent(t *testing.T) {
	store, _, _, registration := createAdmissionFixture(t, nil)
	other := &model.Event{OrganizerID: 1, Title: "其他活动", EventTime: time.Now().Add(72 * time.Hour).Format(model.TimeFormat), Location: "其他", Capacity: 1}
	if err := store.CreateEvent(other); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.CheckIn(other.ID, registration.Admission.CredentialCode, "test-admin", time.Now()); !errors.Is(err, model.ErrAdmissionNotFound) {
		t.Fatalf("cross-event credential was accepted: %v", err)
	}
}

func TestAdmissionMigrationCreatesForeignKeysAndTriggers(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, table := range []string{"admissions", "checkins"} {
		var name string
		if err := store.db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name); err != nil {
			t.Fatalf("missing %s table: %v", table, err)
		}
	}
	var triggerCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'trigger' AND name LIKE 'checkins_immutable_%'`).Scan(&triggerCount); err != nil || triggerCount != 2 {
		t.Fatalf("immutable triggers missing: count=%d err=%v", triggerCount, err)
	}
}
