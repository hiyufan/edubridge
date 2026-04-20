<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '../utils/request'

const currentWeek = ref(1)
const maxWeek = 20
const scheduleData = ref(null)
const loading = ref(false)

const weeks = computed(() => Array.from({ length: maxWeek }, (_, i) => i + 1))

const days = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']

const periods = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]

// 课程颜色配置 - Apple 风格
const courseColors = [
  { bg: '#007AFF', light: 'rgba(0, 122, 255, 0.15)' },
  { bg: '#34C759', light: 'rgba(52, 199, 89, 0.15)' },
  { bg: '#FF9500', light: 'rgba(255, 149, 0, 0.15)' },
  { bg: '#AF52DE', light: 'rgba(175, 82, 222, 0.15)' },
  { bg: '#FF2D55', light: 'rgba(255, 45, 85, 0.15)' },
  { bg: '#5AC8FA', light: 'rgba(90, 200, 250, 0.15)' },
  { bg: '#FF3B30', light: 'rgba(255, 59, 48, 0.15)' },
  { bg: '#5856D6', light: 'rgba(88, 86, 214, 0.15)' }
]

// 根据课程名称获取固定颜色
const getCourseColor = (name) => {
  if (!name) return courseColors[0]
  const hash = name.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0)
  return courseColors[hash % courseColors.length]
}

const fetchSchedule = async (week) => {
  loading.value = true
  try {
    const url = week ? `/schedule?week=${week}` : '/schedule'
    const res = await request.get(url)
    scheduleData.value = res.data
    // 仅在首次加载（后端自动判断当前周）时使用后端返回的周数
    // 切换周数时不再覆盖，保持 UI 和 URL 一致
    if (!week && res.data.currentWeek && res.data.currentWeek > 0) {
      currentWeek.value = res.data.currentWeek
    }
  } catch (error) {
    ElMessage.error('获取课表失败')
  } finally {
    loading.value = false
  }
}

const changeWeek = (week) => {
  currentWeek.value = week
  fetchSchedule(week)
}

onMounted(() => {
  // 首次加载：让后端计算真实当前周，不传 week 参数
  fetchSchedule()
})
</script>

<template>
  <div class="schedule-page">
    <!-- Week Selector Card -->
    <div class="week-card animate-apple-fade-in">
      <div class="week-header">
        <div class="week-info" v-if="scheduleData">
          <span class="week-student">{{ scheduleData.studentName }}</span>
          <span class="week-divider">·</span>
          <span class="week-class">{{ scheduleData.className }}</span>
        </div>
        <div class="week-info" v-else>
          <span class="week-loading">加载中...</span>
        </div>
      </div>

      <div class="week-selector">
        <button
          class="week-nav-btn"
          :disabled="currentWeek <= 1"
          @click="changeWeek(Math.max(1, currentWeek - 1))"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="15 18 9 12 15 6"/>
          </svg>
        </button>

        <div class="week-display">
          <span class="week-num">第 {{ currentWeek }} 周</span>
        </div>

        <button
          class="week-nav-btn"
          :disabled="currentWeek >= maxWeek"
          @click="changeWeek(Math.min(maxWeek, currentWeek + 1))"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"/>
          </svg>
        </button>
      </div>

      <!-- Week Pills -->
      <div class="week-pills">
        <button
          v-for="w in weeks"
          :key="w"
          class="week-pill"
          :class="{ active: currentWeek === w }"
          @click="changeWeek(w)"
        >
          {{ w }}
        </button>
      </div>
    </div>

    <!-- Schedule Grid -->
    <div class="schedule-card animate-apple-fade-in stagger-2">
      <!-- Skeleton Loading (shown inside the card body) -->
      <div v-if="loading" class="skeleton-overlay">
        <div class="sk-header">
          <div class="sk-corner"></div>
          <div v-for="day in days" :key="day" class="sk-day-header"></div>
        </div>
        <div class="sk-body">
          <div
            v-for="period in periods"
            :key="`sk-num-${period}`"
            class="sk-period-num"
            :style="{ gridColumn: 1, gridRow: period + 1 }"
          ></div>
          <div
            v-for="period in periods"
            :key="`sk-row-${period}`"
            class="sk-period-row"
            :style="{ gridColumn: '2 / -1', gridRow: period + 1 }"
          >
            <div
              v-for="(day, dayIdx) in days"
              :key="`sk-cell-${period}-${dayIdx}`"
              class="sk-cell"
              :class="{ 'is-weekend': day === '周六' || day === '周日' }"
            ></div>
          </div>
        </div>
      </div>

      <!-- Real Content -->
      <template v-else>
        <!-- Schedule Body: flat grid — header row + period rows, NO nesting -->
        <div class="schedule-body">
          <!-- Header Row (row 1): corner + day names -->
          <div class="schedule-corner" style="gridColumn: 1; gridRow: 1;"></div>
          <div
            v-for="(day, dayIdx) in days"
            :key="`h-${dayIdx}`"
            class="day-header-cell"
            :class="{ 'is-weekend': day === '周六' || day === '周日' }"
            :style="{ gridColumn: dayIdx + 2, gridRow: 1 }"
          >
            {{ day }}
          </div>

          <!-- Period Rows (rows 2-13): period number + day cells + course cards -->
          <div
            v-for="period in periods"
            :key="`row-${period}`"
            class="period-row"
            :style="{ gridColumn: '2 / -1', gridRow: period + 1 }"
          >
            <!-- Day cells as background stripes -->
            <div
              v-for="(day, dayIdx) in days"
              :key="`cell-${period}-${dayIdx}`"
              class="day-cell"
              :class="{ 'is-weekend': day === '周六' || day === '周日' }"
            ></div>
          </div>

          <!-- Period number labels (col 1, rows 2-13) -->
          <div
            v-for="period in periods"
            :key="`label-${period}`"
            class="period-num-cell"
            :style="{ gridColumn: 1, gridRow: period + 1 }"
          >
            <span class="period-num">{{ period }}</span>
          </div>

          <!-- Course Cards: flat in grid, each in its exact column + row span -->
          <div
            v-for="course in scheduleData?.courses || []"
            :key="`${course.dayOfWeek}-${course.periodStart}`"
            class="course-card"
            :style="{
              gridColumn: course.dayOfWeek + 1,
              gridRow: `${course.periodStart + 1} / span ${course.periods}`,
              backgroundColor: getCourseColor(course.name).light,
              borderLeftColor: getCourseColor(course.name).bg
            }"
          >
            <span class="course-name">{{ course.name }}</span>
            <span class="course-room">{{ course.room }}</span>
            <div class="course-preview">
              <div class="preview-content">
                <div class="preview-name">{{ course.name }}</div>
                <div class="preview-room">{{ course.room }}</div>
                <div class="preview-teacher">{{ course.teacher }}</div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>

    <!-- Empty State -->
    <div v-if="!loading && scheduleData && scheduleData.courses.length === 0" class="empty-state">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="empty-icon">
        <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
        <line x1="16" y1="2" x2="16" y2="6"/>
        <line x1="8" y1="2" x2="8" y2="6"/>
        <line x1="3" y1="10" x2="21" y2="10"/>
      </svg>
      <p>本周暂无课程安排</p>
    </div>
  </div>
</template>

<style scoped>
.schedule-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* Week Card */
.week-card {
  background: white;
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04), 0 4px 12px rgba(0, 0, 0, 0.03);
}

.week-header {
  margin-bottom: 16px;
}

.week-info {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 15px;
}

.week-student {
  font-weight: 600;
  color: #1d1d1f;
}

.week-divider {
  color: #c7c7cc;
}

.week-class {
  color: #8e8e93;
}

.week-loading {
  color: #8e8e93;
}

.week-selector {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 24px;
  margin-bottom: 20px;
}

.week-nav-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: none;
  background: #f2f2f7;
  border-radius: 50%;
  cursor: pointer;
  transition: all 0.2s ease;
}

.week-nav-btn svg {
  width: 18px;
  height: 18px;
  color: #007AFF;
}

.week-nav-btn:hover:not(:disabled) {
  background: #e5e5ea;
}

.week-nav-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.week-display {
  min-width: 100px;
  text-align: center;
}

.week-num {
  font-size: 20px;
  font-weight: 700;
  color: #1d1d1f;
  letter-spacing: -0.023em;
}

.week-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
}

.week-pill {
  min-width: 32px;
  height: 32px;
  padding: 0 10px;
  border: none;
  background: #f2f2f7;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  color: #636366;
  cursor: pointer;
  transition: all 0.2s ease;
}

.week-pill:hover:not(.active) {
  background: #e5e5ea;
}

.week-pill.active {
  background: #007AFF;
  color: white;
  font-weight: 600;
}

/* Schedule Card */
.schedule-card {
  background: white;
  border-radius: 16px;
  overflow: visible;
  position: relative;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04), 0 4px 12px rgba(0, 0, 0, 0.03);
  isolation: isolate;
}

/* Schedule Body: ONE flat grid — no nesting */
.schedule-body {
  display: grid;
  grid-template-columns: 44px repeat(7, minmax(0, 1fr));
  grid-template-rows: 44px repeat(12, 64px);
  position: relative;
}

/* Header Row (row 1) */
.schedule-corner {
  grid-column: 1;
  grid-row: 1;
  background: #fafafa;
  border-right: 0.5px solid rgba(0, 0, 0, 0.05);
  border-bottom: 0.5px solid rgba(0, 0, 0, 0.05);
}

.day-header-cell {
  grid-row: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: #8e8e93;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  background: #fafafa;
  border-bottom: 0.5px solid rgba(0, 0, 0, 0.05);
  border-right: 0.5px solid rgba(0, 0, 0, 0.05);
}

.day-header-cell:last-of-type {
  border-right: none;
}

.day-header-cell.is-weekend {
  color: #ff9500;
}

/* Period Rows (rows 2-13): flex row of day cells */
.period-row {
  grid-column: 2 / -1;
  display: flex;
  border-bottom: 0.5px solid rgba(0, 0, 0, 0.05);
}

.day-cell {
  flex: 1;
  border-right: 0.5px solid rgba(0, 0, 0, 0.05);
  min-height: 64px;
}

.day-cell:last-child {
  border-right: none;
}

.day-cell.is-weekend {
  background: rgba(255, 149, 0, 0.03);
}

/* Period Number Labels (col 1, rows 2-13) */
.period-num-cell {
  grid-column: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  border-bottom: 0.5px solid rgba(0, 0, 0, 0.05);
  border-right: 0.5px solid rgba(0, 0, 0, 0.05);
  background: #fafafa;
}

.period-num {
  font-size: 13px;
  font-weight: 600;
  color: #8e8e93;
}

/* Course Cards: in the flat grid */
.course-card {
  border-radius: 10px;
  border-left: 3px solid;
  padding: 6px 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  overflow: visible;
  cursor: pointer;
  z-index: 1;
  position: relative;
  width: 100%;
  height: 100%;
  box-sizing: border-box;
  transition: transform 0.5s cubic-bezier(0.25, 0.1, 0.25, 1),
              box-shadow 0.5s cubic-bezier(0.25, 0.1, 0.25, 1),
              border-left-width 0.3s ease,
              filter 0.5s ease;
  will-change: transform, box-shadow;
}

.course-card:hover {
  transform: scale(1.03);
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.14), 0 6px 16px rgba(0, 0, 0, 0.06);
  z-index: 100;
  border-left-width: 4px;
  filter: brightness(1.02);
}

.course-card:active {
  transform: scale(1.01);
}

/* Preview popup - Apple signature spring animation */
.course-preview {
  position: absolute;
  left: 50%;
  bottom: calc(100% + 14px);
  transform: translateX(-50%) scale(0.8) translateY(12px);
  opacity: 0;
  pointer-events: none;
  transition: transform 0.55s cubic-bezier(0.34, 1.56, 0.64, 1),
              opacity 0.4s cubic-bezier(0.25, 0.1, 0.25, 1);
  z-index: 200;
  min-width: 200px;
  max-width: 260px;
  will-change: transform, opacity;
}

.course-card:hover .course-preview {
  transform: translateX(-50%) scale(1) translateY(0);
  opacity: 1;
  pointer-events: auto;
  z-index: 300;
}

.preview-content {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(24px) saturate(180%);
  -webkit-backdrop-filter: blur(24px) saturate(180%);
  border-radius: 16px;
  padding: 16px 18px;
  box-shadow: 0 24px 64px rgba(0, 0, 0, 0.18), 0 8px 24px rgba(0, 0, 0, 0.08),
              0 0 0 0.5px rgba(0, 0, 0, 0.08);
}

.preview-name {
  font-size: 16px;
  font-weight: 700;
  color: #1d1d1f;
  margin-bottom: 8px;
  letter-spacing: -0.022em;
  line-height: 1.25;
}

.preview-room {
  font-size: 14px;
  color: #007AFF;
  font-weight: 500;
  margin-bottom: 4px;
  letter-spacing: -0.01em;
}

.preview-teacher {
  font-size: 13px;
  color: #8e8e93;
  letter-spacing: -0.005em;
}

/* Arrow - Apple frosted glass triangle */
.course-preview::after {
  content: '';
  position: absolute;
  left: 50%;
  bottom: -7px;
  transform: translateX(-50%) rotate(45deg);
  width: 14px;
  height: 14px;
  background: rgba(255, 255, 255, 0.95);
  box-shadow: 3px 3px 6px rgba(0, 0, 0, 0.06);
  backdrop-filter: blur(4px);
}

.course-name {
  font-size: 12px;
  font-weight: 600;
  color: #1d1d1f;
  word-wrap: break-word;
  word-break: break-word;
  hyphens: auto;
  text-align: center;
  line-height: 1.3;
}

.course-room {
  font-size: 10px;
  color: #8e8e93;
  word-wrap: break-word;
  word-break: break-word;
  hyphens: auto;
  text-align: center;
  line-height: 1.3;
  margin-top: 2px;
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: #8e8e93;
}

.empty-icon {
  width: 64px;
  height: 64px;
  margin-bottom: 16px;
  opacity: 0.5;
}

.empty-state p {
  font-size: 15px;
}

/* Skeleton Loading */
.skeleton-overlay {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 10;
  border-radius: 16px;
  background: white;
  display: flex;
  flex-direction: column;
}

.sk-header {
  display: grid;
  grid-template-columns: 44px repeat(7, minmax(0, 1fr));
  grid-template-rows: 44px;
  flex-shrink: 0;
}

.sk-corner {
  background: #fafafa;
  border-right: 0.5px solid rgba(0, 0, 0, 0.05);
  border-bottom: 0.5px solid rgba(0, 0, 0, 0.05);
}

.sk-day-header {
  background: #fafafa;
  border-bottom: 0.5px solid rgba(0, 0, 0, 0.05);
  border-right: 0.5px solid rgba(0, 0, 0, 0.05);
}

.sk-body {
  display: grid;
  grid-template-columns: 44px repeat(7, minmax(0, 1fr));
  grid-template-rows: repeat(12, 64px);
  flex: 1;
}

.sk-period-num {
  grid-column: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fafafa;
  border-bottom: 0.5px solid rgba(0, 0, 0, 0.05);
  border-right: 0.5px solid rgba(0, 0, 0, 0.05);
}

.sk-period-num::after {
  content: '';
  width: 20px;
  height: 20px;
  border-radius: 4px;
  background: linear-gradient(90deg, #e8e8e8 25%, #f5f5f5 50%, #e8e8e8 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

.sk-period-row {
  grid-column: 2 / -1;
  display: flex;
  border-bottom: 0.5px solid rgba(0, 0, 0, 0.05);
}

.sk-cell {
  flex: 1;
  border-right: 0.5px solid rgba(0, 0, 0, 0.05);
  margin: 4px;
  border-radius: 8px;
  background: linear-gradient(90deg, #f0f0f0 25%, #f8f8f8 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

.sk-cell:last-child {
  border-right: none;
}

.sk-cell.is-weekend {
  background: linear-gradient(90deg, rgba(255, 149, 0, 0.05) 25%, rgba(255, 149, 0, 0.08) 50%, rgba(255, 149, 0, 0.05) 75%);
  background-size: 200% 100%;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
</style>
