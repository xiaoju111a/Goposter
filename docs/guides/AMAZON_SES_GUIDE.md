# 亚马逊SES(Simple Email Service)配置指南

## 概述

亚马逊SES是Amazon Web Services(AWS)提供的云端邮件发送服务，具有高可靠性、高扩展性和成本效益。本指南将帮助您配置邮箱服务器使用亚马逊SES作为SMTP中继。

## 前置条件

1. ✅ AWS账号（已完成）
2. ✅ 已创建SES SMTP凭证（已完成）
3. ✅ 已验证发送域名或邮箱地址
4. ✅ 账号脱离沙盒模式（用于发送给未验证邮箱）

## 已配置信息

### 当前SES配置
- **SMTP端点**: email-smtp.us-east-1.amazonaws.com
- **端口**: 587 (TLS)
- **用户名**: AKIAXA7QUP4N5BIV7
- **密码**: BDm9CIr6aVzWuVcH/4SPcpCBUcXMBT9YMUL
- **状态**: ✅ 已验证连接成功

### 支持的AWS区域

您的邮箱服务器支持以下AWS SES区域：

| 区域 | SMTP端点 |
|------|----------|
| US East (N. Virginia) | email-smtp.us-east-1.amazonaws.com |
| US West (Oregon) | email-smtp.us-west-2.amazonaws.com |
| Europe (Ireland) | email-smtp.eu-west-1.amazonaws.com |
| Asia Pacific (Singapore) | email-smtp.ap-southeast-1.amazonaws.com |

## 配置步骤


### 1. 直接编辑配置文件

编辑 `./backend/data/smtp_relay.json`：

```json
{
  "enabled": true,
  "host": "email-smtp.us-east-1.amazonaws.com",
  "port": 587,
  "username": "AKIAXA7QUN5B3XIV7",
  "password": "BDm9CIr6aVzWuVcH/4aPBUPUG3xcXMBT9YMUL",
  "use_tls": true
}
```

## 发送限制

### 沙盒模式限制
- 只能发送给已验证的邮箱地址
- 每日发送限额: 200封
- 最大发送速率: 1封/秒

### 生产模式限制
- 可发送给任何有效邮箱地址
- 发送限额: 根据账号等级递增
- 发送速率: 可申请提升

## 验证域名

### 1. 在AWS SES控制台验证域名

1. 登录 [AWS SES控制台](https://console.aws.amazon.com/ses/)
2. 选择正确的区域 (us-east-1)
3. 点击"Verified identities" > "Create identity"
4. 选择"Domain"，输入您的域名 (如: ygocard.org)
5. 启用DKIM签名
6. 记录DNS配置要求

### 2. DNS配置

在您的DNS提供商添加以下记录：

#### MX记录 (如果要接收邮件)
```
记录类型: MX
主机名: @
值: 10 mail.ygocard.org
```

#### SPF记录
```
记录类型: TXT
主机名: @
值: v=spf1 include:amazonses.com ~all
```

#### DKIM记录
AWS会提供3个DKIM记录，类似：
```
记录类型: CNAME
主机名: abc123def._domainkey
值: abc123def.dkim.amazonses.com
```

#### DMARC记录
```
记录类型: TXT
主机名: _dmarc
值: v=DMARC1; p=none; rua=mailto:dmarc@ygocard.org
```

## 监控和日志

### 1. 查看发送日志
```bash
tail -f server.log | grep "SMTP中继"
```

### 2. SES控制台监控
- 发送统计
- 退信和投诉
- 信誉指标

## 故障排除

### 常见错误

#### 1. 认证失败
```
错误: SMTP认证失败
解决: 检查用户名和密码是否正确
```

#### 2. 发送被拒绝
```
错误: MessageRejected
原因: 邮箱地址未验证（沙盒模式）
解决: 验证收件人邮箱或申请脱离沙盒
```

#### 3. 发送限额超出
```
错误: Sending quota exceeded
解决: 等待配额重置或申请提升限额
```

### 调试命令

#### 测试DNS记录
```bash
# SPF记录
nslookup -type=TXT ygocard.org

# DKIM记录
nslookup -type=CNAME abc123def._domainkey.ygocard.org

# DMARC记录
nslookup -type=TXT _dmarc.ygocard.org
```

#### 测试SMTP连接
```bash
# 使用内置测试工具
go run test_ses.go

# 或通过API
curl -X POST http://localhost:8080/api/relay/test
```

## 安全最佳实践

1. **保护SMTP凭证**
   - 不要在代码中硬编码凭证
   - 定期轮换访问密钥
   - 使用最小权限原则

2. **监控使用情况**
   - 设置CloudWatch告警
   - 监控退信率和投诉率
   - 定期查看发送统计

3. **配置事件发布**
   - 设置SNS通知
   - 监控退信和投诉事件
   - 自动处理无效邮箱

## 成本优化

### 定价信息 (美国东部)
- 前1000封邮件: $0.10
- 之后每1000封: $0.10
- 附件存储: $0.023/GB/月

### 优化建议
1. 定期清理无效邮箱列表
2. 使用bounce handling自动处理退信
3. 监控发送统计避免浪费配额

## API参考

### 获取中继状态
```bash
GET /api/relay/status
```

### 更新配置
```bash
POST /api/relay/config
Content-Type: application/json

{
  "enabled": true,
  "host": "email-smtp.us-east-1.amazonaws.com",
  "port": 587,
  "username": "YOUR_USERNAME",
  "password": "YOUR_PASSWORD",
  "use_tls": true
}
```

## 相关链接

- [AWS SES开发者指南](https://docs.aws.amazon.com/ses/)
- [SES SMTP接口](https://docs.aws.amazon.com/ses/latest/dg/send-email-smtp.html)
- [域名验证](https://docs.aws.amazon.com/ses/latest/dg/verify-domains.html)
- [脱离沙盒模式](https://docs.aws.amazon.com/ses/latest/dg/request-production-access.html)