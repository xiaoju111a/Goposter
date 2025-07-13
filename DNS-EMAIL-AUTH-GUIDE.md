# 📧 邮件验证DNS配置指南

## 🎯 目标
通过配置SPF、DKIM、DMARC记录来解决外部邮件发送问题，提高邮件送达率和信誉度。

## 🔍 问题原因
外部邮件发送失败的主要原因：
1. ❌ **缺少SPF记录** - 邮件服务商无法验证发送方身份
2. ❌ **没有DKIM签名** - 邮件缺乏数字签名验证
3. ❌ **没有DMARC策略** - 缺少邮件策略指导

## 🚀 解决方案

### 第一步：获取DNS配置信息
```bash
# 访问API获取配置
curl http://localhost:8080/api/dns/config
```

### 第二步：配置DNS记录

#### 2.1 SPF记录 (发送方策略框架)
```dns
类型: TXT
主机记录: @
记录值: v=spf1 a mx ip4:[你的服务器IP] ~all
TTL: 600
```

**示例**:
```
ygocard.org TXT "v=spf1 a mx ip4:1.2.3.4 ~all"
```

#### 2.2 DKIM记录 (数字签名)
```dns
类型: TXT
主机记录: default._domainkey
记录值: v=DKIM1;k=rsa;p=[公钥内容]
TTL: 600
```

**注意**: 系统会自动生成DKIM密钥对，通过API获取完整的DKIM记录。

#### 2.3 DMARC记录 (邮件验证策略)
```dns
类型: TXT
主机记录: _dmarc
记录值: v=DMARC1; p=none; rua=mailto:dmarc@ygocard.org; ruf=mailto:dmarc@ygocard.org; fo=1
TTL: 600
```

### 第三步：DNS配置验证

#### 验证SPF记录
```bash
dig TXT ygocard.org | grep spf
```

#### 验证DKIM记录
```bash
dig TXT default._domainkey.ygocard.org
```

#### 验证DMARC记录
```bash
dig TXT _dmarc.ygocard.org
```

### 第四步：重启邮件服务器

```bash
# 停止当前服务器
sudo pkill -f "go run"

# 重新启动
sudo go run *.go ygocard.org localhost 25 143 8080
```

## 📋 各DNS提供商配置方法

### 阿里云DNS
1. 登录 [阿里云DNS控制台](https://dns.console.aliyun.com)
2. 找到 `ygocard.org` 域名，点击"解析设置"
3. 点击"添加记录"，逐一添加上述三个TXT记录

### 腾讯云DNSPod
1. 登录 [DNSPod控制台](https://console.dnspod.cn)
2. 找到 `ygocard.org`，点击域名进入解析页面
3. 点击"添加记录"，选择TXT类型

### Cloudflare
1. 登录 [Cloudflare控制台](https://dash.cloudflare.com)
2. 选择 `ygocard.org` 域名
3. 进入"DNS"标签页，添加TXT记录

### GoDaddy
1. 登录 [GoDaddy管理中心](https://dcc.godaddy.com)
2. 点击"DNS" → "Manage Zones"
3. 找到 `ygocard.org`，添加TXT记录

## 🎉 预期效果

配置完成后，你的邮件系统将具备：

✅ **SPF验证** - 邮件服务商可以验证发送方IP
✅ **DKIM签名** - 每封邮件都包含数字签名
✅ **DMARC策略** - 提供邮件处理指导
✅ **提高送达率** - 减少进入垃圾箱的概率
✅ **建立信誉** - 逐步建立发送方信誉

## 🔧 测试验证

### 测试邮件发送
```bash
curl -X POST http://localhost:8080/api/send \
  -H "Content-Type: application/json" \
  -d '{"from":"admin@ygocard.org","to":"test@gmail.com","subject":"测试验证邮件","body":"这是配置DNS验证后的测试邮件"}'
```

### 在线验证工具
- **MX Toolbox**: https://mxtoolbox.com/spf.aspx
- **DKIM Validator**: https://dkimvalidator.com/
- **DMARC Analyzer**: https://www.dmarcanalyzer.com/

## ⚠️ 注意事项

1. **DNS传播时间**: 通常需要2-24小时完全生效
2. **IP信誉建立**: 新IP需要逐步建立发送信誉
3. **发送量控制**: 初期建议控制发送量，避免被标记为垃圾邮件
4. **监控报告**: 定期检查DMARC报告，了解邮件送达情况

## 🔍 故障排除

### 如果邮件仍然失败
1. **检查端口25**: 确认ISP没有封锁端口25
2. **验证DNS记录**: 使用在线工具验证DNS配置
3. **查看邮件头**: 检查接收方邮件头中的验证信息
4. **IP信誉**: 检查IP是否在黑名单中

### 常见错误
- **SPF记录格式错误**: 确保语法正确
- **DKIM密钥不匹配**: 重新生成密钥对
- **DMARC策略太严格**: 初期使用 `p=none`

配置完成后，你的YgoCard邮箱将能够成功发送邮件到Gmail、Outlook等外部邮箱！
