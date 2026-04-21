import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'
import { useUserStore } from '../stores/user'

const request = axios.create({
  baseURL: '/api',
  timeout: 20000,
  withCredentials: true
})

// Token 刷新状态管理
let isRefreshing = false
let refreshQueue = []

const processRefreshQueue = (token) => {
  refreshQueue.forEach(({ resolve, reject }) => {
    if (token) {
      resolve(token)
    } else {
      reject(new Error('Refresh failed'))
    }
  })
  refreshQueue = []
}

// VUE-1 修复：页面初始化时，如果有 RefreshToken Cookie 则静默刷新
const attemptSilentRefresh = async () => {
  if (isRefreshing) return false
  isRefreshing = true
  try {
    const { data } = await axios.post('/api/auth/refresh', {}, {
      withCredentials: true,
      baseURL: '/api'
    })
    if (data.status === 1) {
      const userStore = useUserStore()
      userStore.setUser({ token: data.token, uid: userStore.uid })
      processRefreshQueue(data.token)
      isRefreshing = false
      return true
    }
  } catch {
    processRefreshQueue(null)
  }
  isRefreshing = false
  return false
}

// 导出静默刷新方法，供 main.js 在初始化时调用
export { attemptSilentRefresh }

// 请求拦截器
request.interceptors.request.use(
  (config) => {
    // VUE-1 修复：从 Pinia 内存读取 token，不从 localStorage
    const userStore = useUserStore()
    if (userStore.token) {
      config.headers.Authorization = `Bearer ${userStore.token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器
request.interceptors.response.use(
  (response) => {
    const res = response.data

    // 验证码接口
    if (response.config.url === '/captcha' && res.sessionId) {
      localStorage.setItem('sessionId', res.sessionId)
    }

    // 业务逻辑错误
    if (res.status === 0) {
      // VUE-8 修复：TOKEN_EXPIRED 业务错误码不弹 toast，静默处理
      if (res.code !== 'TOKEN_EXPIRED') {
        ElMessage.error(res.info || res.message || '请求失败')
      }
      return Promise.reject(new Error(res.info || res.message))
    }

    return res
  },
  async (error) => {
    const originalConfig = error.config

    // 401 处理：尝试刷新 Token
    if (error.response?.status === 401 && !originalConfig._retry) {
      originalConfig._retry = true

      if (isRefreshing) {
        // 已在刷新中，将请求加入队列
        return new Promise((resolve, reject) => {
          refreshQueue.push({
            resolve: (token) => {
              originalConfig.headers.Authorization = `Bearer ${token}`
              resolve(request(originalConfig))
            },
            reject: (err) => {
              reject(err)
            }
          })
        })
      }

      isRefreshing = true

      try {
        const { data } = await axios.post('/api/auth/refresh', {}, {
          withCredentials: true,
          baseURL: '/api'
        })

        if (data.status === 1) {
          // VUE-1 修复：刷新成功后更新 Pinia 内存，不写 localStorage
          const userStore = useUserStore()
          userStore.setUser({ token: data.token, uid: userStore.uid })
          processRefreshQueue(data.token)
          originalConfig.headers.Authorization = `Bearer ${data.token}`
          return request(originalConfig)
        }
      } catch (refreshError) {
        processRefreshQueue(null)
        // 刷新失败时清空 Pinia token
        const userStore = useUserStore()
        userStore.logout()
        ElMessage.error('登录已过期，请重新登录')
        router.push('/login')
        return Promise.reject(refreshError)
      } finally {
        isRefreshing = false
      }
    }

    // VUE-8 修复：从响应 body 读取业务错误码，区分处理
    const code = error.response?.data?.code
    const serverMessage = error.response?.data?.info || error.response?.data?.message
    if (error.response?.status === 400) {
      // 业务错误，不一定是 TOKEN_EXPIRED
      if (code !== 'TOKEN_EXPIRED') {
        ElMessage.error(serverMessage || '请求失败')
      }
    } else if (error.response?.status !== 401) {
      ElMessage.error(error.message || '网络错误')
    }

    return Promise.reject(error)
  }
)

export default request
