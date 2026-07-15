import { get, post, put, del } from './client'
import type { Organizer } from './types'

export interface CreateOrganizerReq {
  name: string
  description?: string
  contact?: string
  logo_url?: string
  address?: string
  website?: string
  tags?: string
}

export interface UpdateOrganizerReq {
  name?: string
  description?: string
  contact?: string
  logo_url?: string
  address?: string
  website?: string
  tags?: string
}

export function listOrganizers(params?: { page?: number; page_size?: number }) {
  return get<Organizer[]>('/organizers', params)
}

export function getOrganizer(id: number) {
  return get<Organizer>(`/organizers/${id}`)
}

export function createOrganizer(data: CreateOrganizerReq) {
  return post<Organizer>('/organizers', data)
}

export function updateOrganizer(id: number, data: UpdateOrganizerReq) {
  return put<Organizer>(`/organizers/${id}`, data)
}

export function deleteOrganizer(id: number) {
  return del(`/organizers/${id}`)
}
