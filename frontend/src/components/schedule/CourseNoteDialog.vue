<script setup>
// 课程备注（保存在本机浏览器）
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getNote, saveNote } from '../../utils/notes'
import { DAY_NAMES } from '../../utils/schedule'

const props = defineProps({
  course: { type: Object, default: null }
})
const visible = defineModel({ type: Boolean, default: false })
const emit = defineEmits(['saved'])

const text = ref('')
watch(visible, (open) => {
  if (open && props.course) {
    text.value = getNote(props.course.name, props.course.dayOfWeek, props.course.periodStart)
  }
})

const save = (value) => {
  const c = props.course
  if (!c) return
  saveNote(c.name, c.dayOfWeek, c.periodStart, value)
  visible.value = false
  emit('saved')
  ElMessage.success(value ? '备注已保存' : '备注已删除')
}
</script>

<template>
  <el-dialog v-model="visible" :title="course ? '课程备注 - ' + course.name : '课程备注'" width="400px" :close-on-click-modal="false">
    <div class="note-dialog-content">
      <div v-if="course" class="note-course-info">
        <span class="note-course-name">{{ course.name }}</span>
        <span class="note-course-detail">{{ course.room }} · {{ DAY_NAMES[course.dayOfWeek - 1] }} · 第{{ course.periodStart }}节</span>
      </div>
      <el-input v-model="text" type="textarea" :rows="4" placeholder="添加课程备注，如：需要带教材、实验课地点变更等" maxlength="200" show-word-limit />
    </div>
    <template #footer>
      <div class="note-dialog-footer">
        <el-button v-if="text" type="danger" plain @click="save('')">删除备注</el-button>
        <el-button type="primary" @click="save(text)">保存</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
/* Dialog */
.note-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.note-course-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.note-course-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-dark);
}

.note-course-detail {
  font-size: 13px;
  color: var(--color-text-muted);
}

.note-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
