/**
 * 课程提醒通知工具
 * 使用浏览器 Notification API + Service Worker
 */

const NOTIFY_BEFORE_MINUTES = 15 // 提前多少分钟提醒

/**
 * 检查通知权限状态
 */
export function getNotifyPermission() {
  if (!('Notification' in window)) return 'unsupported'
  return Notification.permission // 'granted' | 'denied' | 'default'
}

/**
 * 请求通知权限
 */
export async function requestNotifyPermission() {
  if (!('Notification' in window)) return 'unsupported'
  const result = await Notification.requestPermission()
  return result
}

/**
 * 发送浏览器通知
 */
export function sendNotification(title, options = {}) {
  if (getNotifyPermission() !== 'granted') return null
  const notif = new Notification(title, {
    icon: '/favicon.ico',
    badge: '/favicon.ico',
    silent: false,
    requireInteraction: false,
    ...options,
  })
  // 5秒后自动关闭
  setTimeout(() => notif.close(), 5000)
  return notif
}

/**
 * 根据课程数据设置提醒
 * @param {Array} courses - 课程列表
 */
export function scheduleCourseReminders(courses) {
  if (getNotifyPermission() !== 'granted') return

  // 清除之前的提醒
  clearAllReminders()

  const now = new Date()

  courses.forEach(course => {
    // 计算课程开始时间
    const { startTime } = getCourseNextOccurrence(course, now)
    if (!startTime) return

    // 计算提醒时间
    const notifyTime = new Date(startTime.getTime() - NOTIFY_BEFORE_MINUTES * 60 * 1000)
    const delay = notifyTime.getTime() - now.getTime()

    if (delay > 0 && delay < 24 * 60 * 60 * 1000) {
      // 只提醒24小时内的课程
      const timerId = setTimeout(() => {
        sendNotification(`📚 ${course.name} 即将上课`, {
          body: `${course.room || '待定'} · 还有 ${NOTIFY_BEFORE_MINUTES} 分钟`,
          tag: `course-${course.dayOfWeek}-${course.periodStart}`,
        })
      }, delay)
      reminderTimers.push(timerId)
    }
  })
}

// 存储所有定时器 ID
const reminderTimers = []

/**
 * 清除所有提醒
 */
export function clearAllReminders() {
  reminderTimers.forEach(id => clearTimeout(id))
  reminderTimers.length = 0
}

/**
 * 计算课程下一次出现的时间
 * @param {Object} course - 课程对象
 * @param {Date} from - 从什么时候开始计算
 */
export function getCourseNextOccurrence(course, from = new Date()) {
  const days = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  const now = new Date(from)

  // 找出下一个符合 dayOfWeek 的日期
  let targetDay = course.dayOfWeek // 1=周一, 7=周日
  let targetDate = new Date(now)

  // 调整 targetDay: 教务系统1=周一 -> JS中0=周日,1=周一...
  // 所以 targetDay=1(周一) -> JS中1, targetDay=7(周日) -> JS中0
  const jsDayOfWeek = targetDay === 7 ? 0 : targetDay

  const currentDay = now.getDay()
  let daysUntil = jsDayOfWeek - currentDay
  if (daysUntil < 0) daysUntil += 7 // 下周

  targetDate.setDate(now.getDate() + daysUntil)

  // 解析上课时间
  const timeMap = {
    1:  { h: 8,  m: 0  },
    2:  { h: 8,  m: 55 },
    3:  { h: 10, m: 0  },
    4:  { h: 10, m: 55 },
    5:  { h: 14, m: 0  },
    6:  { h: 14, m: 55 },
    7:  { h: 15, m: 50 },
    8:  { h: 16, m: 45 },
    9:  { h: 18, m: 30 },
    10: { h: 19, m: 25 },
    11: { h: 20, m: 20 },
    12: { h: 21, m: 15 },
  }

  const periodTime = timeMap[course.periodStart]
  if (!periodTime) return { startTime: null, endTime: null }

  targetDate.setHours(periodTime.h, periodTime.m, 0, 0)

  const endMinutes = (course.periodStart - 1 + course.periods) * 55
  const endHour = Math.floor(8 * 60 + endMinutes) / 60
  const endTime = new Date(targetDate.getTime() + course.periods * 55 * 60 * 1000)

  return { startTime: targetDate, endTime }
}
