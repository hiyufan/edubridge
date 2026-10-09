import { computed } from 'vue'

const PALETTE = [
  { bg: '#78716C', light: '#E7E2DA' }, { bg: '#A16207', light: '#FEF3C7' },
  { bg: '#15803D', light: '#DCFCE7' }, { bg: '#B45309', light: '#FEF3C7' },
  { bg: '#0369A1', light: '#E0F2FE' }, { bg: '#7C3AED', light: '#EDE9FE' },
  { bg: '#C2410C', light: '#FFEDD5' }, { bg: '#65A30D', light: '#ECFCCB' }
]

/**
 * 按课程名稳定地分配颜色（同名课程同色）
 * @param {import('vue').Ref<Array<{name: string}>>} courses
 */
export function useCourseColors(courses) {
  const colorMap = computed(() => {
    const m = new Map()
    for (const c of courses.value || []) {
      if (!m.has(c.name)) {
        const hash = c.name.split('').reduce((a, ch) => a + ch.charCodeAt(0), 0)
        m.set(c.name, PALETTE[hash % PALETTE.length])
      }
    }
    return m
  })
  return (name) => colorMap.value.get(name) || PALETTE[0]
}
