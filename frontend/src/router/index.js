import { createRouter, createWebHistory } from 'vue-router'
import Login from '../views/Login.vue'
import MainLayout from '../layout/MainLayout.vue'
import { useUserStore } from '../stores/user'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: Login
    },
    {
      path: '/',
      component: MainLayout,
      children: [
        {
          path: '',
          redirect: '/schedule'
        },
        {
          path: 'schedule',
          name: 'Schedule',
          component: () => import('../views/Schedule.vue')
        },
        {
          path: 'today',
          name: 'Today',
          component: () => import('../views/Today.vue')
        },
        {
          path: 'score',
          name: 'Score',
          component: () => import('../views/Score.vue')
        },
        {
          path: 'profile',
          name: 'Profile',
          component: () => import('../views/Profile.vue')
        }
      ]
    }
  ]
})

router.beforeEach((to, from, next) => {
  // 从 Pinia 内存读取 token（VUE-1 修复：不再从 localStorage 读取）
  const userStore = useUserStore()
  const hasToken = !!userStore.token
  if (!hasToken && to.path !== '/login') {
    next('/login')
  } else if (hasToken && to.path === '/login') {
    next('/')
  } else {
    next()
  }
})

export default router
