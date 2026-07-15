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

export type OrganizationRole = 'owner' | 'admin' | 'editor' | 'checker' | 'finance'

export interface OrganizationContext {
  organization_id: number
  organization_name: string
  organization_slug: string
  organization_status: string
  membership_status: 'active' | 'revoked'
  role: OrganizationRole
  principal_type: 'organization_member'
  capabilities: string[]
}

export interface OrganizationWorkspace {
  id: number
  name: string
  slug: string
  status: string
  profile: Organizer
  created_at: string
  updated_at: string
}

export interface OrganizationMember {
  id: number
  user_id: number
  name: string
  contact: string
  role: OrganizationRole
  status: 'active' | 'revoked'
  created_at: string
  updated_at: string
}

export interface OrganizationInvitation {
  id: number
  email: string
  role: Exclude<OrganizationRole, 'owner'>
  status: 'pending' | 'accepted' | 'revoked' | 'expired'
  expires_at: string
  accepted_at?: string
  revoked_at?: string
  invited_by_user_id: number
  created_at: string
  updated_at: string
}

export interface CreateOrganizationReq {
  name: string
  slug: string
  profile_name: string
  profile_description: string
  profile_contact: string
  profile_logo_url: string
  profile_address: string
  profile_website: string
  profile_tags: string
}

export interface Event {
  id: number
  organizer_id: number
  organizer_name?: string
  title: string
  description: string
  cover_url: string
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
  identity_status?: 'verified' | 'backfilled' | 'legacy'
  created_at: string
  admission?: Admission
}

export interface Admission {
  id: number
  event_id: number
  ticket_name?: string
  credential_code: string
  credential: string
  status: 'active' | 'revoked'
  issued_at: string
  revoked_at?: string
  checked_in_at?: string
}

export interface MyAdmission extends Admission {
  event_title: string
  event_time: string
  location: string
  event_status: string
}

export interface MyActivity {
  id: number
  kind: 'admission' | 'registration'
  registration_id?: number
  event_id: number
  event_title: string
  event_time: string
  location: string
  event_status: string
  ticket_id?: number
  ticket_name?: string
  joined_at: string
  admission?: Admission
}

export interface Checkin {
  id: number
  admission_id: number
  event_id: number
  credential_code?: string
  user_name?: string
  user_contact?: string
  checked_in_at: string
  checked_in_by: string
}

export interface CheckinResult {
  checkin: Checkin
  already_checked_in: boolean
}

export interface MyRegistration {
  id: number
  event_id: number
  event_title: string
  event_time: string
  location: string
  event_status: string
  ticket_id?: number
  ticket_name?: string
  created_at: string
}

export type NotificationType =
  | 'registration_confirmed'
  | 'registration_cancelled'
  | 'event_updated'
  | 'event_reminder_24h'

export interface Notification {
  id: number
  event_id?: number
  type: NotificationType
  title: string
  body: string
  action_url: string
  read_at?: string
  created_at: string
}

export interface NotificationUnreadCount {
  unread: number
}

export interface NotificationsMarkedRead {
  updated: number
}

export interface RegistrationStatus {
  registered: boolean
  admission?: Admission
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

export interface PostDetail {
  post: Post
  replies: Reply[]
}

export interface APIResp<T = any> {
  code: number
  error_code?: string
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
  cover_url: string
  event_time: string
  location: string
  capacity: number
  price: number
}

export interface UpdateEventReq {
  organizer_id?: number
  title?: string
  description?: string
  cover_url?: string
  event_time?: string
  location?: string
  capacity?: number
  price?: number
  status?: string
}

export interface RegisterReq {
  ticket_id?: number
}

export interface CreatePostReq {
  title: string
  content: string
}

export interface CreateReplyReq {
  content: string
}

export type ContentTargetType = 'post' | 'reply'
export type ContentReportStatus = 'open' | 'resolved' | 'dismissed'
export type ContentReportCategory = 'spam' | 'abuse' | 'illegal' | 'privacy' | 'other'

export interface CreateContentReportReq {
  category: ContentReportCategory
  detail: string
}

export interface ContentReportReceipt {
  id: number
  target_type: ContentTargetType
  target_id: number
  category: ContentReportCategory
  status: ContentReportStatus
  created_at: string
}

export interface ContentReport extends ContentReportReceipt {
  event_id: number
  post_id: number
  reporter_user_id: number
  reporter_name: string
  detail: string
  resolved_at?: string
  resolved_by: string
  resolution_note: string
  target_author_name: string
  target_title: string
  target_content: string
  target_moderation_status: 'visible' | 'removed'
}

export interface ContentModerationAction {
  id: number
  report_id?: number
  event_id: number
  post_id: number
  target_type: ContentTargetType
  target_id: number
  action: 'remove' | 'restore' | 'dismiss'
  actor: string
  reason: string
  created_at: string
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
  recovery_email?: string
  recovery_email_verified_at?: string
  created_at: string
}

export interface LoginResp {
  token: string
  user: User
}
