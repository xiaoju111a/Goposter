# 🎮 YgoCard.org 域名解析配置完整教程

## 📋 前置条件

1. **拥有 ygocard.org 域名** (从域名注册商购买)
2. **云服务器/VPS** 已准备好并获得公网IP
3. **具备域名DNS管理权限**

## 🌐 域名解析详细步骤

### 步骤1: 登录域名管理控制台

根据你的域名注册商，登录对应的控制台：

#### 常见域名注册商
- **阿里云 (万网)**: https://dns.console.aliyun.com
- **腾讯云 DNSPod**: https://console.dnspod.cn
- **GoDaddy**: https://dcc.godaddy.com/manage/dns
- **Namecheap**: https://ap.www.namecheap.com/domains/list
- **Cloudflare**: https://dash.cloudflare.com

### 步骤2: 添加DNS记录

在DNS管理页面，添加以下记录：

#### 2.1 A记录 (必需)
```
记录类型: A
主机记录: mail
记录值: [你的服务器公网IP]
TTL: 600 (10分钟)
```

**示例**:
- 主机记录: `mail`
- 记录值: `1.2.3.4` (替换为你的实际IP)
- 完整域名: `mail.ygocard.org`

#### 2.2 MX记录 (必需)
```
记录类型: MX
主机记录: @
记录值: mail.ygocard.org
优先级: 10
TTL: 600
```

**说明**: MX记录告诉邮件系统，发送到 `@ygocard.org` 的邮件应该投递到 `mail.ygocard.org`

#### 2.3 TXT记录 - SPF (推荐)
```
记录类型: TXT
主机记录: @
记录值: "v=spf1 a mx ~all"
TTL: 600
```

**说明**: SPF记录防止邮件被标记为垃圾邮件

## 🔍 DNS配置验证

配置完成后，使用以下命令验证：

### 检查A记录
```bash
# Windows
nslookup mail.ygocard.org

# Linux/Mac
dig A mail.ygocard.org
```

**期望结果**:
```
mail.ygocard.org.    600    IN    A    1.2.3.4
```

### 检查MX记录
```bash
# Windows
nslookup -type=MX ygocard.org

# Linux/Mac
dig MX ygocard.org
```

**期望结果**:
```
ygocard.org.    600    IN    MX    10 mail.ygocard.org.
```

### 检查SPF记录
```bash
# Windows
nslookup -type=TXT ygocard.org

# Linux/Mac
dig TXT ygocard.org
```

**期望结果**:
```
ygocard.org.    600    IN    TXT    "v=spf1 a mx ~all"
```

## ⏰ DNS生效时间

- **本地生效**: 10分钟-2小时
- **全球生效**: 2-24小时
- **完全传播**: 最多48小时

## 🖥️ 不同注册商的具体操作

### 阿里云 (万网)
1. 登录 [阿里云DNS控制台](https://dns.console.aliyun.com)
2. 找到 `ygocard.org` 域名，点击"解析设置"
3. 点击"添加记录"
4. 依次添加A记录、MX记录、TXT记录

### 腾讯云 DNSPod
1. 登录 [DNSPod控制台](https://console.dnspod.cn)
2. 找到 `ygocard.org`，点击域名进入解析页面
3. 点击"添加记录"
4. 选择记录类型并填写相应信息

### GoDaddy
1. 登录 [GoDaddy管理中心](https://dcc.godaddy.com)
2. 点击"DNS" → "Manage Zones"
3. 找到 `ygocard.org`，点击DNS图标
4. 添加相应的DNS记录

### Cloudflare
1. 登录 [Cloudflare控制台](https://dash.cloudflare.com)
2. 选择 `ygocard.org` 域名
3. 进入"DNS"标签页
4. 点击"Add record"添加记录

## 🔧 常见配置问题

### 问题1: DNS不生效
**解决方案**:
- 等待更长时间 (最多48小时)
- 清除本地DNS缓存
- 使用不同的DNS查询工具验证

### 问题2: MX记录配置错误
**检查项**:
- 主机记录应该是 `@` 而不是空
- 记录值应该是 `mail.ygocard.org` (有结尾的点更好)
- 优先级设置为10

### 问题3: A记录指向错误
**检查项**:
- 确认服务器IP地址正确
- 确认IP是公网IP而不是内网IP
- 测试IP能否正常访问

## 📱 移动端DNS管理

大部分域名注册商都提供手机App：
- **阿里云**: 阿里云App
- **腾讯云**: 腾讯云助手
- **GoDaddy**: GoDaddy Mobile

## 🛠️ 高级配置 (可选)

### DKIM记录
```
记录类型: TXT
主机记录: default._domainkey
记录值: "v=DKIM1; k=rsa; p=公钥内容..."
```

### DMARC记录
```
记录类型: TXT
主机记录: _dmarc
记录值: "v=DMARC1; p=none; rua=mailto:dmarc@ygocard.org"
```

### CAA记录
```
记录类型: CAA
主机记录: @
记录值: 0 issue "letsencrypt.org"
```

## ✅ 配置检查清单

- [ ] A记录: `mail.ygocard.org` → 服务器IP
- [ ] MX记录: `ygocard.org` → `mail.ygocard.org`
- [ ] TXT记录: SPF配置
- [ ] DNS生效验证
- [ ] 服务器端口开放确认
- [ ] 邮箱服务器启动测试

完成以上配置后，全球任何邮箱都可以发送邮件到 `任意名称@ygocard.org`！
