package dto

import (
	"github.com/qw2261/soulmarker/event_go/internal/authorization"
	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/privacy"
)

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
	return UserResponse{
		ID: user.ID, Name: user.Name, Contact: user.Contact,
		RecoveryEmail: user.RecoveryEmail, RecoveryEmailVerifiedAt: user.RecoveryEmailVerifiedAt,
		CreatedAt: user.CreatedAt,
	}
}

func Notification(notification *model.Notification) NotificationResponse {
	return NotificationResponse{
		ID: notification.ID, EventID: notification.EventID, Type: notification.Type,
		Title: notification.Title, Body: notification.Body, ActionURL: notification.ActionURL,
		ReadAt: notification.ReadAt, CreatedAt: notification.CreatedAt,
	}
}

func Notifications(notifications []*model.Notification) []NotificationResponse {
	result := make([]NotificationResponse, 0, len(notifications))
	for _, notification := range notifications {
		result = append(result, Notification(notification))
	}
	return result
}

func OrganizationContext(value authorization.OrganizationContext) OrganizationContextResponse {
	capabilities := make([]string, 0, len(value.Capabilities))
	for _, capability := range value.Capabilities {
		capabilities = append(capabilities, string(capability))
	}
	return OrganizationContextResponse{
		OrganizationID: value.OrganizationID, OrganizationName: value.OrganizationName,
		OrganizationSlug: value.OrganizationSlug, OrganizationStatus: value.OrganizationStatus,
		MembershipStatus: value.MembershipStatus, Role: value.Role,
		PrincipalType: authorization.PrincipalTypeOrganizationMember, Capabilities: capabilities,
	}
}

func OrganizationContexts(values []authorization.OrganizationContext) []OrganizationContextResponse {
	result := make([]OrganizationContextResponse, 0, len(values))
	for _, value := range values {
		result = append(result, OrganizationContext(value))
	}
	return result
}

func OrganizationWorkspace(organization *model.Organization, profile *model.OrganizerProfile) OrganizationWorkspaceResponse {
	return OrganizationWorkspaceWithPII(organization, profile, true)
}

func OrganizationWorkspaceWithPII(organization *model.Organization, profile *model.OrganizerProfile, full bool) OrganizationWorkspaceResponse {
	return OrganizationWorkspaceResponse{
		ID: organization.ID, Name: organization.Name, Slug: organization.Slug, Status: organization.Status,
		Profile: OrganizerWithPII(profile, full), CreatedAt: organization.CreatedAt, UpdatedAt: organization.UpdatedAt,
	}
}

func OrganizationMember(member *model.OrganizationMember) OrganizationMemberResponse {
	return OrganizationMemberResponse{
		ID: member.ID, UserID: member.UserID, Name: member.UserName, Contact: member.UserContact,
		Role: member.Role, Status: member.Status, CreatedAt: member.CreatedAt, UpdatedAt: member.UpdatedAt,
	}
}

func OrganizationMembers(members []*model.OrganizationMember) []OrganizationMemberResponse {
	return OrganizationMembersWithPII(members, true)
}

func OrganizationMembersWithPII(members []*model.OrganizationMember, full bool) []OrganizationMemberResponse {
	result := make([]OrganizationMemberResponse, 0, len(members))
	for _, member := range members {
		mapped := OrganizationMember(member)
		if !full {
			mapped.Name = privacy.MaskName(mapped.Name)
			mapped.Contact = privacy.MaskContact(mapped.Contact)
		}
		result = append(result, mapped)
	}
	return result
}

func OrganizationInvitation(invitation *model.OrganizationInvitation) OrganizationInvitationResponse {
	return OrganizationInvitationWithPII(invitation, true)
}

func OrganizationInvitationWithPII(invitation *model.OrganizationInvitation, full bool) OrganizationInvitationResponse {
	email := invitation.Email
	if !full {
		email = privacy.MaskContact(email)
	}
	return OrganizationInvitationResponse{
		ID: invitation.ID, Email: email, Role: invitation.Role, Status: invitation.Status,
		ExpiresAt: invitation.ExpiresAt, AcceptedAt: invitation.AcceptedAt, RevokedAt: invitation.RevokedAt,
		InvitedByUserID: invitation.InvitedByUserID, CreatedAt: invitation.CreatedAt, UpdatedAt: invitation.UpdatedAt,
	}
}

func OrganizationInvitations(invitations []*model.OrganizationInvitation) []OrganizationInvitationResponse {
	return OrganizationInvitationsWithPII(invitations, true)
}

func OrganizationInvitationsWithPII(invitations []*model.OrganizationInvitation, full bool) []OrganizationInvitationResponse {
	result := make([]OrganizationInvitationResponse, 0, len(invitations))
	for _, invitation := range invitations {
		result = append(result, OrganizationInvitationWithPII(invitation, full))
	}
	return result
}

func OrganizationAudit(entry *model.OrganizationAuditLog) OrganizationAuditResponse {
	return OrganizationAuditResponse{
		ID: entry.ID, OrganizationID: entry.OrganizationID, ActorType: entry.ActorType,
		ActorID: entry.ActorID, Action: entry.Action, ResourceType: entry.ResourceType,
		ResourceID: entry.ResourceID, RequestID: entry.RequestID, Outcome: entry.Outcome,
		HTTPStatus: entry.HTTPStatus, CreatedAt: entry.CreatedAt,
	}
}

func OrganizationAudits(entries []*model.OrganizationAuditLog) []OrganizationAuditResponse {
	result := make([]OrganizationAuditResponse, 0, len(entries))
	for _, entry := range entries {
		result = append(result, OrganizationAudit(entry))
	}
	return result
}

func Organizer(organizer *model.Organizer) OrganizerResponse {
	return OrganizerWithPII(organizer, true)
}

func OrganizerWithPII(organizer *model.Organizer, full bool) OrganizerResponse {
	contact := organizer.Contact
	if !full {
		contact = privacy.MaskContact(contact)
	}
	return OrganizerResponse{
		ID: organizer.ID, Name: organizer.Name, Description: organizer.Description,
		Contact: contact, LogoURL: organizer.LogoURL, Address: organizer.Address,
		Website: organizer.Website, Tags: organizer.Tags, EventCount: organizer.EventCount,
		CreatedAt: organizer.CreatedAt, UpdatedAt: organizer.UpdatedAt,
	}
}

func Organizers(organizers []*model.Organizer) []OrganizerResponse {
	return OrganizersWithPII(organizers, true)
}

func OrganizersWithPII(organizers []*model.Organizer, full bool) []OrganizerResponse {
	result := make([]OrganizerResponse, 0, len(organizers))
	for _, organizer := range organizers {
		result = append(result, OrganizerWithPII(organizer, full))
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
	return CheckinWithPII(checkin, true)
}

func CheckinWithPII(checkin *model.Checkin, full bool) CheckinResponse {
	credentialCode, userName, userContact := checkin.CredentialCode, checkin.UserName, checkin.UserContact
	if !full {
		credentialCode = ""
		userName = privacy.MaskName(userName)
		userContact = privacy.MaskContact(userContact)
	}
	return CheckinResponse{
		ID: checkin.ID, AdmissionID: checkin.AdmissionID, EventID: checkin.EventID,
		CredentialCode: credentialCode, UserName: userName,
		UserContact: userContact, CheckedInAt: checkin.CheckedInAt, CheckedInBy: checkin.CheckedInBy,
	}
}

func Checkins(checkins []*model.Checkin) []CheckinResponse {
	return CheckinsWithPII(checkins, true)
}

func CheckinsWithPII(checkins []*model.Checkin, full bool) []CheckinResponse {
	result := make([]CheckinResponse, 0, len(checkins))
	for _, checkin := range checkins {
		result = append(result, CheckinWithPII(checkin, full))
	}
	return result
}

func Registrations(registrations []*model.Registration) []RegistrationResponse {
	return RegistrationsWithPII(registrations, true)
}

func RegistrationsWithPII(registrations []*model.Registration, full bool) []RegistrationResponse {
	result := make([]RegistrationResponse, 0, len(registrations))
	for _, registration := range registrations {
		mapped := Registration(registration)
		if !full {
			mapped.Name = privacy.MaskName(mapped.Name)
			mapped.Contact = privacy.MaskContact(mapped.Contact)
			mapped.Admission = nil
		}
		result = append(result, mapped)
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

func ContentReportReceipt(report *model.ContentReport) ContentReportReceiptResponse {
	return ContentReportReceiptResponse{
		ID: report.ID, TargetType: report.TargetType, TargetID: report.TargetID,
		Category: report.Category, Status: report.Status, CreatedAt: report.CreatedAt,
	}
}

func ContentReport(report *model.ContentReport) ContentReportResponse {
	return ContentReportResponse{
		ID: report.ID, EventID: report.EventID, PostID: report.PostID, TargetType: report.TargetType,
		TargetID: report.TargetID, ReporterUserID: report.ReporterUserID, ReporterName: report.ReporterName,
		Category: report.Category, Detail: report.Detail, Status: report.Status, CreatedAt: report.CreatedAt,
		ResolvedAt: report.ResolvedAt, ResolvedBy: report.ResolvedBy, ResolutionNote: report.ResolutionNote,
		TargetAuthorName: report.TargetAuthorName, TargetTitle: report.TargetTitle,
		TargetContent: report.TargetContent, TargetModerationStatus: report.TargetModerationStatus,
	}
}

func ContentReports(reports []*model.ContentReport) []ContentReportResponse {
	result := make([]ContentReportResponse, 0, len(reports))
	for _, report := range reports {
		result = append(result, ContentReport(report))
	}
	return result
}

func ContentModerationAction(action *model.ContentModerationAction) ContentModerationActionResponse {
	return ContentModerationActionResponse{
		ID: action.ID, ReportID: action.ReportID, EventID: action.EventID, PostID: action.PostID,
		TargetType: action.TargetType, TargetID: action.TargetID, Action: action.Action,
		Actor: action.Actor, Reason: action.Reason, CreatedAt: action.CreatedAt,
	}
}

func ContentModerationActions(actions []*model.ContentModerationAction) []ContentModerationActionResponse {
	result := make([]ContentModerationActionResponse, 0, len(actions))
	for _, action := range actions {
		result = append(result, ContentModerationAction(action))
	}
	return result
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
