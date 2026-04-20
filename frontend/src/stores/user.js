import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const uid = ref(localStorage.getItem('uid') || '')

  function setUser(data) {
    token.value = data.token
    uid.value = data.uid
    localStorage.setItem('token', data.token)
    localStorage.setItem('uid', data.uid)
  }

  function logout() {
    token.value = ''
    uid.value = ''
    localStorage.removeItem('token')
    localStorage.removeItem('uid')
  }

  return { token, uid, setUser, logout }
})
