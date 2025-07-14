# SMTP中继快速启动指南

## 功能概述

您的邮箱服务器现已集成完整的SMTP中继功能，支持通过腾讯云邮件推送等第三方服务发送外部邮件。

## 主要功能

1. **智能路由**: 自动识别内部/外部邮件，内部邮件直接投递，外部邮件通过中继发送
2. **多提供商支持**: 预配置腾讯云SES、QQ邮箱、163/126邮箱、Gmail等
3. **Web界面管理**: 通过友好的Web界面配置和管理SMTP中继
4. **连接测试**: 实时测试SMTP中继连接状态
5. **详细日志**: 完整的错误处理和性能监控日志
6. **备用策略**: 直连失败时自动尝试中继发送

## 快速配置步骤

### 1. 启动邮箱服务器
```bash
# 开发模式（端口2525）
./mailserver ygocard.org localhost 2525 1143 8080

# 生产模式（端口25，需要sudo权限）
sudo ./mailserver ygocard.org your-hostname.com 25 143 8080
```

### 2. 访问Web界面
打开浏览器访问: http://localhost:8080

### 3. 配置SMTP中继

#### 使用腾讯云邮件推送（推荐）
1. 在Web界面找到"SMTP中继配置"部分
2. 从下拉菜单选择"腾讯云邮件推送"
3. 填写您的SMTP用户名和密码
4. 勾选"启用SMTP中继"
5. 点击"测试连接"验证配置
6. 点击"保存配置"

#### 手动配置
1. 填写SMTP服务器信息：
   - **SMTP主机**: smtp.qcloudmail.com
   - **端口**: 587
   - **用户名**: 您的SMTP用户名
   - **密码**: 您的SMTP密码
   - **使用TLS**: 勾选
2. 勾选"启用SMTP中继"
3. 保存配置

### 4. 测试邮件发送
1. 在Web界面的"发送邮件"部分
2. 填写邮件信息并发送外部邮件
3. 查看服务器日志确认中继工作正常

## 配置文件

SMTP中继配置自动保存在 `./data/smtp_relay.json`，格式如下：

```json
{
  "enabled": true,
  "host": "smtp.qcloudmail.com",
  "port": 587,
  "username": "your-smtp-user@yourdomain.com",
  "password": "your-smtp-password",
  "use_tls": true
}
```

## API接口

### 获取中继配置
```bash
curl http://localhost:8080/api/relay/config
```

### 更新中继配置
```bash
curl -X POST http://localhost:8080/api/relay/config \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "host": "smtp.qcloudmail.com",
    "port": 587,
    "username": "user@domain.com",
    "password": "password",
    "use_tls": true
  }'
```

### 测试中继连接
```bash
curl -X POST http://localhost:8080/api/relay/test
```

### 获取中继状态
```bash
curl http://localhost:8080/api/relay/status
```

## 日志监控

所有SMTP中继活动都会记录详细日志，包括：
- 连接建立时间和耗时
- 认证状态
- 邮件发送状态和大小
- 错误信息和故障排除

查看日志：
```bash
tail -f server.log | grep "SMTP中继"
```

## 故障排除

### 连接测试失败
1. 检查网络连接和防火墙设置
2. 确认SMTP服务器地址和端口
3. 验证用户名和密码
4. 检查TLS设置

### 邮件发送失败
1. 查看详细错误日志
2. 确认发件人地址在验证域名内
3. 检查是否超出发送限制
4. 验证收件人地址有效性

### 邮件进入垃圾箱
1. 配置正确的DNS记录（SPF、DKIM、DMARC）
2. 使用合适的发件人地址
3. 避免垃圾邮件关键词
4. 建立良好的发送信誉

## 性能优化

1. **连接复用**: 系统自动管理SMTP连接
2. **超时控制**: 10秒连接超时避免长时间等待
3. **错误重试**: 支持多MX记录和备用策略
4. **异步发送**: Web界面发送为异步处理

## 安全建议

1. 使用强密码并定期更换
2. 限制SMTP中继的发送权限
3. 监控发送日志防止滥用
4. 定期备份配置文件
5. 使用HTTPS访问Web界面（生产环境）

## 扩展功能

您可以通过修改代码添加更多SMTP提供商：
1. 在 `smtp_relay.go` 的 `PresetConfigs` 中添加配置
2. 在Web界面的下拉菜单中添加选项
3. 重新编译和部署

## 技术支持

如遇问题请查看：
1. `server.log` 服务器日志
2. `TENCENT_SES_GUIDE.md` 腾讯云配置指南
3. 各邮件服务商的官方文档