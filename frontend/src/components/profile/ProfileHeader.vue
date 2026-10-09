<script setup>
import { computed } from 'vue'

const props = defineProps({
  uid: { type: String, default: '' },
  studentName: { type: String, default: '' },
  className: { type: String, default: '' }
})

const avatarLetter = computed(() => (props.uid || 'U').charAt(0).toUpperCase())
</script>

<template>
  <div class="profile-card animate-warm-fade-in">
    <div class="profile-avatar">
      <span class="avatar-letter">{{ avatarLetter }}</span>
      <div class="avatar-ring"></div>
    </div>
    <div class="profile-info">
      <h2 class="profile-name">{{ studentName || uid || '用户' }}</h2>
      <p class="profile-role">{{ className || '学生' }}</p>
    </div>
    <div class="profile-status">
      <span class="status-dot"></span>
      <span class="status-text">已登录</span>
    </div>
  </div>
</template>

<style scoped>
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

@media (min-width: 1024px) {
  .profile-card {
    padding: 36px 28px 28px;
    border-radius: var(--radius-xl);
  }

  .profile-name {
    font-size: 26px;
  }
}
</style>
