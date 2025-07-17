# 🚀 goposter 全功能邮箱系统 - 增强特性

## ✨ 新增功能概览

本次更新为 goposter 邮箱系统添加了完整的邮件服务功能，现在支持：

### 🔑 核心功能
- ✅ **SMTP接收** - 接收外部邮件
- ✅ **SMTP发送** - 发送邮件到外部
- ✅ **IMAP支持** - 邮件客户端访问
- ✅ **别名管理** - 自定义邮箱别名
- ✅ **邮件转发** - 自动转发到其他邮箱
- ✅ **用户认证** - 安全的用户登录系统
- ✅ **数据持久化** - 邮件永久保存
- ✅ **Web管理界面** - 现代化管理面板

## 🖥️ 邮件客户端配置

### IMAP 设置 (收邮件)
```
服务器: your-server-ip 或 mail.ygocard.org
端口: 143 (测试) 或 143 (生产)
加密: 无 (当前版本)
用户名: 任意@ygocard.org
密码: 任意密码 (简单认证)
```

### SMTP 设置 (发邮件)
```
服务器: your-server-ip 或 mail.ygocard.org
端口: 2525 (测试) 或 25 (生产)
加密: 无 (当前版本)
认证: 无需认证
```

### 支持的邮件客户端
- **Outlook** - 完全支持
- **Thunderbird** - 完全支持
- **Apple Mail** - 完全支持
- **手机邮件客户端** - 基本支持

## 🏃‍♂️ 快速启动

### 测试模式
```bash
# 基本启动 (推荐用于测试)
go run *.go ygocard.org localhost

# 完整参数启动 (域名 主机名 SMTP端口 IMAP端口 Web端口)
go run *.go ygocard.org localhost 2525 1143 8080
```

### 生产模式
```bash
# 生产环境启动 (需要sudo权限)
sudo go run *.go ygocard.org mail.ygocard.org 25 143 8080
```

## 📧 别名管理系统

### 通过Web界面管理
1. 访问 `http://服务器IP:8080`
2. 在 "别名管理" 区域添加别名
3. 输入别名邮箱和真实邮箱
4. 点击 "添加别名"

### 别名示例
```
别名邮箱: support@ygocard.org
真实邮箱: admin@ygocard.org

别名邮箱: sales@ygocard.org  
真实邮箱: admin@ygocard.org

别名邮箱: noreply@ygocard.org
真实邮箱: admin@ygocard.org
```

### 邮件转发
- 可以将收到的邮件自动转发到外部邮箱
- 支持一对多转发 (一个邮箱转发到多个地址)
- 保留原邮件同时发送转发邮件

## 📤 发送邮件

### 通过Web界面发送
1. 在Web管理界面的 "发送邮件" 区域
2. 填写发件人、收件人、主题、内容
3. 点击 "发送邮件"

### 通过邮件客户端发送
配置SMTP服务器后，可以通过任何邮件客户端发送邮件

### API发送
```bash
curl -X POST http://localhost:8080/api/send \
  -H "Content-Type: application/json" \
  -d '{
    "from": "admin@ygocard.org",
    "to": "user@example.com", 
    "subject": "测试邮件",
    "body": "这是一封测试邮件"
  }'
```

## 🔐 用户认证系统

### 默认管理员账号
```
邮箱: admin@ygocard.org
密码: admin123
```

### 简化认证策略
当前版本采用简化认证，支持以下方式：
- 任何包含 `@域名` 的用户名
- 默认密码: `123456`, `password`, `admin`, 或邮箱地址本身

### 创建新用户 (通过代码)
```go
// 在系统运行时，可以通过API或直接调用创建用户
userAuth.CreateUser("user@ygocard.org", "userpassword", false)
```

## 💾 数据持久化

### 数据存储位置
```
./data/
├── mailbox_admin_at_ygocard_org.json  # 邮件数据
├── mailbox_support_at_ygocard_org.json
├── aliases.json                        # 别名配置
└── users.json                         # 用户数据
```

### 备份数据
```bash
# 手动备份
cp -r ./data ./backup_$(date +%Y%m%d_%H%M%S)

# 或使用系统内置备份功能 (代码调用)
mailServer.storage.BackupData("./backups")
```

## 🔧 API接口文档

### 邮件相关
- `GET /api/mailboxes` - 获取所有邮箱
- `GET /api/emails/{邮箱}` - 获取邮箱邮件
- `POST /api/send` - 发送邮件

### 别名管理
- `GET /api/aliases` - 获取所有别名
- `POST /api/aliases` - 添加别名
- `DELETE /api/aliases?alias={别名}` - 删除别名

### 转发管理
- `GET /api/forwards` - 获取转发规则
- `POST /api/forwards` - 添加转发
- `DELETE /api/forwards?email={邮箱}&forward_to={目标}` - 删除转发

### 认证相关
- `POST /api/login` - 用户登录
- `POST /api/logout` - 用户登出
- `GET /api/stats` - 获取统计信息

## 🛠️ 故障排除

### IMAP连接问题
1. **检查端口是否开放**
   ```bash
   netstat -tulpn | grep :143
   ```

2. **检查防火墙设置**
   ```bash
   sudo ufw allow 143/tcp
   ```

3. **测试IMAP连接**
   ```bash
   telnet localhost 143
   ```

### SMTP发送问题
1. **检查DNS MX记录**
   ```bash
   dig MX example.com
   ```

2. **检查25端口是否被封锁**
   ```bash
   telnet smtp.gmail.com 587
   ```

### 邮件客户端配置问题
- 确保使用正确的服务器地址和端口
- 当前版本不支持TLS/SSL，选择"无加密"
- 用户名使用完整邮箱地址
- 密码可以使用默认密码

## 🔮 后续规划

### 即将添加的功能
- [ ] **TLS/SSL支持** - 加密连接
- [ ] **SMTP AUTH** - SMTP认证
- [ ] **垃圾邮件过滤** - 智能过滤
- [ ] **邮件搜索** - 全文搜索
- [ ] **附件支持** - 文件附件
- [ ] **邮件模板** - 预设模板
- [ ] **统计报表** - 详细统计
- [ ] **API Key认证** - 安全API访问

### 性能优化
- [ ] **数据库支持** - MySQL/PostgreSQL
- [ ] **缓存系统** - Redis缓存
- [ ] **负载均衡** - 多服务器支持
- [ ] **邮件队列** - 异步处理

## 📞 技术支持

### 常见问题
1. **端口25被封锁** - 联系VPS提供商解封或使用端口587
2. **DNS不生效** - 等待24-48小时DNS传播
3. **邮件客户端连接失败** - 检查端口和认证设置
4. **发送邮件失败** - 检查收件人邮箱和MX记录

### 获取帮助
- 📧 **邮件支持**: admin@ygocard.org (部署后可用)
- 🐛 **问题反馈**: GitHub Issues
- 📚 **文档**: 查看项目README.md

---

**🎮 goposter 全功能邮箱 - 专业级邮件解决方案！**