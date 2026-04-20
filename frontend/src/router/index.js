import { createRouter, createWebHistory } from 'vue-router'
import Login from '../views/Login.vue'
import MainLayout from '../layout/MainLayout.vue'

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
  const token = localStorage.getItem('token')
  if (!token && to.path !== '/login') {
    next('/login')
  } else if (token && to.path === '/login') {
    next('/')
  } else {
    next()
  }
})

export default router
