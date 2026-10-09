<script setup>
defineProps({
  title: { type: String, required: true }
})
</script>

<template>
  <div class="settings-section animate-warm-fade-in stagger-2">
    <div class="apple-section-header">{{ title }}</div>
    <div class="apple-grouped-list">
      <slot />
    </div>
  </div>
</template>
