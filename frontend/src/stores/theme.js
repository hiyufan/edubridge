import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  // 三种模式：light / dark / auto
  const mode = ref(localStorage.getItem('theme_mode') || 'auto')
  const isDark = ref(false)
  // 主题色（默认蓝色）
  const primaryColor = ref(localStorage.getItem('primary_color') || '#007AFF')

  const PRESET_COLORS = [
    { label: '蓝色',   value: '#007AFF' },
    { label: '绿色',   value: '#34C759' },
    { label: '紫色',   value: '#AF52DE' },
    { label: '橙色',   value: '#FF9500' },
    { label: '粉色',   value: '#FF2D55' },
    { label: '青色',   value: '#5AC8FA' },
    { label: '靛蓝',   value: '#5856D6' },
    { label: '深红',   value: '#BF5AF2' },
  ]

  function setPrimaryColor(color) {
    primaryColor.value = color
    localStorage.setItem('primary_color', color)
    document.documentElement.style.setProperty('--color-primary', color)
    document.documentElement.style.setProperty('--color-primary-light', color + '26') // 15% opacity
  }

  function applyTheme() {
    const root = document.documentElement
    if (mode.value === 'dark') {
      isDark.value = true
      root.setAttribute('data-theme', 'dark')
    } else if (mode.value === 'light') {
      isDark.value = false
      root.setAttribute('data-theme', 'light')
    } else {
      // auto: 跟随系统
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
      isDark.value = prefersDark
      root.setAttribute('data-theme', prefersDark ? 'dark' : 'light')
    }
  }

  function setMode(newMode) {
    mode.value = newMode
    localStorage.setItem('theme_mode', newMode)
    applyTheme()
  }

  function toggle() {
    if (mode.value === 'dark' || (mode.value === 'auto' && isDark.value)) {
      setMode('light')
    } else {
      setMode('dark')
    }
  }

  // 初始化
  applyTheme()
  // 初始化主题色
  setPrimaryColor(primaryColor.value)

  // 监听系统主题变化（仅 auto 模式下生效）
  const mq = window.matchMedia('(prefers-color-scheme: dark)')
  mq.addEventListener('change', () => {
    if (mode.value === 'auto') applyTheme()
  })

  return { mode, isDark, setMode, toggle, applyTheme,
    primaryColor, setPrimaryColor, PRESET_COLORS }
})
