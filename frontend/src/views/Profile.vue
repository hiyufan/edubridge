<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '../stores/user'
import { useThemeStore } from '../stores/theme'
import { getNotifyPermission, requestNotifyPermission } from '../utils/notifications'
import request from '../utils/request'

const router = useRouter()
const userStore = useUserStore()
const themeStore = useThemeStore()

const studentName = ref('')
const className = ref('')
const notifyPermission = ref('default')

// 功能 07: iCal 订阅信息
const iCalTokenInfo = ref(null)

// 功能 09: Webhook 配置
const webhookUrl = ref('')
const webhookSecret = ref('')
const webhookInfo = ref(null)

onMounted(async () => {
  try {
    const res = await request.get('/auth/me')
    if (res.data?.name) studentName.value = res.data.name
    if (res.data?.className) className.value = res.data.className
    if (studentName.value) {
      userStore.setUser({ ...userStore, name: studentName.value, className: className.value })
    }
  } catch (e) {
    // 忽略，Profile 降级显示 uid
  }
  notifyPermission.value = getNotifyPermission()

  // 功能 07: 获取 iCal 订阅信息
  try {
    const icalRes = await request.get('/schedule/ical/token-info')
    iCalTokenInfo.value = icalRes.data
  } catch {}

  // 功能 09: 获取 Webhook 信息
  try {
    const whRes = await request.get('/webhook/info')
    webhookInfo.value = whRes.data
    webhookUrl.value = whRes.data?.url || ''
    webhookSecret.value = whRes.data?.secret || ''
  } catch {}
})

// 功能 07: 重新生成 iCal token
const regenerateICalToken = async () => {
  try {
    const res = await request.post('/schedule/ical/token')
    iCalTokenInfo.value = res.data
    ElMessage.success('订阅链接已重新生成')
  } catch {
    ElMessage.error('生成失败')
  }
}

// 功能 09: 保存 Webhook
const saveWebhook = async () => {
  if (!webhookUrl.value) {
    ElMessage.warning('请输入 Webhook URL')
    return
  }
  try {
    await request.post('/webhook/register', {
      url: webhookUrl.value,
      secret: webhookSecret.value
    })
    ElMessage.success('Webhook 配置已保存')
    // 重新获取
    const whRes = await request.get('/webhook/info')
    webhookInfo.value = whRes.data
  } catch {
    ElMessage.error('保存失败')
  }
}

const handleNotify = async () => {
  if (notifyPermission.value === 'granted') {
    ElMessage.info('已开启课程提醒')
    return
  }
  if (notifyPermission.value === 'denied') {
    ElMessage.warning('通知已被浏览器拒绝，请在设置中开启')
    return
  }
  const result = await requestNotifyPermission()
  notifyPermission.value = result
  if (result === 'granted') {
    ElMessage.success('已开启课程提醒')
  }
}

const handleLogout = async () => {
  try {
    await ElMessageBox.confirm('确定要退出登录吗？退出后将返回登录页面。', '退出登录', {
      confirmButtonText: '退出',
      cancelButtonText: '取消',
      confirmButtonClass: 'logout-confirm-btn',
      cancelButtonClass: 'logout-cancel-btn',
      type: 'warning',
      customClass: 'apple-message-box'
    })
    userStore.logout()
    ElMessage.success({
      message: '已安全退出',
      duration: 1500
    })
    setTimeout(() => {
      router.push('/login')
    }, 500)
  } catch {
    // 用户取消
  }
}

// 获取用户头像首字母
const getAvatarLetter = () => {
  return (userStore.uid || 'U').charAt(0).toUpperCase()
}
</script>

<template>
  <div class="profile-page">
    <!-- Profile Header Card -->
    <div class="profile-card animate-warm-fade-in">
      <div class="profile-avatar">
        <span class="avatar-letter">{{ getAvatarLetter() }}</span>
        <div class="avatar-ring"></div>
      </div>
      <div class="profile-info">
        <h2 class="profile-name">{{ studentName || userStore.uid || '用户' }}</h2>
        <p class="profile-role">{{ className || '学生' }}</p>
      </div>
      <div class="profile-status">
        <span class="status-dot"></span>
        <span class="status-text">已登录</span>
      </div>
    </div>

    <!-- Settings Groups -->
    <div class="settings-section animate-warm-fade-in stagger-1">
      <div class="apple-section-header">账户信息</div>
      <div class="apple-grouped-list">
        <div class="apple-grouped-item">
          <div class="item-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
              <circle cx="12" cy="7" r="4"/>
            </svg>
          </div>
          <div class="item-content">
            <span class="item-label">学号</span>
            <span class="item-value">{{ userStore.uid || '-' }}</span>
          </div>
        </div>
        <div class="apple-grouped-item">
          <div class="item-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
              <circle cx="12" cy="7" r="4"/>
            </svg>
          </div>
          <div class="item-content">
            <span class="item-label">姓名</span>
            <span class="item-value">{{ studentName || '-' }}</span>
          </div>
        </div>
        <div class="apple-grouped-item">
          <div class="item-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
              <circle cx="9" cy="7" r="4"/>
              <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
              <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
            </svg>
          </div>
          <div class="item-content">
            <span class="item-label">班级</span>
            <span class="item-value">{{ className || '-' }}</span>
          </div>
        </div>
        <div class="apple-grouped-item">
          <div class="item-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
              <polyline points="22,6 12,13 2,6"/>
            </svg>
          </div>
          <div class="item-content">
            <span class="item-label">身份类型</span>
            <span class="item-value">学生</span>
          </div>
        </div>
        <div class="apple-grouped-item">
          <div class="item-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
          </div>
          <div class="item-content">
            <span class="item-label">登录状态</span>
            <span class="item-value status-active">已认证</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Theme Settings -->
    <div class="settings-section animate-warm-fade-in stagger-2">
      <div class="apple-section-header">外观</div>
      <div class="apple-grouped-list">
        <div class="apple-grouped-item">
          <div class="item-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <circle cx="12" cy="12" r="5"/>
              <line x1="12" y1="1" x2="12" y2="3"/>
              <line x1="12" y1="21" x2="12" y2="23"/>
              <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/>
              <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/>
              <line x1="1" y1="12" x2="3" y2="12"/>
              <line x1="21" y1="12" x2="23" y2="12"/>
              <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/>
              <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
            </svg>
          </div>
          <div class="item-content" style="flex-direction:row;align-items:center;gap:8px;">
            <span class="item-label">主题模式</span>
            <div class="theme-modes">
              <button
                v-for="m in [{v:'light',l:'浅'},{v:'dark',l:'深'},{v:'auto',l:'自动'}]"
                :key="m.v"
                class="theme-mode-btn"
                :class="{ active: themeStore.mode === m.v }"
                @click="themeStore.setMode(m.v)"
              >{{ m.l }}</button>
            </div>
          </div>
        </div>
        <div class="apple-grouped-item" style="flex-wrap:wrap;gap:8px;">
          <div class="item-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <circle cx="13.5" cy="6.5" r="3.5"/>
              <circle cx="17.5" cy="10.5" r="2.5"/>
              <circle cx="8.5" cy="7.5" r="4.5"/>
              <circle cx="6.5" cy="12.5" r="5"/>
            </svg>
          </div>
          <div class="item-content" style="flex-direction:row;align-items:center;gap:6px;flex:1;">
            <span class="item-label">主题色</span>
            <div class="color-swatches">
              <button
                v-for="c in themeStore.PRESET_COLORS"
                :key="c.value"
                class="color-swatch"
                :style="{ background: c.value }"
                :class="{ active: themeStore.primaryColor === c.value }"
                @click="themeStore.setPrimaryColor(c.value)"
              ></button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Notification Settings -->
    <div class="settings-section animate-warm-fade-in stagger-2">
      <div class="apple-section-header">通知</div>
      <div class="apple-grouped-list">
        <div class="apple-grouped-item item-clickable" @click="handleNotify">
          <div class="item-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/>
              <path d="M13.73 21a2 2 0 0 1-3.46 0"/>
            </svg>
          </div>
          <div class="item-content">
            <span class="item-label">课程提醒</span>
            <span class="item-value">{{
              notifyPermission === 'granted' ? '已开启' :
              notifyPermission === 'denied' ? '已被拒绝' : '未开启'
            }}</span>
          </div>
          <div class="item-arrow">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="9 18 15 12 9 6"/>
            </svg>
          </div>
        </div>
      </div>
    </div>

    <!-- 功能 07: iCal 订阅信息 -->
    <div class="settings-section animate-warm-fade-in stagger-2" v-if="iCalTokenInfo">
      <div class="apple-section-header">日历订阅</div>
      <div class="apple-grouped-list">
        <div class="apple-grouped-item">
          <div class="item-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>
            </svg>
          </div>
          <div class="item-content">
            <span class="item-label">订阅状态</span>
            <span class="item-value">已激活 · 有效期至 {{ iCalTokenInfo.expireAt }}</span>
          </div>
        </div>
        <div class="apple-grouped-item item-clickable" @click="regenerateICalToken">
          <div class="item-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <polyline points="23 4 23 10 17 10"/>
              <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
            </svg>
          </div>
          <div class="item-content">
            <span class="item-label">重新生成订阅链接</span>
          </div>
          <div class="item-arrow">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="9 18 15 12 9 6"/>
            </svg>
          </div>
        </div>
      </div>
    </div>

    <!-- 功能 09: Webhook 配置 -->
    <div class="settings-section animate-warm-fade-in stagger-2">
      <div class="apple-section-header">Webhook 推送</div>
      <div class="apple-grouped-list">
        <div class="apple-grouped-item">
          <div class="item-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
              <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
            </svg>
          </div>
          <div class="item-content" style="flex-direction:column;gap:8px;">
            <span class="item-label">Webhook URL</span>
            <el-input v-model="webhookUrl" placeholder="https://example.com/webhook" size="small" />
          </div>
        </div>
        <div class="apple-grouped-item">
          <div class="item-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
          </div>
          <div class="item-content" style="flex-direction:column;gap:8px;">
            <span class="item-label">密钥（可选）</span>
            <el-input v-model="webhookSecret" placeholder="签名密钥" size="small" />
          </div>
        </div>
        <div class="apple-grouped-item item-clickable" @click="saveWebhook">
          <div class="item-icon" style="background:rgba(0,122,255,0.1);">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/>
              <polyline points="17 21 17 13 7 13 7 21"/>
              <polyline points="7 3 7 8 15 8"/>
            </svg>
          </div>
          <div class="item-content">
            <span class="item-label" style="color:#007AFF;">保存 Webhook 配置</span>
          </div>
        </div>
        <!-- Webhook 历史 -->
        <template v-if="webhookInfo?.history?.length">
          <div v-for="(h, idx) in webhookInfo.history.slice(0, 3)" :key="idx" class="webhook-history-item">
            <span class="wh-time">{{ h.time }}</span>
            <span class="wh-summary">{{ h.summary }}</span>
            <span :class="['wh-status', h.success ? 'success' : 'fail']">
              {{ h.success ? '成功' : '失败' }}
            </span>
          </div>
        </template>
      </div>
    </div>

    <!-- Security Settings -->
    <div class="settings-section animate-warm-fade-in stagger-2">
      <div class="apple-section-header">安全设置</div>
      <div class="apple-grouped-list">
        <div class="apple-grouped-item item-clickable">
          <div class="item-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <circle cx="12" cy="12" r="3"/>
              <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
            </svg>
          </div>
          <div class="item-content">
            <span class="item-label">修改密码</span>
          </div>
          <div class="item-arrow">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="9 18 15 12 9 6"/>
            </svg>
          </div>
        </div>
        <div class="apple-grouped-item item-clickable">
          <div class="item-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
            </svg>
          </div>
          <div class="item-content">
            <span class="item-label">隐私政策</span>
          </div>
          <div class="item-arrow">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="9 18 15 12 9 6"/>
            </svg>
          </div>
        </div>
        <div class="apple-grouped-item item-clickable">
          <div class="item-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <circle cx="12" cy="12" r="10"/>
              <line x1="12" y1="16" x2="12" y2="12"/>
              <line x1="12" y1="8" x2="12.01" y2="8"/>
            </svg>
          </div>
          <div class="item-content">
            <span class="item-label">关于我们</span>
          </div>
          <div class="item-arrow">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="9 18 15 12 9 6"/>
            </svg>
          </div>
        </div>
      </div>
    </div>

    <!-- Version Info -->
    <div class="version-info animate-warm-fade-in stagger-3">
      <span>教务系统 v1.0.0</span>
    </div>

    <!-- Logout Button -->
    <div class="logout-section animate-warm-fade-in stagger-4">
      <button class="apple-btn logout-btn" @click="handleLogout">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" class="logout-icon">
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
          <polyline points="16 17 21 12 16 7"/>
          <line x1="21" y1="12" x2="9" y2="12"/>
        </svg>
        退出登录
      </button>
    </div>
  </div>
</template>

<style scoped>
.profile-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* Profile Card */
.profile-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  background: white;
  border-radius: var(--radius-lg);
  padding: 32px 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04), 0 1px 2px rgba(0, 0, 0, 0.03);
  position: relative;
  overflow: hidden;
  border: 1px solid var(--color-border-light);
}

.profile-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 80px;
  background: var(--color-primary);
  border-radius: var(--radius-lg) var(--radius-lg) 0 0;
}

.profile-avatar {
  position: relative;
  width: 88px;
  height: 88px;
  margin-top: 12px;
  z-index: 1;
}

.avatar-letter {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary);
  border-radius: 50%;
  font-size: 36px;
  font-weight: 600;
  color: white;
  letter-spacing: -0.02em;
}

.avatar-ring {
  position: absolute;
  inset: -4px;
  border-radius: 50%;
  border: 3px solid white;
  box-shadow: 0 2px 8px rgba(120, 113, 108, 0.15);
}

.profile-info {
  text-align: center;
  margin-top: 16px;
  z-index: 1;
}

.profile-name {
  font-size: 22px;
  font-weight: 700;
  color: var(--color-text);
  letter-spacing: -0.024em;
}

.profile-role {
  font-size: 14px;
  color: var(--color-text-muted);
  margin-top: 4px;
  font-weight: 500;
}

.profile-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  padding: 6px 14px;
  background: rgba(120, 113, 108, 0.08);
  border-radius: 20px;
}

.status-dot {
  width: 8px;
  height: 8px;
  background: #A16207;
  border-radius: 50%;
  animation: statusPulse 2s ease-in-out infinite;
}

@keyframes statusPulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.status-text {
  font-size: 13px;
  font-weight: 600;
  color: #A16207;
}

/* Settings Section */
.settings-section {
  display: flex;
  flex-direction: column;
}

.apple-grouped-list {
  background: white;
  border-radius: var(--radius-lg);
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  border: 1px solid var(--color-border-light);
}

.apple-grouped-item {
  display: flex;
  align-items: center;
  padding: 14px 16px;
  background: white;
  border-bottom: 0.5px solid var(--color-border);
}

.apple-grouped-item:last-child {
  border-bottom: none;
}

.item-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  margin-right: 12px;
  background: var(--color-bg);
  border-radius: var(--radius-md);
}

.item-icon svg {
  width: 18px;
  height: 18px;
  color: var(--color-primary);
}

.item-content {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.item-label {
  font-size: 15px;
  font-weight: 500;
  color: var(--color-text);
}

.item-value {
  font-size: 13px;
  color: var(--color-text-muted);
  margin-top: 2px;
}

.item-value.status-active {
  color: #A16207;
  font-weight: 600;
}

.item-arrow {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
}

.item-arrow svg {
  width: 16px;
  height: 16px;
  color: var(--color-border);
}

.item-clickable {
  cursor: pointer;
  transition: background 0.2s ease;
}

.item-clickable:hover {
  background: var(--color-bg);
}

/* Version Info */
.version-info {
  text-align: center;
  font-size: 12px;
  color: var(--color-text-muted);
  padding: 8px;
}

/* Logout Button */
.logout-section {
  padding: 8px 0;
}

.logout-btn {
  width: 100%;
  height: 54px;
  background: rgba(220, 38, 38, 0.06);
  color: #DC2626;
  border: none;
  font-weight: 600;
  border-radius: var(--radius-md);
}

.logout-btn:hover {
  background: rgba(220, 38, 38, 0.1);
}

.logout-icon {
  width: 20px;
  height: 20px;
  margin-right: 8px;
}

/* Theme Mode Buttons */
.theme-modes {
  display: flex;
  gap: 4px;
  margin-left: auto;
}

.theme-mode-btn {
  padding: 4px 12px;
  border: 1.5px solid var(--color-border);
  border-radius: 20px;
  background: white;
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: all 0.2s ease;
}

.theme-mode-btn.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
  font-weight: 600;
}

.theme-mode-btn:hover:not(.active) {
  border-color: var(--color-primary);
  color: var(--color-primary);
}

/* Color Swatches */
.color-swatches {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  margin-left: auto;
}

.color-swatch {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 1px 4px rgba(0,0,0,0.15);
}

.color-swatch.active {
  border-color: white;
  box-shadow: 0 0 0 2px currentColor, 0 2px 6px rgba(0,0,0,0.2);
  transform: scale(1.15);
}

.color-swatch:hover:not(.active) {
  transform: scale(1.1);
}

/* 功能 09: Webhook 历史 */
.webhook-history-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  font-size: 12px;
  border-top: 0.5px solid rgba(0,0,0,0.06);
}

.wh-time {
  color: #8e8e93;
  flex-shrink: 0;
}

.wh-summary {
  flex: 1;
  color: #636366;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.wh-status {
  font-weight: 600;
  flex-shrink: 0;
}

.wh-status.success { color: #A16207; }
.wh-status.fail { color: #DC2626; }

/* =====================
   PC Responsive (lg+)
   ===================== */
@media (min-width: 1024px) {
  .profile-page {
    display: grid;
    grid-template-columns: 380px 1fr;
    grid-template-rows: auto auto auto;
    gap: 24px;
    max-width: 1100px;
    align-items: start;
  }

  /* Header spans full width on left column */
  .profile-card {
    grid-column: 1;
    grid-row: 1;
    flex-direction: column;
    align-items: center;
    text-align: center;
    padding: 36px 28px 28px;
    border-radius: var(--radius-xl);
  }

  .profile-info {
    margin-top: 16px;
  }

  .profile-name {
    font-size: 26px;
  }

  /* Settings sections stack in right column */
  .settings-section {
    grid-column: 2;
    grid-row: 1 / span 3;
    align-items: flex-start;
  }

  .settings-section:nth-child(2) {
    grid-row: 1;
  }

  .apple-grouped-list {
    border-radius: var(--radius-xl);
  }

  .apple-grouped-item {
    padding: 16px 20px;
  }

  .apple-section-header {
    font-size: 12px;
    padding: 0 4px 8px;
    letter-spacing: 0.08em;
  }

  .item-icon {
    width: 36px;
    height: 36px;
    margin-right: 16px;
  }

  .item-icon svg {
    width: 20px;
    height: 20px;
  }

  .item-label {
    font-size: 15px;
  }

  .item-value {
    font-size: 14px;
  }

  .theme-mode-btn {
    padding: 5px 16px;
    font-size: 13px;
    border-radius: 20px;
  }

  .color-swatch {
    width: 28px;
    height: 28px;
  }

  .logout-section {
    grid-column: 1;
    grid-row: 2;
    padding: 0;
  }

  .logout-btn {
    border-radius: var(--radius-lg);
    height: 48px;
    font-size: 15px;
  }

  .version-info {
    grid-column: 1;
    grid-row: 3;
    padding: 8px 0 0;
  }
}
</style>

<style>
/* Global overrides for Element Plus MessageBox */
.apple-message-box .el-message-box__headerbtn .el-message-box__close {
  color: var(--color-text-muted);
}

.apple-message-box .el-message-box__title {
  font-size: 17px;
  font-weight: 600;
  color: var(--color-text);
}

.apple-message-box .el-message-box__message {
  font-size: 14px;
  color: var(--color-text-muted);
}

.apple-message-box .el-button--primary {
  background: #DC2626 !important;
  border-color: #DC2626 !important;
}

.apple-message-box .logout-confirm-btn {
  background: #DC2626 !important;
  border-color: #DC2626 !important;
}
</style>
