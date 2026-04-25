<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElDialog, ElAlert, ElNotification } from 'element-plus'
import { useScheduleStore } from '../stores/schedule'
import { storeToRefs } from 'pinia'
import { generateICal, downloadICal } from '../utils/ical'
import { getNote, saveNote, hasNote } from '../utils/notes'
import request from '../utils/request'

const scheduleStore = useScheduleStore()
const { scheduleData, loading } = storeToRefs(scheduleStore)

const currentWeek = ref(1)
const maxWeek = ref(20)
const selectedCourse = ref(null)

const courseColors = [
  { bg: '#78716C', light: '#E7E2DA' }, { bg: '#A16207', light: '#FEF3C7' },
  { bg: '#15803D', light: '#DCFCE7' }, { bg: '#B45309', light: '#FEF3C7' },
  { bg: '#0369A1', light: '#E0F2FE' }, { bg: '#7C3AED', light: '#EDE9FE' },
  { bg: '#C2410C', light: '#FFEDD5' }, { bg: '#65A30D', light: '#ECFCCB' },
]

const courseColorMap = computed(() => {
  const m = new Map()
  if (!scheduleData.value?.courses) return m
  for (const c of scheduleData.value.courses) {
    if (!m.has(c.name)) {
      const hash = c.name.split('').reduce((a, ch) => a + ch.charCodeAt(0), 0)
      m.set(c.name, courseColors[hash % courseColors.length])
    }
  }
  return m
})

const getCourseColor = (name) => courseColorMap.value.get(name) || courseColors[0]

// Mobile: group courses by day of week
const coursesByDay = computed(() => {
  const map = {}
  for (const d of days) map[d] = []
  for (const course of (scheduleData.value?.courses || [])) {
    const dayName = days[course.dayOfWeek - 1]
    if (map[dayName]) map[dayName].push(course)
  }
  return map
})

const weeks = computed(() => Array.from({ length: maxWeek }, (_, i) => i + 1))
const days = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']
const periods = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]

const changeWeek = async (week) => {
  currentWeek.value = week
  try {
    await scheduleStore.fetchSchedule(week)
  } catch {
    ElMessage.error('获取课表失败')
  }
}

const noteCourse = ref(null)
const noteText = ref('')
const showNoteDialog = ref(false)

const openNoteDialog = (course) => {
  selectedCourse.value = course
  noteText.value = getNote(course.name, course.dayOfWeek, course.periodStart)
  showNoteDialog.value = true
}

const handleSaveNote = () => {
  if (!noteCourse.value) return
  saveNote(noteCourse.value.name, noteCourse.value.dayOfWeek, noteCourse.value.periodStart, noteText.value)
  showNoteDialog.value = false
  noteCourse.value = null
  ElMessage.success(noteText.value ? '备注已保存' : '备注已删除')
}

const handleDeleteNote = () => {
  noteText.value = ''
  handleSaveNote()
}

const courseHasNote = (course) => hasNote(course.name, course.dayOfWeek, course.periodStart)

const showICalDialog = ref(false)
const iCalToken = ref('')
const iCalUrl = ref('')
const iCalWebcal = ref('')
const iCalExpireAt = ref('')
const qrCanvas = ref(null)

const handleExportICal = () => {
  if (!scheduleData.value?.courses?.length) {
    ElMessage.warning('课表为空，无法导出')
    return
  }
  try {
    const content = generateICal(scheduleData.value, scheduleData.value.studentName || '')
    downloadICal(content, '课程表.ics')
    ElMessage.success('已导出日历文件')
  } catch {
    ElMessage.error('导出失败')
  }
}

const handleSubscribeICal = async () => {
  try {
    const res = await request.post('/schedule/ical/token')
    iCalToken.value = res.data.token
    iCalUrl.value = res.data.url
    iCalWebcal.value = res.data.webcal
    iCalExpireAt.value = res.data.expireAt
    showICalDialog.value = true
    setTimeout(() => {
      if (qrCanvas.value) generateQR(qrCanvas.value, iCalUrl.value)
    }, 100)
  } catch {
    ElMessage.error('获取订阅链接失败')
  }
}

const copyToClipboard = async (text, msg) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(msg || '已复制')
  } catch {
    ElMessage.warning('复制失败，请手动复制')
  }
}

function generateQR(canvas, text) {
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  const size = 200
  const cell = Math.floor(size / 25)
  canvas.width = size
  canvas.height = size
  ctx.fillStyle = '#fff'
  ctx.fillRect(0, 0, size, size)
  ctx.fillStyle = '#000'
  const modules = simpleQRMatrix(text)
  for (let r = 0; r < modules.length; r++) {
    for (let c = 0; c < modules[r].length; c++) {
      if (modules[r][c]) {
        ctx.fillRect(c * cell, r * cell, cell - 1, cell - 1)
      }
    }
  }
}

function simpleQRMatrix(text) {
  const size = 25
  const m = Array.from({ length: size }, () => Array(size).fill(false))
  const place = (x, y, w, h, v) => {
    for (let r = y; r < y + h && r < size; r++)
      for (let c = x; c < x + w && c < size; c++) m[r][c] = v
  }
  place(0, 0, 7, 7, true); place(1, 1, 5, 5, false); place(2, 2, 3, 3, true)
  place(size - 7, 0, 7, 7, true); place(size - 6, 1, 5, 5, false); place(size - 5, 2, 3, 3, true)
  place(0, size - 7, 7, 7, true); place(1, size - 6, 5, 5, false); place(2, size - 5, 3, 3, true)
  let hash = 0
  for (let i = 0; i < text.length; i++) hash = ((hash << 5) - hash + text.charCodeAt(i)) | 0
  const seed = Math.abs(hash)
  for (let r = 9; r < size - 9; r++) {
    for (let c = 9; c < size - 9; c++) {
      m[r][c] = ((seed >>> (r + c) % 32) & 1) === 1
    }
  }
  return m
}

const conflicts = ref([])

const isConflictCourse = (course) => {
  return conflicts.value.some(c => c.courseA === course.name || c.courseB === course.name)
}

const getConflictInfo = (course) => {
  const cf = conflicts.value.find(c => c.courseA === course.name || c.courseB === course.name)
  if (!cf) return ''
  const other = cf.courseA === course.name ? cf.courseB : cf.courseA
  return `与「${other}」在第 ${cf.conflictWeeks} 周存在时间冲突`
}

onMounted(async () => {
  try {
    const data = await scheduleStore.fetchSchedule()
    if (data?.currentWeek && data.currentWeek > 0) {
      currentWeek.value = data.currentWeek
    }
    try {
      const cfRes = await request.get('/schedule/conflicts')
      conflicts.value = cfRes.data || []
    } catch {}
    try {
      const diffRes = await request.get('/schedule/diff')
      const diff = diffRes.data
      if (diff && (diff.added?.length || diff.removed?.length || diff.changed?.length)) {
        const msgs = []
        if (diff.added?.length) msgs.push(`新增 ${diff.added.length} 门课程`)
        if (diff.removed?.length) msgs.push(`删除 ${diff.removed.length} 门课程`)
        if (diff.changed?.length) msgs.push(`变更 ${diff.changed.length} 处`)
        ElNotification({
          title: '课表有变动',
          message: msgs.join('、'),
          type: 'info',
          duration: 5000
        })
      }
    } catch {}
  } catch {
    ElMessage.error('获取课表失败')
  }
})
</script>

<template>
  <div class="schedule-page">
    <!-- Conflict Alert -->
    <ElAlert
      v-if="conflicts.length > 0"
      type="warning"
      :title="`检测到 ${conflicts.length} 处课程冲突`"
      :closable="false"
      show-icon
      class="conflict-alert"
    />

    <!-- PC Layout: Header with week selector and actions -->
    <div class="schedule-header-card academic-card academic-fade-in">
      <div class="header-top">
        <div class="student-info" v-if="scheduleData">
          <h1 class="student-name">{{ scheduleData.studentName }}</h1>
          <span class="class-name">{{ scheduleData.className }}</span>
        </div>
        <div class="header-actions">
          <button class="action-btn" @click="handleSubscribeICal" title="日历订阅">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>
            </svg>
            <span>订阅</span>
          </button>
          <button class="action-btn" @click="handleExportICal" title="导出日历">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/>
              <line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
            <span>导出</span>
          </button>
        </div>
      </div>

      <div class="week-selector-row">
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
          <span class="week-num font-heading">第 {{ currentWeek }} 周</span>
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
    </div>

    <!-- PC: Main grid with course detail sidebar -->
    <div class="schedule-content-grid">
      <!-- PC Schedule Grid (hidden on mobile) -->
      <div class="schedule-grid-card academic-card academic-fade-in stagger-2 schedule-grid-pc">
        <!-- Skeleton Loading -->
        <div v-if="loading" class="skeleton-overlay">
          <div class="sk-header">
            <div class="sk-corner"></div>
            <div v-for="day in days" :key="day" class="sk-day-header"></div>
          </div>
          <div class="sk-body">
            <div v-for="period in periods" :key="`sk-num-${period}`" class="sk-period-num"></div>
            <div v-for="period in periods" :key="`sk-row-${period}`" class="sk-row">
              <div v-for="(day, dayIdx) in days" :key="`sk-cell-${period}-${dayIdx}`" class="sk-cell"></div>
            </div>
          </div>
        </div>

        <!-- Real Content -->
        <template v-else>
          <div class="schedule-body">
            <!-- Header Row -->
            <div class="schedule-corner" style="gridColumn: 1; gridRow: 1;"></div>
            <div
              v-for="(day, dayIdx) in days"
              :key="`h-${dayIdx}`"
              class="day-header-cell font-heading"
              :class="{ 'is-weekend': day === '周六' || day === '周日' }"
              :style="{ gridColumn: dayIdx + 2, gridRow: 1 }"
            >
              {{ day }}
            </div>

            <!-- Period Rows background -->
            <div
              v-for="period in periods"
              :key="`row-${period}`"
              class="period-row"
              :style="{ gridColumn: '2 / -1', gridRow: period + 1 }"
            >
              <div
                v-for="(day, dayIdx) in days"
                :key="`cell-${period}-${dayIdx}`"
                class="day-cell"
                :class="{ 'is-weekend': day === '周六' || day === '周日' }"
              ></div>
            </div>

            <!-- Period number labels -->
            <div
              v-for="period in periods"
              :key="`label-${period}`"
              class="period-num-cell"
              :style="{ gridColumn: 1, gridRow: period + 1 }"
            >
              <span class="period-num font-mono">{{ period }}</span>
            </div>

            <!-- Course Cards -->
            <div
              v-for="course in scheduleData?.courses || []"
              :key="`${course.dayOfWeek}-${course.periodStart}`"
              class="course-card"
              :class="{ 'is-conflict': isConflictCourse(course) }"
              :style="{
                gridColumn: course.dayOfWeek + 1,
                gridRow: `${course.periodStart + 1} / span ${course.periods}`,
                backgroundColor: getCourseColor(course.name).light,
                borderLeftColor: getCourseColor(course.name).bg
              }"
              @click="openNoteDialog(course)"
            >
              <span class="course-name">{{ course.name }}</span>
              <span class="course-room">{{ course.room }}</span>
              <span v-if="courseHasNote(course)" class="note-indicator" title="有备注">✎</span>
              <div class="course-preview">
                <div class="preview-content">
                  <div class="preview-name">{{ course.name }}</div>
                  <div class="preview-room">{{ course.room }}</div>
                  <div class="preview-teacher">{{ course.teacher }}</div>
                  <div v-if="isConflictCourse(course)" class="preview-conflict">
                    ⚠️ {{ getConflictInfo(course) }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </template>
      </div>

      <!-- Mobile: Day-by-day list (hidden on PC) -->
      <div class="schedule-list-mobile academic-fade-in stagger-2">
        <!-- Skeleton -->
        <div v-if="loading" class="mobile-skeleton">
          <div v-for="i in 3" :key="i" class="mobile-day-skeleton"></div>
        </div>
        <template v-else>
          <div
            v-for="(courses, day) in coursesByDay"
            :key="day"
            class="mobile-day-section"
          >
            <div class="mobile-day-header" :class="{ weekend: day === '周六' || day === '周日' }">
              <span class="mobile-day-name">{{ day }}</span>
              <span class="mobile-day-count">{{ courses.length }}节课</span>
            </div>
            <div v-if="courses.length === 0" class="mobile-day-empty">无课</div>
            <div v-else class="mobile-course-list">
              <div
                v-for="course in courses"
                :key="`${course.dayOfWeek}-${course.periodStart}`"
                class="mobile-course-card"
                :style="{
                  borderLeftColor: getCourseColor(course.name).bg,
                  backgroundColor: getCourseColor(course.name).light
                }"
                @click="openNoteDialog(course)"
              >
                <div class="mobile-course-left">
                  <span class="mobile-course-period">第{{ course.periodStart }}节</span>
                  <span class="mobile-course-duration">{{ course.periods > 1 ? `连上${course.periods}节` : '' }}</span>
                </div>
                <div class="mobile-course-info">
                  <span class="mobile-course-name">{{ course.name }}</span>
                  <span class="mobile-course-room">{{ course.room || '待定' }}</span>
                </div>
                <span v-if="courseHasNote(course)" class="note-indicator">✎</span>
              </div>
            </div>
          </div>
        </template>
      </div>

      <!-- PC: Course Detail Sidebar -->
      <div class="course-detail-sidebar academic-card academic-fade-in stagger-3" v-if="selectedCourse">
        <div class="detail-header">
          <h3 class="detail-title font-heading">{{ selectedCourse.name }}</h3>
          <button class="detail-close" @click="selectedCourse = null">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"/>
              <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>
        <div class="detail-body">
          <div class="detail-item">
            <span class="detail-label">上课地点</span>
            <span class="detail-value">{{ selectedCourse.room || '待定' }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">授课教师</span>
            <span class="detail-value">{{ selectedCourse.teacher || '待定' }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">上课时间</span>
            <span class="detail-value">周{{ ['一','二','三','四','五','六','日'][selectedCourse.dayOfWeek - 1] }} 第{{ selectedCourse.periodStart }}节</span>
          </div>
          <div class="detail-item" v-if="selectedCourse.periods > 1">
            <span class="detail-label">课程节数</span>
            <span class="detail-value">{{ selectedCourse.periods }}节连上</span>
          </div>
        </div>
        <div class="detail-actions">
          <button class="detail-note-btn" @click="openNoteDialog(selectedCourse)">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
              <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
            </svg>
            {{ courseHasNote(selectedCourse) ? '查看备注' : '添加备注' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-if="!loading && scheduleData && scheduleData.courses.length === 0" class="empty-state academic-card academic-fade-in">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="empty-icon">
        <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
        <line x1="16" y1="2" x2="16" y2="6"/>
        <line x1="8" y1="2" x2="8" y2="6"/>
        <line x1="3" y1="10" x2="21" y2="10"/>
      </svg>
      <p>本周暂无课程安排</p>
    </div>

    <!-- Note Dialog -->
    <ElDialog
      v-model="showNoteDialog"
      :title="noteCourse ? '课程备注 - ' + noteCourse.name : '课程备注'"
      width="400px"
      :close-on-click-modal="false"
    >
      <div class="note-dialog-content">
        <div v-if="noteCourse" class="note-course-info">
          <span class="note-course-name">{{ noteCourse.name }}</span>
          <span class="note-course-detail">{{ noteCourse.room }} · 周{{ ['一','二','三','四','五','六','日'][noteCourse.dayOfWeek - 1] }} · 第{{ noteCourse.periodStart }}节</span>
        </div>
        <el-input
          v-model="noteText"
          type="textarea"
          :rows="4"
          placeholder="添加课程备注，如：需要带教材、实验课地点变更等"
          maxlength="200"
          show-word-limit
        />
      </div>
      <template #footer>
        <div class="note-dialog-footer">
          <el-button v-if="noteText" type="danger" plain @click="handleDeleteNote">删除备注</el-button>
          <el-button type="primary" @click="handleSaveNote">保存</el-button>
        </div>
      </template>
    </ElDialog>

    <!-- iCal Dialog -->
    <ElDialog v-model="showICalDialog" title="日历订阅" width="420px">
      <div class="ical-dialog-content">
        <p class="ical-hint">扫描下方二维码或在日历中添加以下订阅地址：</p>
        <div class="ical-qr-wrap">
          <canvas ref="qrCanvas" class="ical-qr-canvas"></canvas>
        </div>
        <div class="ical-links">
          <div class="ical-link-item">
            <span class="ical-link-label">HTTPS 订阅</span>
            <div class="ical-link-row">
              <el-input :value="iCalUrl" readonly size="small" />
              <el-button size="small" @click="copyToClipboard(iCalUrl, 'HTTPS链接已复制')">复制</el-button>
            </div>
          </div>
          <div class="ical-link-item">
            <span class="ical-link-label">WebCal 订阅</span>
            <div class="ical-link-row">
              <el-input :value="iCalWebcal" readonly size="small" />
              <el-button size="small" @click="copyToClipboard(iCalWebcal, 'WebCal链接已复制')">复制</el-button>
            </div>
          </div>
        </div>
        <p class="ical-expire">有效期至：{{ iCalExpireAt }}</p>
      </div>
    </ElDialog>
  </div>
</template>

<style scoped>
.schedule-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

/* Header Card */
.schedule-header-card {
  padding: 24px 28px;
}

.header-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.student-info {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.student-name {
  font-family: var(--font-heading);
  font-size: 24px;
  font-weight: 600;
  color: var(--color-text-dark);
  margin: 0;
}

.class-name {
  font-size: 14px;
  color: var(--color-text-muted);
}

.header-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  background: white;
  cursor: pointer;
  transition: all 0.15s ease;
  font-size: 13px;
  color: var(--color-text-muted);
}

.action-btn:hover {
  background: var(--color-bg);
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.action-btn svg {
  width: 16px;
  height: 16px;
}

/* Week Selector Row */
.week-selector-row {
  display: flex;
  align-items: center;
  gap: 16px;
}

.week-nav-btn {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border-light);
  background: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
  color: var(--color-text-muted);
}

.week-nav-btn:hover:not(:disabled) {
  background: var(--color-bg);
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.week-nav-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.week-nav-btn svg {
  width: 18px;
  height: 18px;
}

.week-display {
  min-width: 80px;
  text-align: center;
}

.week-num {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-dark);
}

.week-pills {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  margin-left: 8px;
}

.week-pill {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border-light);
  background: white;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-muted);
  transition: all 0.15s ease;
}

.week-pill:hover {
  background: var(--color-bg);
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.week-pill.active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: white;
}

/* Schedule Content Grid — PC */
.schedule-content-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 24px;
  align-items: start;
}

@media (min-width: 1200px) {
  .schedule-content-grid {
    grid-template-columns: 1fr 320px;
  }
}

@media (min-width: 1440px) {
  .schedule-content-grid {
    grid-template-columns: 1fr 340px;
  }
}

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

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 40px;
  text-align: center;
}

.empty-icon {
  width: 64px;
  height: 64px;
  color: var(--color-text-muted);
  margin-bottom: 16px;
  opacity: 0.4;
}

.empty-state p {
  font-size: 16px;
  color: var(--color-text-muted);
}

/* Dialog */
.note-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.note-course-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.note-course-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-dark);
}

.note-course-detail {
  font-size: 13px;
  color: var(--color-text-muted);
}

.note-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.ical-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.ical-hint {
  font-size: 14px;
  color: var(--color-text-muted);
  text-align: center;
}

.ical-qr-wrap {
  display: flex;
  justify-content: center;
  padding: 16px;
  background: white;
  border-radius: var(--radius-lg);
}

.ical-qr-canvas {
  width: 200px;
  height: 200px;
}

.ical-links {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.ical-link-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.ical-link-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-muted);
}

.ical-link-row {
  display: flex;
  gap: 8px;
}

.ical-expire {
  font-size: 12px;
  color: var(--color-text-muted);
  text-align: center;
}

.conflict-alert {
  border-radius: var(--radius-lg) !important;
}

/* PC responsive */
@media (min-width: 1024px) {
  .schedule-body {
    grid-template-rows: 44px repeat(12, 72px);
  }
}
</style>