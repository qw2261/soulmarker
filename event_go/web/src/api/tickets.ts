import { get, post, put, del } from './client'
import type { Ticket, CreateTicketReq, UpdateTicketReq } from './types'

export function listTickets(
  eventId: number,
  params?: { page?: number; page_size?: number }
) {
  return get<Ticket[]>(`/api/events/${eventId}/tickets`, params)
}

export function getTicket(eventId: number, ticketId: number) {
  return get<Ticket>(`/api/events/${eventId}/tickets/${ticketId}`)
}

export function createTicket(eventId: number, data: CreateTicketReq) {
  return post<Ticket>(`/api/events/${eventId}/tickets`, data)
}

export function updateTicket(
  eventId: number,
  ticketId: number,
  data: UpdateTicketReq
) {
  return put<Ticket>(`/api/events/${eventId}/tickets/${ticketId}`, data)
}

export function deleteTicket(eventId: number, ticketId: number) {
  return del(`/api/events/${eventId}/tickets/${ticketId}`)
}
