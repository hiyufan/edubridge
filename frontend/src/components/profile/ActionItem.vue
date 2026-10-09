<script setup>
// 设置分组中的蓝色操作行（保存、发送测试等），busy 时显示 busyText 并忽略点击
const props = defineProps({
  text: { type: String, required: true },
  busyText: { type: String, default: '处理中…' },
  busy: { type: Boolean, default: false }
})
const emit = defineEmits(['click'])
const onClick = () => {
  if (!props.busy) emit('click')
}
</script>

<template>
  <div class="apple-grouped-item item-clickable" @click="onClick">
    <div v-if="$slots.icon" class="item-icon" style="background: rgba(0, 122, 255, 0.1)">
      <slot name="icon" />
    </div>
    <div class="item-content">
      <span class="item-label item-action">{{ busy ? busyText : text }}</span>
    </div>
  </div>
</template>
