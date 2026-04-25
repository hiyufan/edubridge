import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import router from './router'
import './style.css'
import App from './App.vue'
import { attemptSilentRefresh } from './utils/request'
import { useUserStore } from './stores/user'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(ElementPlus)

const initAuth = async () => {
  const userStore = useUserStore()
  if (userStore.hasValidToken()) {
    attemptSilentRefresh()
  }
}

initAuth()

app.mount('#app')
