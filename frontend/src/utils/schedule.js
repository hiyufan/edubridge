import { ElNotification } from 'element-plus'
import request from './request'

export const DAY_NAMES = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']
export const PERIODS = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]

export const isWeekend = (dayName) => dayName === '周六' || dayName === '周日'

/** 课程的唯一键（同一门课在同一时段） */
export const courseKey = (c) => `${c.name}|${c.dayOfWeek}|${c.periodStart}`

/**
 * 后台检测到课表变动时弹出一次提示（同一次变动只提示一次）
 */
export async function notifyScheduleDiffOnce() {
  const diff = (await request.get('/schedule/diff')).data
  if (!diff || !(diff.added?.length || diff.removed?.length || diff.changed?.length)) return

  let seen = ''
  try { seen = localStorage.getItem('schedule_diff_seen') || '' } catch {}
  if (diff.detectedAt === seen) return
  try { localStorage.setItem('schedule_diff_seen', diff.detectedAt) } catch {}

  const msgs = []
  if (diff.added?.length) msgs.push(`加课 ${diff.added.length} 节`)
  if (diff.removed?.length) msgs.push(`减课/停课 ${diff.removed.length} 节`)
  if (diff.changed?.length) msgs.push(`调整 ${diff.changed.length} 节`)
  ElNotification({ title: '课表有变动', message: msgs.join('、'), type: 'info', duration: 5000 })
}

/**
 * 冲突检测结果 -> 课程键到说明文字的映射
 * @param {Array<{courseA: object, courseB: object, conflictWeeks: number[]}>} pairs
 */
export function conflictMap(pairs) {
  const m = new Map()
  for (const p of pairs || []) {
    const weeks = (p.conflictWeeks || []).join('、')
    m.set(courseKey(p.courseA), `与「${p.courseB.name}」在第 ${weeks} 周存在时间冲突`)
    m.set(courseKey(p.courseB), `与「${p.courseA.name}」在第 ${weeks} 周存在时间冲突`)
  }
  return m
}
