<script setup>
import { computed } from 'vue'

const props = defineProps({
  studentName: { type: String, default: '' },
  className: { type: String, default: '' },
  week: { type: Number, required: true },
  maxWeek: { type: Number, default: 20 }
})
const emit = defineEmits(['update:week', 'subscribe', 'export'])

const weeks = computed(() => Array.from({ length: props.maxWeek }, (_, i) => i + 1))
const go = (w) => emit('update:week', Math.min(props.maxWeek, Math.max(1, w)))
</script>

<template>
  <div class="schedule-header-card academic-card academic-fade-in">
    <div class="header-top">
      <div v-if="studentName || className" class="student-info">
        <h1 class="student-name">{{ studentName }}</h1>
        <span class="class-name">{{ className }}</span>
      </div>
      <div class="header-actions">
        <button class="action-btn" title="日历订阅" @click="emit('subscribe')">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" /></svg>
          <span>订阅</span>
        </button>
        <button class="action-btn" title="导出日历" @click="emit('export')">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" /><polyline points="7 10 12 15 17 10" /><line x1="12" y1="15" x2="12" y2="3" /></svg>
          <span>导出</span>
        </button>
      </div>
    </div>

    <div class="week-selector-row">
      <button class="week-nav-btn" :disabled="week <= 1" @click="go(week - 1)">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6" /></svg>
      </button>
      <div class="week-display">
        <span class="week-num font-heading">第 {{ week }} 周</span>
      </div>
      <button class="week-nav-btn" :disabled="week >= maxWeek" @click="go(week + 1)">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6" /></svg>
      </button>
      <div class="week-pills">
        <button v-for="w in weeks" :key="w" class="week-pill" :class="{ active: week === w }" @click="go(w)">{{ w }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
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
  flex-wrap: wrap;
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

/* 手机：周次单独一行，左右滑动 */
@media (max-width: 1023px) {
  .week-pills {
    flex-basis: 100%;
    flex-wrap: nowrap;
    overflow-x: auto;
    margin-left: 0;
    padding-bottom: 4px;
    scrollbar-width: none;
  }

  .week-pills::-webkit-scrollbar {
    display: none;
  }

  .week-pill {
    flex-shrink: 0;
  }
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
</style>
