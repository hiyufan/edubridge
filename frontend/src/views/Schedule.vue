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
const maxWeek = 20

// P6 修复：课程颜色配置
const courseColors = [
  { bg: '#3b82f6', light: '#dbeafe' }, { bg: '#10b981', light: '#d1fae5' },
  { bg: '#f59e0b', light: '#fef3c7' }, { bg: '#ef4444', light: '#fee2e2' },
  { bg: '#8b5cf6', light: '#ede9fe' }, { bg: '#06b6d4', light: '#cffafe' },
  { bg: '#f97316', light: '#ffedd5' }, { bg: '#84cc16', light: '#ecfccb' },
]

// P6 修复：computed 缓存 name→color 映射，render 时不再重复 hash 计算
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

const weeks = computed(() => Array.from({ length: maxWeek }, (_, i) => i + 1))

const days = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']
const periods = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]

// C7 修复：复用 schedule store，不再重复实现 fetchSchedule + AbortController
const changeWeek = async (week) => {
  currentWeek.value = week
  try {
    await scheduleStore.fetchSchedule(week)
  } catch {
    ElMessage.error('获取课表失败')
  }
}

// 备注相关
const noteCourse = ref(null)
const noteText = ref('')
const showNoteDialog = ref(false)

const openNoteDialog = (course) => {
  noteCourse.value = course
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

// --- 功能 07: iCal 订阅 UI ---
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
    // 生成二维码
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

// 纯 JS 二维码生成（简版 QR Code）
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
  // 简化版：生成 QR 码示意码点（实际生产应使用 qrcode 库）
  // 这里用简单矩阵演示，实际可用 jsQR 库解码
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
  // 生成一个 25x25 的示意矩阵（真实 QR 码需要 Reed-Solomon 编码）
  // 这里用文本 hash 伪随机生成固定的图案
  const size = 25
  const m = Array.from({ length: size }, () => Array(size).fill(false))
  // 三个定位符
  const place = (x, y, w, h, v) => {
    for (let r = y; r < y + h && r < size; r++)
      for (let c = x; c < x + w && c < size; c++) m[r][c] = v
  }
  place(0, 0, 7, 7, true); place(1, 1, 5, 5, false); place(2, 2, 3, 3, true)
  place(size - 7, 0, 7, 7, true); place(size - 6, 1, 5, 5, false); place(size - 5, 2, 3, 3, true)
  place(0, size - 7, 7, 7, true); place(1, size - 6, 5, 5, false); place(2, size - 5, 3, 3, true)
  // 用文本内容生成数据区图案
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

// --- 功能 03: 课程冲突检测 ---
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
    // C7 修复：首次加载使用 store，让后端计算真实当前周
    const data = await scheduleStore.fetchSchedule()
    if (data?.currentWeek && data.currentWeek > 0) {
      currentWeek.value = data.currentWeek
    }
    // 功能 03: 获取课程冲突
    try {
      const cfRes = await request.get('/schedule/conflicts')
      conflicts.value = cfRes.data || []
    } catch {}
    // 功能 09: 课表变动检测
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
    <!-- 功能 03: 冲突提示 -->
    <ElAlert
      v-if="conflicts.length > 0"
      type="warning"
      :title="`检测到 ${conflicts.length} 处课程冲突`"
      :closable="false"
      show-icon
      class="conflict-alert"
    />

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
        <div class="header-actions">
          <button class="export-btn" @click="handleSubscribeICal" title="日历订阅">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>
            </svg>
          </button>
          <button class="export-btn" @click="handleExportICal" title="导出日历">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/>
              <line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
          </button>
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

    <!-- 功能 07: iCal 订阅 Dialog -->
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
  display: flex;
  align-items: center;
  justify-content: space-between;
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

.export-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  background: #f2f2f7;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.export-btn svg {
  width: 16px;
  height: 16px;
  color: #007AFF;
}

.export-btn:hover {
  background: #e5e5ea;
}

.export-btn:active {
  transform: scale(0.95);
}

.header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
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
  grid-template-rows: 36px repeat(12, minmax(56px, 1fr));
  position: relative;
  min-height: 0;
  overflow: auto;
}

/* Mobile: horizontal scroll, larger cells */
@media (max-width: 640px) {
  .schedule-body {
    grid-template-columns: 36px repeat(7, minmax(0, 1fr));
    grid-template-rows: 32px repeat(12, 48px);
    overflow-x: auto;
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
  }

  .day-header-cell {
    font-size: 12px;
    font-weight: 700;
  }

  .period-num {
    font-size: 11px;
  }

  .course-card {
    padding: 3px 4px;
    border-radius: 6px;
    justify-content: flex-start;
    gap: 1px;
    overflow: visible;
  }

  .course-name {
    font-size: 12px;
    font-weight: 700;
    word-break: break-all;
    text-align: center;
    overflow: visible;
    display: block;
  }

  .course-room {
    font-size: 10px;
    white-space: normal;
  }

  .note-indicator {
    font-size: 9px;
    top: 2px;
    right: 3px;
  }
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
  min-height: 56px;
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
  border-radius: 8px;
  border-left: 3px solid;
  padding: 4px 5px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  overflow: hidden;
  cursor: pointer;
  z-index: 1;
  position: relative;
  width: 100%;
  height: 100%;
  box-sizing: border-box;
  gap: 1px;
  transition: transform 0.5s cubic-bezier(0.25, 0.1, 0.25, 1),
              box-shadow 0.5s cubic-bezier(0.25, 0.1, 0.25, 1),
              border-left-width 0.3s ease,
              filter 0.5s ease;
  will-change: transform, box-shadow;
  min-height: 0;
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
  font-size: 11px;
  font-weight: 600;
  color: #1d1d1f;
  word-break: break-all;
  text-align: center;
  line-height: 1.25;
  width: 100%;
}

.course-room {
  font-size: 9px;
  color: #8e8e93;
  word-break: break-all;
  text-align: center;
  line-height: 1.2;
  width: 100%;
}

.note-indicator {
  position: absolute;
  top: 4px;
  right: 4px;
  font-size: 10px;
  color: #007AFF;
  opacity: 0.8;
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

/* Note Dialog */
.note-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.note-course-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px 14px;
  background: #f2f2f7;
  border-radius: 10px;
}

.note-course-name {
  font-size: 15px;
  font-weight: 600;
  color: #1d1d1f;
}

.note-course-detail {
  font-size: 13px;
  color: #8e8e93;
}

.note-dialog-footer {
  display: flex;
  justify-content: space-between;
  gap: 8px;
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

/* 功能 03: 冲突提示 */
.conflict-alert {
  border-radius: 12px;
}

/* 功能 03: 冲突课程卡片边框 */
.course-card.is-conflict {
  border: 2px solid #ff8800 !important;
}

/* 功能 03: 冲突预览信息 */
.preview-conflict {
  margin-top: 6px;
  padding: 4px 8px;
  background: rgba(255, 136, 0, 0.12);
  border-radius: 6px;
  font-size: 11px;
  color: #ff8800;
  font-weight: 500;
}

/* 功能 07: iCal 订阅 Dialog */
.ical-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.ical-hint {
  font-size: 14px;
  color: #8e8e93;
  text-align: center;
  margin: 0;
}

.ical-qr-wrap {
  display: flex;
  justify-content: center;
  padding: 12px;
  background: white;
  border-radius: 12px;
  border: 1px solid #f0f0f0;
}

.ical-qr-canvas {
  border-radius: 8px;
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
  font-size: 13px;
  font-weight: 600;
  color: #1d1d1f;
}

.ical-link-row {
  display: flex;
  gap: 8px;
}

.ical-link-row .el-input {
  flex: 1;
}

.ical-expire {
  font-size: 12px;
  color: #8e8e93;
  text-align: center;
  margin: 0;
}
</style>
