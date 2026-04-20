# 前端开发指南

## 快速开始

```bash
cd frontend
npm install
npm run dev
```

访问 `http://localhost:5173`，开发服务器会自动代理 `/api` 请求到 `http://localhost:3000`。

## 项目结构

```
frontend/src/
├── main.js              # Vue 入口
├── App.vue              # 根组件
├── style.css            # 全局样式
├── router/
│   └── index.js         # 路由配置
├── stores/
│   ├── user.js          # 用户状态（token、uid）
│   └── theme.js         # 主题状态
├── utils/
│   ├── request.js       # Axios 封装（请求拦截、Token 自动携带）
│   ├── periods.js       # 课节时间配置
│   ├── notifications.js # 浏览器通知
│   ├── ical.js          # 日历导出
│   └── notes.js         # 备注工具
├── views/
│   ├── Login.vue        # 登录页
│   ├── Schedule.vue     # 课表页
│   ├── Score.vue        # 成绩页
│   ├── Today.vue        # 今日课表
│   └── Profile.vue      # 个人中心
└── layout/
    └── MainLayout.vue   # 主布局（顶部 + 底部导航）
```

## 开发规范

### 组件写法

使用 Vue 3 Composition API + `<script setup>` 语法：

```vue
<script setup>
import { ref, onMounted } from 'vue'
import request from '../utils/request'

const data = ref(null)

onMounted(async () => {
  const res = await request.get('/schedule')
  data.value = res.data
})
</script>
```

### 样式

- 全局样式使用 Tailwind CSS 工具类
- 组件样式使用 `<style scoped>`
- 主色调: `blue-600`（Element Plus 蓝色系）

### 请求封装

`request.js` 已封装好 Axios 实例：

- 自动携带 `Authorization: Bearer <token>` 头
- 401 错误自动跳转登录页
- 业务错误自动弹出 `ElMessage` 提示

### API 代理

`vite.config.js` 中已配置，开发环境所有 `/api` 请求代理到 `http://localhost:3000`。

## 主要组件说明

### Login.vue

- 组件挂载时请求 `/api/captcha` 获取验证码
- 登录成功后保存 Token 到 localStorage，跳转主页

### Schedule.vue

- 默认请求当前周课表
- 支持周次切换（1~20 周）
- 提供「查看全学期」功能

### Score.vue

- 从 `/api/score/semesters` 获取可选学期
- 学期下拉筛选 + 成绩表格展示

### Profile.vue

- 显示学生基本信息（姓名、班级、学号）
- 退出登录按钮
