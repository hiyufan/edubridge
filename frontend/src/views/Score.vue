<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '../utils/request'

const allScores = ref([])
const selectedSemester = ref('')
const loading = ref(false)

// 从全部成绩中派生学期列表
const semesterListFromScores = computed(() => {
  const semesters = [...new Set(allScores.value.map(s => `${s.year}-${s.semester}`))]
  return semesters.sort()
})

// 根据选中学期过滤成绩
const scoreList = computed(() => {
  if (!selectedSemester.value) return allScores.value
  return allScores.value.filter(s => `${s.year}-${s.semester}` === selectedSemester.value)
})

// 当前学期索引
const currentIndex = computed(() => {
  return semesterListFromScores.value.indexOf(selectedSemester.value)
})

// 切换到上一个学期
const prevSemester = () => {
  if (currentIndex.value > 0) {
    selectedSemester.value = semesterListFromScores.value[currentIndex.value - 1]
  }
}

// 切换到下一个学期
const nextSemester = () => {
  if (currentIndex.value < semesterListFromScores.value.length - 1) {
    selectedSemester.value = semesterListFromScores.value[currentIndex.value + 1]
  }
}

// 获取全部成绩
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

// GPA 详情计算
const gpaDetail = computed(() => {
  const validScores = scoreList.value.filter(s => s.gpa > 0 && s.credit > 0)
  if (validScores.length === 0) {
    return { weighted: '0.00', simple: '0.00', passRate: '0%', totalCredits: '0.0' }
  }

  const totalCredits = validScores.reduce((sum, s) => sum + s.credit, 0)
  const weightedSum = validScores.reduce((sum, s) => sum + s.gpa * s.credit, 0)
  const simpleSum = validScores.reduce((sum, s) => sum + s.gpa, 0)
  const passCount = scoreList.value.filter(s => parseFloat(s.grade) >= 60 || s.gpa > 0).length

  return {
    weighted: (weightedSum / totalCredits).toFixed(2),
    simple: (simpleSum / validScores.length).toFixed(2),
    passRate: ((passCount / scoreList.value.length) * 100).toFixed(0) + '%',
    totalCredits: totalCredits.toFixed(1)
  }
})

// 获取 GPA 颜色
const getGpaColor = (gpaVal) => {
  const num = parseFloat(gpaVal)
  if (isNaN(num)) return '#8e8e93'
  if (num >= 3.5) return '#34C759'
  if (num >= 2.5) return '#007AFF'
  if (num >= 1.5) return '#FF9500'
  return '#FF3B30'
}

// 获取成绩颜色
const getGradeInfo = (grade) => {
  const num = parseFloat(grade)
  if (isNaN(num)) return { color: '#8e8e93', bg: 'rgba(142, 142, 147, 0.12)', label: '无成绩' }
  if (num >= 90) return { color: '#34C759', bg: 'rgba(52, 199, 89, 0.12)', label: '优秀' }
  if (num >= 80) return { color: '#007AFF', bg: 'rgba(0, 122, 255, 0.12)', label: '良好' }
  if (num >= 70) return { color: '#FF9500', bg: 'rgba(255, 149, 0, 0.12)', label: '中等' }
  if (num >= 60) return { color: '#FF3B30', bg: 'rgba(255, 59, 48, 0.12)', label: '及格' }
  return { color: '#FF3B30', bg: 'rgba(255, 59, 48, 0.12)', label: '不及格' }
}

// 初始化
onMounted(async () => {
  await fetchAllScores()
  if (semesterListFromScores.value.length > 0) {
    selectedSemester.value = semesterListFromScores.value[0]
  }
})
</script>

<template>
  <div class="score-page">
    <!-- Stats Card -->
    <div class="stats-card animate-apple-fade-in">
      <div class="stat-main">
        <div class="stat-gpa">
          <span class="stat-gpa-value">{{ gpaDetail.weighted }}</span>
          <span class="stat-gpa-label">加权绩点 GPA</span>
        </div>
      </div>
      <div class="stat-divider"></div>
      <div class="stat-secondary">
        <div class="stat-item">
          <span class="stat-value">{{ scoreList.length }}</span>
          <span class="stat-label">课程数</span>
        </div>
        <div class="stat-item">
          <span class="stat-value">{{ gpaDetail.totalCredits }}</span>
          <span class="stat-label">总学分</span>
        </div>
        <div class="stat-item">
          <span class="stat-value">{{ gpaDetail.passRate }}</span>
          <span class="stat-label">通过率</span>
        </div>
      </div>
    </div>

    <!-- GPA Detail Card -->
    <div class="gpa-detail-card animate-apple-fade-in stagger-1">
      <div class="gpa-detail-header">绩点分析</div>
      <div class="gpa-detail-row">
        <span class="gpa-detail-label">加权平均绩点</span>
        <span class="gpa-detail-value" :style="{ color: getGpaColor(gpaDetail.weighted) }">
          {{ gpaDetail.weighted }}
        </span>
      </div>
      <div class="gpa-detail-row">
        <span class="gpa-detail-label">算术平均绩点</span>
        <span class="gpa-detail-value">{{ gpaDetail.simple }}</span>
      </div>
      <div class="gpa-detail-row">
        <span class="gpa-detail-label">课程通过率</span>
        <span class="gpa-detail-value">{{ gpaDetail.passRate }}</span>
      </div>
      <!-- GPA Progress Bar -->
      <div class="gpa-bar-wrap">
        <div class="gpa-bar-bg">
          <div
            class="gpa-bar-fill"
            :style="{
              width: `${Math.min((parseFloat(gpaDetail.weighted) / 4.0) * 100, 100)}%`,
              background: getGpaColor(gpaDetail.weighted)
            }"
          ></div>
        </div>
        <div class="gpa-bar-labels">
          <span>0</span><span>1.0</span><span>2.0</span><span>3.0</span><span>4.0</span>
        </div>
      </div>
    </div>

    <!-- Semester Selector - Apple Tab Style -->
    <div class="semester-card animate-apple-fade-in stagger-2" v-if="semesterListFromScores.length > 0">
      <div class="semester-picker">
        <button
          class="semester-nav-btn"
          :class="{ disabled: currentIndex <= 0 }"
          @click="prevSemester"
          :disabled="currentIndex <= 0"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="15 18 9 12 15 6"/>
          </svg>
        </button>
        <div class="semester-display">
          <transition name="semester-slide" mode="out-in">
            <div :key="selectedSemester" class="semester-content">
              <span class="semester-year">{{ selectedSemester.split('-')[0] }}-{{ selectedSemester.split('-')[1] }}</span>
              <span class="semester-term">{{ selectedSemester.split('-')[2] === '1' ? '上学期' : '下学期' }}</span>
            </div>
          </transition>
        </div>
        <button
          class="semester-nav-btn"
          :class="{ disabled: currentIndex >= semesterListFromScores.length - 1 }"
          @click="nextSemester"
          :disabled="currentIndex >= semesterListFromScores.length - 1"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="9 18 15 12 9 6"/>
          </svg>
        </button>
      </div>
      <div class="semester-indicator">
        <span
          v-for="(sem, idx) in semesterListFromScores"
          :key="sem"
          class="indicator-dot"
          :class="{ active: idx === currentIndex }"
        />
      </div>
    </div>

    <!-- Score List -->
    <div v-loading="loading" class="score-list animate-apple-fade-in stagger-3">
      <!-- Skeleton Loading -->
      <template v-if="loading">
        <div v-for="i in 5" :key="i" class="score-item skeleton-item">
          <div class="score-left">
            <div class="skeleton-text skeleton-title"></div>
            <div class="skeleton-text skeleton-meta"></div>
            <div class="skeleton-tags">
              <div class="skeleton-tag"></div>
              <div class="skeleton-tag"></div>
            </div>
          </div>
          <div class="score-right">
            <div class="skeleton-grade"></div>
          </div>
        </div>
      </template>

      <!-- Real Content -->
      <template v-else>
        <div
          v-for="(score, idx) in scoreList"
          :key="idx"
          class="score-item"
        >
          <div class="score-left">
            <div class="score-name">{{ score.course }}</div>
            <div class="score-meta">
              <span class="score-teacher">{{ score.teacher || '未知教师' }}</span>
              <span class="score-dot">·</span>
              <span class="score-credit">{{ score.credit }}学分</span>
            </div>
            <div class="score-tags">
              <span class="apple-badge" :style="{ background: 'rgba(88, 86, 214, 0.1)', color: '#5856D6' }">
                {{ score.nature || '必修' }}
              </span>
              <span class="apple-badge" :style="{ background: 'rgba(255, 149, 0, 0.1)', color: '#FF9500' }">
                {{ score.type || '普通' }}
              </span>
            </div>
          </div>
          <div class="score-right">
            <div
              class="score-grade"
              :style="{ background: getGradeInfo(score.grade).bg, color: getGradeInfo(score.grade).color }"
            >
              <span class="grade-value">{{ score.grade }}</span>
              <span class="grade-label">{{ getGradeInfo(score.grade).label }}</span>
            </div>
            <div
              class="score-gpa"
              :style="{ color: getGpaColor(score.gpa) }"
            >
              {{ Number(score.gpa).toFixed(1) }}
            </div>
          </div>
        </div>
      </template>

      <!-- Empty State -->
      <div v-if="scoreList.length === 0 && !loading" class="empty-state">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="empty-icon">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
          <polyline points="14 2 14 8 20 8"/>
          <line x1="12" y1="12" x2="12" y2="18"/>
          <line x1="9" y1="15" x2="15" y2="15"/>
        </svg>
        <p>暂无成绩数据</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.score-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* Stats Card */
.stats-card {
  display: flex;
  align-items: center;
  background: linear-gradient(135deg, #007AFF 0%, #5856D6 100%);
  border-radius: 20px;
  padding: 28px 24px;
  color: white;
  box-shadow: 0 8px 32px rgba(0, 122, 255, 0.3);
}

.stat-main {
  flex: 1;
}

.stat-gpa {
  display: flex;
  flex-direction: column;
}

.stat-gpa-value {
  font-size: 56px;
  font-weight: 700;
  letter-spacing: -0.04em;
  line-height: 1;
}

.stat-gpa-label {
  font-size: 14px;
  font-weight: 500;
  opacity: 0.85;
  margin-top: 6px;
}

.stat-divider {
  width: 1px;
  height: 60px;
  background: rgba(255, 255, 255, 0.3);
  margin: 0 24px;
}

.stat-secondary {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.stat-item {
  display: flex;
  flex-direction: column;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.stat-label {
  font-size: 12px;
  font-weight: 500;
  opacity: 0.75;
}

/* GPA Detail Card */
.gpa-detail-card {
  background: white;
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04), 0 4px 12px rgba(0,0,0,0.03);
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.gpa-detail-header {
  font-size: 13px;
  font-weight: 600;
  color: #8e8e93;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.gpa-detail-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.gpa-detail-label {
  font-size: 15px;
  color: #1d1d1f;
}

.gpa-detail-value {
  font-size: 17px;
  font-weight: 700;
  color: #1d1d1f;
}

.gpa-bar-wrap {
  margin-top: 4px;
}

.gpa-bar-bg {
  height: 8px;
  background: #f2f2f7;
  border-radius: 4px;
  overflow: hidden;
}

.gpa-bar-fill {
  height: 100%;
  border-radius: 4px;
  transition: width 0.8s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.gpa-bar-labels {
  display: flex;
  justify-content: space-between;
  margin-top: 4px;
  font-size: 11px;
  color: #c7c7cc;
}

/* Semester Card */
.semester-card {
  background: white;
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04), 0 4px 12px rgba(0,0,0,0.03);
}

/* Apple Tab Style Semester Picker */
.semester-picker {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.semester-nav-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: rgba(0, 122, 255, 0.08);
  border: none;
  cursor: pointer;
  transition: all 0.25s cubic-bezier(0.25, 0.1, 0.25, 1);
  flex-shrink: 0;
}

.semester-nav-btn svg {
  width: 18px;
  height: 18px;
  color: #007AFF;
  transition: transform 0.2s ease;
}

.semester-nav-btn:hover:not(.disabled) {
  background: rgba(0, 122, 255, 0.15);
  transform: scale(1.05);
}

.semester-nav-btn:hover:not(.disabled) svg {
  transform: translateX(-1px);
}

.semester-nav-btn:active:not(.disabled) {
  transform: scale(0.95);
}

.semester-nav-btn.disabled {
  background: rgba(118, 118, 128, 0.08);
  cursor: not-allowed;
}

.semester-nav-btn.disabled svg {
  color: #C7C7CC;
}

.semester-display {
  flex: 1;
  text-align: center;
  overflow: hidden;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.semester-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.semester-year {
  font-size: 15px;
  font-weight: 600;
  color: #1d1d1f;
  letter-spacing: -0.01em;
}

.semester-term {
  font-size: 13px;
  font-weight: 500;
  color: #8e8e93;
}

/* Semester slide transition */
.semester-slide-enter-active,
.semester-slide-leave-active {
  transition: all 0.3s cubic-bezier(0.25, 0.1, 0.25, 1);
}

.semester-slide-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.semester-slide-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

/* Semester indicator dots */
.semester-indicator {
  display: flex;
  justify-content: center;
  gap: 6px;
  margin-top: 14px;
}

.indicator-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: rgba(118, 118, 128, 0.25);
  transition: all 0.3s cubic-bezier(0.25, 0.1, 0.25, 1);
}

.indicator-dot.active {
  width: 18px;
  border-radius: 3px;
  background: #007AFF;
}

/* Score List */
.score-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.score-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: white;
  border-radius: 16px;
  padding: 16px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04), 0 2px 8px rgba(0,0,0,0.02);
  transition: all 0.2s ease;
}

.score-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(0,0,0,0.08);
}

.score-left {
  flex: 1;
  min-width: 0;
}

.score-name {
  font-size: 16px;
  font-weight: 600;
  color: #1d1d1f;
  letter-spacing: -0.016em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 4px;
}

.score-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #8e8e93;
  margin-bottom: 10px;
}

.score-dot {
  opacity: 0.5;
}

.score-tags {
  display: flex;
  gap: 6px;
}

.apple-badge {
  display: inline-flex;
  align-items: center;
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 600;
}

.score-right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
}

.score-grade {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px 14px;
  border-radius: 12px;
  min-width: 64px;
}

.grade-value {
  font-size: 20px;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.grade-label {
  font-size: 10px;
  font-weight: 500;
  opacity: 0.8;
  margin-top: 2px;
}

.score-gpa {
  font-size: 14px;
  font-weight: 600;
}

/* Skeleton Loading */
.skeleton-item {
  pointer-events: none;
}

.skeleton-text {
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 4px;
}

.skeleton-title {
  width: 60%;
  height: 18px;
  margin-bottom: 8px;
}

.skeleton-meta {
  width: 40%;
  height: 14px;
  margin-bottom: 10px;
}

.skeleton-tags {
  display: flex;
  gap: 6px;
}

.skeleton-tag {
  width: 50px;
  height: 20px;
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 6px;
}

.skeleton-grade {
  width: 64px;
  height: 60px;
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 12px;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
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
</style>
