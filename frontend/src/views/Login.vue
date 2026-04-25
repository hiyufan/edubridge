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

    request.get('/schedule?week=1').catch(() => {})
    request.get('/score').catch(() => {})
  } catch (error) {
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <!-- Left brand panel (desktop) -->
    <div class="brand-panel">
      <div class="brand-inner">
        <div class="brand-logo">
          <svg viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
            <rect x="4" y="8" width="40" height="32" rx="4" fill="#78716C"/>
            <rect x="10" y="14" width="28" height="3" rx="1.5" fill="white" opacity="0.9"/>
            <rect x="10" y="20" width="20" height="2.5" rx="1.25" fill="white" opacity="0.6"/>
            <rect x="10" y="25" width="24" height="2.5" rx="1.25" fill="white" opacity="0.6"/>
            <rect x="10" y="30" width="16" height="2.5" rx="1.25" fill="white" opacity="0.4"/>
            <path d="M38 8 L44 12 L44 12 L38 16" stroke="#A16207" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/>
            <path d="M38 32 L44 36 L44 36 L38 40" stroke="#A16207" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/>
          </svg>
        </div>
        <h1 class="brand-title">教务系统</h1>
        <p class="brand-subtitle">University Academic Portal</p>
        <div class="brand-divider"></div>
        <p class="brand-desc">简洁、高效的课程与成绩管理平台</p>
      </div>
      <div class="brand-decoration">
        <div class="deco-line deco-1"></div>
        <div class="deco-line deco-2"></div>
        <div class="deco-line deco-3"></div>
      </div>
    </div>

    <!-- Right login form -->
    <div class="form-panel">
      <div class="form-container warm-scale-in">
        <div class="form-header">
          <h2 class="form-title">登录</h2>
          <p class="form-hint">请输入您的账号信息</p>
        </div>

        <!-- Login Type Selector -->
        <div class="login-type-selector">
          <div class="warm-segmented">
            <button
              v-for="(type, idx) in loginTypes"
              :key="type.value"
              class="warm-segmented-item"
              :class="{ active: activeLoginType === idx }"
              @click="activeLoginType = idx"
            >
              {{ type.label }}
            </button>
          </div>
        </div>

        <!-- Form -->
        <div class="login-form">
          <div class="form-item warm-fade-in stagger-1">
            <label class="form-label">学号 / 账号</label>
            <input
              v-model="loginForm.username"
              type="text"
              class="warm-input"
              placeholder="请输入学号"
              autocomplete="username"
            />
          </div>

          <div class="form-item warm-fade-in stagger-2">
            <label class="form-label">密码</label>
            <input
              v-model="loginForm.password"
              type="password"
              class="warm-input"
              placeholder="请输入密码"
              autocomplete="current-password"
              @keyup.enter="handleLogin"
            />
          </div>

          <div class="form-item warm-fade-in stagger-3">
            <label class="form-label">验证码</label>
            <div class="captcha-wrapper">
              <input
                v-model="loginForm.captcha"
                type="text"
                class="warm-input captcha-input"
                placeholder="请输入验证码"
                maxlength="4"
                @keyup.enter="handleLogin"
              />
              <div class="captcha-image" @click="fetchCaptcha">
                <img :src="captchaUrl" alt="验证码" />
                <div class="captcha-overlay">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                    <circle cx="12" cy="12" r="3"/>
                  </svg>
                  <span>点击刷新</span>
                </div>
              </div>
            </div>
          </div>

          <button
            class="warm-btn warm-btn-primary login-btn warm-fade-in stagger-4"
            :class="{ loading }"
            :disabled="loading"
            @click="handleLogin"
          >
            <span v-if="!loading">登 录</span>
            <span v-else class="loading-text">登录中...</span>
          </button>
        </div>

        <!-- Footer -->
        <div class="login-footer warm-fade-in stagger-5">
          <p>登录即表示同意 <a href="#">服务条款</a> 和 <a href="#">隐私政策</a></p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  background: var(--color-bg);
}

/* Brand Panel - Left */
.brand-panel {
  display: none;
  width: 45%;
  background: var(--color-primary);
  position: relative;
  overflow: hidden;
}

@media (min-width: 768px) {
  .brand-panel {
    display: flex;
    align-items: center;
    justify-content: center;
  }
}

.brand-inner {
  position: relative;
  z-index: 1;
  padding: 48px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.brand-logo svg {
  width: 64px;
  height: 64px;
}

.brand-title {
  font-family: var(--font-serif);
  font-size: 36px;
  font-weight: 700;
  color: white;
  letter-spacing: 0.02em;
  margin-top: 8px;
}

.brand-subtitle {
  font-size: 13px;
  font-weight: 400;
  color: rgba(255, 255, 255, 0.5);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.brand-divider {
  width: 40px;
  height: 2px;
  background: var(--color-accent);
  margin: 8px 0;
}

.brand-desc {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.6);
  line-height: 1.6;
  max-width: 280px;
}

/* Brand decoration lines */
.brand-decoration {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.deco-line {
  position: absolute;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 2px;
}

.deco-1 {
  width: 100%;
  height: 1px;
  top: 30%;
}

.deco-2 {
  width: 1px;
  height: 100%;
  left: 30%;
}

.deco-3 {
  width: 200px;
  height: 200px;
  border: 1px solid rgba(255, 255, 255, 0.03);
  border-radius: 50%;
  top: 50%;
  left: 60%;
  transform: translate(-50%, -50%);
}

/* Form Panel - Right */
.form-panel {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px 24px;
}

.form-container {
  width: 100%;
  max-width: 380px;
}

.form-header {
  margin-bottom: 28px;
}

.form-title {
  font-family: var(--font-serif);
  font-size: 26px;
  font-weight: 700;
  color: var(--color-text);
  letter-spacing: 0.01em;
}

.form-hint {
  font-size: 14px;
  color: var(--color-text-muted);
  margin-top: 4px;
}

.login-type-selector {
  display: flex;
  margin-bottom: 24px;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.form-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-secondary);
}

.captcha-wrapper {
  display: flex;
  gap: 10px;
}

.captcha-input {
  flex: 1;
}

.captcha-image {
  position: relative;
  width: 140px;
  height: 50px;
  border-radius: var(--radius-md);
  overflow: hidden;
  cursor: pointer;
  flex-shrink: 0;
  border: 1px solid var(--color-border);
  background: white;
  transition: border-color 0.2s ease;
}

.captcha-image:hover {
  border-color: var(--color-primary);
}

.captcha-image img {
  width: 140px;
  height: 50px;
  object-fit: contain;
  display: block;
}

.captcha-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  font-size: 11px;
  color: white;
  background: rgba(0, 0, 0, 0.35);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.captcha-image:hover .captcha-overlay {
  opacity: 1;
}

.captcha-overlay svg {
  width: 12px;
  height: 12px;
}

.login-btn {
  width: 100%;
  height: 50px;
  margin-top: 4px;
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0.1em;
}

.login-btn.loading {
  pointer-events: none;
  opacity: 0.7;
}

.loading-text {
  letter-spacing: 0.1em;
}

.login-footer {
  margin-top: 24px;
  text-align: center;
  font-size: 12px;
  color: var(--color-text-muted);
}

.login-footer a {
  color: var(--color-primary);
  text-decoration: none;
  font-weight: 500;
}

.login-footer a:hover {
  text-decoration: underline;
}
</style>
