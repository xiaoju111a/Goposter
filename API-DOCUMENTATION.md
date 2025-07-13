# FreeAgent 邮箱服务器 API 文档

## 项目概述

FreeAgent 邮箱服务器是一个基于 Go 语言开发的现代化邮箱系统，配备 React 前端管理界面。系统支持完整的邮件收发、邮箱管理、SMTP 中继等功能。

### 技术栈
- **后端**: Go (Golang) - 邮件服务器核心
- **前端**: React + Vite - 管理界面  
- **协议**: SMTP、IMAP
- **中继**: Amazon SES SMTP 中继
- **域名**: freeagent.live

## 项目结构

```
mail/
├── 后端 Go 服务器
│   ├── main.go              # 主服务器程序
│   ├── smtp_sender.go       # SMTP 发送功能
│   ├── smtp_relay.go        # SMTP 中继管理
│   ├── imap.go             # IMAP 服务器
│   ├── email_parser.go      # 邮件解析
│   ├── storage.go          # 数据存储
│   ├── auth.go             # 用户认证
│   ├── mailbox_manager.go   # 邮箱管理
│   ├── alias.go            # 邮箱别名
│   ├── email_auth.go       # 邮件认证
│   └── relay_config.go     # 中继配置
│
├── 前端 React 应用
│   ├── src/
│   │   ├── App.jsx         # 主应用组件
│   │   ├── components/     # React 组件
│   │   │   ├── MailboxCard.jsx    # 邮箱卡片
│   │   │   ├── EmailItem.jsx      # 邮件项目
│   │   │   ├── SendEmail.jsx      # 发送邮件
│   │   │   ├── CreateMailbox.jsx  # 创建邮箱
│   │   │   └── Stats.jsx          # 统计信息
│   │   ├── utils/          # 工具类
│   │   │   ├── api.js      # API 接口
│   │   │   └── cache.js    # 缓存管理
│   │   ├── index.css       # 样式文件
│   │   └── main.jsx        # 入口文件
│   └── vite.config.js      # Vite 配置
│
└── 配置文件
    ├── data/               # 数据存储目录
    └── 各种配置和文档文件
```

## 服务架构

### 端口配置
- **SMTP 服务**: 端口 25 (接收邮件)
- **IMAP 服务**: 端口 143 (客户端访问)
- **API 后端**: 端口 9090 (REST API)
- **React 前端**: 端口 8080 (Web 界面)

### 数据流
```
用户浏览器 (8080) → React 前端 → API 代理 → Go 后端 (9090)
                                              ↓
外部邮件 → SMTP (25) → Go 邮件处理 → 存储
                                              ↓
邮件客户端 ← IMAP (143) ← Go IMAP 服务 ← 存储
```

## API 接口文档

### 基础信息
- **Base URL**: `http://localhost:9090/api`
- **Content-Type**: `application/json`
- **CORS**: 已启用，允许跨域访问

### 1. 邮箱管理 API

#### 1.1 获取邮箱列表
```http
GET /api/mailboxes
```

**响应示例:**
```json
[
  "admin@freeagent.live",
  "support@freeagent.live", 
  "info@freeagent.live"
]
```

**cURL 示例:**
```bash
curl -X GET http://localhost:9090/api/mailboxes
```

#### 1.2 创建邮箱
```http
POST /api/mailboxes/create
```

**请求体:**
```json
{
  "username": "newuser",
  "password": "secure123",
  "description": "新用户邮箱"
}
```

**响应示例:**
```json
{
  "status": "success",
  "message": "邮箱创建成功",
  "email": "newuser@freeagent.live"
}
```

**cURL 示例:**
```bash
curl -X POST http://localhost:9090/api/mailboxes/create \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "mypassword123",
    "description": "测试邮箱"
  }'
```

#### 1.3 邮箱管理
```http
GET/POST/DELETE /api/mailboxes/manage
```

支持邮箱的增删改查操作。

### 2. 邮件管理 API

#### 2.1 获取邮箱邮件
```http
GET /api/emails/{mailbox}
```

**路径参数:**
- `mailbox`: 邮箱地址，需要 URL 编码

**响应示例:**
```json
[
  {
    "ID": "1",
    "From": "sender@example.com",
    "To": "admin@freeagent.live",
    "Subject": "测试邮件",
    "Body": "邮件内容...",
    "Date": "2025-01-13T10:30:00Z",
    "Size": 1024
  }
]
```

**cURL 示例:**
```bash
curl -X GET "http://localhost:9090/api/emails/admin%40freeagent.live"
```

#### 2.2 删除邮件
```http
DELETE /api/emails/delete/{mailbox}/{emailId}
```

**路径参数:**
- `mailbox`: 邮箱地址，需要 URL 编码
- `emailId`: 邮件 ID

**cURL 示例:**
```bash
curl -X DELETE "http://localhost:9090/api/emails/delete/admin%40freeagent.live/1"
```

#### 2.3 发送邮件
```http
POST /api/send
```

**请求体:**
```json
{
  "from": "admin@freeagent.live",
  "to": "recipient@example.com",
  "subject": "邮件主题",
  "body": "邮件内容"
}
```

**响应示例:**
```json
{
  "status": "success",
  "message": "邮件发送成功"
}
```

**cURL 示例:**
```bash
curl -X POST http://localhost:9090/api/send \
  -H "Content-Type: application/json" \
  -d '{
    "from": "admin@freeagent.live",
    "to": "test@example.com",
    "subject": "测试邮件",
    "body": "这是一封测试邮件"
  }'
```

### 3. 系统统计 API

#### 3.1 获取系统统计
```http
GET /api/stats
```

**响应示例:**
```json
{
  "total_mailboxes": 7,
  "total_emails": 15,
  "storage_used": "2.5MB",
  "uptime": "2h30m"
}
```

**cURL 示例:**
```bash
curl -X GET http://localhost:9090/api/stats
```

### 4. 用户认证 API

#### 4.1 登录
```http
POST /api/login
```

**请求体:**
```json
{
  "username": "admin",
  "password": "admin123"
}
```

**响应示例:**
```json
{
  "status": "success",
  "session_id": "abc123xyz",
  "user": "admin@freeagent.live"
}
```

#### 4.2 登出
```http
POST /api/logout
```

**请求头:**
```
Session-ID: abc123xyz
```

### 5. SMTP 中继 API

#### 5.1 获取中继状态
```http
GET /api/relay/status
```

**响应示例:**
```json
{
  "enabled": true,
  "provider": "Amazon SES",
  "server": "email-smtp.us-east-1.amazonaws.com:587",
  "status": "connected"
}
```

#### 5.2 获取中继配置
```http
GET /api/relay/config
```

#### 5.3 更新中继配置
```http
POST /api/relay/config
```

**请求体:**
```json
{
  "enabled": true,
  "server": "email-smtp.us-east-1.amazonaws.com",
  "port": 587,
  "username": "AKIA...",
  "password": "BG8X..."
}
```

#### 5.4 测试中继连接
```http
POST /api/relay/test
```

#### 5.5 获取支持的中继提供商
```http
GET /api/relay/providers
```

### 6. DNS 配置 API

#### 6.1 获取 DNS 配置
```http
GET /api/dns/config
```

**响应示例:**
```json
{
  "domain": "freeagent.live",
  "mx_record": "mail.freeagent.live",
  "spf_record": "v=spf1 a mx ~all",
  "dkim_enabled": true
}
```

## React 前端功能

### 页面结构
1. **📮 邮箱管理**: 查看所有邮箱和邮件
2. **📧 发送邮件**: 发送新邮件
3. **✉️ 创建邮箱**: 创建新的邮箱账户

### 组件说明

#### App.jsx - 主应用
- 管理全局状态和路由
- 三个标签页切换
- 统一的数据刷新机制

#### MailboxCard.jsx - 邮箱卡片
- 显示单个邮箱信息
- 邮件列表展示
- 邮件删除功能

#### SendEmail.jsx - 发送邮件
- 表单验证
- 支持自定义发件人
- 错误处理和状态反馈

#### CreateMailbox.jsx - 创建邮箱
- 用户名验证（仅支持字母数字和特定符号）
- 自动添加 @freeagent.live 后缀
- 密码生成器

#### Stats.jsx - 统计组件
- 实时显示系统统计
- 邮箱数量和邮件数量

### API 工具类 (api.js)

提供统一的 API 调用接口：

```javascript
import { api, sendEmail, createMailbox } from './utils/api.js';

// 获取邮箱列表
const mailboxes = await api.getMailboxes();

// 获取邮件
const emails = await api.getEmails('admin@freeagent.live');

// 发送邮件
await sendEmail({
  from: 'admin@freeagent.live',
  to: 'test@example.com',
  subject: '测试',
  body: '内容'
});

// 创建邮箱
await createMailbox({
  username: 'newuser',
  password: 'password123',
  description: '描述'
});
```

## 部署和运行

### 启动命令

1. **启动 Go 后端服务器:**
```bash
cd /root/mail
sudo go run *.go freeagent.live mail.freeagent.live 25 143 9090
```

2. **启动 React 前端:**
```bash
cd /root/mail/frontend
npm run dev -- --port 8080 --host 0.0.0.0
```

### 访问地址
- **Web 管理界面**: http://localhost:8080
- **API 文档测试**: http://localhost:9090/api/

### 环境要求
- Go 1.21+
- Node.js 18+
- 网络端口: 25, 143, 8080, 9090

## 安全配置

### 默认账户
- **管理员邮箱**: admin@freeagent.live
- **用户名**: admin (会自动转换为 admin@freeagent.live)
- **默认密码**: admin123

**登录方式:**
- 可以使用用户名 `admin` 或完整邮箱 `admin@freeagent.live`
- 密码统一为 `admin123`

### DNS 配置要求
```
# A 记录
mail.freeagent.live -> [服务器IP]

# MX 记录  
freeagent.live -> mail.freeagent.live (优先级10)

# SPF 记录
freeagent.live -> "v=spf1 a mx ~all"
```

### 防火墙端口
- 25 (SMTP)
- 143 (IMAP) 
- 8080 (Web界面)
- 9090 (API)

## 错误码说明

| HTTP 状态码 | 说明 |
|------------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 404 | 资源不存在 |
| 405 | 方法不允许 |
| 500 | 服务器内部错误 |

## 代码统计

- **总代码行数**: 5,811 行
- **Go 后端**: 5,160 行
- **React 前端**: 651 行
- **配置文件**: 22 个

## 特性总结

✅ **完整邮件系统**
- SMTP 接收/发送
- IMAP 客户端支持
- 邮箱别名系统

✅ **现代化前端**
- React + Vite
- 响应式设计
- 实时数据更新

✅ **强大的 API**
- RESTful 设计
- 完整的邮箱管理
- SMTP 中继支持

✅ **生产就绪**
- Amazon SES 集成
- DNS 配置支持
- 安全认证机制

---

**文档版本**: 1.0  
**更新时间**: 2025-01-13  
**联系方式**: admin@freeagent.live