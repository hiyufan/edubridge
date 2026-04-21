import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import router from './router'
import './style.css'
import App from './App.vue'
import { attemptSilentRefresh } from './utils/request'

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(ElementPlus)

// VUE-1 修复：页面加载时，如果有 RefreshToken Cookie 则静默刷新，恢复登录状态
// 在 mount 之前调用，此时 Pinia 已初始化完毕
attemptSilentRefresh()

app.mount('#app')
