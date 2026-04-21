import { defineStore } from 'pinia'
import { ref } from 'vue'
import { clearSessionId } from '../utils/request'

export const useUserStore = defineStore('user', () => {
  // VUE-1 修复：Token 仅存 Pinia 内存，不写入 localStorage，防止 XSS 攻击
  const token = ref('')
  const uid = ref('')

  function setUser(data) {
    token.value = data.token
    uid.value = data.uid
  }

  function logout() {
    token.value = ''
    uid.value = ''
    clearSessionId()
  }

  return { token, uid, setUser, logout }
})
