<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getPeriodTime } from '../utils/periods'
import { useScheduleStore } from '../stores/schedule'

const scheduleStore = useScheduleStore()
const todayDate = ref('')
const todayWeekday = ref('')

const days = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

// VUE-7 修复：今日课程需要自己请求真实当前周数据，不复用 Schedule 缓存
const todayCourses = ref([])
const todayLoading = ref(true)
const todayScheduleData = ref(null)

const fetchTodayData = async () => {
  todayLoading.value = true
  try {
    const data = await scheduleStore.fetchSchedule()
    todayScheduleData.value = data || scheduleStore.scheduleData
    const weekday = new Date().getDay() // 0=周日, 1=周一...
    const dayOfWeek = weekday === 0 ? 7 : weekday
    todayCourses.value = (todayScheduleData.value?.courses || [])
      .filter(c => c.dayOfWeek === dayOfWeek)
      .sort((a, b) => a.periodStart - b.periodStart)
  } catch {
    todayCourses.value = []
  } finally {
    todayLoading.value = false
  }
}

// 无课提示
const freeMessage = computed(() => {
  const count = todayCourses.value.length
  if (todayLoading.value) return ''
  if (count === 0) return '今日无课，好好休息 🎉'
  return `今日共 ${count} 节课`
})

// 课程颜色
const courseColors = [
  { bg: '#78716C', light: 'rgba(120, 113, 108, 0.1)' },
  { bg: '#A16207', light: 'rgba(161, 98, 7, 0.1)' },
  { bg: '#15803D', light: 'rgba(21, 128, 61, 0.1)' },
  { bg: '#B45309', light: 'rgba(180, 83, 9, 0.1)' },
  { bg: '#0369A1', light: 'rgba(3, 105, 161, 0.1)' },
  { bg: '#7C3AED', light: 'rgba(124, 58, 237, 0.1)' },
]

// P6 修复：computed 缓存 name→color 映射，render 时不再重复 hash 计算
const courseColorMap = computed(() => {
  const m = new Map()
  const courses = todayScheduleData.value?.courses
  if (!courses) return m
  for (const c of courses) {
    if (!m.has(c.name)) {
      const hash = c.name.split('').reduce((a, ch) => a + ch.charCodeAt(0), 0)
      m.set(c.name, courseColors[hash % courseColors.length])
    }
  }
  return m
})

const getCourseColor = (name) => courseColorMap.value.get(name) || courseColors[0]

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
  fetchTodayData()
})
</script>

<template>
  <div class="today-page">
    <!-- Date Header -->
    <div class="date-header animate-warm-fade-in">
      <div class="date-main">
        <span class="date-month-day">{{ todayDate }}</span>
        <span class="date-weekday" :class="{ weekend: todayWeekday === '周六' || todayWeekday === '周日' }">
          {{ todayWeekday }}
        </span>
      </div>
      <div class="date-semester" v-if="todayScheduleData">
        <span>{{ todayScheduleData.semester }}</span>
        <span class="date-dot">·</span>
        <span>第 {{ todayScheduleData.currentWeek }} 周</span>
      </div>
      <div class="date-semester" v-else>
        <span class="loading-dots">加载中</span>
      </div>
    </div>

    <!-- Course Summary -->
    <div class="summary-card animate-warm-fade-in stagger-1">
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
    <div v-if="todayLoading" class="courses-list">
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
        class="course-card animate-warm-fade-in"
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
    <div v-else class="empty-today animate-warm-fade-in stagger-2">
      <div class="empty-emoji">☀️</div>
      <div class="empty-text">今日无课安排</div>
      <div class="empty-sub">好好享受难得的空闲时间吧</div>
    </div>

    <!-- Week Pills -->
    <div class="week-nav animate-warm-fade-in stagger-3">
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
  background: var(--color-primary);
  border-radius: var(--radius-lg);
  padding: 24px 24px 20px;
  color: white;
  position: relative;
  overflow: hidden;
}

.date-header::before {
  content: '';
  position: absolute;
  top: -40px;
  right: -20px;
  width: 160px;
  height: 160px;
  background: rgba(255, 255, 255, 0.04);
  border-radius: 50%;
}

.date-header::after {
  content: '';
  position: absolute;
  bottom: -30px;
  right: 60px;
  width: 100px;
  height: 100px;
  background: rgba(255, 255, 255, 0.03);
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
  color: #FCD34D;
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
  background: rgba(120, 113, 108, 0.1);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
}

.summary-icon svg {
  width: 20px;
  height: 20px;
  color: var(--color-primary);
}

.summary-count {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text);
  letter-spacing: -0.02em;
  line-height: 1;
}

.summary-label {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-top: 2px;
}

.summary-tip {
  font-size: 13px;
  color: var(--color-text-muted);
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
  color: var(--color-text);
}

.time-divider {
  width: 1px;
  flex: 1;
  background: var(--color-border);
  min-height: 12px;
}

.time-end {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text);
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
  font-weight: 600;
  color: var(--color-text);
  letter-spacing: 0;
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
  color: var(--color-text-muted);
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
  color: var(--color-primary);
  background: rgba(120, 113, 108, 0.08);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
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
  color: var(--color-primary);
  text-decoration: none;
  padding: 10px 16px;
  border-radius: var(--radius-md);
  background: white;
  border: 1px solid var(--color-border);
  transition: all 0.15s ease;
}

.week-link:hover {
  background: var(--color-bg);
  border-color: var(--color-primary);
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

/* =====================
   PC Responsive (lg+)
   ===================== */
@media (min-width: 1024px) {
  .today-page {
    gap: 28px;
    max-width: 1100px;
  }

  /* Header expands wider */
  .date-header {
    border-radius: var(--radius-xl);
    padding: 36px 40px 32px;
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
  }

  .date-header::before {
    width: 260px;
    height: 260px;
    top: -80px;
    right: -40px;
  }

  .date-header::after {
    width: 160px;
    height: 160px;
    bottom: -60px;
    right: 120px;
  }

  .date-main {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
    z-index: 1;
  }

  .date-month-day {
    font-size: 52px;
  }

  .date-weekday {
    font-size: 20px;
  }

  .date-semester {
    font-size: 15px;
    z-index: 1;
  }

  /* Summary card — large horizontal dashboard card */
  .summary-card {
    padding: 24px 32px;
    border-radius: var(--radius-xl);
  }

  .summary-icon {
    width: 52px;
    height: 52px;
    border-radius: var(--radius-lg);
  }

  .summary-icon svg {
    width: 26px;
    height: 26px;
  }

  .summary-count {
    font-size: 44px;
  }

  .summary-label {
    font-size: 14px;
    margin-top: 4px;
  }

  .summary-tip {
    font-size: 15px;
  }

  /* Courses — 2-column grid on PC */
  .courses-list {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 16px;
  }

  .course-card {
    padding: 20px;
    border-radius: var(--radius-xl);
    border-left-width: 5px;
  }

  .course-time-col {
    min-width: 52px;
  }

  .time-start, .time-end {
    font-size: 15px;
  }

  .course-name {
    font-size: 17px;
  }

  .meta-item {
    font-size: 14px;
  }

  .meta-item svg {
    width: 14px;
    height: 14px;
  }

  .period-badge {
    font-size: 32px;
    top: 16px;
    right: 18px;
  }

  .scope-tag {
    font-size: 12px;
    padding: 3px 10px;
  }

  .week-link {
    padding: 12px 24px;
    font-size: 15px;
    border-radius: var(--radius-lg);
  }

  .week-link svg {
    width: 18px;
    height: 18px;
  }

  /* Skeleton also 2-col */
  .courses-list:has(.course-card-skeleton) {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
