import { get, post, put, del } from './client'
import type {
  Event,
  CreateEventReq,
  UpdateEventReq,
  Registration,
  RegisterReq,
  CancelRegistrationReq,
} from './types'

export interface ListEventsParams {
  status?: string
  price_type?: string
  q?: string
  page?: number
  page_size?: number
}

export function listEvents(params?: ListEventsParams) {
  return get<Event[]>('/api/events', params)
}

export function getEvent(id: number) {
  return get<Event>(`/api/events/${id}`)
}

export function createEvent(data: CreateEventReq) {
  return post<Event>('/api/events', data)
}

export function updateEvent(id: number, data: UpdateEventReq) {
  return put<Event>(`/api/events/${id}`, data)
}

export function deleteEvent(id: number) {
  return del(`/api/events/${id}`)
}

export function registerEvent(id: number, data: RegisterReq) {
  return post<Registration>(`/api/events/${id}/register`, data)
}

export function cancelRegistration(id: number, data: CancelRegistrationReq) {
  return del(`/api/events/${id}/register`, data)
}

export function listRegistrations(
  id: number,
  params?: { page?: number; page_size?: number }
) {
  return get<Registration[]>(`/api/events/${id}/registrations`, params)
}
