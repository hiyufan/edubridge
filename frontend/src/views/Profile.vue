<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '../stores/user'
import request from '../utils/request'
import '../styles/settings.css'
import ProfileHeader from '../components/profile/ProfileHeader.vue'
import AccountSection from '../components/profile/AccountSection.vue'
import AppearanceSection from '../components/profile/AppearanceSection.vue'
import BrowserNotifySection from '../components/profile/BrowserNotifySection.vue'
import MonitorSection from '../components/profile/MonitorSection.vue'
import ReminderSection from '../components/profile/ReminderSection.vue'
import NotifyChannelsSection from '../components/profile/NotifyChannelsSection.vue'
import ICalSection from '../components/profile/ICalSection.vue'
import WebhookSection from '../components/profile/WebhookSection.vue'
import AboutSection from '../components/profile/AboutSection.vue'

const router = useRouter()
const userStore = useUserStore()

const studentName = ref('')
const className = ref('')

onMounted(async () => {
  try {
    const res = await request.get('/auth/me')
    studentName.value = res.data?.name || ''
    className.value = res.data?.className || ''
  } catch {
    // 忽略，降级显示学号
  }
})

// 退出：通知后端作废登录凭证、停止后台监控，再清理本地状态
const handleLogout = async () => {
  try {
    await ElMessageBox.confirm('确定要退出登录吗？退出后将不再推送课表变动和上课提醒。', '退出登录', {
      confirmButtonText: '退出',
      cancelButtonText: '取消',
      confirmButtonClass: 'logout-confirm-btn',
      cancelButtonClass: 'logout-cancel-btn',
      type: 'warning',
      customClass: 'apple-message-box'
    })
  } catch {
    return // 用户取消
  }
  try {
    await request.post('/auth/logout')
  } catch {
    // 后端失败也继续退出本地登录
  }
  userStore.logout()
  ElMessage.success({ message: '已安全退出', duration: 1500 })
  setTimeout(() => router.push('/login'), 500)
}
</script>

<template>
  <div class="profile-page">
    <aside class="profile-side">
      <ProfileHeader :uid="userStore.uid" :student-name="studentName" :class-name="className" />
      <div class="logout-section animate-warm-fade-in stagger-4">
        <button class="apple-btn logout-btn" @click="handleLogout">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" class="logout-icon">
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
            <polyline points="16 17 21 12 16 7" />
            <line x1="21" y1="12" x2="9" y2="12" />
          </svg>
          退出登录
        </button>
      </div>
      <div class="version-info animate-warm-fade-in stagger-3">教务系统 v1.0.0</div>
    </aside>

    <main class="profile-main">
      <AccountSection :uid="userStore.uid" :student-name="studentName" :class-name="className" />
      <AppearanceSection />
      <BrowserNotifySection />
      <MonitorSection />
      <ReminderSection />
      <NotifyChannelsSection />
      <ICalSection />
      <WebhookSection />
      <AboutSection />
    </main>
  </div>
</template>

<style scoped>
.profile-page,
.profile-side,
.profile-main {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* 手机：版本号和退出按钮放在最底部 */
.profile-side {
  display: contents;
}

.logout-section {
  order: 3;
  padding: 8px 0;
}

.version-info {
  order: 2;
  text-align: center;
  font-size: 12px;
  color: var(--color-text-muted);
  padding: 8px;
}

.profile-main {
  order: 1;
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

/* 桌面：左侧个人信息，右侧设置 */
@media (min-width: 1024px) {
  .profile-page {
    display: grid;
    grid-template-columns: 380px 1fr;
    gap: 24px;
    max-width: 1100px;
    align-items: start;
  }

  .profile-side {
    display: flex;
    position: sticky;
    top: 24px;
  }

  .logout-section {
    padding: 0;
  }

  .logout-btn {
    border-radius: var(--radius-lg);
    height: 48px;
    font-size: 15px;
  }

  .version-info {
    padding: 0;
  }
}
</style>

<style>
/* Element Plus MessageBox 全局覆盖 */
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

.apple-message-box .el-button--primary,
.apple-message-box .logout-confirm-btn {
  background: #DC2626 !important;
  border-color: #DC2626 !important;
}
</style>
