<script setup>
import { useThemeStore } from '../../stores/theme'
import SettingsSection from './SettingsSection.vue'

const themeStore = useThemeStore()
const modes = [{ v: 'light', l: '浅' }, { v: 'dark', l: '深' }, { v: 'auto', l: '自动' }]
</script>

<template>
  <SettingsSection title="外观">
    <div class="apple-grouped-item">
      <div class="item-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><circle cx="12" cy="12" r="5" /><line x1="12" y1="1" x2="12" y2="3" /><line x1="12" y1="21" x2="12" y2="23" /><line x1="4.22" y1="4.22" x2="5.64" y2="5.64" /><line x1="18.36" y1="18.36" x2="19.78" y2="19.78" /><line x1="1" y1="12" x2="3" y2="12" /><line x1="21" y1="12" x2="23" y2="12" /><line x1="4.22" y1="19.78" x2="5.64" y2="18.36" /><line x1="18.36" y1="5.64" x2="19.78" y2="4.22" /></svg>
      </div>
      <div class="item-content row">
        <span class="item-label">主题模式</span>
        <div class="theme-modes">
          <button
            v-for="m in modes"
            :key="m.v"
            class="theme-mode-btn"
            :class="{ active: themeStore.mode === m.v }"
            @click="themeStore.setMode(m.v)"
          >{{ m.l }}</button>
        </div>
      </div>
    </div>
    <div class="apple-grouped-item" style="flex-wrap: wrap; gap: 8px">
      <div class="item-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><circle cx="13.5" cy="6.5" r="3.5" /><circle cx="17.5" cy="10.5" r="2.5" /><circle cx="8.5" cy="7.5" r="4.5" /><circle cx="6.5" cy="12.5" r="5" /></svg>
      </div>
      <div class="item-content row" style="gap: 6px; flex: 1">
        <span class="item-label">主题色</span>
        <div class="color-swatches">
          <button
            v-for="c in themeStore.PRESET_COLORS"
            :key="c.value"
            class="color-swatch"
            :style="{ background: c.value }"
            :class="{ active: themeStore.primaryColor === c.value }"
            @click="themeStore.setPrimaryColor(c.value)"
          ></button>
        </div>
      </div>
    </div>
  </SettingsSection>
</template>

<style scoped>
.row {
  flex-direction: row;
  align-items: center;
  gap: 8px;
}

.theme-modes {
  display: flex;
  gap: 4px;
  margin-left: auto;
}

.theme-mode-btn {
  padding: 4px 12px;
  border: 1.5px solid var(--color-border);
  border-radius: 20px;
  background: white;
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: all 0.2s ease;
}

.theme-mode-btn.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
  font-weight: 600;
}

.theme-mode-btn:hover:not(.active) {
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.color-swatches {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  margin-left: auto;
}

.color-swatch {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.15);
}

.color-swatch.active {
  border-color: white;
  box-shadow: 0 0 0 2px currentColor, 0 2px 6px rgba(0, 0, 0, 0.2);
  transform: scale(1.15);
}

.color-swatch:hover:not(.active) {
  transform: scale(1.1);
}

@media (min-width: 1024px) {
  .theme-mode-btn {
    padding: 5px 16px;
    font-size: 13px;
  }

  .color-swatch {
    width: 28px;
    height: 28px;
  }
}
</style>
