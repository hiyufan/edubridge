<script setup>
// 手机端：按星期分组的课程列表（桌面端隐藏）
import { computed } from 'vue'
import { DAY_NAMES, isWeekend, courseKey } from '../../utils/schedule'

const props = defineProps({
  courses: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  colorOf: { type: Function, required: true },
  hasNote: { type: Function, required: true }
})
defineEmits(['select'])

const byDay = computed(() => {
  const map = Object.fromEntries(DAY_NAMES.map((d) => [d, []]))
  for (const c of props.courses) {
    const day = DAY_NAMES[c.dayOfWeek - 1]
    if (day) map[day].push(c)
  }
  return map
})
</script>

<template>
  <div class="schedule-list-mobile academic-fade-in stagger-2">
    <div v-if="loading" class="mobile-skeleton">
      <div v-for="i in 3" :key="i" class="mobile-day-skeleton"></div>
    </div>
    <template v-else>
      <div v-for="(list, day) in byDay" :key="day" class="mobile-day-section">
        <div class="mobile-day-header" :class="{ weekend: isWeekend(day) }">
          <span class="mobile-day-name">{{ day }}</span>
          <span class="mobile-day-count">{{ list.length }}节课</span>
        </div>
        <div v-if="list.length === 0" class="mobile-day-empty">无课</div>
        <div v-else class="mobile-course-list">
          <div
            v-for="course in list"
            :key="courseKey(course)"
            class="mobile-course-card"
            :style="{ borderLeftColor: colorOf(course.name).bg, backgroundColor: colorOf(course.name).light }"
            @click="$emit('select', course)"
          >
            <div class="mobile-course-left">
              <span class="mobile-course-period">第{{ course.periodStart }}节</span>
              <span class="mobile-course-duration">{{ course.periods > 1 ? `连上${course.periods}节` : '' }}</span>
            </div>
            <div class="mobile-course-info">
              <span class="mobile-course-name">{{ course.name }}</span>
              <span class="mobile-course-room">{{ course.room || '待定' }}</span>
            </div>
            <span v-if="hasNote(course)" class="note-indicator">✎</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
/* =====================
   Mobile List View — hidden on PC
   ===================== */
.schedule-list-mobile {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
@media (min-width: 1024px) {
  .schedule-list-mobile {
    display: none;
  }
}

.mobile-day-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.mobile-day-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 14px;
  background: white;
  border-radius: var(--radius-md);
  border-left: 4px solid var(--color-primary);
}
.mobile-day-header.weekend {
  border-left-color: var(--color-amber);
}

.mobile-day-name {
  font-family: var(--font-serif);
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text);
}
.mobile-day-count {
  font-size: 12px;
  color: var(--color-text-muted);
  font-weight: 500;
}
.mobile-day-empty {
  padding: 12px 14px;
  font-size: 13px;
  color: var(--color-text-muted);
  background: rgba(255,255,255,0.6);
  border-radius: var(--radius-md);
  text-align: center;
}
.mobile-course-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.mobile-course-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border-left: 4px solid;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.15s ease;
  position: relative;
}
.mobile-course-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);
}
.mobile-course-left {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 42px;
  flex-shrink: 0;
}
.mobile-course-period {
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text);
  font-family: var(--font-mono);
}
.mobile-course-duration {
  font-size: 10px;
  color: var(--color-text-muted);
  margin-top: 2px;
}
.mobile-course-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.mobile-course-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.mobile-course-room {
  font-size: 12px;
  color: var(--color-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.mobile-skeleton {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.mobile-day-skeleton {
  height: 80px;
  background: linear-gradient(90deg, #e7e2da 25%, #f7f4ef 50%, #e7e2da 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: var(--radius-lg);
}

.note-indicator {
  position: absolute;
  top: 6px;
  right: 8px;
  font-size: 12px;
  opacity: 0.5;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
</style>
