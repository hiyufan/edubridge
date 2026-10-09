<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../../stores/user'
import request from '../../utils/request'
import { relativeTime } from '../../utils/time'
import SettingsSection from './SettingsSection.vue'

const router = useRouter()
const userStore = useUserStore()

const status = ref(null)
const expanded = ref(new Set())

const eventLabels = {
  'schedule-diff': { label: '课表变动', color: '#007AFF' },
  'score-new': { label: '新成绩', color: '#34C759' },
  'session-expired': { label: '需重新登录', color: '#FF3B30' }
}

onMounted(async () => {
  try {
    status.value = (await request.get('/monitor/status')).data
  } catch {}
})

const toggle = (idx) => {
  const next = new Set(expanded.value)
  next.has(idx) ? next.delete(idx) : next.add(idx)
  expanded.value = next
}

const relogin = () => {
  if (status.value?.monitoring) return
  userStore.logout()
  router.push('/login')
}
</script>

<template>
  <SettingsSection v-if="status" title="课表监控">
    <div class="apple-grouped-item" :class="{ 'item-clickable': !status.monitoring }" @click="relogin">
      <div class="item-content channel-row">
        <div class="channel-head">
          <span class="item-label">监控状态</span>
          <span class="item-value" :style="{ color: status.monitoring ? '#34C759' : '#FF3B30' }">
            {{ status.monitoring ? '运行中' : '已暂停，点此重新登录' }}
          </span>
        </div>
      </div>
    </div>
    <div class="apple-grouped-item">
      <div class="item-content channel-row">
        <div class="channel-head">
          <span class="item-label">上次检查</span>
          <span class="item-value" :style="{ color: status.lastCheckError ? '#FF3B30' : '' }">
            {{ relativeTime(status.lastCheck) }}{{ status.lastCheck ? (status.lastCheckError ? ' ✗' : ' ✓') : '' }}
          </span>
        </div>
        <span v-if="status.lastCheckError" class="channel-hint">{{ status.lastCheckError }}</span>
        <span v-if="status.checkMinutes" class="channel-hint">每 {{ status.checkMinutes }} 分钟检查一次课表和成绩</span>
      </div>
    </div>
    <div v-if="!status.notifyChannel" class="apple-grouped-item">
      <div class="item-content">
        <span class="channel-hint" style="color: #FF9500">还没有开启通知方式，有变动时只能在这里看到</span>
      </div>
    </div>
    <div class="apple-grouped-item">
      <div class="item-content channel-row">
        <span class="item-label">最近动态</span>
        <span v-if="!status.history?.length" class="channel-hint">暂无。课表或成绩有变化时会记录在这里</span>
        <div v-for="(h, idx) in status.history" :key="idx" class="history-item" @click="toggle(idx)">
          <div class="history-head">
            <span class="history-tag" :style="{ color: eventLabels[h.event]?.color, borderColor: eventLabels[h.event]?.color }">
              {{ eventLabels[h.event]?.label || h.event }}
            </span>
            <span class="channel-hint">{{ relativeTime(h.time) }}</span>
          </div>
          <div class="history-text" :class="{ expanded: expanded.has(idx) }">{{ h.text }}</div>
        </div>
      </div>
    </div>
  </SettingsSection>
</template>

<style scoped>
.history-item {
  padding: 8px 0;
  border-top: 1px solid var(--el-border-color-lighter);
  cursor: pointer;
}

.history-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}

.history-tag {
  font-size: 11px;
  padding: 1px 6px;
  border: 1px solid;
  border-radius: 4px;
}

.history-text {
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.history-text.expanded {
  display: block;
}
</style>
