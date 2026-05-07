import { get, post } from './client'
import type { Post, Reply, CreatePostReq, CreateReplyReq } from './types'

export function listPosts(
  eventId: number,
  params?: { page?: number; page_size?: number }
) {
  return get<Post[]>(`/api/events/${eventId}/posts`, params)
}

export function getPost(eventId: number, postId: number) {
  return get<Post>(`/api/events/${eventId}/posts/${postId}`)
}

export function createPost(eventId: number, data: CreatePostReq) {
  return post<Post>(`/api/events/${eventId}/posts`, data)
}

export function createReply(
  eventId: number,
  postId: number,
  data: CreateReplyReq
) {
  return post<Reply>(`/api/events/${eventId}/posts/${postId}/replies`, data)
}
