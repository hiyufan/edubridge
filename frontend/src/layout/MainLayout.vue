<script setup>
import { useRouter, useRoute } from 'vue-router'
import { computed } from 'vue'

const router = useRouter()
const route = useRoute()

const activeTab = computed(() => route.path)

const tabs = [
  {
    path: '/schedule',
    label: '课表查询',
    icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>`
  },
  {
    path: '/today',
    label: '今日课程',
    icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>`
  },
  {
    path: '/score',
    label: '成绩查询',
    icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>`
  },
  {
    path: '/profile',
    label: '个人中心',
    icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>`
  }
]

const handleTabChange = (path) => {
  router.push(path)
}
</script>

<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="sidebar-header">
        <div class="logo">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <rect x="2" y="3" width="20" height="18" rx="2"/>
            <path d="M8 10h8M8 14h5"/>
          </svg>
        </div>
        <div class="brand">
          <span class="brand-name">教务系统</span>
          <span class="brand-sub">Academic Portal</span>
        </div>
      </div>

      <nav class="sidebar-nav">
        <button
          v-for="tab in tabs"
          :key="tab.path"
          class="nav-item"
          :class="{ active: activeTab === tab.path }"
          @click="handleTabChange(tab.path)"
        >
          <span class="nav-icon" v-html="tab.icon"></span>
          <span class="nav-label">{{ tab.label }}</span>
        </button>
      </nav>
    </aside>

    <main class="main">
      <header class="topbar">
        <h1 class="page-title">{{ tabs.find(t => t.path === activeTab)?.label || '首页' }}</h1>
      </header>

      <div class="content">
        <router-view />
      </div>
    </main>
  </div>
</template>

<style scoped>
.layout {
  display: flex;
  min-height: 100vh;
  background: #F7F8FA;
}

.sidebar {
  width: 220px;
  background: #fff;
  border-right: 1px solid #E8E8E8;
  display: flex;
  flex-direction: column;
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
}

.sidebar-header {
  padding: 16px 20px;
  display: flex;
  align-items: center;
  gap: 12px;
  border-bottom: 1px solid #E8E8E8;
}

.logo {
  width: 36px;
  height: 36px;
  background: #1677FF;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.logo svg {
  width: 20px;
  height: 20px;
  color: #fff;
}

.brand {
  display: flex;
  flex-direction: column;
}

.brand-name {
  font-size: 15px;
  font-weight: 600;
  color: #1F1F1F;
}

.brand-sub {
  font-size: 11px;
  color: #999;
}

.sidebar-nav {
  flex: 1;
  padding: 12px 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: none;
  background: transparent;
  border-radius: 6px;
  cursor: pointer;
  transition: all 150ms ease;
  width: 100%;
  text-align: left;
}

.nav-item:hover {
  background: #F7F8FA;
}

.nav-item.active {
  background: #E6F4FF;
  color: #1677FF;
}

.nav-icon {
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.nav-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.nav-item:not(.active) .nav-icon {
  color: #666;
}

.nav-item.active .nav-icon {
  color: #1677FF;
}

.nav-label {
  font-size: 14px;
  font-weight: 500;
}

.nav-item:not(.active) .nav-label {
  color: #666;
}

.nav-item.active .nav-label {
  color: #1677FF;
}

.main {
  flex: 1;
  margin-left: 220px;
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.topbar {
  height: 52px;
  background: #fff;
  border-bottom: 1px solid #E8E8E8;
  padding: 0 24px;
  display: flex;
  align-items: center;
}

.page-title {
  font-size: 15px;
  font-weight: 600;
  color: #1F1F1F;
}

.content {
  flex: 1;
  padding: 24px;
}
</style>
