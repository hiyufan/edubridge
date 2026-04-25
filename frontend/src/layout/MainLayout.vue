<script setup>
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()
const sidebarCollapsed = ref(false)

const activeTab = computed(() => route.path)

const navItems = [
  {
    path: '/schedule',
    label: '课表',
    icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
      <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
      <line x1="16" y1="2" x2="16" y2="6"/>
      <line x1="8" y1="2" x2="8" y2="6"/>
      <line x1="3" y1="10" x2="21" y2="10"/>
      <path d="M8 14h.01M12 14h.01M16 14h.01M8 18h.01M12 18h.01"/>
    </svg>`,
  },
  {
    path: '/today',
    label: '今日',
    icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
      <circle cx="12" cy="12" r="10"/>
      <polyline points="12 6 12 12 16 14"/>
    </svg>`,
  },
  {
    path: '/score',
    label: '成绩',
    icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
      <polyline points="14 2 14 8 20 8"/>
      <line x1="16" y1="13" x2="8" y2="13"/>
      <line x1="16" y1="17" x2="8" y2="17"/>
    </svg>`,
  },
  {
    path: '/profile',
    label: '我的',
    icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
      <circle cx="12" cy="7" r="4"/>
    </svg>`,
  },
]

const handleTabChange = (path) => {
  router.push(path)
}
</script>

<template>
  <div class="layout-container">
    <!-- PC Sidebar (lg+) -->
    <aside
      class="sidebar-pc"
      :class="{ collapsed: sidebarCollapsed }"
    >
      <!-- Logo -->
      <div class="sidebar-header">
        <div class="sidebar-logo">
          <svg viewBox="0 0 24 24" fill="none" class="logo-icon">
            <rect x="2" y="3" width="20" height="18" rx="3" stroke="currentColor" stroke-width="1.5"/>
            <path d="M8 10h8M8 14h5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
        </div>
        <span v-if="!sidebarCollapsed" class="sidebar-title">教务系统</span>
      </div>

      <!-- Collapse toggle -->
      <button
        class="sidebar-toggle"
        @click="sidebarCollapsed = !sidebarCollapsed"
        :title="sidebarCollapsed ? '展开侧栏' : '收起侧栏'"
      >
        <svg v-if="!sidebarCollapsed" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="toggle-icon">
          <path d="M15 18l-6-6 6-6"/>
        </svg>
        <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="toggle-icon">
          <path d="M9 18l6-6-6-6"/>
        </svg>
      </button>

      <!-- Nav items -->
      <nav class="sidebar-nav">
        <button
          v-for="item in navItems"
          :key="item.path"
          class="nav-item"
          :class="{ active: activeTab === item.path }"
          @click="handleTabChange(item.path)"
          :title="sidebarCollapsed ? item.label : undefined"
        >
          <div class="nav-item-indicator" v-if="activeTab === item.path"></div>
          <span class="nav-icon" v-html="item.icon"></span>
          <span v-if="!sidebarCollapsed" class="nav-label">{{ item.label }}</span>
        </button>
      </nav>
    </aside>

    <!-- Mobile Tab Bar -->
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

    <!-- Mobile Tab Bar -->
    <nav class="tab-bar">
      <button
        v-for="tab in navItems"
        :key="tab.path"
        class="tab-item"
        :class="{ active: activeTab === tab.path }"
        @click="handleTabChange(tab.path)"
      >
        <span class="tab-icon" v-html="tab.icon"></span>
        <span class="tab-label">{{ tab.label }}</span>
      </button>
    </nav>
  </div>
</template>

<style scoped>
.layout-container {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: var(--color-bg);
}

/* =====================
   PC Sidebar (lg+ only)
   ===================== */
.sidebar-pc {
  display: none;
}

@media (min-width: 1024px) {
  .sidebar-pc {
    display: flex;
    flex-direction: column;
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    width: 240px;
    background: #1E293B;
    border-right: 1px solid rgba(255, 255, 255, 0.06);
    z-index: 200;
    transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .sidebar-pc.collapsed {
    width: 64px;
  }

  .sidebar-header {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 24px 20px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  }

  .sidebar-logo {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .logo-icon {
    width: 26px;
    height: 26px;
    color: #94A3B8;
  }

  .sidebar-title {
    font-family: var(--font-heading);
    font-size: 16px;
    font-weight: 600;
    color: #F8FAFC;
    white-space: nowrap;
    letter-spacing: 0.02em;
  }

  .sidebar-toggle {
    position: absolute;
    right: -12px;
    top: 72px;
    width: 24px;
    height: 24px;
    border-radius: 50%;
    background: #334155;
    border: 1px solid rgba(255, 255, 255, 0.1);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: all 0.2s ease;
    z-index: 10;
  }

  .sidebar-toggle:hover {
    background: #475569;
    transform: scale(1.05);
  }

  .toggle-icon {
    width: 12px;
    height: 12px;
    color: #F8FAFC;
  }

  .sidebar-nav {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 16px 12px;
    flex: 1;
  }

  .nav-item {
    position: relative;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 11px 12px;
    border-radius: 8px;
    border: none;
    background: transparent;
    cursor: pointer;
    transition: all 0.2s ease;
    color: #94A3B8;
    text-align: left;
    width: 100%;
  }

  .nav-item:hover:not(.active) {
    background: rgba(255, 255, 255, 0.05);
    color: #F8FAFC;
  }

  .nav-item.active {
    background: rgba(255, 255, 255, 0.08);
    color: #F8FAFC;
  }

  .nav-item-indicator {
    position: absolute;
    left: 0;
    top: 50%;
    transform: translateY(-50%);
    width: 3px;
    height: 18px;
    background: #22C55E;
    border-radius: 0 2px 2px 0;
  }

  .nav-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 22px;
    height: 22px;
  }

  .nav-icon :deep(svg) {
    width: 100%;
    height: 100%;
  }

  .nav-label {
    font-size: 14px;
    font-weight: 500;
    white-space: nowrap;
  }
}

/* =====================
   Mobile Tab Bar
   ===================== */
@media (min-width: 1024px) {
  .nav-bar,
  .tab-bar {
    display: none !important;
  }
}

.nav-bar {
  position: sticky;
  top: 0;
  z-index: 100;
  background: var(--color-card);
  border-bottom: 1px solid var(--color-border);
  display: block;
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
  font-family: var(--font-heading);
  font-size: 17px;
  font-weight: 700;
  color: var(--color-text-dark);
  letter-spacing: 0.02em;
}

.nav-logo {
  width: 22px;
  height: 22px;
  color: var(--color-primary);
}

.main-content {
  flex: 1;
  padding: 20px 16px;
  padding-bottom: calc(80px + env(safe-area-inset-bottom));
  max-width: 600px;
  margin: 0 auto;
  width: 100%;
}

@media (min-width: 1024px) {
  .main-content {
    padding: 32px 40px;
    padding-bottom: 32px;
    max-width: none;
    margin: 0;
    margin-left: 240px;
    transition: margin-left 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .sidebar-pc.collapsed ~ .main-content,
  .layout-container:has(.sidebar-pc.collapsed) .main-content {
    margin-left: 64px;
  }
}

.tab-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 100;
  display: flex;
  justify-content: space-around;
  align-items: flex-end;
  height: calc(64px + env(safe-area-inset-bottom));
  padding-bottom: env(safe-area-inset-bottom);
  background: var(--color-card);
  border-top: 1px solid var(--color-border);
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
  transition: all 0.15s ease;
  color: var(--color-text-muted);
}

.tab-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  transition: all 0.15s ease;
}

.tab-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.tab-label {
  font-size: 10px;
  font-weight: 500;
  letter-spacing: 0;
  transition: all 0.15s ease;
}

.tab-item.active {
  color: var(--color-primary);
}

.tab-item.active .tab-label {
  font-weight: 600;
}

.tab-item:hover:not(.active) {
  color: var(--color-text-secondary);
}
</style>
