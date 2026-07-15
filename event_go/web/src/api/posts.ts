import { del, get, post, put } from './client'
import type {
  Post,
  Reply,
  PostDetail,
  CreatePostReq,
  CreateReplyReq,
  CreateContentReportReq,
  ContentReportReceipt,
} from './types'

export function listPosts(
  eventId: number,
  params?: { page?: number; page_size?: number }
) {
  return get<Post[]>(`/events/${eventId}/posts`, params)
}

export function getPost(eventId: number, postId: number) {
  return get<PostDetail>(`/events/${eventId}/posts/${postId}`)
}

export function createPost(eventId: number, data: CreatePostReq) {
  return post<Post>(`/events/${eventId}/posts`, data)
}

export function createReply(
  eventId: number,
  postId: number,
  data: CreateReplyReq
) {
  return post<Reply>(`/events/${eventId}/posts/${postId}/replies`, data)
}

export function reportPost(eventId: number, postId: number, data: CreateContentReportReq) {
  return post<ContentReportReceipt>(`/events/${eventId}/posts/${postId}/reports`, data)
}

export function reportReply(eventId: number, postId: number, replyId: number, data: CreateContentReportReq) {
  return post<ContentReportReceipt>(`/events/${eventId}/posts/${postId}/replies/${replyId}/reports`, data)
}

export function removePost(eventId: number, postId: number, reason: string) {
  return del(`/events/${eventId}/posts/${postId}`, { reason })
}

export function restorePost(eventId: number, postId: number, reason: string) {
  return put(`/events/${eventId}/posts/${postId}/restore`, { reason })
}

export function removeReply(eventId: number, postId: number, replyId: number, reason: string) {
  return del(`/events/${eventId}/posts/${postId}/replies/${replyId}`, { reason })
}

export function restoreReply(eventId: number, postId: number, replyId: number, reason: string) {
  return put(`/events/${eventId}/posts/${postId}/replies/${replyId}/restore`, { reason })
}
