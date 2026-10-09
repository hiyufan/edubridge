<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '../../utils/request'
import SettingsSection from './SettingsSection.vue'
import ActionItem from './ActionItem.vue'

const reminder = ref({ daily: false, dailyTime: '07:00', dailyTomorrow: false, beforeClass: false, beforeMinutes: 15 })
const saving = ref(false)
const beforeOptions = [5, 10, 15, 20, 30, 60]

onMounted(async () => {
  try {
    const res = await request.get('/notify/reminder')
    if (res.data) reminder.value = { ...reminder.value, ...res.data }
  } catch {}
})

const save = async () => {
  saving.value = true
  try {
    await request.put('/notify/reminder', reminder.value)
    ElMessage.success('提醒设置已保存')
  } catch {
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <SettingsSection title="上课提醒">
    <div class="apple-grouped-item">
      <div class="item-content channel-row">
        <div class="channel-head">
          <span class="item-label">每日课表</span>
          <el-switch v-model="reminder.daily" size="small" />
        </div>
        <div v-if="reminder.daily" class="reminder-row">
          <span class="channel-hint">每天</span>
          <el-time-select v-model="reminder.dailyTime" start="05:00" step="00:30" end="23:30" size="small" :clearable="false" style="width: 110px" />
          <span class="channel-hint">推送</span>
          <el-radio-group v-model="reminder.dailyTomorrow" size="small">
            <el-radio-button :value="false">当天</el-radio-button>
            <el-radio-button :value="true">明天</el-radio-button>
          </el-radio-group>
          <span class="channel-hint">的课</span>
        </div>
      </div>
    </div>
    <div class="apple-grouped-item">
      <div class="item-content channel-row">
        <div class="channel-head">
          <span class="item-label">课前提醒</span>
          <el-switch v-model="reminder.beforeClass" size="small" />
        </div>
        <div v-if="reminder.beforeClass" class="reminder-row">
          <span class="channel-hint">每节课开始前</span>
          <el-select v-model="reminder.beforeMinutes" size="small" style="width: 90px">
            <el-option v-for="m in beforeOptions" :key="m" :label="`${m} 分钟`" :value="m" />
          </el-select>
        </div>
      </div>
    </div>
    <ActionItem text="保存提醒设置" busy-text="保存中…" :busy="saving" @click="save" />
  </SettingsSection>
</template>

<style scoped>
.reminder-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}
</style>
