<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '../utils/request'

const allScores = ref([])
const selectedSemester = ref('')
const loading = ref(false)

const semesterListFromScores = computed(() => {
  const semesters = [...new Set(allScores.value.map(s => `${s.year}-${s.semester}`))]
  return semesters.sort()
})

const scoreList = computed(() => {
  if (!selectedSemester.value) return allScores.value
  return allScores.value.filter(s => `${s.year}-${s.semester}` === selectedSemester.value)
})

const currentIndex = computed(() => {
  return semesterListFromScores.value.indexOf(selectedSemester.value)
})

const prevSemester = () => {
  if (currentIndex.value > 0) {
    selectedSemester.value = semesterListFromScores.value[currentIndex.value - 1]
  }
}

const nextSemester = () => {
  if (currentIndex.value < semesterListFromScores.value.length - 1) {
    selectedSemester.value = semesterListFromScores.value[currentIndex.value + 1]
  }
}

const fetchAllScores = async () => {
  loading.value = true
  try {
    const res = await request.get('/score')
    allScores.value = res.data
  } catch (error) {
    ElMessage.error('获取成绩失败')
  } finally {
    loading.value = false
  }
}

const gpaDetail = computed(() => {
  const validScores = scoreList.value.filter(s => s.gpa > 0 && s.credit > 0)
  if (validScores.length === 0) {
    return { weighted: '0.00', totalCredits: '0.0' }
  }

  const totalCredits = validScores.reduce((sum, s) => sum + s.credit, 0)
  const weightedSum = validScores.reduce((sum, s) => sum + s.gpa * s.credit, 0)

  return {
    weighted: (weightedSum / totalCredits).toFixed(2),
    totalCredits: totalCredits.toFixed(1)
  }
})

const getGradeColor = (grade) => {
  const num = parseFloat(grade)
  if (isNaN(num)) return '#999'
  if (num >= 90) return '#52C41A'
  if (num >= 80) return '#1677FF'
  if (num >= 70) return '#FAAD14'
  if (num >= 60) return '#FA8C16'
  return '#FF4D4F'
}

onMounted(async () => {
  await fetchAllScores()
  if (semesterListFromScores.value.length > 0) {
    selectedSemester.value = semesterListFromScores.value[0]
  }
})
</script>

<template>
  <div class="score-page">
    <div class="card stats-card">
      <div class="stat-item">
        <span class="stat-value">{{ gpaDetail.weighted }}</span>
        <span class="stat-label">加权 GPA</span>
      </div>
      <div class="stat-divider"></div>
      <div class="stat-item">
        <span class="stat-value">{{ scoreList.length }}</span>
        <span class="stat-label">课程数</span>
      </div>
      <div class="stat-divider"></div>
      <div class="stat-item">
        <span class="stat-value">{{ gpaDetail.totalCredits }}</span>
        <span class="stat-label">总学分</span>
      </div>
    </div>

    <div v-if="semesterListFromScores.length > 1" class="card semester-card">
      <div class="semester-picker">
        <button class="nav-btn" :disabled="currentIndex <= 0" @click="prevSemester">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>
        </button>
        <span class="semester-label">{{ selectedSemester }}</span>
        <button class="nav-btn" :disabled="currentIndex >= semesterListFromScores.length - 1" @click="nextSemester">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>
        </button>
      </div>
    </div>

    <div v-loading="loading" class="score-list">
      <template v-if="loading">
        <div v-for="i in 5" :key="i" class="card score-item">
          <div class="skeleton-line w-60"></div>
          <div class="skeleton-line w-40"></div>
        </div>
      </template>

      <template v-else>
        <div
          v-for="(score, idx) in scoreList"
          :key="idx"
          class="card score-item"
        >
          <div class="score-info">
            <span class="score-name">{{ score.course }}</span>
            <span class="score-meta">
              <span>{{ score.credit }}学分</span>
              <span v-if="score.nature" class="score-tag">{{ score.nature }}</span>
            </span>
          </div>
          <div class="score-grade">
            <span class="grade-value" :style="{ color: getGradeColor(score.grade) }">
              {{ score.grade || '-' }}
            </span>
            <span class="grade-gpa">GPA {{ Number(score.gpa).toFixed(1) }}</span>
          </div>
        </div>
      </template>

      <div v-if="scoreList.length === 0 && !loading" class="empty-state">
        <p>暂无成绩数据</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.score-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.card {
  background: #FFFFFF;
  border-radius: 8px;
  border: 1px solid #E8E8E8;
}

.stats-card {
  display: flex;
  align-items: center;
  padding: 24px;
}

.stat-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}

.stat-value {
  font-size: 32px;
  font-weight: 600;
  color: #1F1F1F;
}

.stat-label {
  font-size: 13px;
  color: #666;
}

.stat-divider {
  width: 1px;
  height: 40px;
  background: #E8E8E8;
}

.semester-card {
  padding: 14px 20px;
}

.semester-picker {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
}

.nav-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid #E8E8E8;
  background: #FFFFFF;
  border-radius: 6px;
  cursor: pointer;
  transition: all 150ms ease;
}

.nav-btn:hover:not(:disabled) {
  border-color: #1677FF;
  color: #1677FF;
}

.nav-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.nav-btn svg {
  width: 16px;
  height: 16px;
  color: #666;
}

.nav-btn:hover:not(:disabled) svg {
  color: #1677FF;
}

.semester-label {
  font-size: 14px;
  font-weight: 500;
  color: #1F1F1F;
  min-width: 100px;
  text-align: center;
}

.score-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.score-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  transition: all 150ms ease;
}

.score-item:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.score-info {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.score-name {
  font-size: 15px;
  font-weight: 500;
  color: #1F1F1F;
}

.score-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #666;
}

.score-tag {
  padding: 2px 6px;
  background: #F7F8FA;
  border-radius: 4px;
  font-size: 11px;
}

.score-grade {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
}

.grade-value {
  font-size: 20px;
  font-weight: 600;
}

.grade-gpa {
  font-size: 12px;
  color: #999;
}

.skeleton-line {
  height: 14px;
  background: #F7F8FA;
  border-radius: 4px;
  margin-bottom: 8px;
}

.skeleton-line.w-60 {
  width: 60%;
}

.skeleton-line.w-40 {
  width: 40%;
}

.empty-state {
  padding: 60px;
  text-align: center;
  color: #666;
}
</style>
