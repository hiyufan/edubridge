import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'

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

// 请求拦截器
request.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
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
      ElMessage.error(res.info || '请求失败')
      return Promise.reject(new Error(res.info))
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
          localStorage.setItem('token', data.token)
          processRefreshQueue(data.token)
          originalConfig.headers.Authorization = `Bearer ${data.token}`
          return request(originalConfig)
        }
      } catch (refreshError) {
        processRefreshQueue(null)
        localStorage.removeItem('token')
        ElMessage.error('登录已过期，请重新登录')
        router.push('/login')
        return Promise.reject(refreshError)
      } finally {
        isRefreshing = false
      }
    }

    if (error.response?.status === 400) {
      ElMessage.error(error.response?.data?.info || '请求失败')
    } else if (error.response?.status !== 401) {
      ElMessage.error(error.message || '网络错误')
    }

    return Promise.reject(error)
  }
)

export default request
