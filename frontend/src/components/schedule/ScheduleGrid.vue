<script setup>
// 桌面端：周课表网格（手机端隐藏）
import { DAY_NAMES, PERIODS, isWeekend, courseKey } from '../../utils/schedule'

defineProps({
  courses: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  colorOf: { type: Function, required: true },
  hasNote: { type: Function, required: true },
  conflicts: { type: Map, default: () => new Map() }
})
defineEmits(['select'])
</script>

<template>
  <div class="schedule-grid-card academic-card academic-fade-in stagger-2">
    <div v-if="loading" class="skeleton-overlay">
      <div class="sk-header">
        <div class="sk-corner"></div>
        <div v-for="day in DAY_NAMES" :key="day" class="sk-day-header"></div>
      </div>
      <div class="sk-body">
        <div v-for="p in PERIODS" :key="`sk-num-${p}`" class="sk-period-num"></div>
        <div v-for="p in PERIODS" :key="`sk-row-${p}`" class="sk-row">
          <div v-for="(day, i) in DAY_NAMES" :key="`sk-cell-${p}-${i}`" class="sk-cell"></div>
        </div>
      </div>
    </div>

    <div v-else class="schedule-body">
      <div class="schedule-corner" style="grid-column: 1; grid-row: 1"></div>
      <div
        v-for="(day, i) in DAY_NAMES"
        :key="`h-${i}`"
        class="day-header-cell font-heading"
        :class="{ 'is-weekend': isWeekend(day) }"
        :style="{ gridColumn: i + 2, gridRow: 1 }"
      >{{ day }}</div>

      <div v-for="p in PERIODS" :key="`row-${p}`" class="period-row" :style="{ gridColumn: '2 / -1', gridRow: p + 1 }">
        <div v-for="(day, i) in DAY_NAMES" :key="`cell-${p}-${i}`" class="day-cell" :class="{ 'is-weekend': isWeekend(day) }"></div>
      </div>

      <div v-for="p in PERIODS" :key="`label-${p}`" class="period-num-cell" :style="{ gridColumn: 1, gridRow: p + 1 }">
        <span class="period-num font-mono">{{ p }}</span>
      </div>

      <div
        v-for="course in courses"
        :key="courseKey(course)"
        class="course-card"
        :class="{ 'is-conflict': conflicts.has(courseKey(course)) }"
        :style="{
          gridColumn: course.dayOfWeek + 1,
          gridRow: `${course.periodStart + 1} / span ${course.periods}`,
          backgroundColor: colorOf(course.name).light,
          borderLeftColor: colorOf(course.name).bg
        }"
        @click="$emit('select', course)"
      >
        <span class="course-name">{{ course.name }}</span>
        <span class="course-room">{{ course.room }}</span>
        <span v-if="hasNote(course)" class="note-indicator" title="有备注">✎</span>
        <div class="course-preview">
          <div class="preview-content">
            <div class="preview-name">{{ course.name }}</div>
            <div class="preview-room">{{ course.room }}</div>
            <div class="preview-teacher">{{ course.teacher }}</div>
            <div v-if="conflicts.has(courseKey(course))" class="preview-conflict">⚠️ {{ conflicts.get(courseKey(course)) }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* =====================
   PC Schedule Grid — hidden on mobile
   ===================== */
.schedule-grid-card {
  display: none;
}
@media (min-width: 1024px) {
  .schedule-grid-card {
    display: block;
  }
}

/* =====================
   Schedule Grid Card
   ===================== */
.schedule-grid-card {
  padding: 0;
  overflow: visible;
  border-radius: var(--radius-lg);
}

/* Schedule Body — Mobile-first grid */
.schedule-body {
  display: grid;
  grid-template-columns: 40px repeat(7, minmax(44px, 1fr));
  grid-template-rows: 40px repeat(12, 56px);
  gap: 1px;
  background: var(--color-border-light);
  position: relative;
  overflow-x: auto;
  min-width: 580px; /* 保证7列在手机窄屏下不压扁 */
}

/* PC: Larger cells */
@media (min-width: 1024px) {
  .schedule-body {
    grid-template-columns: 52px repeat(7, minmax(0, 1fr));
    grid-template-rows: 44px repeat(12, 72px);
  }
}

/* Corner cell */
.schedule-corner {
  background: var(--color-bg);
}

/* Day header cells */
.day-header-cell {
  background: var(--color-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-muted);
  letter-spacing: 0.05em;
}

.day-header-cell.is-weekend {
  color: var(--color-amber);
}

/* Period rows */
.period-row {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 1px;
}

.day-cell {
  background: white;
}

.day-cell.is-weekend {
  background: rgba(241, 245, 249, 0.5);
}

/* Period number labels */
.period-num-cell {
  background: var(--color-bg);
  display: flex;
  align-items: center;
  justify-content: center;
}

.period-num {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-muted);
}

/* Course Cards */
.course-card {
  position: relative;
  border-left: 4px solid;
  border-radius: var(--radius-md);
  padding: 8px 10px;
  margin: 2px;
  cursor: pointer;
  transition: all 0.15s ease;
  overflow: hidden;
  z-index: 1;
}

.course-card:hover {
  transform: scale(1.02);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  z-index: 2;
}

.course-card.is-conflict {
  box-shadow: 0 0 0 2px var(--color-danger);
}

.course-name {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-dark);
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.course-room {
  display: block;
  font-size: 11px;
  color: var(--color-text-muted);
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.note-indicator {
  position: absolute;
  top: 6px;
  right: 8px;
  font-size: 12px;
  opacity: 0.5;
}

.course-preview {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: inherit;
  border-radius: var(--radius-md);
  opacity: 0;
  transition: opacity 0.15s ease;
  pointer-events: none;
  z-index: 10;
}

.course-card:hover .course-preview {
  opacity: 1;
  pointer-events: auto;
}

.preview-content {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.preview-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-dark);
}

.preview-room, .preview-teacher {
  font-size: 12px;
  color: var(--color-text-muted);
}

.preview-conflict {
  font-size: 11px;
  color: var(--color-danger);
  margin-top: 4px;
}

/* Skeleton */
.skeleton-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: white;
  z-index: 5;
  padding: 16px;
}

.sk-header {
  display: grid;
  grid-template-columns: 48px repeat(7, 1fr);
  gap: 1px;
  margin-bottom: 1px;
}

.sk-corner {
  background: var(--color-bg);
  height: 44px;
}

.sk-day-header {
  background: var(--color-bg);
  height: 44px;
  border-radius: var(--radius-sm);
}

.sk-body {
  display: grid;
  grid-template-columns: 48px repeat(7, 1fr);
  grid-template-rows: repeat(12, 64px);
  gap: 1px;
}

.sk-period-num {
  background: var(--color-bg);
  border-radius: var(--radius-sm);
}

.sk-row {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 1px;
}

.sk-cell {
  background: var(--color-bg);
  border-radius: var(--radius-sm);
  opacity: 0.5;
}

/* PC responsive */
@media (min-width: 1024px) {
  .schedule-body {
    grid-template-rows: 44px repeat(12, 72px);
  }
}
</style>
