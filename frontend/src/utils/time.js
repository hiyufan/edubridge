/**
 * 相对时间：刚刚 / N 分钟前 / N 小时前 / M月D日 HH:mm
 * @param {string|null} iso
 */
export function relativeTime(iso) {
  if (!iso) return '尚未进行'
  const diff = (Date.now() - new Date(iso).getTime()) / 1000
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`
  const d = new Date(iso)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getMonth() + 1}月${d.getDate()}日 ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
