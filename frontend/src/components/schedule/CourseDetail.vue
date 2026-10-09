<script setup>
// 桌面端：选中课程的详情侧栏
import { DAY_NAMES } from '../../utils/schedule'

defineProps({
  course: { type: Object, required: true },
  hasNote: { type: Boolean, default: false }
})
defineEmits(['close', 'note'])
</script>

<template>
  <div class="course-detail-sidebar academic-card academic-fade-in stagger-3">
    <div class="detail-header">
      <h3 class="detail-title font-heading">{{ course.name }}</h3>
      <button class="detail-close" @click="$emit('close')">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
      </button>
    </div>
    <div class="detail-body">
      <div class="detail-item">
        <span class="detail-label">上课地点</span>
        <span class="detail-value">{{ course.room || '待定' }}</span>
      </div>
      <div class="detail-item">
        <span class="detail-label">授课教师</span>
        <span class="detail-value">{{ course.teacher || '待定' }}</span>
      </div>
      <div class="detail-item">
        <span class="detail-label">上课时间</span>
        <span class="detail-value">{{ DAY_NAMES[course.dayOfWeek - 1] }} 第{{ course.periodStart }}节</span>
      </div>
      <div v-if="course.periods > 1" class="detail-item">
        <span class="detail-label">课程节数</span>
        <span class="detail-value">{{ course.periods }}节连上</span>
      </div>
    </div>
    <div class="detail-actions">
      <button class="detail-note-btn" @click="$emit('note')">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" /><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" /></svg>
        {{ hasNote ? '查看备注' : '添加备注' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
/* Course Detail Sidebar */
.course-detail-sidebar {
  position: sticky;
  top: 24px;
  padding: 0;
}

.detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  border-bottom: 1px solid var(--color-border-light);
}

.detail-title {
  font-family: var(--font-heading);
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-dark);
  margin: 0;
}

.detail-close {
  width: 28px;
  height: 28px;
  border-radius: var(--radius-full);
  border: none;
  background: var(--color-bg);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
  transition: all 0.15s ease;
}

.detail-close:hover {
  background: var(--color-border-light);
  color: var(--color-text-dark);
}

.detail-close svg {
  width: 14px;
  height: 14px;
}

.detail-body {
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.detail-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-muted);
  letter-spacing: 0.05em;
}

.detail-value {
  font-size: 14px;
  color: var(--color-text-dark);
}

.detail-actions {
  padding: 16px 24px;
  border-top: 1px solid var(--color-border-light);
}

.detail-note-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px 16px;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  background: white;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: var(--color-accent);
  transition: all 0.15s ease;
}

.detail-note-btn:hover {
  background: rgba(34, 197, 94, 0.05);
  border-color: var(--color-accent);
}

.detail-note-btn svg {
  width: 16px;
  height: 16px;
}
</style>
