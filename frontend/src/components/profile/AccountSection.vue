<script setup>
import SettingsSection from './SettingsSection.vue'
import SettingsItem from './SettingsItem.vue'

defineProps({
  uid: { type: String, default: '' },
  studentName: { type: String, default: '' },
  className: { type: String, default: '' }
})
</script>

<template>
  <SettingsSection title="账户信息">
    <SettingsItem label="学号" :value="uid || '-'">
      <template #icon>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" /></svg>
      </template>
    </SettingsItem>
    <SettingsItem label="姓名" :value="studentName || '-'">
      <template #icon>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" /></svg>
      </template>
    </SettingsItem>
    <SettingsItem label="班级" :value="className || '-'">
      <template #icon>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" /><path d="M23 21v-2a4 4 0 0 0-3-3.87" /><path d="M16 3.13a4 4 0 0 1 0 7.75" /></svg>
      </template>
    </SettingsItem>
    <SettingsItem label="身份类型" value="学生">
      <template #icon>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z" /><polyline points="22,6 12,13 2,6" /></svg>
      </template>
    </SettingsItem>
    <SettingsItem label="登录状态">
      <template #icon>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path d="M7 11V7a5 5 0 0 1 10 0v4" /></svg>
      </template>
      <span class="item-value status-active">已认证</span>
    </SettingsItem>
  </SettingsSection>
</template>

<style scoped>
.status-active {
  color: #A16207;
  font-weight: 600;
}
</style>
