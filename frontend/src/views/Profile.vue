<script setup>
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '../stores/user'

const router = useRouter()
const userStore = useUserStore()

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
    <div class="profile-card animate-apple-fade-in">
      <div class="profile-avatar">
        <span class="avatar-letter">{{ getAvatarLetter() }}</span>
        <div class="avatar-ring"></div>
      </div>
      <div class="profile-info">
        <h2 class="profile-name">{{ userStore.uid || '用户' }}</h2>
        <p class="profile-role">学生</p>
      </div>
      <div class="profile-status">
        <span class="status-dot"></span>
        <span class="status-text">已登录</span>
      </div>
    </div>

    <!-- Settings Groups -->
    <div class="settings-section animate-apple-fade-in stagger-1">
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

    <div class="settings-section animate-apple-fade-in stagger-2">
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
    <div class="version-info animate-apple-fade-in stagger-3">
      <span>教务系统 v1.0.0</span>
    </div>

    <!-- Logout Button -->
    <div class="logout-section animate-apple-fade-in stagger-4">
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
  border-radius: 20px;
  padding: 32px 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04), 0 4px 12px rgba(0, 0, 0, 0.03);
  position: relative;
  overflow: hidden;
}

.profile-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 80px;
  background: linear-gradient(135deg, #007AFF 0%, #5856D6 100%);
  border-radius: 20px 20px 0 0;
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
  background: linear-gradient(135deg, #007AFF, #5856D6);
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
  box-shadow: 0 4px 16px rgba(0, 122, 255, 0.3);
}

.profile-info {
  text-align: center;
  margin-top: 16px;
  z-index: 1;
}

.profile-name {
  font-size: 22px;
  font-weight: 700;
  color: #1d1d1f;
  letter-spacing: -0.024em;
}

.profile-role {
  font-size: 14px;
  color: #8e8e93;
  margin-top: 4px;
  font-weight: 500;
}

.profile-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  padding: 6px 14px;
  background: rgba(52, 199, 89, 0.1);
  border-radius: 20px;
}

.status-dot {
  width: 8px;
  height: 8px;
  background: #34C759;
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
  color: #34C759;
}

/* Settings Section */
.settings-section {
  display: flex;
  flex-direction: column;
}

.apple-grouped-list {
  background: white;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04), 0 4px 12px rgba(0, 0, 0, 0.03);
}

.apple-grouped-item {
  display: flex;
  align-items: center;
  padding: 14px 16px;
  background: white;
  border-bottom: 0.5px solid rgba(0, 0, 0, 0.06);
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
  background: #f2f2f7;
  border-radius: 8px;
}

.item-icon svg {
  width: 18px;
  height: 18px;
  color: #007AFF;
}

.item-content {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.item-label {
  font-size: 15px;
  font-weight: 500;
  color: #1d1d1f;
}

.item-value {
  font-size: 13px;
  color: #8e8e93;
  margin-top: 2px;
}

.item-value.status-active {
  color: #34C759;
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
  color: #c7c7cc;
}

.item-clickable {
  cursor: pointer;
  transition: background 0.2s ease;
}

.item-clickable:hover {
  background: rgba(0, 122, 255, 0.03);
}

/* Version Info */
.version-info {
  text-align: center;
  font-size: 12px;
  color: #c7c7cc;
  padding: 8px;
}

/* Logout Button */
.logout-section {
  padding: 8px 0;
}

.logout-btn {
  width: 100%;
  height: 54px;
  background: rgba(255, 59, 48, 0.08);
  color: #FF3B30;
  border: none;
  font-weight: 600;
}

.logout-btn:hover {
  background: rgba(255, 59, 48, 0.15);
}

.logout-icon {
  width: 20px;
  height: 20px;
  margin-right: 8px;
}
</style>

<style>
/* Global overrides for Element Plus MessageBox */
.apple-message-box .el-message-box__headerbtn .el-message-box__close {
  color: #8e8e93;
}

.apple-message-box .el-message-box__title {
  font-size: 17px;
  font-weight: 600;
  color: #1d1d1f;
}

.apple-message-box .el-message-box__message {
  font-size: 14px;
  color: #636366;
}

.apple-message-box .el-button--primary {
  background: #FF3B30 !important;
  border-color: #FF3B30 !important;
}

.apple-message-box .logout-confirm-btn {
  background: #FF3B30 !important;
  border-color: #FF3B30 !important;
}
</style>
