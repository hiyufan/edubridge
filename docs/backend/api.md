# 后端 API 参考

## 基础信息

- **Base URL**: `http://localhost:3000`（开发环境）
- **Content-Type**: `application/json`（除验证码接口外）
- **认证方式**: JWT Bearer Token

## 认证机制

### 双 Token 流程

```
登录 → Access Token (2h) + Refresh Token (30d in HttpOnly Cookie)
         ↓
    API 请求           → 401 → 自动用 Refresh Token 续期
```

### Token 续期

```bash
POST /api/auth/refresh
Cookie: refreshToken=<token>
```

成功返回新的 Access Token。

---

## 公开接口

### GET /api/captcha

获取验证码图片。

**响应**: `image/png` 二进制数据

**Headers**:
```
Set-Cookie: sessionId=<uuid>; Path=/
Content-Type: image/png
```

> 前端需从 `Set-Cookie` 中提取 `sessionId`，后续登录时作为 `sessionId` 参数传入。

---

### POST /api/auth/login

用户登录。

**请求体**:
```json
{
  "sessionId": "uuid-from-captcha",
  "username": "2405030544",
  "password": "your-password",
  "captcha": "abcd",
  "loginType": "xsxh"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `sessionId` | string | ✅ | 验证码对应的 session ID |
| `username` | string | ✅ | 学号/账号 |
| `password` | string | ✅ | 密码 |
| `captcha` | string | ✅ | 验证码（4位） |
| `loginType` | string | ❌ | 登录类型: `xsxh`(学生，默认) / `zjh`(教职工) / `gkksh`(考生) |

**成功响应**:
```json
{
  "status": 1,
  "info": "登录成功",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "uid": "2405030544",
  "expiresIn": 7200
}
```

**失败响应**:
```json
{
  "status": 0,
  "info": "验证码错误！"
}
```

---

### POST /api/auth/refresh

用 Refresh Token 续期 Access Token。

**请求**: 无需请求体，但需携带 `refreshToken` Cookie。

**成功响应**:
```json
{
  "status": 1,
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresIn": 7200
}
```

**失败响应**:
```json
{
  "status": 0,
  "info": "登录已过期，请重新登录"
}
```

---

## 受保护接口

以下接口需要在请求头携带：
```
Authorization: Bearer <access-token>
```

---

### GET /api/auth/me

获取当前登录用户信息。

**响应**:
```json
{
  "status": 1,
  "data": {
    "uid": "2405030544"
  }
}
```

---

### POST /api/auth/logout

登出。

**响应**:
```json
{
  "status": 1,
  "info": "已退出登录"
}
```

---

### GET /api/schedule

获取单周课表。

**参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `week` | int | ❌ | 周次（1~20），默认当前周 |

**响应**:
```json
{
  "status": 1,
  "data": {
    "semester": "2025-2026第2学期",
    "className": "2024人工智能技术应用5班",
    "studentName": "陈雨钒",
    "week": 7,
    "currentWeek": 7,
    "semesterStart": "2026-02-16",
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

**字段说明**:

| 字段 | 类型 | 说明 |
|------|------|------|
| `semester` | string | 学期名称 |
| `className` | string | 班级名称 |
| `studentName` | string | 学生姓名 |
| `week` | int | 教务系统周号 (DQZ) |
| `currentWeek` | int | 真实当前周（根据开学日期计算） |
| `semesterStart` | string | 学期第一周周一，格式 `YYYY-MM-DD` |
| `courses[].dayOfWeek` | int | 星期几（1=周一，7=周日） |
| `courses[].periodStart` | int | 第几节课开始（1~12） |
| `courses[].periods` | int | 跨几节课 |

---

### GET /api/schedule/full

获取全学期课表（所有周的课程汇总）。

**参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `maxWeek` | int | ❌ | 最大周次，默认 20 |

**响应**:
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

### GET /api/score

获取成绩。

**参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `semester` | string | ❌ | 学期（如 `2025-2026-2`），不传返回全部学期 |

**响应**:
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

**字段说明**:

| 字段 | 类型 | 说明 |
|------|------|------|
| `year` | string | 学年（如 `2025-2026`） |
| `semester` | string | 学期（`1` 或 `2`） |
| `course` | string | 课程名 |
| `nature` | string | 课程性质（必修/选修等） |
| `credit` | float | 学分 |
| `grade` | any | 成绩（数字或等级） |
| `gpa` | float | 绩点 |
| `type` | string | 考试类型（正常/补考等） |

---

### GET /api/score/semesters

获取可选学期列表。

**响应**:
```json
{
  "status": 1,
  "data": ["2025-2026-2", "2025-2026-1", "2024-2025-2", "2024-2025-1"]
}
```

---

### GET /api/schedule/conflicts

检测课程时间冲突（同一时段多门课程）。

**响应**:
```json
{
  "status": 1,
  "data": [
    {
      "dayOfWeek": 1,
      "periodStart": 1,
      "courses": [
        { "name": "高等数学", "room": "A101", "teacher": "张三" },
        { "name": "大学物理", "room": "B201", "teacher": "李四" }
      ]
    }
  ]
}
```

---

### GET /api/schedule/ical

生成 iCalendar 格式日历订阅地址。

**响应**: `text/calendar; charset=utf-8`

返回标准 iCalendar 格式，可导入 Google Calendar / Apple Calendar / Outlook。

---

### GET /api/webhook

查询当前用户注册的所有 Webhook。

**响应**:
```json
{
  "status": 1,
  "data": [
    {
      "id": "wh_xxx",
      "url": "https://example.com/webhook",
      "events": ["score", "schedule"],
      "createdAt": "2026-04-21T12:00:00Z"
    }
  ]
}
```

---

### POST /api/webhook

注册一个新的 Webhook 回调。

**请求体**:
```json
{
  "url": "https://example.com/webhook",
  "events": ["score", "schedule"],
  "secret": "可选签名密钥"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `url` | string | ✅ | 回调地址 |
| `events` | string[] | ✅ | 事件类型: `score`(成绩变动) / `schedule`(课表变动) |
| `secret` | string | ❌ | HMAC 签名密钥 |

**响应**:
```json
{
  "status": 1,
  "info": "Webhook 注册成功",
  "data": { "id": "wh_xxx" }
}
```

---

### DELETE /api/webhook/:id

删除指定的 Webhook。

**响应**:
```json
{
  "status": 1,
  "info": "Webhook 已删除"
}
```

---

### GET /api/webhook/info

获取当前 Webhook 的签名密钥（仅首次调用时返回密钥内容）。

**响应**:
```json
{
  "status": 1,
  "data": {
    "secret": "whs_xxx",
    "created": true
  }
}
```

---

## 通用响应格式

```json
{
  "status": 1,
  "info": "操作成功信息",
  "data": { ... }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `status` | int | `1`=成功，`0`=失败 |
| `info` | string | 操作结果描述 |
| `data` | object | 成功时返回的数据 |

---

## 错误码

| HTTP Status | info 示例 | 说明 |
|-------------|-----------|------|
| 400 | `缺少必要参数` | 请求参数缺失 |
| 401 | `Token 无效或已过期` | Token 验证失败 |
| 401 | `登录已过期，请重新登录` | Refresh Token 过期 |
| 429 | `登录尝试次数过多，请稍后再试` | 登录限流触发 |
| 500 | `获取课表失败` | 教务系统请求失败 |

---

## 健康检查

### GET /health

```json
{ "ok": 1 }
```

### GET /

```json
{ "name": "jw-server-go", "version": "1.0.0" }
```
