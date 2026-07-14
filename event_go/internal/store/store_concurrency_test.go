package store

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func setupConcurrentStore(t *testing.T) *Store {
	t.Helper()
	s, err := OpenStore(filepath.Join(t.TempDir(), "concurrency.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.CreateOrganizer(&model.Organizer{Name: "并发门店"}); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestConcurrentRegistrationNeverExceedsCapacity(t *testing.T) {
	s := setupConcurrentStore(t)
	event := newTestEvent("并发容量")
	event.Capacity = 5
	if err := s.CreateEvent(event); err != nil {
		t.Fatal(err)
	}

	results := make(chan error, 20)
	var wg sync.WaitGroup
	users := make([]*model.User, 0, 20)
	for i := 1; i <= 20; i++ {
		user := &model.User{Name: fmt.Sprintf("用户%d", i), Contact: fmt.Sprintf("user%d@example.com", i), PasswordHash: "hash"}
		if err := s.CreateUser(user); err != nil {
			t.Fatal(err)
		}
		users = append(users, user)
	}
	for _, user := range users {
		wg.Add(1)
		go func(user *model.User) {
			defer wg.Done()
			results <- s.Register(&model.Registration{
				EventID: event.ID,
				UserID:  &user.ID,
				Name:    user.Name,
				Contact: user.Contact,
			})
		}(user)
	}
	wg.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		if !errors.Is(err, model.ErrFull) {
			t.Fatalf("unexpected registration error: %v", err)
		}
	}
	if successes != event.Capacity {
		t.Fatalf("expected %d successful registrations, got %d", event.Capacity, successes)
	}
	registrations, total, err := s.ListRegistrations(event.ID, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != event.Capacity || len(registrations) != event.Capacity {
		t.Fatalf("capacity exceeded: total=%d len=%d", total, len(registrations))
	}
}

func TestConcurrentTicketRegistrationNeverMakesStockNegative(t *testing.T) {
	s := setupConcurrentStore(t)
	event := newTestEvent("并发库存")
	event.Capacity = 20
	if err := s.CreateEvent(event); err != nil {
		t.Fatal(err)
	}
	ticket := &model.Ticket{EventID: event.ID, Name: "限量票", Stock: 3}
	if err := s.CreateTicket(ticket); err != nil {
		t.Fatal(err)
	}

	results := make(chan error, 10)
	var wg sync.WaitGroup
	users := make([]*model.User, 0, 10)
	for i := 1; i <= 10; i++ {
		user := &model.User{Name: fmt.Sprintf("票用户%d", i), Contact: fmt.Sprintf("ticket%d@example.com", i), PasswordHash: "hash"}
		if err := s.CreateUser(user); err != nil {
			t.Fatal(err)
		}
		users = append(users, user)
	}
	for _, user := range users {
		wg.Add(1)
		go func(user *model.User) {
			defer wg.Done()
			ticketID := ticket.ID
			results <- s.Register(&model.Registration{
				EventID:  event.ID,
				UserID:   &user.ID,
				Name:     user.Name,
				Contact:  user.Contact,
				TicketID: &ticketID,
			})
		}(user)
	}
	wg.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		if !errors.Is(err, model.ErrTicketSoldOut) {
			t.Fatalf("unexpected ticket registration error: %v", err)
		}
	}
	if successes != 3 {
		t.Fatalf("expected 3 successful ticket registrations, got %d", successes)
	}
	updated, err := s.GetTicket(ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Stock != 0 {
		t.Fatalf("expected stock 0, got %d", updated.Stock)
	}
}

func TestConcurrentCancellationRestoresTicketExactlyOnce(t *testing.T) {
	s := setupConcurrentStore(t)
	event := newTestEvent("并发取消")
	if err := s.CreateEvent(event); err != nil {
		t.Fatal(err)
	}
	ticket := &model.Ticket{EventID: event.ID, Name: "可退票", Stock: 1}
	if err := s.CreateTicket(ticket); err != nil {
		t.Fatal(err)
	}
	user := &model.User{Name: "取消用户", Contact: "cancel@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	ticketID := ticket.ID
	if err := s.Register(&model.Registration{
		EventID: event.ID, UserID: &user.ID, Name: user.Name, Contact: user.Contact, TicketID: &ticketID,
	}); err != nil {
		t.Fatal(err)
	}

	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- s.CancelRegistrationByUserID(event.ID, user.ID)
		}()
	}
	wg.Wait()
	close(results)

	successes := 0
	notRegistered := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, model.ErrNotRegistered):
			notRegistered++
		default:
			t.Fatalf("unexpected cancellation error: %v", err)
		}
	}
	if successes != 1 || notRegistered != 1 {
		t.Fatalf("unexpected cancellation results: success=%d not_registered=%d", successes, notRegistered)
	}
	updated, err := s.GetTicket(ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Stock != 1 {
		t.Fatalf("ticket stock restored more than once: %d", updated.Stock)
	}
}
