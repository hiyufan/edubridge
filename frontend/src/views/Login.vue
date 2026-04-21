<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import request from '../utils/request'
import { useUserStore } from '../stores/user'

const router = useRouter()
const userStore = useUserStore()

// B12 修复：不再从 localStorage 读取 sessionId，login 时直接从 request 对象获取实时值

const loginForm = ref({
  username: '',
  password: '',
  captcha: '',
  loginType: 'xsxh'
})

const captchaUrl = ref('')
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
    const res = await request.post('/auth/login', payload)
    userStore.setUser({
      token: res.token,
      uid: res.uid
    })
    ElMessage.success('登录成功')
    router.push('/')

    // 登录成功后后台预加载数据，不阻塞跳转
    request.get('/schedule?week=1').catch(() => {})
    request.get('/score').catch(() => {})
  } catch (error) {
    // 错误已在拦截器处理
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-container">
    <!-- Animated Background -->
    <div class="login-bg">
      <div class="bg-gradient"></div>
      <div class="bg-noise"></div>
    </div>

    <!-- Login Card -->
    <div class="login-card animate-apple-scale-in">
      <!-- Logo & Title -->
      <div class="login-header">
        <div class="login-logo">
          <svg viewBox="0 0 100 100" class="logo-icon">
            <defs>
              <linearGradient id="logoGrad" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" style="stop-color:#0A84FF"/>
                <stop offset="100%" style="stop-color:#5AC8FA"/>
              </linearGradient>
            </defs>
            <rect x="15" y="20" width="70" height="60" rx="12" fill="url(#logoGrad)"/>
            <rect x="25" y="35" width="50" height="4" rx="2" fill="white" opacity="0.9"/>
            <rect x="25" y="45" width="35" height="4" rx="2" fill="white" opacity="0.7"/>
            <rect x="25" y="55" width="42" height="4" rx="2" fill="white" opacity="0.5"/>
            <rect x="25" y="65" width="28" height="4" rx="2" fill="white" opacity="0.3"/>
          </svg>
        </div>
        <h1 class="login-title">教务系统</h1>
        <p class="login-subtitle">University Academic Portal</p>
      </div>

      <!-- Login Type Selector -->
      <div class="login-type-selector">
        <div class="apple-segmented">
          <button
            v-for="(type, idx) in loginTypes"
            :key="type.value"
            class="apple-segmented-item"
            :class="{ active: activeLoginType === idx }"
            @click="activeLoginType = idx"
          >
            {{ type.label }}
          </button>
        </div>
      </div>

      <!-- Form -->
      <div class="login-form">
        <div class="form-item animate-apple-fade-in stagger-1">
          <label class="form-label">学号 / 账号</label>
          <input
            v-model="loginForm.username"
            type="text"
            class="apple-input"
            placeholder="请输入学号"
            autocomplete="username"
          />
        </div>

        <div class="form-item animate-apple-fade-in stagger-2">
          <label class="form-label">密码</label>
          <input
            v-model="loginForm.password"
            type="password"
            class="apple-input"
            placeholder="请输入密码"
            autocomplete="current-password"
            @keyup.enter="handleLogin"
          />
        </div>

        <div class="form-item animate-apple-fade-in stagger-3">
          <label class="form-label">验证码</label>
          <div class="captcha-wrapper">
            <input
              v-model="loginForm.captcha"
              type="text"
              class="apple-input captcha-input"
              placeholder="请输入验证码"
              maxlength="4"
              @keyup.enter="handleLogin"
            />
            <div class="captcha-image" @click="fetchCaptcha">
              <img :src="captchaUrl" alt="验证码" />
              <div class="captcha-hint">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                  <circle cx="12" cy="12" r="3"/>
                </svg>
                点击刷新
              </div>
            </div>
          </div>
        </div>

        <button
          class="apple-btn apple-btn-primary login-btn animate-apple-fade-in stagger-4"
          :class="{ loading }"
          :disabled="loading"
          @click="handleLogin"
        >
          <span v-if="!loading">登 录</span>
          <span v-else class="loading-dots">
            <span></span><span></span><span></span>
          </span>
        </button>
      </div>

      <!-- Footer -->
      <div class="login-footer animate-apple-fade-in stagger-5">
        <p>登录即表示同意 <a href="#">服务条款</a> 和 <a href="#">隐私政策</a></p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  position: relative;
  overflow: hidden;
}

.login-bg {
  position: absolute;
  inset: 0;
  z-index: 0;
}

.bg-gradient {
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, #e0e5ec 0%, #f5f7fa 50%, #e8eef5 100%);
}

.bg-noise {
  position: absolute;
  inset: 0;
  opacity: 0.4;
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 200 200' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noise'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.65' numOctaves='3' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%' height='100%' filter='url(%23noise)'/%3E%3C/svg%3E");
}

.login-card {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 400px;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: saturate(180%) blur(40px);
  -webkit-backdrop-filter: saturate(180%) blur(40px);
  border-radius: 24px;
  padding: 40px 32px;
  box-shadow:
    0 25px 50px -12px rgba(0, 0, 0, 0.08),
    0 0 0 0.5px rgba(0, 0, 0, 0.05),
    inset 0 0 0 0.5px rgba(255, 255, 255, 0.5);
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.login-logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  margin-bottom: 20px;
  background: linear-gradient(145deg, #f5f7fa, #e8eef5);
  border-radius: 22px;
  box-shadow:
    8px 8px 20px rgba(0, 0, 0, 0.06),
    -8px -8px 20px rgba(255, 255, 255, 0.8),
    inset 0 0 0 0.5px rgba(255, 255, 255, 0.5);
}

.logo-icon {
  width: 50px;
  height: 50px;
}

.login-title {
  font-size: 28px;
  font-weight: 700;
  letter-spacing: -0.03em;
  color: #1d1d1f;
  margin-bottom: 4px;
}

.login-subtitle {
  font-size: 13px;
  font-weight: 500;
  color: #86868b;
  letter-spacing: 0.02em;
}

.login-type-selector {
  display: flex;
  justify-content: center;
  margin-bottom: 28px;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-label {
  font-size: 13px;
  font-weight: 600;
  color: #86868b;
  letter-spacing: 0.02em;
}

.captcha-wrapper {
  display: flex;
  gap: 12px;
}

.captcha-input {
  flex: 1;
}

.captcha-image {
  position: relative;
  width: 120px;
  height: 44px;
  border-radius: 10px;
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.2s ease;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  flex-shrink: 0;
}

.captcha-image:hover {
  transform: scale(1.02);
}

.captcha-image img {
  width: 120px;
  height: 44px;
  object-fit: fill;
}

.captcha-hint {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  font-size: 11px;
  color: white;
  background: rgba(0, 0, 0, 0.3);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.captcha-image:hover .captcha-hint {
  opacity: 1;
}

.captcha-hint svg {
  width: 12px;
  height: 12px;
}

.login-btn {
  width: 100%;
  height: 54px;
  margin-top: 8px;
  font-size: 17px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.login-btn.loading {
  pointer-events: none;
}

.loading-dots {
  display: flex;
  gap: 4px;
}

.loading-dots span {
  width: 6px;
  height: 6px;
  background: white;
  border-radius: 50%;
  animation: loadingPulse 1.2s ease-in-out infinite;
}

.loading-dots span:nth-child(2) {
  animation-delay: 0.2s;
}

.loading-dots span:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes loadingPulse {
  0%, 100% {
    opacity: 0.4;
    transform: scale(0.8);
  }
  50% {
    opacity: 1;
    transform: scale(1);
  }
}

.login-footer {
  margin-top: 28px;
  text-align: center;
  font-size: 12px;
  color: #86868b;
}

.login-footer a {
  color: #0A84FF;
  text-decoration: none;
}

.login-footer a:hover {
  text-decoration: underline;
}

/* Hide Element Plus defaults */
:deep(.el-input__prefix) {
  display: none;
}
</style>
