<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { storeToRefs } from 'pinia'
import { useScheduleStore } from '../stores/schedule'
import { useCourseColors } from '../composables/courseColors'
import { generateICal, downloadICal } from '../utils/ical'
import { hasNote } from '../utils/notes'
import { conflictMap, notifyScheduleDiffOnce } from '../utils/schedule'
import request from '../utils/request'
import ScheduleHeader from '../components/schedule/ScheduleHeader.vue'
import ScheduleGrid from '../components/schedule/ScheduleGrid.vue'
import ScheduleDayList from '../components/schedule/ScheduleDayList.vue'
import CourseDetail from '../components/schedule/CourseDetail.vue'
import CourseNoteDialog from '../components/schedule/CourseNoteDialog.vue'
import ICalSubscribeDialog from '../components/schedule/ICalSubscribeDialog.vue'

const MAX_WEEK = 20

const scheduleStore = useScheduleStore()
const { scheduleData, loading } = storeToRefs(scheduleStore)
const courses = computed(() => scheduleData.value?.courses || [])
const colorOf = useCourseColors(courses)

const currentWeek = ref(1)
const conflicts = ref(new Map())

const changeWeek = async (week) => {
  currentWeek.value = week
  try {
    await scheduleStore.fetchSchedule(week)
  } catch {}
}

// 课程详情与备注（备注存在本机，noteVersion 变化时刷新“有备注”标记）
const selectedCourse = ref(null)
const showNote = ref(false)
const noteVersion = ref(0)
const courseHasNote = (c) => noteVersion.value >= 0 && hasNote(c.name, c.dayOfWeek, c.periodStart)
const openNote = (course) => {
  selectedCourse.value = course
  showNote.value = true
}

// 日历：导出 .ics 文件 / 生成订阅链接
const ical = ref({ show: false, url: '', webcal: '', expireAt: '' })

const exportICal = () => {
  if (!courses.value.length) {
    ElMessage.warning('课表为空，无法导出')
    return
  }
  try {
    downloadICal(generateICal(scheduleData.value, scheduleData.value.studentName || ''), '课程表.ics')
    ElMessage.success('已导出日历文件')
  } catch {
    ElMessage.error('导出失败')
  }
}

const subscribeICal = async () => {
  try {
    const { url, webcal, expireAt } = (await request.post('/schedule/ical/token')).data
    ical.value = { show: true, url, webcal, expireAt }
  } catch {}
}

onMounted(async () => {
  try {
    const data = await scheduleStore.fetchSchedule()
    if (data?.currentWeek > 0) currentWeek.value = data.currentWeek
  } catch {
    return
  }
  request.get('/schedule/conflicts').then((res) => { conflicts.value = conflictMap(res.data) }).catch(() => {})
  notifyScheduleDiffOnce().catch(() => {})
})
</script>

<template>
  <div class="schedule-page">
    <el-alert
      v-if="conflicts.size > 0"
      type="warning"
      :title="`检测到 ${conflicts.size} 门课程存在时间冲突`"
      :closable="false"
      show-icon
      class="conflict-alert"
    />

    <ScheduleHeader
      :student-name="scheduleData?.studentName"
      :class-name="scheduleData?.className"
      :week="currentWeek"
      :max-week="MAX_WEEK"
      @update:week="changeWeek"
      @subscribe="subscribeICal"
      @export="exportICal"
    />

    <div class="schedule-content-grid">
      <ScheduleGrid
        :courses="courses"
        :loading="loading"
        :color-of="colorOf"
        :has-note="courseHasNote"
        :conflicts="conflicts"
        @select="openNote"
      />
      <ScheduleDayList :courses="courses" :loading="loading" :color-of="colorOf" :has-note="courseHasNote" @select="openNote" />
      <CourseDetail
        v-if="selectedCourse"
        :course="selectedCourse"
        :has-note="courseHasNote(selectedCourse)"
        @close="selectedCourse = null"
        @note="showNote = true"
      />
    </div>

    <div v-if="!loading && scheduleData && courses.length === 0" class="empty-state academic-card academic-fade-in">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="empty-icon">
        <rect x="3" y="4" width="18" height="18" rx="2" ry="2" /><line x1="16" y1="2" x2="16" y2="6" /><line x1="8" y1="2" x2="8" y2="6" /><line x1="3" y1="10" x2="21" y2="10" />
      </svg>
      <p>本周暂无课程安排</p>
    </div>

    <CourseNoteDialog v-model="showNote" :course="selectedCourse" @saved="noteVersion++" />
    <ICalSubscribeDialog v-model="ical.show" :url="ical.url" :webcal="ical.webcal" :expire-at="ical.expireAt" />
  </div>
</template>

<style scoped>
.schedule-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
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

.conflict-alert {
  border-radius: var(--radius-lg) !important;
}
</style>
