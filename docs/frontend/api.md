# API 接口文档

本文档为前端开发提供完整的 API 参考。所有接口前缀 `/api`。

## 认证说明

除公开接口外，所有业务接口均需在请求头携带 JWT Token：

```
Authorization: Bearer <token>
```

Token 通过登录接口获取，有效期 24 小时，过期后需重新登录。

---

## 公开接口

### 获取验证码

```
GET /api/captcha
```

返回验证码图片（PNG），同时通过 `Set-Cookie` 设置 `sessionId`，前端需保存该值用于登录。

**响应示例：**
```
HTTP/1.1 200 OK
Content-Type: image/png
Set-Cookie: sessionId=abc123; Path=/

<binary image data>
```

---

### 登录

```
POST /api/auth/login
Content-Type: application/json
```

**请求体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `username` | string | 学号 |
| `password` | string | 密码 |
| `captcha` | string | 验证码（4位） |
| `loginType` | string | 登录类型: `xsxh`(学生) / `zjh`(教职工) / `gkksh`(考生) |
| `sessionId` | string | 验证码 session ID（从 cookie 获取） |

**成功响应：**
```json
{
  "status": 1,
  "info": "登录成功",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "uid": "2405030544"
}
```

**失败响应：**
```json
{
  "status": 0,
  "info": "验证码错误！"
}
```

---

### 刷新 Token

```
POST /api/auth/refresh
```

无需登录，利用 Refresh Token 自动获取新的 Access Token。

---

## 受保护接口

以下接口需要在请求头携带 `Authorization: Bearer <token>`。

---

### 获取当前用户

```
GET /api/auth/me
```

**响应：**
```json
{
  "status": 1,
  "data": {
    "uid": "2405030544"
  }
}
```

---

### 登出

```
POST /api/auth/logout
```

**响应：**
```json
{
  "status": 1,
  "info": "已退出"
}
```

---

### 获取单周课表

```
GET /api/schedule?week=7
```

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `week` | int | 否 | 周次（1~20），不传默认当前周 |

**响应：**
```json
{
  "status": 1,
  "data": {
    "semester": "2025-2026第2学期",
    "className": "2024人工智能技术应用5班",
    "studentName": "陈雨钒",
    "week": 7,
    "courses": [
      {
        "name": "深度学习及应用",
        "teacher": "陈政廷",
        "room": "F505信息安全实训室(本部教学楼)",
        "dayOfWeek": 2,
        "periodStart": 1,
        "periods": 4
      }
    ]
  }
}
```

**字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `dayOfWeek` | int | 星期几（1=周一，7=周日） |
| `periodStart` | int | 第几节课开始（1~12） |
| `periods` | int | 跨几节课 |

---

### 获取全学期课表

```
GET /api/schedule/full?maxWeek=20
```

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `maxWeek` | int | 否 | 最大周次，默认 20 |

**响应：**
```json
{
  "status": 1,
  "data": {
    "semester": "2025-2026第2学期",
    "className": "2024人工智能技术应用5班",
    "studentName": "陈雨钒",
    "currentWeek": 7,
    "totalWeeks": 20,
    "fetchedWeeks": 20,
    "courses": [
      {
        "name": "深度学习及应用",
        "teacher": "陈政廷",
        "room": "F505信息安全实训室(本部教学楼)",
        "dayOfWeek": 2,
        "periodStart": 1,
        "periods": 4,
        "weeks": [1, 2, 3, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16]
      }
    ]
  }
}
```

> `weeks` 数组表示该课程在哪些周有课。

---

### 获取成绩

```
GET /api/score?semester=2025-2026-2
```

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `semester` | string | 否 | 学期（如 `2025-2026-2`），不传返回全部 |

**响应：**
```json
{
  "status": 1,
  "data": [
    {
      "year": "2025-2026",
      "semester": "2",
      "className": "2024人工智能技术应用5班",
      "course": "深度学习及应用",
      "nature": "必修",
      "credit": 3.0,
      "teacher": "陈政廷",
      "grade": 87,
      "gpa": 3.70,
      "type": "正常"
    }
  ]
}
```

---

### 获取可选学期列表

```
GET /api/score/semesters
```

**响应：**
```json
{
  "status": 1,
  "data": ["2025-2026-2", "2025-2026-1", "2024-2025-2", "2024-2025-1"]
}
```

---

## 课节时间说明

一天划分为以下课节：

| 课节 | 时间 |
|------|------|
| 第1~4节 | 上午 |
| 中午1 / 中午2 | 午休（不排课） |
| 第5~8节 | 下午 |
| 晚上1~4节 | 晚上 |
