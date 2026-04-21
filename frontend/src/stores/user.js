import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useUserStore = defineStore('user', () => {
  // VUE-1 修复：Token 仅存 Pinia 内存，不写入 localStorage，防止 XSS 攻击
  const token = ref('')
  const uid = ref('')

  function setUser(data) {
    token.value = data.token
    uid.value = data.uid
  }

  async function logout() {
    token.value = ''
    uid.value = ''
    // NB4 修复：request.sessionId 也应置空
    const request = await import('../utils/request').then(m => m.default)
    request.sessionId = ''
  }

  return { token, uid, setUser, logout }
})
