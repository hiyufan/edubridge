<script setup>
// 设置分组中的一行：可选图标 + 标题/值 + 可选右箭头；默认插槽放额外内容（表单等）
defineProps({
  label: { type: String, default: '' },
  value: { type: String, default: '' },
  arrow: { type: Boolean, default: false },
  clickable: { type: Boolean, default: false }
})
defineEmits(['click'])
</script>

<template>
  <div class="apple-grouped-item" :class="{ 'item-clickable': clickable || arrow }" @click="$emit('click')">
    <div v-if="$slots.icon" class="item-icon">
      <slot name="icon" />
    </div>
    <div class="item-content">
      <span v-if="label" class="item-label">{{ label }}</span>
      <span v-if="value" class="item-value">{{ value }}</span>
      <slot />
    </div>
    <div v-if="arrow" class="item-arrow">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="9 18 15 12 9 6" />
      </svg>
    </div>
  </div>
</template>
