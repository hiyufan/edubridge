/**
 * 课表 iCal (.ics) 导出工具
 * 生成符合 RFC 5545 的日历文件
 */

import { getPeriodTime } from './periods'

// RFC 5545 日期时间格式
const fmt = {
  DTSTART: (d) => formatICSDate(d),
  DTEND:   (d) => formatICSDate(d),
  DTSTAMP: ()  => formatICSDate(new Date()),
}

/** 将 Date 转为 ICS 格式 YYYYMMDDTHHMMSS */
function formatICSDate(date) {
  const pad = n => String(n).padStart(2, '0')
  const y = date.getFullYear()
  const mo = pad(date.getMonth() + 1)
  const d = pad(date.getDate())
  const h = pad(date.getHours())
  const mi = pad(date.getMinutes())
  const s = pad(date.getSeconds())
  return `${y}${mo}${d}T${h}${mi}${s}`
}

/** 转义 ICS 特殊字符 */
function escapeICS(str) {
  if (!str) return ''
  return String(str)
    .replace(/\\/g, '\\\\')
    .replace(/;/g, '\\;')
    .replace(/,/g, '\\,')
    .replace(/\n/g, '\\n')
}

/** 获取课程下一节课的日期 */
function getNextCourseDate(dayOfWeek, periodStart) {
  const now = new Date()
  const dayMap = [0, 1, 2, 3, 4, 5, 6] // JS: 0=周日
  const targetJS = dayOfWeek === 7 ? 0 : dayOfWeek

  const next = new Date(now)
  next.setHours(0, 0, 0, 0)
  const diff = (targetJS - now.getDay() + 7) % 7 || 7
  next.setDate(now.getDate() + diff)

  const timeMap = {
    1:  [8,  0], 2:  [8,  55], 3:  [10, 0], 4:  [10, 55],
    5:  [14, 0], 6:  [14, 55], 7:  [15, 50], 8:  [16, 45],
    9:  [18, 30], 10: [19, 25], 11: [20, 20], 12: [21, 15],
  }
  const [h, m] = timeMap[periodStart] || [8, 0]
  next.setHours(h, m, 0, 0)
  return next
}

/**
 * 生成 iCal 文件内容
 * @param {Object} scheduleData - 课表数据
 * @param {string} studentName - 学生姓名
 * @returns {string} .ics 文件文本
 */
export function generateICal(scheduleData, studentName = '') {
  const uidBase = Date.now()
  const lines = [
    'BEGIN:VCALENDAR',
    'VERSION:2.0',
    'PRODID:-//JW System//课表导出//ZH',
    'CALSCALE:GREGORIAN',
    'METHOD:PUBLISH',
    'X-WR-CALNAME:' + escapeICS((studentName ? studentName + ' ' : '') + '课程表'),
    'X-WR-TIMEZONE:Asia/Shanghai',
    'X-APPLE-CALENDAR-COLOR:#007AFF',
  ]

  const courses = scheduleData?.courses || []
  const seenKeys = new Set()
  let uidCounter = 0

  courses.forEach(course => {
    // 按 (dayOfWeek, periodStart, name) 去重
    const key = `${course.dayOfWeek}-${course.periodStart}-${course.name}`
    if (seenKeys.has(key)) return
    seenKeys.add(key)

    const startDate = getNextCourseDate(course.dayOfWeek, course.periodStart)
    const periodTime = getPeriodTime(course.periodStart)
    const endMinutes = course.periods * 55
    const endDate = new Date(startDate.getTime() + endMinutes * 60 * 1000)

    // BYDAY: MO/TU/WE/TH/FR/SA/SU
    const dayNames = ['SU', 'MO', 'TU', 'WE', 'TH', 'FR', 'SA']
    const byDay = dayNames[course.dayOfWeek === 7 ? 0 : course.dayOfWeek]

    // RRULE: 直到 2025-08-01 学期结束
    const until = '20250801T000000Z'

    const uid = `${uidBase}-${uidCounter++}@jw-system`

    lines.push('BEGIN:VEVENT')
    lines.push(`UID:${uid}`)
    lines.push(`DTSTAMP:${fmt.DTSTAMP()}`)
    lines.push(`DTSTART:${fmt.DTSTART(startDate)}`)
    lines.push(`DTEND:${fmt.DTEND(endDate)}`)
    lines.push(`RRULE:FREQ=WEEKLY;BYDAY=${byDay};UNTIL=${until}`)
    lines.push(`SUMMARY:${escapeICS(course.name)}`)
    lines.push(`LOCATION:${escapeICS(course.room || '')}`)
    lines.push(`DESCRIPTION:${escapeICS(course.teacher ? '教师: ' + course.teacher : '')}`)
    lines.push(`CATEGORIES:${escapeICS(course.room ? '课程' : '')}`)
    lines.push('STATUS:CONFIRMED')
    lines.push('TRANSP:OPAQUE')
    lines.push('END:VEVENT')
  })

  lines.push('END:VCALENDAR')
  return lines.join('\r\n')
}

/**
 * 触发浏览器下载 .ics 文件
 */
export function downloadICal(content, filename = 'schedule.ics') {
  const blob = new Blob([content], { type: 'text/calendar;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
