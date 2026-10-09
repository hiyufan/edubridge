import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'
import { useUserStore } from '../stores/user'

const request = axios.create({
  baseURL: '/api',
  timeout: 20000,
  withCredentials: true
})

request.sessionId = ''

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

const doRefresh = async () => {
  try {
    const { data } = await axios.post('/api/auth/refresh', {}, {
      withCredentials: true
    })
    if (data.status === 1) {
      const userStore = useUserStore()
      userStore.setUser({ token: data.token, uid: userStore.uid })
      processRefreshQueue(data.token)
      return data.token
    }
  } catch {
    processRefreshQueue(null)
  }
  return null
}

const attemptSilentRefresh = async () => {
  if (isRefreshing) {
    return new Promise((resolve, reject) => {
      refreshQueue.push({
        resolve: (token) => resolve(token),
        reject: (err) => reject(err)
      })
    })
  }

  isRefreshing = true
  const token = await doRefresh()
  isRefreshing = false

  if (!token) {
    const userStore = useUserStore()
    userStore.logout()
    router.push('/login')
  }

  return token
}

export { attemptSilentRefresh }

export const clearSessionId = () => { request.sessionId = '' }

request.interceptors.request.use(
  (config) => {
    const userStore = useUserStore()
    if (userStore.token) {
      config.headers.Authorization = `Bearer ${userStore.token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

request.interceptors.response.use(
  (response) => {
    const res = response.data

    if (response.config.url === '/captcha' && res.sessionId) {
      request.sessionId = res.sessionId
    }

    if (res.status === 0) {
      ElMessage.error(res.info || '请求失败')
      return Promise.reject(new Error(res.info))
    }

    return res
  },
  async (error) => {
    const originalConfig = error.config

    // 教务系统登录已失效（JWT 仍有效，刷新也没用）：回到登录页重新输入验证码
    if (error.response?.data?.code === 'JW_SESSION_EXPIRED') {
      const userStore = useUserStore()
      userStore.logout()
      ElMessage.warning(error.response.data.info || '教务系统登录已失效，请重新登录')
      router.push('/login')
      return Promise.reject(error)
    }

    if (error.response?.status === 401 && !originalConfig._retry) {
      originalConfig._retry = true

      if (isRefreshing) {
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

      const token = await attemptSilentRefresh()
      if (token) {
        originalConfig.headers.Authorization = `Bearer ${token}`
        return request(originalConfig)
      }
    }

    // 后端统一返回 {status: 0, info}，info 可直接展示
    const serverMessage = error.response?.data?.info
    if (error.response?.status !== 401 && !axios.isCancel(error)) {
      ElMessage.error(serverMessage || (error.response ? '请求失败' : '网络错误，请检查网络连接'))
    }

    return Promise.reject(error)
  }
)

export default request
