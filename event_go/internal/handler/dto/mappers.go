package dto

import "github.com/qw2261/soulmarker/event_go/internal/model"

func (r UpdateOrganizerRequest) Command() model.UpdateOrganizerReq {
	return model.UpdateOrganizerReq(r)
}

func (r UpdateEventRequest) Command() model.UpdateEventReq {
	return model.UpdateEventReq(r)
}

func (r UpdateTicketRequest) Command() model.UpdateTicketReq {
	return model.UpdateTicketReq(r)
}

func User(user *model.User) UserResponse {
	return UserResponse{ID: user.ID, Name: user.Name, Contact: user.Contact, CreatedAt: user.CreatedAt}
}

func Organizer(organizer *model.Organizer) OrganizerResponse {
	return OrganizerResponse{
		ID: organizer.ID, Name: organizer.Name, Description: organizer.Description,
		Contact: organizer.Contact, LogoURL: organizer.LogoURL, Address: organizer.Address,
		Website: organizer.Website, Tags: organizer.Tags, EventCount: organizer.EventCount,
		CreatedAt: organizer.CreatedAt, UpdatedAt: organizer.UpdatedAt,
	}
}

func Organizers(organizers []*model.Organizer) []OrganizerResponse {
	result := make([]OrganizerResponse, 0, len(organizers))
	for _, organizer := range organizers {
		result = append(result, Organizer(organizer))
	}
	return result
}

func Event(event *model.Event) EventResponse {
	return EventResponse{
		ID: event.ID, OrganizerID: event.OrganizerID, OrganizerName: event.OrganizerName,
		Title: event.Title, Description: event.Description, EventTime: event.EventTime,
		Location: event.Location, Capacity: event.Capacity, Price: event.Price, Status: event.Status,
		CreatedAt: event.CreatedAt, UpdatedAt: event.UpdatedAt,
	}
}

func Events(events []*model.Event) []EventResponse {
	result := make([]EventResponse, 0, len(events))
	for _, event := range events {
		result = append(result, Event(event))
	}
	return result
}

func Ticket(ticket *model.Ticket) TicketResponse {
	return TicketResponse{
		ID: ticket.ID, EventID: ticket.EventID, Name: ticket.Name, Price: ticket.Price,
		Stock: ticket.Stock, CreatedAt: ticket.CreatedAt, UpdatedAt: ticket.UpdatedAt,
	}
}

func Tickets(tickets []*model.Ticket) []TicketResponse {
	result := make([]TicketResponse, 0, len(tickets))
	for _, ticket := range tickets {
		result = append(result, Ticket(ticket))
	}
	return result
}

func Registration(registration *model.Registration) RegistrationResponse {
	return RegistrationResponse{
		ID: registration.ID, EventID: registration.EventID, Name: registration.Name,
		Contact: registration.Contact, TicketID: registration.TicketID, TicketName: registration.TicketName,
		IdentityStatus: registration.IdentityStatus, CreatedAt: registration.CreatedAt,
	}
}

func Registrations(registrations []*model.Registration) []RegistrationResponse {
	result := make([]RegistrationResponse, 0, len(registrations))
	for _, registration := range registrations {
		result = append(result, Registration(registration))
	}
	return result
}

func MyRegistration(registration *model.MyRegistration) MyRegistrationResponse {
	return MyRegistrationResponse{
		ID: registration.ID, EventID: registration.EventID, EventTitle: registration.EventTitle,
		EventTime: registration.EventTime, Location: registration.Location, EventStatus: registration.EventStatus,
		TicketID: registration.TicketID, TicketName: registration.TicketName, CreatedAt: registration.CreatedAt,
	}
}

func MyRegistrations(registrations []*model.MyRegistration) []MyRegistrationResponse {
	result := make([]MyRegistrationResponse, 0, len(registrations))
	for _, registration := range registrations {
		result = append(result, MyRegistration(registration))
	}
	return result
}

func Post(post *model.Post) PostResponse {
	return PostResponse{
		ID: post.ID, EventID: post.EventID, AuthorName: post.AuthorName, Title: post.Title,
		Content: post.Content, ReplyCount: post.ReplyCount, CreatedAt: post.CreatedAt,
	}
}

func Posts(posts []*model.Post) []PostResponse {
	result := make([]PostResponse, 0, len(posts))
	for _, post := range posts {
		result = append(result, Post(post))
	}
	return result
}

func Reply(reply *model.Reply) ReplyResponse {
	return ReplyResponse{
		ID: reply.ID, PostID: reply.PostID, AuthorName: reply.AuthorName,
		Content: reply.Content, CreatedAt: reply.CreatedAt,
	}
}

func Replies(replies []*model.Reply) []ReplyResponse {
	result := make([]ReplyResponse, 0, len(replies))
	for _, reply := range replies {
		result = append(result, Reply(reply))
	}
	return result
}

func PostDetail(post *model.Post, replies []*model.Reply) PostDetailResponse {
	return PostDetailResponse{Post: Post(post), Replies: Replies(replies)}
}

func IdentityMigrationReport(report *model.IdentityMigrationReport) IdentityMigrationReportResponse {
	stats := func(value model.IdentityEntityStats) IdentityEntityStatsResponse {
		return IdentityEntityStatsResponse(value)
	}
	records := make([]LegacyIdentityRecordResponse, 0, len(report.LegacyRecords))
	for _, record := range report.LegacyRecords {
		records = append(records, LegacyIdentityRecordResponse(record))
	}
	return IdentityMigrationReportResponse{
		Registrations: stats(report.Registrations), Posts: stats(report.Posts), Replies: stats(report.Replies),
		LegacyRecords: records,
	}
}
