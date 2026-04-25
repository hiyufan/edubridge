<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import request from '../utils/request'
import { useUserStore } from '../stores/user'

const router = useRouter()
const userStore = useUserStore()

const loginForm = ref({
  username: '',
  password: '',
  captcha: '',
  loginType: 'xsxh'
})

const captchaUrl = ref('')
const captchaKey = ref(0)
const loading = ref(false)

const loginTypes = [
  { label: '学生', value: 'xsxh' },
  { label: '教职工', value: 'zjh' },
  { label: '考生', value: 'gkksh' }
]

const activeLoginType = ref(0)

const fetchCaptcha = async () => {
  try {
    const res = await request.get('/captcha')
    captchaKey.value++
    captchaUrl.value = res.data
  } catch (error) {
    ElMessage.error('获取验证码失败')
  }
}

onMounted(() => {
  fetchCaptcha()
})

const handleLogin = async () => {
  if (!loginForm.value.username || !loginForm.value.password || !loginForm.value.captcha) {
    ElMessage.warning('请填写完整信息')
    return
  }
  if (!request.sessionId) {
    ElMessage.warning('验证码已过期，请刷新验证码')
    return
  }

  loading.value = true
  try {
    const payload = {
      username: loginForm.value.username,
      password: loginForm.value.password,
      captcha: loginForm.value.captcha,
      loginType: loginTypes[activeLoginType.value].value,
      sessionId: request.sessionId
    }
    console.log('[Login] Sending request...')
    const res = await request.post('/auth/login', payload)
    console.log('[Login] Response:', res)
    console.log('[Login] Token:', res.token)
    userStore.setUser({
      token: res.token,
      uid: res.uid
    })
    console.log('[Login] User set, navigating to /')
    ElMessage.success('登录成功')
    router.push('/').then(() => {
      console.log('[Login] Navigation complete')
    }).catch(err => {
      console.error('[Login] Navigation error:', err)
    })

    request.get('/schedule?week=1').catch(() => {})
    request.get('/score').catch(() => {})
  } catch (error) {
    console.error('[Login] Error:', error)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-container">
      <div class="login-left">
        <div class="brand">
          <div class="logo">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <rect x="2" y="3" width="20" height="18" rx="2"/>
              <path d="M8 10h8M8 14h5"/>
            </svg>
          </div>
          <div class="brand-text">
            <h1>教务系统</h1>
            <p>Academic Administration Portal</p>
          </div>
        </div>
        <div class="features">
          <div class="feature">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <rect x="3" y="4" width="18" height="18" rx="2"/>
              <line x1="16" y1="2" x2="16" y2="6"/>
              <line x1="8" y1="2" x2="8" y2="6"/>
              <line x1="3" y1="10" x2="21" y2="10"/>
            </svg>
            <span>课表查询</span>
          </div>
          <div class="feature">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
              <polyline points="14 2 14 8 20 8"/>
            </svg>
            <span>成绩查询</span>
          </div>
          <div class="feature">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
            </svg>
            <span>安全可靠</span>
          </div>
        </div>
      </div>

      <div class="login-right">
        <div class="login-box">
          <div class="login-header">
            <h2>用户登录</h2>
            <p>请输入您的账号信息</p>
          </div>

          <div class="login-type-tabs">
            <button
              v-for="(type, idx) in loginTypes"
              :key="type.value"
              class="type-tab"
              :class="{ active: activeLoginType === idx }"
              @click="activeLoginType = idx"
            >
              {{ type.label }}
            </button>
          </div>

          <form class="login-form" @submit.prevent="handleLogin">
            <div class="form-group">
              <label>账号</label>
              <input
                v-model="loginForm.username"
                type="text"
                class="input"
                placeholder="请输入学号/工号"
                autocomplete="username"
              />
            </div>

            <div class="form-group">
              <label>密码</label>
              <input
                v-model="loginForm.password"
                type="password"
                class="input"
                placeholder="请输入密码"
                autocomplete="current-password"
              />
            </div>

            <div class="form-group">
              <label>验证码</label>
              <div class="captcha-row">
                <input
                  v-model="loginForm.captcha"
                  type="text"
                  class="input captcha-input"
                  placeholder="请输入验证码"
                  maxlength="4"
                />
                <div class="captcha-img" @click="fetchCaptcha">
                  <img :key="captchaKey" :src="captchaUrl" alt="验证码" />
                </div>
              </div>
            </div>

            <button type="submit" class="submit-btn" :disabled="loading">
              <span v-if="!loading">登 录</span>
              <span v-else>登录中...</span>
            </button>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.login-container {
  display: flex;
  width: 900px;
  max-width: 100%;
  background: #fff;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
}

.login-left {
  flex: 1;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 48px 40px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.brand {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 48px;
}

.logo {
  width: 48px;
  height: 48px;
  background: rgba(255,255,255,0.2);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.logo svg {
  width: 26px;
  height: 26px;
  color: #fff;
}

.brand-text h1 {
  font-size: 22px;
  font-weight: 600;
  color: #fff;
  margin-bottom: 4px;
}

.brand-text p {
  font-size: 13px;
  color: rgba(255,255,255,0.8);
}

.features {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.feature {
  display: flex;
  align-items: center;
  gap: 14px;
  color: #fff;
}

.feature svg {
  width: 20px;
  height: 20px;
  opacity: 0.9;
}

.feature span {
  font-size: 14px;
  font-weight: 500;
}

.login-right {
  width: 420px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 40px;
}

.login-box {
  width: 100%;
  max-width: 320px;
}

.login-header {
  text-align: center;
  margin-bottom: 28px;
}

.login-header h2 {
  font-size: 20px;
  font-weight: 600;
  color: #1F1F1F;
  margin-bottom: 6px;
}

.login-header p {
  font-size: 13px;
  color: #666;
}

.login-type-tabs {
  display: flex;
  background: #F7F8FA;
  padding: 4px;
  border-radius: 8px;
  margin-bottom: 24px;
}

.type-tab {
  flex: 1;
  padding: 10px 16px;
  font-size: 13px;
  font-weight: 500;
  color: #666;
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: all 150ms ease;
}

.type-tab:hover {
  color: #333;
}

.type-tab.active {
  background: #fff;
  color: #1677FF;
  box-shadow: 0 2px 4px rgba(0,0,0,0.08);
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 13px;
  font-weight: 500;
  color: #333;
}

.input {
  width: 100%;
  padding: 11px 12px;
  font-size: 14px;
  color: #1F1F1F;
  background: #fff;
  border: 1px solid #E8E8E8;
  border-radius: 6px;
  transition: all 150ms ease;
  outline: none;
}

.input:focus {
  border-color: #1677FF;
  box-shadow: 0 0 0 2px rgba(22, 119, 255, 0.1);
}

.input::placeholder {
  color: #999;
}

.captcha-row {
  display: flex;
  gap: 10px;
}

.captcha-input {
  flex: 1;
}

.captcha-img {
  width: 120px;
  height: 42px;
  border: 1px solid #E8E8E8;
  border-radius: 6px;
  overflow: hidden;
  cursor: pointer;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
}

.captcha-img:hover {
  border-color: #1677FF;
}

.captcha-img img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.submit-btn {
  width: 100%;
  padding: 12px;
  font-size: 14px;
  font-weight: 500;
  color: #fff;
  background: #1677FF;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: all 150ms ease;
  margin-top: 4px;
}

.submit-btn:hover:not(:disabled) {
  background: #4096FF;
}

.submit-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

@media (max-width: 768px) {
  .login-container {
    flex-direction: column;
    width: 100%;
  }

  .login-left {
    padding: 32px 24px;
  }

  .features {
    display: none;
  }

  .login-right {
    width: 100%;
    padding: 32px 24px;
  }
}
</style>
