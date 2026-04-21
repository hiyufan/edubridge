import { defineStore } from 'pinia'
import { ref } from 'vue'
import request from '../utils/request'

export const useScheduleStore = defineStore('schedule', () => {
  const scheduleData = ref(null)
  const loading = ref(false)
  const scheduleController = ref(null)

  async function fetchSchedule(week) {
    // VUE-7 修复：取消上一个 pending 请求，避免旧响应覆盖新数据
    if (scheduleController.value) {
      scheduleController.value.abort()
    }
    scheduleController.value = new AbortController()

    loading.value = true
    try {
      const url = week ? `/schedule?week=${week}` : '/schedule'
      const res = await request.get(url, {
        signal: scheduleController.value.signal
      })
      scheduleData.value = res.data
      return res.data
    } catch (error) {
      if (error.name === 'CanceledError' || error.code === 'ERR_CANCELED') {
        return null
      }
      throw error
    } finally {
      loading.value = false
    }
  }

  function clearSchedule() {
    scheduleData.value = null
  }

  return { scheduleData, loading, fetchSchedule, clearSchedule }
})
