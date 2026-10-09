<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '../../utils/request'
import SettingsSection from './SettingsSection.vue'
import SettingsItem from './SettingsItem.vue'

// 只有生成过订阅链接（在课表页）才显示
const info = ref(null)

onMounted(async () => {
  try {
    info.value = (await request.get('/schedule/ical/token-info')).data
  } catch {}
})

const regenerate = async () => {
  try {
    info.value = (await request.post('/schedule/ical/token')).data
    ElMessage.success('订阅链接已重新生成')
  } catch {
    ElMessage.error('生成失败')
  }
}
</script>

<template>
  <SettingsSection v-if="info" title="日历订阅">
    <SettingsItem label="订阅状态" :value="`已激活 · 有效期至 ${info.expireAt}`">
      <template #icon>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" /></svg>
      </template>
    </SettingsItem>
    <SettingsItem label="重新生成订阅链接" arrow @click="regenerate">
      <template #icon>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><polyline points="23 4 23 10 17 10" /><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" /></svg>
      </template>
    </SettingsItem>
  </SettingsSection>
</template>
