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
│   ├── schedule.js      # 课表状态（含冲突数据）
│   └── theme.js         # 主题状态
├── utils/
│   ├── request.js       # Axios 封装（请求拦截、Token 自动携带）
│   ├── periods.js       # 课节时间配置
│   ├── notifications.js # 浏览器通知
│   ├── ical.js          # 日历导出
│   └── notes.js         # 备注工具
├── views/
│   ├── Login.vue        # 登录页
│   ├── Schedule.vue     # 课表页（含冲突检测警告）
│   ├── Score.vue        # 成绩页（含 Canvas GPA 趋势图）
│   ├── Today.vue        # 今日课表
│   └── Profile.vue      # 个人中心（iCal 订阅、Webhook 管理）
└── layout/
    └── MainLayout.vue   # 主布局（顶部 + 底部导航）
```

## 页面说明

### 今日课表 `/`

显示当天课程，快速概览。支持按日期切换。

### 课表 `/schedule`

- 按周切换（1~20 周），自动识别当前周
- 冲突检测：同一时段有重复课程时显示红色警告卡片
- 全学期视图：展开查看所有周课程

### 成绩 `/score`

- **GPA 趋势图**：原生 Canvas 绘制，贝塞尔曲线 + 渐变填充 + 数据点发光效果，高清适配
- 学期筛选：下拉选择学期，支持学期切换动画
- 成绩列表：卡片展示每门课程（含绩点颜色映射）
- **GPA 模拟器**：拖动滑块调整分数，实时预测 GPA 变化

### 个人中心 `/profile`

- iCal 订阅：一键复制订阅地址，导入 Google Calendar / Apple Calendar / Outlook
- Webhook 管理：注册、查看、删除回调通知

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

- 自动携带 `Authorization: Bearer ***` 头
- 401 错误自动跳转登录页
- 业务错误自动弹出 `ElMessage` 提示
- 使用 `AbortController` 取消竞态请求

### Canvas 图表

GPA 趋势图使用原生 Canvas API 绘制，不依赖第三方图表库。关键实现：

- `devicePixelRatio` 高清适配
- 贝塞尔曲线 (`bezierCurveTo`) 平滑连接数据点
- 线性渐变 (`createLinearGradient`) 填充曲线下方区域
- 径向渐发 (`createRadialGradient`) 实现数据点外发光
