<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '../../utils/request'
import SettingsSection from './SettingsSection.vue'
import SettingsItem from './SettingsItem.vue'
import ActionItem from './ActionItem.vue'

const url = ref('')
const secret = ref('')

onMounted(async () => {
  try {
    const data = (await request.get('/webhook/info')).data
    url.value = data?.url || ''
    secret.value = data?.secret || ''
  } catch {}
})

const save = async () => {
  if (!url.value) {
    ElMessage.warning('请输入 Webhook URL')
    return
  }
  try {
    await request.post('/webhook/register', { url: url.value, secret: secret.value })
    ElMessage.success('Webhook 配置已保存')
  } catch {}
}
</script>

<template>
  <SettingsSection title="Webhook 推送">
    <SettingsItem label="Webhook URL">
      <template #icon>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" /><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" /></svg>
      </template>
      <el-input v-model="url" placeholder="https://example.com/webhook" size="small" class="field" />
    </SettingsItem>
    <SettingsItem label="密钥（可选）">
      <template #icon>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path d="M7 11V7a5 5 0 0 1 10 0v4" /></svg>
      </template>
      <el-input v-model="secret" placeholder="签名密钥" size="small" class="field" />
    </SettingsItem>
    <ActionItem text="保存 Webhook 配置" @click="save">
      <template #icon>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" /><polyline points="17 21 17 13 7 13 7 21" /><polyline points="7 3 7 8 15 8" /></svg>
      </template>
    </ActionItem>
  </SettingsSection>
</template>

<style scoped>
.field {
  margin-top: 8px;
}
</style>
