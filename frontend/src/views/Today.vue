<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '../utils/request'
import { getPeriodTime, getCourseTimeRange } from '../utils/periods'

const scheduleData = ref(null)
const loading = ref(false)
const todayDate = ref('')
const todayWeekday = ref('')

const days = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

// 课程颜色
const courseColors = [
  { bg: '#007AFF', light: 'rgba(0, 122, 255, 0.15)', text: '#007AFF' },
  { bg: '#34C759', light: 'rgba(52, 199, 89, 0.15)', text: '#34C759' },
  { bg: '#FF9500', light: 'rgba(255, 149, 0, 0.15)', text: '#FF9500' },
  { bg: '#AF52DE', light: 'rgba(175, 82, 222, 0.15)', text: '#AF52DE' },
  { bg: '#FF2D55', light: 'rgba(255, 45, 85, 0.15)', text: '#FF2D55' },
  { bg: '#5AC8FA', light: 'rgba(90, 200, 250, 0.15)', text: '#5AC8FA' },
]

const getCourseColor = (name) => {
  if (!name) return courseColors[0]
  const hash = name.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0)
  return courseColors[hash % courseColors.length]
}

// 按节次排序的今日课程
const todayCourses = computed(() => {
  if (!scheduleData.value?.courses) return []
  const weekday = new Date().getDay() // 0=周日, 1=周一...
  const dayOfWeek = weekday === 0 ? 7 : weekday // 转换为1=周一...7=周日
  return scheduleData.value.courses
    .filter(c => c.dayOfWeek === dayOfWeek)
    .sort((a, b) => a.periodStart - b.periodStart)
})

// 无课提示
const freeMessage = computed(() => {
  if (!scheduleData.value?.courses) return ''
  const weekday = new Date().getDay()
  const dayOfWeek = weekday === 0 ? 7 : weekday
  const count = scheduleData.value.courses.filter(c => c.dayOfWeek === dayOfWeek).length
  if (count === 0) return '今日无课，好好休息 🎉'
  return `今日共 ${count} 节课`
})

const fetchTodaySchedule = async () => {
  loading.value = true
  try {
    const res = await request.get('/schedule')
    scheduleData.value = res.data
  } catch (error) {
    ElMessage.error('获取课表失败')
  } finally {
    loading.value = false
  }
}

const formatDate = () => {
  const now = new Date()
  const month = now.getMonth() + 1
  const date = now.getDate()
  const weekday = days[now.getDay()]
  todayDate.value = `${month}月${date}日`
  todayWeekday.value = weekday
}

onMounted(() => {
  formatDate()
  fetchTodaySchedule()
})
</script>

<template>
  <div class="today-page">
    <!-- Date Header -->
    <div class="date-header animate-apple-fade-in">
      <div class="date-main">
        <span class="date-month-day">{{ todayDate }}</span>
        <span class="date-weekday" :class="{ weekend: todayWeekday === '周六' || todayWeekday === '周日' }">
          {{ todayWeekday }}
        </span>
      </div>
      <div class="date-semester" v-if="scheduleData">
        <span>{{ scheduleData.semester }}</span>
        <span class="date-dot">·</span>
        <span>第 {{ scheduleData.currentWeek }} 周</span>
      </div>
      <div class="date-semester" v-else>
        <span class="loading-dots">加载中</span>
      </div>
    </div>

    <!-- Course Summary -->
    <div class="summary-card animate-apple-fade-in stagger-1">
      <div class="summary-left">
        <div class="summary-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
            <path d="M8 2v4M16 2v4M3 10h18M5 4h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z"/>
          </svg>
        </div>
        <div class="summary-info">
          <div class="summary-count">{{ todayCourses.length }}</div>
          <div class="summary-label">今日课程</div>
        </div>
      </div>
      <div class="summary-tip">{{ freeMessage }}</div>
    </div>

    <!-- Skeleton -->
    <div v-if="loading" class="courses-list">
      <div v-for="i in 3" :key="i" class="course-card-skeleton">
        <div class="sk-line sk-time"></div>
        <div class="sk-card"></div>
      </div>
    </div>

    <!-- Course List -->
    <div v-else-if="todayCourses.length > 0" class="courses-list">
      <div
        v-for="(course, idx) in todayCourses"
        :key="`${course.dayOfWeek}-${course.periodStart}`"
        class="course-card animate-apple-fade-in"
        :class="`stagger-${idx + 2}`"
        :style="{
          backgroundColor: getCourseColor(course.name).light,
          borderLeftColor: getCourseColor(course.name).bg
        }"
      >
        <!-- Time column -->
        <div class="course-time-col">
          <div class="time-start">{{ getPeriodTime(course.periodStart).start }}</div>
          <div class="time-divider"></div>
          <div class="time-end">{{ getPeriodTime(course.periodStart + course.periods - 1).end }}</div>
        </div>

        <!-- Info column -->
        <div class="course-info-col">
          <div class="course-name">{{ course.name }}</div>
          <div class="course-meta">
            <span class="meta-item">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
                <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/>
                <circle cx="12" cy="10" r="3"/>
              </svg>
              {{ course.room || '待定' }}
            </span>
            <span class="meta-item">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
                <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                <circle cx="12" cy="7" r="4"/>
              </svg>
              {{ course.teacher || '待定' }}
            </span>
          </div>
          <div class="course-scope">
            <span class="scope-tag">{{ getPeriodTime(course.periodStart).start }} – {{ getPeriodTime(course.periodStart + course.periods - 1).end }}</span>
            <span class="scope-tag">第{{ course.periodStart }}–{{ course.periodStart + course.periods - 1 }}节</span>
          </div>
        </div>

        <!-- Period Badge -->
        <div class="period-badge" :style="{ color: getCourseColor(course.name).bg }">
          {{ course.periodStart }}
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else class="empty-today animate-apple-fade-in stagger-2">
      <div class="empty-emoji">☀️</div>
      <div class="empty-text">今日无课安排</div>
      <div class="empty-sub">好好享受难得的空闲时间吧</div>
    </div>

    <!-- Week Pills -->
    <div class="week-nav animate-apple-fade-in stagger-3">
      <router-link to="/schedule" class="week-link">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
          <rect x="3" y="4" width="18" height="18" rx="2"/>
          <path d="M8 2v4M16 2v4M3 10h18"/>
        </svg>
        查看完整课表
      </router-link>
    </div>
  </div>
</template>

<style scoped>
.today-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* Date Header */
.date-header {
  background: linear-gradient(135deg, #007AFF 0%, #5856D6 100%);
  border-radius: 20px;
  padding: 24px 24px 20px;
  color: white;
  position: relative;
  overflow: hidden;
}

.date-header::before {
  content: '';
  position: absolute;
  top: -30px;
  right: -30px;
  width: 120px;
  height: 120px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 50%;
}

.date-header::after {
  content: '';
  position: absolute;
  bottom: -20px;
  right: 40px;
  width: 80px;
  height: 80px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 50%;
}

.date-main {
  display: flex;
  align-items: baseline;
  gap: 10px;
  margin-bottom: 6px;
  position: relative;
  z-index: 1;
}

.date-month-day {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.03em;
  line-height: 1;
}

.date-weekday {
  font-size: 16px;
  font-weight: 500;
  opacity: 0.85;
}

.date-weekday.weekend {
  color: #FFD60A;
}

.date-semester {
  font-size: 13px;
  opacity: 0.7;
  display: flex;
  align-items: center;
  gap: 6px;
  position: relative;
  z-index: 1;
}

.date-dot {
  opacity: 0.5;
}

/* Summary Card */
.summary-card {
  background: white;
  border-radius: 16px;
  padding: 16px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04), 0 4px 12px rgba(0, 0, 0, 0.03);
}

.summary-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.summary-icon {
  width: 40px;
  height: 40px;
  background: rgba(0, 122, 255, 0.1);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.summary-icon svg {
  width: 20px;
  height: 20px;
  color: #007AFF;
}

.summary-count {
  font-size: 28px;
  font-weight: 700;
  color: #1d1d1f;
  letter-spacing: -0.03em;
  line-height: 1;
}

.summary-label {
  font-size: 12px;
  color: #8e8e93;
  margin-top: 2px;
}

.summary-tip {
  font-size: 13px;
  color: #8e8e93;
  text-align: right;
}

/* Course Cards */
.courses-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.course-card {
  background: white;
  border-radius: 16px;
  border-left: 4px solid;
  padding: 16px;
  display: flex;
  align-items: stretch;
  gap: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04), 0 4px 12px rgba(0, 0, 0, 0.03);
  position: relative;
}

.course-time-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-width: 44px;
  gap: 4px;
}

.time-start {
  font-size: 13px;
  font-weight: 600;
  color: #1d1d1f;
}

.time-divider {
  width: 1px;
  flex: 1;
  background: rgba(0, 0, 0, 0.1);
  min-height: 12px;
}

.time-end {
  font-size: 13px;
  font-weight: 600;
  color: #1d1d1f;
}

.course-info-col {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
  justify-content: center;
}

.course-name {
  font-size: 16px;
  font-weight: 700;
  color: #1d1d1f;
  letter-spacing: -0.02em;
}

.course-meta {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #8e8e93;
}

.meta-item svg {
  width: 13px;
  height: 13px;
}

.course-scope {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.scope-tag {
  font-size: 11px;
  font-weight: 500;
  color: #007AFF;
  background: rgba(0, 122, 255, 0.08);
  padding: 2px 8px;
  border-radius: 6px;
}

.period-badge {
  position: absolute;
  top: 12px;
  right: 14px;
  font-size: 22px;
  font-weight: 800;
  opacity: 0.15;
  letter-spacing: -0.03em;
}

/* Empty State */
.empty-today {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 20px;
  background: white;
  border-radius: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04), 0 4px 12px rgba(0, 0, 0, 0.03);
}

.empty-emoji {
  font-size: 48px;
  margin-bottom: 12px;
}

.empty-text {
  font-size: 17px;
  font-weight: 600;
  color: #1d1d1f;
  margin-bottom: 6px;
}

.empty-sub {
  font-size: 14px;
  color: #8e8e93;
}

/* Week Nav */
.week-nav {
  display: flex;
  justify-content: center;
}

.week-link {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 500;
  color: #007AFF;
  text-decoration: none;
  padding: 10px 16px;
  border-radius: 10px;
  background: rgba(0, 122, 255, 0.08);
  transition: background 0.2s ease;
}

.week-link:hover {
  background: rgba(0, 122, 255, 0.15);
}

.week-link svg {
  width: 16px;
  height: 16px;
}

/* Skeleton */
.course-card-skeleton {
  display: flex;
  gap: 16px;
  background: white;
  border-radius: 16px;
  padding: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.sk-line {
  background: linear-gradient(90deg, #f0f0f0 25%, #f8f8f8 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 4px;
}

.sk-time {
  width: 40px;
  height: 60px;
  flex-shrink: 0;
}

.sk-card {
  flex: 1;
  height: 60px;
  background: linear-gradient(90deg, #f0f0f0 25%, #f8f8f8 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 8px;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
</style>
