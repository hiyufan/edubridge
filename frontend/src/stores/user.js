import { defineStore } from 'pinia'
import { ref } from 'vue'
import { clearSessionId } from '../utils/request'

const TOKEN_KEY = 'auth_token'
const UID_KEY = 'auth_uid'

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem(TOKEN_KEY) || '')
  const uid = ref(localStorage.getItem(UID_KEY) || '')

  function setUser(data) {
    token.value = data.token
    uid.value = data.uid
    if (data.token) {
      localStorage.setItem(TOKEN_KEY, data.token)
    } else {
      localStorage.removeItem(TOKEN_KEY)
    }
    if (data.uid) {
      localStorage.setItem(UID_KEY, data.uid)
    } else {
      localStorage.removeItem(UID_KEY)
    }
  }

  function logout() {
    token.value = ''
    uid.value = ''
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(UID_KEY)
    clearSessionId()
  }

  function hasValidToken() {
    return !!token.value
  }

  return { token, uid, setUser, logout, hasValidToken }
})
