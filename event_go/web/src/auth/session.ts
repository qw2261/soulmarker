export const USER_SESSION_EXPIRED_EVENT = 'soulmark:user-session-expired'

export function isCurrentUserSessionToken(token: unknown) {
  const currentToken = localStorage.getItem('user_token')
  return Boolean(currentToken) && token === currentToken
}

export function expireUserSession(redirect: string) {
  localStorage.removeItem('user_token')
  localStorage.removeItem('user_info')
  window.dispatchEvent(new CustomEvent(USER_SESSION_EXPIRED_EVENT, { detail: { redirect } }))
}
