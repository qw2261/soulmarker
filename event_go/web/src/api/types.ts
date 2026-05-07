export interface Organizer {
  id: number
  name: string
  description: string
  contact: string
  logo_url: string
  address: string
  website: string
  tags: string
  event_count?: number
  created_at: string
  updated_at: string
}

export interface Event {
  id: number
  organizer_id: number
  organizer_name?: string
  title: string
  description: string
  event_time: string
  location: string
  capacity: number
  price: number
  status: string
  created_at: string
  updated_at: string
}

export interface Ticket {
  id: number
  event_id: number
  name: string
  price: number
  stock: number
  created_at: string
  updated_at: string
}

export interface Registration {
  id: number
  event_id: number
  name: string
  contact: string
  ticket_id?: number
  ticket_name?: string
  created_at: string
}

export interface Post {
  id: number
  event_id: number
  author_name: string
  title: string
  content: string
  reply_count: number
  replies?: Reply[]
  created_at: string
}

export interface Reply {
  id: number
  post_id: number
  author_name: string
  content: string
  created_at: string
}

export interface APIResp<T = any> {
  code: number
  message: string
  data?: T
  total?: number
  page?: number
  page_size?: number
}

export interface CreateEventReq {
  organizer_id: number
  title: string
  description: string
  event_time: string
  location: string
  capacity: number
  price: number
}

export interface UpdateEventReq {
  title?: string
  description?: string
  event_time?: string
  location?: string
  capacity?: number
  price?: number
  status?: string
}

export interface RegisterReq {
  name: string
  contact: string
  ticket_id?: number
}

export interface CancelRegistrationReq {
  contact: string
}

export interface CreatePostReq {
  author_name: string
  author_contact: string
  title: string
  content: string
}

export interface CreateReplyReq {
  author_name: string
  author_contact: string
  content: string
}

export interface CreateTicketReq {
  name: string
  price: number
  stock: number
}

export interface UpdateTicketReq {
  name?: string
  price?: number
  stock?: number
}

export const EventStatusMap: Record<string, string> = {
  draft: '草稿',
  published: '已发布',
  cancelled: '已取消',
  ended: '已结束',
}

export const EventStatusColors: Record<string, string> = {
  draft: 'info',
  published: 'success',
  cancelled: 'danger',
  ended: 'warning',
}

export interface User {
  id: number
  name: string
  contact: string
  created_at: string
}

export interface LoginResp {
  token: string
  user: User
}
