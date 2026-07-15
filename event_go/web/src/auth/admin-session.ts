export const ADMIN_SESSION_EXPIRED_EVENT = 'soulmark:admin-session-expired'

export function isCurrentAdminSessionToken(token: unknown) {
  const currentToken = localStorage.getItem('admin_token')
  return Boolean(currentToken) && token === currentToken
}

export function expireAdminSession(redirect: string) {
  localStorage.removeItem('admin_token')
  window.dispatchEvent(new CustomEvent(ADMIN_SESSION_EXPIRED_EVENT, { detail: { redirect } }))
}
