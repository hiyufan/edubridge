# jww.p 前端

Vue 3 + Vite 构建的教务系统中间件管理后台。

## 功能页面

| 页面 | 路径 | 说明 |
|------|------|------|
| 登录 | `/login` | 验证码登录 |
| 今日课表 | `/` | 当天课程快速查看 |
| 课表 | `/schedule` | 全学期课表，按周切换 |
| 成绩 | `/score` | GPA 趋势图、成绩列表、绩点模拟器 |
| 个人中心 | `/profile` | iCal 订阅、Webhook 管理 |

## 技术栈

- Vue 3 (Composition API + `<script setup>`)
- Vite
- Pinia（状态管理）
- Vue Router
- Element Plus（UI 组件）
- Tailwind CSS

## 开发

```bash
npm install
npm run dev
```

访问 `http://localhost:5173`

## 构建

```bash
npm run build
```

构建产物输出到 `dist/`，为标准静态文件，可由任意 Web 服务器托管。
