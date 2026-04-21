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

  function logout() {
    token.value = ''
    uid.value = ''
    // B12 修复：不再使用 localStorage 存储 sessionId，无需清理
  }

  return { token, uid, setUser, logout }
})
