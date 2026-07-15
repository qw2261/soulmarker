import { get, post, put, del } from './client'
import type {
  Event,
  CreateEventReq,
  UpdateEventReq,
  Registration,
  RegistrationStatus,
  MyRegistration,
  RegisterReq,
  MyAdmission,
  Checkin,
  CheckinResult,
} from './types'

export interface ListEventsParams {
  status?: string
  price_type?: string
  q?: string
  organizer_id?: number
  page?: number
  page_size?: number
}

export function listEvents(params?: ListEventsParams) {
  return get<Event[]>('/events', params)
}

export function getEvent(id: number) {
  return get<Event>(`/events/${id}`)
}

export function createEvent(data: CreateEventReq) {
  return post<Event>('/events', data)
}

export function updateEvent(id: number, data: UpdateEventReq) {
  return put<Event>(`/events/${id}`, data)
}

export function deleteEvent(id: number) {
  return del(`/events/${id}`)
}

export function registerEvent(id: number, data: RegisterReq) {
  return post<Registration>(`/events/${id}/register`, data)
}

export function cancelRegistration(id: number) {
  return del(`/events/${id}/register`)
}

export function listRegistrations(
  id: number,
  params?: { page?: number; page_size?: number }
) {
  return get<Registration[]>(`/events/${id}/registrations`, params)
}

export function getRegistrationStatus(id: number) {
  return get<RegistrationStatus>('/events/' + id + '/registration')
}

export function listMyRegistrations(params?: { page?: number; page_size?: number }) {
  return get<MyRegistration[]>('/me/registrations', params)
}

export function listMyAdmissions(params?: { page?: number; page_size?: number }) {
  return get<MyAdmission[]>('/me/admissions', params)
}

export function getMyAdmission(eventId: number) {
  return get<MyAdmission>(`/events/${eventId}/admission`)
}

export function checkInAdmission(eventId: number, credential: string) {
  return post<CheckinResult>(`/events/${eventId}/checkins`, { credential })
}

export function listCheckins(
  eventId: number,
  params?: { page?: number; page_size?: number }
) {
  return get<Checkin[]>(`/events/${eventId}/checkins`, params)
}
