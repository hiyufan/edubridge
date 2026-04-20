/**
 * 课程节次时间表配置
 * 可根据学校实际作息调整
 */

// 节次 → 实际时间（24小时制）
export const PERIOD_TIMES = {
  1:  { start: '08:00', end: '08:45' },
  2:  { start: '08:55', end: '09:40' },
  3:  { start: '10:00', end: '10:45' },
  4:  { start: '10:55', end: '11:40' },
  5:  { start: '14:00', end: '14:45' },
  6:  { start: '14:55', end: '15:40' },
  7:  { start: '15:50', end: '16:35' },
  8:  { start: '16:45', end: '17:30' },
  9:  { start: '18:30', end: '19:15' },
  10: { start: '19:25', end: '20:10' },
  11: { start: '20:20', end: '21:05' },
  12: { start: '21:15', end: '22:00' },
}

// 上午/下午/晚上分类
export const PERIOD_SEGMENT = {
  1:  '上午', 2:  '上午',
  3:  '上午', 4:  '上午',
  5:  '下午', 6:  '下午',
  7:  '下午', 8:  '下午',
  9:  '晚上', 10: '晚上',
  11: '晚上', 12: '晚上',
}

/**
 * 获取某节次的起止时间字符串
 * @param {number} period - 节次编号（1-12）
 * @returns {{ start: string, end: string, display: string }}
 */
export function getPeriodTime(period) {
  const time = PERIOD_TIMES[period]
  if (!time) return { start: '', end: '', display: '' }
  return {
    start: time.start,
    end: time.end,
    display: `${time.start}–${time.end}`,
  }
}

/**
 * 格式化节次显示文本
 * @param {number} period - 节次编号
 * @param {number} periods - 连续节数（默认1）
 */
export function formatPeriodLabel(period, periods = 1) {
  if (periods === 1) {
    return `第${period}节`
  }
  const endPeriod = period + periods - 1
  return `第${period}–${endPeriod}节`
}

/**
 * 获取某节课的实际时间段（用于 Today.vue）
 * @param {number} periodStart - 起始节次
 * @param {number} periods - 连续节数
 */
export function getCourseTimeRange(periodStart, periods) {
  const start = PERIOD_TIMES[periodStart]
  if (!start) return ''
  const end = PERIOD_TIMES[periodStart + periods - 1]
  if (!end) return `${start.start}`
  return `${start.start}–${end.end}`
}
