<script setup>
import { useRouter, useRoute } from 'vue-router'
import { computed } from 'vue'

const router = useRouter()
const route = useRoute()

const activeTab = computed(() => route.path)

const tabs = [
  {
    path: '/schedule',
    label: '课表',
    icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
      <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
      <line x1="16" y1="2" x2="16" y2="6"/>
      <line x1="8" y1="2" x2="8" y2="6"/>
      <line x1="3" y1="10" x2="21" y2="10"/>
      <path d="M8 14h.01M12 14h.01M16 14h.01M8 18h.01M12 18h.01"/>
    </svg>`
  },
  {
    path: '/score',
    label: '成绩',
    icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
      <polyline points="14 2 14 8 20 8"/>
      <line x1="16" y1="13" x2="8" y2="13"/>
      <line x1="16" y1="17" x2="8" y2="17"/>
      <polyline points="10 9 9 9 8 9"/>
    </svg>`
  },
  {
    path: '/profile',
    label: '我的',
    icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
      <circle cx="12" cy="7" r="4"/>
    </svg>`
  }
]

const handleTabChange = (path) => {
  router.push(path)
}
</script>

<template>
  <div class="layout-container">
    <!-- Navigation Bar -->
    <header class="nav-bar">
      <div class="nav-content">
        <div class="nav-title">
          <svg viewBox="0 0 24 24" fill="none" class="nav-logo">
            <rect x="2" y="3" width="20" height="18" rx="3" stroke="currentColor" stroke-width="1.5"/>
            <path d="M8 10h8M8 14h5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
          <span>教务系统</span>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="main-content">
      <router-view />
    </main>

    <!-- Tab Bar -->
    <nav class="tab-bar">
      <button
        v-for="tab in tabs"
        :key="tab.path"
        class="tab-item"
        :class="{ active: activeTab === tab.path }"
        @click="handleTabChange(tab.path)"
      >
        <span class="tab-icon" v-html="tab.icon"></span>
        <span class="tab-label">{{ tab.label }}</span>
        <span class="tab-indicator" v-if="activeTab === tab.path"></span>
      </button>
    </nav>
  </div>
</template>

<style scoped>
.layout-container {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: #f2f2f7;
}

/* Navigation Bar */
.nav-bar {
  position: sticky;
  top: 0;
  z-index: 100;
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: saturate(180%) blur(20px);
  -webkit-backdrop-filter: saturate(180%) blur(20px);
  border-bottom: 0.5px solid rgba(0, 0, 0, 0.1);
}

.nav-content {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 52px;
  padding: 0 16px;
}

.nav-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 17px;
  font-weight: 600;
  color: #1d1d1f;
  letter-spacing: -0.022em;
}

.nav-logo {
  width: 22px;
  height: 22px;
  color: #007AFF;
}

/* Main Content */
.main-content {
  flex: 1;
  padding: 20px 16px;
  padding-bottom: calc(80px + env(safe-area-inset-bottom));
  max-width: 600px;
  margin: 0 auto;
  width: 100%;
}

/* Tab Bar */
.tab-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 100;
  display: flex;
  justify-content: space-around;
  align-items: flex-end;
  height: calc(80px + env(safe-area-inset-bottom));
  padding-bottom: env(safe-area-inset-bottom);
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: saturate(180%) blur(20px);
  -webkit-backdrop-filter: saturate(180%) blur(20px);
  border-top: 0.5px solid rgba(0, 0, 0, 0.1);
}

.tab-item {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  padding: 8px 20px;
  border: none;
  background: transparent;
  cursor: pointer;
  transition: all 0.2s ease;
}

.tab-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  color: #8e8e93;
  transition: all 0.25s cubic-bezier(0.25, 0.1, 0.25, 1);
}

.tab-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.tab-label {
  font-size: 10px;
  font-weight: 500;
  color: #8e8e93;
  letter-spacing: 0.02em;
  transition: all 0.2s ease;
}

.tab-indicator {
  position: absolute;
  top: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 20px;
  height: 3px;
  background: #007AFF;
  border-radius: 0 0 2px 2px;
}

/* Active State */
.tab-item.active .tab-icon {
  color: #007AFF;
  transform: translateY(-2px);
}

.tab-item.active .tab-label {
  color: #007AFF;
  font-weight: 600;
}

/* Hover State */
.tab-item:hover:not(.active) .tab-icon {
  color: #636366;
}

.tab-item:hover:not(.active) .tab-label {
  color: #636366;
}
</style>
