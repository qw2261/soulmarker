import axios from 'axios'

export function requestErrorMessage(error: unknown, fallback = '加载失败，请稍后重试') {
  if (typeof navigator !== 'undefined' && !navigator.onLine) {
    return '当前网络不可用，请检查连接后重试'
  }
  if (axios.isAxiosError(error)) {
    if (error.code === 'ECONNABORTED') {
      return '请求超时，请检查网络后重试'
    }
    return error.response?.data?.message || error.message || fallback
  }
  return error instanceof Error && error.message ? error.message : fallback
}
