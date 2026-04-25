<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '../utils/request'

const allScores = ref([])
const selectedSemester = ref('')
const loading = ref(false)

// 功能 05: 成绩统计数据
const scoreStats = ref(null)

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

// 功能 05: 获取成绩统计
const fetchScoreStats = async () => {
  try {
    const res = await request.get('/score/stats')
    scoreStats.value = res.data
  } catch {}
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
  if (num >= 3.5) return '#A16207'
  if (num >= 2.5) return '#78716C'
  if (num >= 1.5) return '#B45309'
  return '#DC2626'
}

// 获取成绩颜色
const getGradeInfo = (grade) => {
  const num = parseFloat(grade)
  if (isNaN(num)) return { color: '#8e8e93', bg: 'rgba(142, 142, 147, 0.12)', label: '无成绩' }
  if (num >= 90) return { color: '#A16207', bg: 'rgba(161, 98, 7, 0.12)', label: '优秀' }
  if (num >= 80) return { color: '#78716C', bg: 'rgba(120, 113, 108, 0.12)', label: '良好' }
  if (num >= 70) return { color: '#B45309', bg: 'rgba(180, 83, 9, 0.12)', label: '中等' }
  if (num >= 60) return { color: '#DC2626', bg: 'rgba(220, 38, 38, 0.12)', label: '及格' }
  return { color: '#DC2626', bg: 'rgba(220, 38, 38, 0.12)', label: '不及格' }
}

// 功能 05: 绘制 GPA 趋势图
const trendCanvas = ref(null)

const drawTrendChart = () => {
  if (!trendCanvas.value || !scoreStats.value?.semesterStats?.length) return
  const canvas = trendCanvas.value
  const ctx = canvas.getContext('2d')
  const dpr = window.devicePixelRatio || 1
  const w = canvas.offsetWidth
  const h = 200

  // 设置高分辨率画布
  canvas.width = w * dpr
  canvas.height = h * dpr
  canvas.style.width = w + 'px'
  canvas.style.height = h + 'px'
  ctx.scale(dpr, dpr)

  const data = scoreStats.value.semesterStats
  const labels = data.map(s => s.semester)
  const values = data.map(s => s.gpa)
  const n = labels.length
  const padding = { top: 24, right: 24, bottom: 40, left: 40 }
  const chartW = w - padding.left - padding.right
  const chartH = h - padding.top - padding.bottom
  const yMin = 0, yMax = 4.0
  const xStep = n > 1 ? chartW / (n - 1) : 0
  const yScale = (v) => padding.top + chartH - ((v - yMin) / (yMax - yMin)) * chartH
  const xPos = (i) => padding.left + i * xStep

  // 坐标点
  const pts = values.map((v, i) => ({ x: xPos(i), y: yScale(v) }))

  // --- 背景 ---
  ctx.fillStyle = '#fff'
  ctx.fillRect(0, 0, w, h)

  // --- 柔和网格线 ---
  ctx.strokeStyle = 'rgba(120, 120, 128, 0.1)'
  ctx.lineWidth = 1
  for (let y = 0; y <= 4; y += 1) {
    const py = yScale(y)
    ctx.beginPath()
    ctx.moveTo(padding.left, py)
    ctx.lineTo(w - padding.right, py)
    ctx.stroke()
  }

  // --- X 轴标签 ---
  ctx.fillStyle = '#8e8e93'
  ctx.font = '11px -apple-system, BlinkMacSystemFont, sans-serif'
  ctx.textAlign = 'center'
  labels.forEach((label, i) => {
    ctx.fillText(label, xPos(i), h - 8)
  })

  // --- Y 轴标签 ---
  ctx.textAlign = 'right'
  ctx.font = '10px -apple-system, BlinkMacSystemFont, sans-serif'
  for (let y = 0; y <= 4; y += 1) {
    ctx.fillText(y.toFixed(1), padding.left - 6, yScale(y) + 4)
  }

  // --- 渐变填充区域 ---
  if (n > 1) {
    const grad = ctx.createLinearGradient(0, padding.top, 0, h - padding.bottom)
    grad.addColorStop(0, 'rgba(0, 122, 255, 0.18)')
    grad.addColorStop(1, 'rgba(0, 122, 255, 0.00)')
    ctx.beginPath()
    ctx.moveTo(pts[0].x, h - padding.bottom)
    pts.forEach(p => ctx.lineTo(p.x, p.y))
    ctx.lineTo(pts[n - 1].x, h - padding.bottom)
    ctx.closePath()
    ctx.fillStyle = grad
    ctx.fill()
  }

  // --- 贝塞尔平滑曲线 ---
  ctx.beginPath()
  if (n === 1) {
    ctx.arc(pts[0].x, pts[0].y, 3, 0, Math.PI * 2)
  } else {
    pts.forEach((p, i) => {
      if (i === 0) {
        ctx.moveTo(p.x, p.y)
      } else {
        const prev = pts[i - 1]
        const cpX = (prev.x + p.x) / 2
        ctx.bezierCurveTo(cpX, prev.y, cpX, p.y, p.x, p.y)
      }
    })
  }
  ctx.strokeStyle = '#78716C'
  ctx.lineWidth = 2.5
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  ctx.stroke()

  // --- 数据点 + 外发光 ---
  pts.forEach((p, i) => {
    // 外发光
    const radGrad = ctx.createRadialGradient(p.x, p.y, 0, p.x, p.y, 10)
    radGrad.addColorStop(0, 'rgba(0, 122, 255, 0.35)')
    radGrad.addColorStop(1, 'rgba(0, 122, 255, 0)')
    ctx.beginPath()
    ctx.arc(p.x, p.y, 10, 0, Math.PI * 2)
    ctx.fillStyle = radGrad
    ctx.fill()

    // 白底
    ctx.beginPath()
    ctx.arc(p.x, p.y, 4.5, 0, Math.PI * 2)
    ctx.fillStyle = '#fff'
    ctx.fill()

    // 蓝点
    ctx.beginPath()
    ctx.arc(p.x, p.y, 3, 0, Math.PI * 2)
    ctx.fillStyle = '#78716C'
    ctx.fill()

    // GPA 数值标签
    ctx.fillStyle = 'rgba(29, 29, 31, 0.75)'
    ctx.font = '500 10px -apple-system, BlinkMacSystemFont, sans-serif'
    ctx.textAlign = 'center'
    const labelY = p.y - 12
    if (labelY > padding.top + 5) {
      ctx.fillText(values[i].toFixed(2), p.x, labelY)
    } else {
      ctx.fillText(values[i].toFixed(2), p.x, p.y + 16)
    }
  })
}

// 功能 05: GPA 详情计算
// 初始化
onMounted(async () => {
  await fetchAllScores()
  await fetchScoreStats()
  if (semesterListFromScores.value.length > 0) {
    selectedSemester.value = semesterListFromScores.value[0]
  }
  setTimeout(drawTrendChart, 100)
})
</script>

<template>
  <div class="score-page">
    <!-- Stats Card -->
    <div class="stats-card animate-warm-fade-in">
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

    <!-- 功能 05: 顶部统计 el-statistic -->
    <div class="stats-row animate-warm-fade-in" v-if="scoreStats">
      <div class="stat-card">
        <span class="stat-card-value">{{ scoreStats.totalCredits || 0 }}</span>
        <span class="stat-card-label">总学分</span>
      </div>
      <div class="stat-card">
        <span class="stat-card-value" :style="{ color: getGpaColor(scoreStats.weightedGPA) }">
          {{ scoreStats.weightedGPA?.toFixed(2) || '0.00' }}
        </span>
        <span class="stat-card-label">加权 GPA</span>
      </div>
      <div class="stat-card">
        <span class="stat-card-value" :style="{ color: scoreStats.failedCount > 0 ? '#DC2626' : '#A16207' }">
          {{ scoreStats.failedCount || 0 }}
        </span>
        <span class="stat-card-label">挂科数</span>
      </div>
    </div>

    <!-- 功能 05: GPA 趋势图 -->
    <div class="trend-card animate-warm-fade-in stagger-1" v-if="scoreStats?.semesterStats?.length">
      <div class="trend-header">GPA 趋势</div>
      <canvas ref="trendCanvas" width="600" height="200" class="trend-canvas"></canvas>
    </div>

    <!-- GPA Detail Card -->
    <div class="gpa-detail-card animate-warm-fade-in stagger-1">
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
    <div class="semester-card animate-warm-fade-in stagger-2" v-if="semesterListFromScores.length > 0">
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
    <div v-loading="loading" class="score-list animate-warm-fade-in stagger-3">
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
              <span class="apple-badge" :style="{ background: 'rgba(120, 113, 108, 0.1)', color: '#78716C' }">
                {{ score.nature || '必修' }}
              </span>
              <span class="apple-badge" :style="{ background: 'rgba(180, 83, 9, 0.1)', color: '#B45309' }">
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
  background: var(--color-primary);
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
  color: var(--color-text-muted);
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
  color: var(--color-text);
}

.gpa-detail-value {
  font-size: 17px;
  font-weight: 700;
  color: var(--color-text);
}

.gpa-bar-wrap {
  margin-top: 4px;
}

.gpa-bar-bg {
  height: 8px;
  background: var(--color-bg);
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
  color: var(--color-border);
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
  color: var(--color-primary);
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
  color: var(--color-text);
  letter-spacing: -0.01em;
}

.semester-term {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-muted);
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
  color: var(--color-text);
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
  color: var(--color-text-muted);
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
  background: linear-gradient(90deg, #E7E2DA 25%, #F7F4EF 50%, #E7E2DA 75%);
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
  background: linear-gradient(90deg, #E7E2DA 25%, #F7F4EF 50%, #E7E2DA 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 6px;
}

.skeleton-grade {
  width: 64px;
  height: 60px;
  background: linear-gradient(90deg, #E7E2DA 25%, #F7F4EF 50%, #E7E2DA 75%);
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
  color: var(--color-text-muted);
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

/* 功能 05: 统计行 */
.stats-row {
  display: flex;
  gap: 12px;
}

.stat-card {
  flex: 1;
  background: white;
  border-radius: 12px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04), 0 4px 12px rgba(0,0,0,0.03);
}

.stat-card-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text);
}

.stat-card-label {
  font-size: 12px;
  color: var(--color-text-muted);
  font-weight: 500;
}

/* 功能 05: GPA 趋势图 */
.trend-card {
  background: white;
  border-radius: 16px;
  padding: 20px 24px 16px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04), 0 4px 12px rgba(0,0,0,0.03);
}

.trend-header {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text);
  margin-bottom: 16px;
  letter-spacing: -0.2px;
}

.trend-canvas {
  width: 100%;
  height: 200px;
  display: block;
}

/* =====================
   PC Responsive (lg+)
   ===================== */
@media (min-width: 1024px) {
  .score-page {
    gap: 28px;
    max-width: 1200px;
  }

  /* Stats row — 4 columns */
  .stats-row {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
  }

  .stat-card {
    padding: 24px 20px;
    border-radius: var(--radius-xl);
  }

  .stat-card-value {
    font-size: 36px;
  }

  .stat-card-label {
    font-size: 13px;
  }

  /* Trend chart — wider */
  .trend-card {
    border-radius: var(--radius-xl);
    padding: 28px 32px 24px;
  }

  .trend-canvas {
    height: 240px;
  }

  /* Score list + simulator — 2 column grid */
  .score-list {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 16px;
    align-items: start;
  }

  .score-item {
    padding: 20px;
    border-radius: var(--radius-xl);
  }

  .score-name {
    font-size: 17px;
  }

  .score-meta {
    font-size: 14px;
  }

  .score-grade {
    padding: 10px 18px;
    border-radius: var(--radius-lg);
  }

  .grade-value {
    font-size: 24px;
  }

  .grade-label {
    font-size: 11px;
  }

  .score-gpa {
    font-size: 15px;
  }

  .apple-badge {
    font-size: 12px;
    padding: 4px 10px;
  }

  /* Skeleton */
  .skeleton-item {
    border-radius: var(--radius-xl);
  }
}
</style>
