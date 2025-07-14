# 🌐 OylCorp.org 真实域名邮箱部署指南

## 📋 前置要求

1. **拥有 oylcorp.org 域名控制权**
2. **云服务器/VPS** (建议2核2G以上)
3. **公网IP地址**
4. **域名DNS管理权限**

## 🔧 DNS 配置步骤

### 1. 配置A记录
```
类型: A
主机记录: mail
记录值: [你的服务器公网IP]
TTL: 600
```

### 2. 配置MX记录
```
类型: MX
主机记录: @
记录值: mail.oylcorp.org
优先级: 10
TTL: 600
```

### 3. 配置SPF记录（可选，防垃圾邮件）
```
类型: TXT
主机记录: @
记录值: "v=spf1 a mx ~all"
TTL: 600
```

### 4. 配置PTR反向解析（可选，提高送达率）
在VPS提供商控制台设置反向DNS:
```
IP: [你的公网IP]
反向解析: mail.oylcorp.org
```

## 🚀 服务器部署

### 1. 服务器环境准备
```bash
# 更新系统
sudo apt update && sudo apt upgrade -y

# 安装Go (如果未安装)
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 创建项目目录
mkdir -p /opt/oylcorp-mail
cd /opt/oylcorp-mail
```

### 2. 上传项目文件
```bash
# 将 main.go 和 sender.go 上传到 /opt/oylcorp-mail/
# 或使用git克隆项目
```

### 3. 配置防火墙
```bash
# Ubuntu/Debian
sudo ufw allow 25/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 8080/tcp
sudo ufw enable

# CentOS/RHEL
sudo firewall-cmd --permanent --add-port=25/tcp
sudo firewall-cmd --permanent --add-port=80/tcp
sudo firewall-cmd --permanent --add-port=443/tcp
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

### 4. 启动邮箱服务器

#### 开发模式测试
```bash
# 测试模式 (端口2525)
go run main.go oylcorp.org

# 访问Web界面
http://[服务器IP]:8080
```

#### 生产模式部署
```bash
# 生产模式 (端口25，需要sudo权限)
sudo go run main.go oylcorp.org 25 8080

# 或编译后运行
go build -o mailserver main.go
sudo ./mailserver oylcorp.org 25 8080
```

### 5. 设置系统服务 (可选)

创建systemd服务文件：
```bash
sudo nano /etc/systemd/system/oylcorp-mail.service
```

服务文件内容：
```ini
[Unit]
Description=OylCorp Mail Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/oylcorp-mail
ExecStart=/opt/oylcorp-mail/mailserver oylcorp.org 25 8080
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启动服务：
```bash
sudo systemctl daemon-reload
sudo systemctl enable oylcorp-mail
sudo systemctl start oylcorp-mail
sudo systemctl status oylcorp-mail
```

## 📧 测试邮件接收

### 1. DNS验证
```bash
# 检查MX记录
dig MX oylcorp.org

# 检查A记录
dig A mail.oylcorp.org

# 检查SPF记录
dig TXT oylcorp.org
```

### 2. 发送测试邮件
从任何邮箱发送邮件到：
- `test@oylcorp.org`
- `admin@oylcorp.org`
- `任意名称@oylcorp.org`

### 3. 查看接收情况
访问Web界面: `http://[服务器IP]:8080`

## 🔍 故障排除

### 常见问题

1. **端口25被ISP封锁**
   - 联系VPS提供商解封端口25
   - 或使用其他端口+邮件中继

2. **DNS解析不生效**
   - 等待DNS传播 (通常2-24小时)
   - 使用多个DNS查询工具验证

3. **防火墙阻拦**
   - 检查云服务商安全组设置
   - 确认本地防火墙配置

4. **权限问题**
   - 端口25需要root权限
   - 使用sudo运行服务

### 日志查看
```bash
# 查看系统服务日志
sudo journalctl -u oylcorp-mail -f

# 查看实时连接
sudo netstat -tulpn | grep :25
```

## 🔒 安全建议

1. **启用HTTPS** (推荐使用Let's Encrypt)
2. **配置TLS加密** SMTP连接
3. **添加用户认证**
4. **设置邮件大小限制**
5. **配置反垃圾邮件规则**
6. **定期备份邮件数据**

## 📈 扩展功能

- **数据库存储**: 使用MySQL/PostgreSQL持久化
- **邮件转发**: 实现邮件路由功能
- **API接口**: 提供RESTful API
- **监控告警**: 集成监控系统
- **负载均衡**: 多服务器部署

部署完成后，oylcorp.org 将能够接收全球任何地方发送的邮件！