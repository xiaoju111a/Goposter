# YgoCard Mail - API 文档

欢迎使用 YgoCard Mail API 文档。本文档旨在为开发者提供与 YgoCard Mail 后端服务进行交互所需的所有信息。

---

## 1. 认证

YgoCard Mail API 使用 **JWT (JSON Web Tokens)** 进行认证。所有需要认证的请求都必须在 HTTP Header 中包含一个有效的 `Authorization` 令牌。

- **Header:** `Authorization`
- **Value:** `Bearer <YOUR_JWT_TOKEN>`

### **认证流程**

1.  客户端使用邮箱和密码调用 `POST /api/login` 端点。
2.  服务器验证凭据。
    - 如果凭据有效且未启用2FA，服务器将返回一个 `access_token` 和一个 `refresh_token`。
    - 如果启用了2FA，服务器将返回一个临时令牌，并要求进行第二因素验证。
3.  客户端在后续请求的 `Authorization` Header 中携带 `access_token`。
4.  `access_token` 过期后，客户端可使用 `refresh_token` 获取新的令牌（刷新逻辑由前端自动处理）。

---

## 2. API 端点详解

### **2.1 认证 (Authentication)**

#### **`POST /api/login`**

用户登录并获取 JWT 令牌。

- **请求体 (Request Body):**
  ```json
  {
    "email": "admin@ygocard.org",
    "password": "admin123"
  }
  ```
- **成功响应 (200 OK):**
  ```json
  {
    "status": "success",
    "token": "ey...",
    "refresh_token": "ey...",
    "message": "登录成功"
  }
  ```
- **2FA 要求响应 (202 OK):**
  ```json
  {
    "status": "2fa_required",
    "message": "请输入两步验证码",
    "temp_token": "ey..."
  }
  ```
- **错误响应 (401 Unauthorized):**
  ```json
  {
    "status": "error",
    "message": "邮箱或密码错误"
  }
  ```

---

### **2.2 两步验证 (Two-Factor Authentication)**

#### **`GET /api/2fa/setup`**

为当前登录用户生成一个新的 2FA 设置。返回一个包含二维码图像（Base64编码）和密钥的对象。

- **认证:** 需要 (Bearer Token)
- **成功响应 (200 OK):**
  ```json
  {
    "status": "success",
    "secret": "JBSWY3DPEHPK3PXP",
    "qr_code": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA..."
  }
  ```

#### **`POST /api/2fa/verify`**

验证用户提供的 2FA 验证码。

- **认证:** 需要 (Bearer Token 或 登录时的 `temp_token`)
- **请求体 (Request Body):**
  ```json
  {
    "token": "123456" // 用户从 Authenticator 应用获取的6位数字码
  }
  ```
- **成功响应 (200 OK, 启用2FA后):**
  ```json
  {
    "status": "success",
    "message": "2FA ��成功启用"
  }
  ```
- **成功响应 (200 OK, 登录验证):**
  ```json
  {
    "status": "success",
    "token": "ey...", // 正式的 access_token
    "refresh_token": "ey...",
    "message": "登录成功"
  }
  ```
- **错误响应 (401 Unauthorized):**
  ```json
  {
    "status": "error",
    "message": "无效的验证码"
  }
  ```

---

### **2.3 邮箱管理 (Mailbox Management)**

#### **`GET /api/mailboxes`**

获取当前用户的所有邮箱列表。

- **认证:** 需要 (Bearer Token)
- **成功响应 (200 OK):**
  ```json
  [
    {
      "name": "INBOX",
      "unread_count": 5,
      "total_count": 120
    },
    {
      "name": "Sent",
      "unread_count": 0,
      "total_count": 42
    }
  ]
  ```

#### **`GET /api/mailboxes/{mailbox_name}/emails`**

获取指定邮箱内的邮件列表。

- **认证:** 需要 (Bearer Token)
- **URL 参数:**
  - `mailbox_name`: 邮箱名称 (例如: `INBOX`, `Sent`)
- **查询参数 (Query Parameters):**
  - `page` (可选): 页码，默认为 `1`。
  - `limit` (可选): 每页数量，默认为 `50`。
- **成功响应 (200 OK):**
  ```json
  {
    "emails": [
      {
        "id": "1678886400.1.1",
        "from": "sender@example.com",
        "to": ["user@ygocard.org"],
        "subject": "会议邀请",
        "snippet": "下周三下午2点...",
        "date": "2025-07-16T12:00:00Z",
        "is_read": false,
        "has_attachment": true
      }
    ],
    "total_pages": 10,
    "current_page": 1
  }
  ```

---

### **2.4 邮件操作 (Email Operations)**

#### **`GET /api/emails/{email_id}`**

获取单封邮件的完整内容。

- **认证:** 需要 (Bearer Token)
- **URL 参数:**
  - `email_id`: 邮件的唯一ID。
- **成功响应 (200 OK):**
  ```json
  {
    "id": "1678886400.1.1",
    "from": "sender@example.com",
    "to": ["user@ygocard.org"],
    "subject": "会议邀请",
    "body_html": "<html><body><p>详细内容...</p></body></html>",
    "body_text": "详细内容...",
    "date": "2025-07-16T12:00:00Z",
    "is_read": true,
    "attachments": [
      {
        "filename": "report.pdf",
        "content_type": "application/pdf",
        "size": 102400,
        "download_url": "/api/emails/1678886400.1.1/attachments/report.pdf"
      }
    ]
  }
  ```

#### **`POST /api/send`**

发送一封新邮件。

- **认证:** 需要 (Bearer Token)
- **请求体 (Request Body):**
  ```json
  {
    "from": "user@ygocard.org",
    "to": ["recipient1@example.com", "recipient2@example.com"],
    "subject": "Hello World",
    "body": "这是一封测试邮件。"
  }
  ```
- **成功响应 (200 OK):**
  ```json
  {
    "status": "success",
    "message": "邮件已成功发送"
  }
  ```
- **错误响应 (400 Bad Request):**
  ```json
  {
    "status": "error",
    "message": "收件人地址无效"
  }
  ```

---

## 3. 管理员端点 (Admin Endpoints)

以下端点需要管理员权限。

#### **`GET /api/admin/users`**
获取所有用户列表。

#### **`POST /api/admin/users`**
创建一个新用户。

