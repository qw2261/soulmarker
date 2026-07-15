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
		CoverURL: event.CoverURL, Location: event.Location, Capacity: event.Capacity, Price: event.Price, Status: event.Status,
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
	response := RegistrationResponse{
		ID: registration.ID, EventID: registration.EventID, Name: registration.Name,
		Contact: registration.Contact, TicketID: registration.TicketID, TicketName: registration.TicketName,
		IdentityStatus: registration.IdentityStatus, CreatedAt: registration.CreatedAt,
	}
	if registration.Admission != nil {
		admission := Admission(registration.Admission)
		response.Admission = &admission
	}
	return response
}

func Admission(admission *model.Admission) AdmissionResponse {
	return AdmissionResponse{
		ID: admission.ID, EventID: admission.EventID, TicketName: admission.TicketName,
		CredentialCode: admission.CredentialCode,
		Credential:     model.AdmissionCredentialPrefix + admission.CredentialCode,
		Status:         admission.Status, IssuedAt: admission.IssuedAt, RevokedAt: admission.RevokedAt,
		CheckedInAt: admission.CheckedInAt,
	}
}

func MyAdmission(admission *model.MyAdmission) MyAdmissionResponse {
	return MyAdmissionResponse{
		AdmissionResponse: Admission(&admission.Admission), EventTitle: admission.EventTitle,
		EventTime: admission.EventTime, Location: admission.Location, EventStatus: admission.EventStatus,
	}
}

func MyAdmissions(admissions []*model.MyAdmission) []MyAdmissionResponse {
	result := make([]MyAdmissionResponse, 0, len(admissions))
	for _, admission := range admissions {
		result = append(result, MyAdmission(admission))
	}
	return result
}

func MyActivity(activity *model.MyActivity) MyActivityResponse {
	response := MyActivityResponse{
		ID: activity.ID, Kind: activity.Kind, RegistrationID: activity.RegistrationID,
		EventID: activity.EventID, EventTitle: activity.EventTitle, EventTime: activity.EventTime,
		Location: activity.Location, EventStatus: activity.EventStatus, TicketID: activity.TicketID,
		TicketName: activity.TicketName, JoinedAt: activity.JoinedAt,
	}
	if activity.Admission != nil {
		admission := Admission(activity.Admission)
		response.Admission = &admission
	}
	return response
}

func MyActivities(activities []*model.MyActivity) []MyActivityResponse {
	result := make([]MyActivityResponse, 0, len(activities))
	for _, activity := range activities {
		result = append(result, MyActivity(activity))
	}
	return result
}

func Checkin(checkin *model.Checkin) CheckinResponse {
	return CheckinResponse{
		ID: checkin.ID, AdmissionID: checkin.AdmissionID, EventID: checkin.EventID,
		CredentialCode: checkin.CredentialCode, UserName: checkin.UserName,
		UserContact: checkin.UserContact, CheckedInAt: checkin.CheckedInAt, CheckedInBy: checkin.CheckedInBy,
	}
}

func Checkins(checkins []*model.Checkin) []CheckinResponse {
	result := make([]CheckinResponse, 0, len(checkins))
	for _, checkin := range checkins {
		result = append(result, Checkin(checkin))
	}
	return result
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
