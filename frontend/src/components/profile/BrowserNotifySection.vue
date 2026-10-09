<script setup>
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { getNotifyPermission, requestNotifyPermission } from '../../utils/notifications'
import SettingsSection from './SettingsSection.vue'
import SettingsItem from './SettingsItem.vue'

// 浏览器本地的课程提醒（页面打开时生效），与服务端推送互不影响
const permission = ref(getNotifyPermission())
const label = computed(() =>
  permission.value === 'granted' ? '已开启' : permission.value === 'denied' ? '已被拒绝' : '未开启'
)

const handleClick = async () => {
  if (permission.value === 'granted') {
    ElMessage.info('已开启课程提醒')
    return
  }
  if (permission.value === 'denied') {
    ElMessage.warning('通知已被浏览器拒绝，请在设置中开启')
    return
  }
  permission.value = await requestNotifyPermission()
  if (permission.value === 'granted') {
    ElMessage.success('已开启课程提醒')
  }
}
</script>

<template>
  <SettingsSection title="通知">
    <SettingsItem label="课程提醒" :value="label" arrow @click="handleClick">
      <template #icon>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" /><path d="M13.73 21a2 2 0 0 1-3.46 0" /></svg>
      </template>
    </SettingsItem>
  </SettingsSection>
</template>
