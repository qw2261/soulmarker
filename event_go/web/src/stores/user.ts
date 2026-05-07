import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User } from '@/api/types'

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('user_token') || '')
  const user = ref<User | null>(null)
  const savedUser = localStorage.getItem('user_info')
  if (savedUser) {
    try { user.value = JSON.parse(savedUser) } catch { /* ignore */ }
  }

  const isLoggedIn = computed(() => !!token.value)

  function setAuth(tokenStr: string, u: User) {
    token.value = tokenStr
    user.value = u
    localStorage.setItem('user_token', tokenStr)
    localStorage.setItem('user_info', JSON.stringify(u))
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('user_token')
    localStorage.removeItem('user_info')
  }

  return { token, user, isLoggedIn, setAuth, logout }
})
