<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElDialog } from 'element-plus'
import { useScheduleStore } from '../stores/schedule'
import { storeToRefs } from 'pinia'
import { getNote, saveNote, hasNote } from '../utils/notes'
import request from '../utils/request'

const scheduleStore = useScheduleStore()
const { scheduleData, loading } = storeToRefs(scheduleStore)
const scheduleTable = ref(null)

const currentWeek = ref(1)
const maxWeek = 20

const CELL_HEIGHT = 52
const CELL_WIDTH = '12.5%'
const HEADER_HEIGHT = 36
const CORNER_WIDTH = 48

const courseColors = [
  '#1677FF', '#52C41A', '#FAAD14', '#FF4D4F', '#722ED1', '#13C2C2', '#FA8C16', '#EB2F96'
]

const courseColorMap = computed(() => {
  const m = new Map()
  if (!scheduleData.value?.courses) return m
  let colorIdx = 0
  for (const c of scheduleData.value.courses) {
    if (!m.has(c.name)) {
      m.set(c.name, courseColors[colorIdx % courseColors.length])
      colorIdx++
    }
  }
  return m
})

const weeks = computed(() => Array.from({ length: maxWeek }, (_, i) => i + 1))
const days = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']
const periods = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]

const getCourseStyle = (course) => {
  const col = course.dayOfWeek + 1
  const row = course.periodStart + 1
  const rowSpan = course.periods
  return {
    gridColumn: `${col} / span 1`,
    gridRow: `${row} / span ${rowSpan}`
  }
}

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

const courseHasNote = (course) => hasNote(course.name, course.dayOfWeek, course.periodStart)

onMounted(async () => {
  try {
    const data = await scheduleStore.fetchSchedule()
    if (data?.currentWeek && data.currentWeek > 0) {
      currentWeek.value = data.currentWeek
    }
  } catch {
    ElMessage.error('获取课表失败')
  }
})
</script>

<template>
  <div class="schedule-page">
    <div class="card week-card">
      <div class="week-info" v-if="scheduleData">
        <span class="student-name">{{ scheduleData.studentName }}</span>
        <span class="class-name">{{ scheduleData.className }}</span>
      </div>
      <div class="week-info" v-else>
        <span class="loading-text">加载中...</span>
      </div>

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

    <div class="card schedule-card">
      <div v-if="loading" class="loading-state">
        <span>加载中...</span>
      </div>

      <template v-else>
        <div class="schedule-table">
          <div class="grid-corner"></div>
          <div v-for="(day, idx) in days" :key="idx" class="grid-day-header">{{ day }}</div>
          <template v-for="period in periods" :key="period">
            <div class="grid-period">{{ period }}</div>
            <div v-for="(day, dayIdx) in days" :key="dayIdx" class="grid-cell"></div>
          </template>
          <div
            v-for="course in scheduleData?.courses || []"
            :key="`course-${course.dayOfWeek}-${course.periodStart}`"
            class="course-cell"
            :style="getCourseStyle(course)"
            @click="openNoteDialog(course)"
          >
            <span class="course-name" :style="{ color: courseColorMap.get(course.name) }">{{ course.name }}</span>
            <span class="course-room">{{ course.room }}</span>
            <span v-if="courseHasNote(course)" class="note-mark">*</span>
          </div>
        </div>
      </template>
    </div>

    <div v-if="!loading && scheduleData && scheduleData.courses.length === 0" class="empty-state">
      <p>本周暂无课程安排</p>
    </div>

    <ElDialog
      v-model="showNoteDialog"
      :title="noteCourse ? '课程备注 - ' + noteCourse.name : '课程备注'"
      width="420px"
    >
      <div class="note-content">
        <div v-if="noteCourse" class="note-info">
          <span class="note-title">{{ noteCourse.name }}</span>
          <span class="note-detail">{{ noteCourse.room }} · 周{{ ['一','二','三','四','五','六','日'][noteCourse.dayOfWeek - 1] }} · 第{{ noteCourse.periodStart }}节</span>
        </div>
        <textarea
          v-model="noteText"
          class="note-textarea"
          rows="4"
          placeholder="添加课程备注"
        ></textarea>
      </div>
      <template #footer>
        <button class="btn btn-default" @click="showNoteDialog = false">取消</button>
        <button class="btn btn-primary" @click="handleSaveNote">保存</button>
      </template>
    </ElDialog>
  </div>
</template>

<style scoped>
.schedule-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.card {
  background: #FFFFFF;
  border-radius: 8px;
  border: 1px solid #E8E8E8;
}

.week-card {
  padding: 20px;
}

.week-info {
  margin-bottom: 20px;
  display: flex;
  gap: 12px;
  align-items: center;
}

.student-name {
  font-weight: 600;
  color: #1F1F1F;
}

.class-name {
  color: #666;
  font-size: 13px;
}

.loading-text {
  color: #999;
  font-size: 13px;
}

.week-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.week-pill {
  min-width: 32px;
  height: 32px;
  padding: 0 10px;
  border: 1px solid #E8E8E8;
  background: #FFFFFF;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  color: #666;
  cursor: pointer;
  transition: all 150ms ease;
}

.week-pill:hover {
  border-color: #1677FF;
  color: #1677FF;
}

.week-pill.active {
  background: #1677FF;
  border-color: #1677FF;
  color: #FFFFFF;
}

.schedule-card {
  overflow: hidden;
}

.loading-state {
  padding: 60px;
  text-align: center;
  color: #666;
}

.schedule-table {
  display: grid;
  grid-template-columns: 48px repeat(7, minmax(0, 1fr));
  grid-template-rows: 36px repeat(12, 52px);
}

.grid-corner {
  background: #F7F8FA;
  border-right: 1px solid #E8E8E8;
  border-bottom: 1px solid #E8E8E8;
}

.grid-day-header {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 500;
  color: #666;
  background: #F7F8FA;
  border-bottom: 1px solid #E8E8E8;
  border-right: 1px solid #E8E8E8;
}

.grid-period {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: #999;
  background: #F7F8FA;
  border-right: 1px solid #E8E8E8;
  border-bottom: 1px solid #E8E8E8;
}

.grid-cell {
  border-right: 1px solid #E8E8E8;
  border-bottom: 1px solid #E8E8E8;
}

.course-cell {
  background: #E6F4FF;
  border-radius: 4px;
  padding: 4px 6px;
  margin: 2px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  justify-content: center;
  overflow: hidden;
  transition: all 150ms ease;
  border-left: 3px solid #1677FF;
}

.course-cell:hover {
  background: #BAE0FF;
  box-shadow: 0 2px 8px rgba(22, 119, 255, 0.2);
}

.course-name {
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.course-room {
  font-size: 10px;
  color: #666;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.note-mark {
  position: absolute;
  top: 2px;
  right: 4px;
  font-size: 10px;
  color: #1677FF;
}

.empty-state {
  padding: 60px;
  text-align: center;
  color: #666;
}

.note-content {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.note-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 14px;
  background: #F7F8FA;
  border-radius: 6px;
}

.note-title {
  font-size: 14px;
  font-weight: 600;
  color: #1F1F1F;
}

.note-detail {
  font-size: 12px;
  color: #666;
}

.note-textarea {
  width: 100%;
  padding: 10px 12px;
  font-size: 14px;
  font-family: inherit;
  border: 1px solid #E8E8E8;
  border-radius: 6px;
  resize: none;
  outline: none;
  transition: border-color 150ms ease;
}

.note-textarea:focus {
  border-color: #1677FF;
}
</style>
