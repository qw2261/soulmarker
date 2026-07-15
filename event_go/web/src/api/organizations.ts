import { del, get, post, put } from './client'
import type {
  Checkin,
  CheckinResult,
  CreateEventReq,
  CreateOrganizationReq,
  CreateTicketReq,
  Event,
  OrganizationContext,
  OrganizationInvitation,
  OrganizationMember,
  OrganizationRole,
  OrganizationWorkspace,
  Registration,
  Ticket,
  UpdateEventReq,
  UpdateTicketReq,
} from './types'

export function listMyOrganizations() {
  return get<OrganizationContext[]>('/me/organizations')
}

export function createOrganization(data: CreateOrganizationReq) {
  return post<OrganizationWorkspace>('/organizations', data)
}

export function acceptOrganizationInvitation(token: string) {
  return post('/organization-invitations/accept', { token })
}

export function getOrganizationSession(organizationId: number) {
  return get<OrganizationContext>(`/organizations/${organizationId}/session`)
}

export function getOrganizationWorkspace(organizationId: number) {
  return get<OrganizationWorkspace>(`/organizations/${organizationId}`)
}

export function listOrganizationMembers(organizationId: number) {
  return get<OrganizationMember[]>(`/organizations/${organizationId}/members`)
}

export function updateOrganizationMember(
  organizationId: number,
  memberId: number,
  role: Exclude<OrganizationRole, 'owner'>,
) {
  return put<OrganizationMember>(`/organizations/${organizationId}/members/${memberId}`, { role })
}

export function revokeOrganizationMember(organizationId: number, memberId: number) {
  return del(`/organizations/${organizationId}/members/${memberId}`)
}

export function listOrganizationInvitations(organizationId: number) {
  return get<OrganizationInvitation[]>(`/organizations/${organizationId}/invitations`)
}

export function createOrganizationInvitation(
  organizationId: number,
  email: string,
  role: Exclude<OrganizationRole, 'owner'>,
) {
  return post<OrganizationInvitation>(`/organizations/${organizationId}/invitations`, { email, role })
}

export function revokeOrganizationInvitation(organizationId: number, invitationId: number) {
  return del(`/organizations/${organizationId}/invitations/${invitationId}`)
}

function eventPath(organizationId: number, eventId?: number) {
  const base = `/organizations/${organizationId}/events`
  return eventId ? `${base}/${eventId}` : base
}

export function listOrganizationEvents(organizationId: number, params?: { page?: number; page_size?: number }) {
  return get<Event[]>(eventPath(organizationId), params)
}

export function getOrganizationEvent(organizationId: number, eventId: number) {
  return get<Event>(eventPath(organizationId, eventId))
}

export function createOrganizationEvent(organizationId: number, data: CreateEventReq) {
  return post<Event>(eventPath(organizationId), data)
}

export function updateOrganizationEvent(organizationId: number, eventId: number, data: UpdateEventReq) {
  return put<Event>(eventPath(organizationId, eventId), data)
}

export function deleteOrganizationEvent(organizationId: number, eventId: number) {
  return del(eventPath(organizationId, eventId))
}

function ticketPath(organizationId: number, eventId: number, ticketId?: number) {
  const base = `${eventPath(organizationId, eventId)}/tickets`
  return ticketId ? `${base}/${ticketId}` : base
}

export function listOrganizationTickets(organizationId: number, eventId: number, params?: { page?: number; page_size?: number }) {
  return get<Ticket[]>(ticketPath(organizationId, eventId), params)
}

export function createOrganizationTicket(organizationId: number, eventId: number, data: CreateTicketReq) {
  return post<Ticket>(ticketPath(organizationId, eventId), data)
}

export function updateOrganizationTicket(organizationId: number, eventId: number, ticketId: number, data: UpdateTicketReq) {
  return put<Ticket>(ticketPath(organizationId, eventId, ticketId), data)
}

export function deleteOrganizationTicket(organizationId: number, eventId: number, ticketId: number) {
  return del(ticketPath(organizationId, eventId, ticketId))
}

export function listOrganizationRegistrations(
  organizationId: number,
  eventId: number,
  params?: { page?: number; page_size?: number },
) {
  return get<Registration[]>(`${eventPath(organizationId, eventId)}/registrations`, params)
}

export function checkInOrganizationAdmission(organizationId: number, eventId: number, credential: string) {
  return post<CheckinResult>(`${eventPath(organizationId, eventId)}/checkins`, { credential })
}

export function listOrganizationCheckins(
  organizationId: number,
  eventId: number,
  params?: { page?: number; page_size?: number },
) {
  return get<Checkin[]>(`${eventPath(organizationId, eventId)}/checkins`, params)
}
