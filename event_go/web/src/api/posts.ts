import { get, post } from './client'
import type { Post, Reply, PostDetail, CreatePostReq, CreateReplyReq } from './types'

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
