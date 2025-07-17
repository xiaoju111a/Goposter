# 🔒 goposter 邮箱服务器端口和防火墙配置指南

## 📋 需要开放的端口

### 必需端口
| 端口 | 协议 | 用途 | 重要性 |
|------|------|------|--------|
| **25** | TCP | SMTP邮件接收 | ⭐⭐⭐ 必须 |
| **8080** | TCP | Web管理界面 | ⭐⭐⭐ 必须 |

### 可选端口
| 端口 | 协议 | 用途 | 重要性 |
|------|------|------|--------|
| **80** | TCP | HTTP访问(可重定向到8080) | ⭐⭐ 推荐 |
| **443** | TCP | HTTPS访问 | ⭐⭐ 推荐 |
| **22** | TCP | SSH远程管理 | ⭐⭐ 推荐 |
| **2525** | TCP | 开发测试SMTP | ⭐ 可选 |

## 🛡️ 防火墙配置

### Ubuntu/Debian (UFW)

#### 基础配置
```bash
# 启用UFW
sudo ufw enable

# 开放SSH (防止锁定)
sudo ufw allow 22/tcp

# 开放邮箱必需端口
sudo ufw allow 25/tcp
sudo ufw allow 8080/tcp

# 开放Web端口 (可选)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# 查看防火墙状态
sudo ufw status verbose
```

#### 安全加固 (限制来源)
```bash
# 只允许特定IP访问Web管理界面
sudo ufw allow from [你的IP] to any port 8080

# 允许所有IP访问邮件端口
sudo ufw allow 25/tcp
```

#### 完整脚本
```bash
#!/bin/bash
# goposter 防火墙配置脚本

echo "🔒 配置 goposter 邮箱服务器防火墙..."

# 重置防火墙规则
sudo ufw --force reset

# 默认策略
sudo ufw default deny incoming
sudo ufw default allow outgoing

# SSH访问 (根据实际情况修改端口)
sudo ufw allow 22/tcp

# 邮箱服务端口
sudo ufw allow 25/tcp comment 'SMTP邮件接收'
sudo ufw allow 8080/tcp comment 'Web管理界面'

# Web服务端口 (可选)
sudo ufw allow 80/tcp comment 'HTTP'
sudo ufw allow 443/tcp comment 'HTTPS'

# 开发测试端口 (可选)
# sudo ufw allow 2525/tcp comment 'SMTP测试'

# 启用防火墙
sudo ufw enable

# 显示状态
sudo ufw status verbose

echo "✅ 防火墙配置完成!"
```

### CentOS/RHEL/Rocky Linux (Firewalld)

#### 基础配置
```bash
# 启动firewalld
sudo systemctl start firewalld
sudo systemctl enable firewalld

# 开放端口
sudo firewall-cmd --permanent --add-port=25/tcp
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --permanent --add-port=80/tcp
sudo firewall-cmd --permanent --add-port=443/tcp

# 重载配置
sudo firewall-cmd --reload

# 查看开放端口
sudo firewall-cmd --list-ports
```

#### 使用服务方式
```bash
# 添加邮件服务
sudo firewall-cmd --permanent --add-service=smtp
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https

# 添加自定义端口
sudo firewall-cmd --permanent --add-port=8080/tcp

# 重载并查看
sudo firewall-cmd --reload
sudo firewall-cmd --list-all
```

#### 完整脚本
```bash
#!/bin/bash
# goposter 防火墙配置脚本 (CentOS/RHEL)

echo "🔒 配置 goposter 邮箱服务器防火墙..."

# 启动并启用firewalld
sudo systemctl start firewalld
sudo systemctl enable firewalld

# 重置到默认区域
sudo firewall-cmd --set-default-zone=public

# 移除不需要的服务
sudo firewall-cmd --permanent --remove-service=dhcpv6-client

# 添加必需端口
sudo firewall-cmd --permanent --add-port=25/tcp  # SMTP
sudo firewall-cmd --permanent --add-port=8080/tcp  # Web管理
sudo firewall-cmd --permanent --add-port=80/tcp   # HTTP
sudo firewall-cmd --permanent --add-port=443/tcp  # HTTPS

# 添加SSH (如果不是22端口请修改)
sudo firewall-cmd --permanent --add-service=ssh

# 重载配置
sudo firewall-cmd --reload

# 显示配置
sudo firewall-cmd --list-all

echo "✅ 防火墙配置完成!"
```

## ☁️ 云服务商安全组配置

### 阿里云ECS

1. 登录 [阿里云控制台](https://ecs.console.aliyun.com)
2. 选择实例 → 安全组 → 配置规则
3. 添加入方向规则：

| 端口范围 | 授权对象 | 协议 | 说明 |
|----------|----------|------|------|
| 25/25 | 0.0.0.0/0 | TCP | SMTP邮件 |
| 8080/8080 | 0.0.0.0/0 | TCP | Web管理 |
| 80/80 | 0.0.0.0/0 | TCP | HTTP |
| 443/443 | 0.0.0.0/0 | TCP | HTTPS |

### 腾讯云CVM

1. 登录 [腾讯云控制台](https://console.cloud.tencent.com/cvm)
2. 实例 → 安全组 → 修改规则
3. 入站规则添加：

```
TCP:25     来源:0.0.0.0/0    SMTP邮件接收
TCP:8080   来源:0.0.0.0/0    Web管理界面
TCP:80     来源:0.0.0.0/0    HTTP服务
TCP:443    来源:0.0.0.0/0    HTTPS服务
```

### AWS EC2

1. 登录 [AWS控制台](https://console.aws.amazon.com/ec2)
2. 实例 → Security Groups → Edit inbound rules
3. 添加规则：

```
Type: Custom TCP    Port: 25     Source: 0.0.0.0/0
Type: Custom TCP    Port: 8080   Source: 0.0.0.0/0
Type: HTTP          Port: 80     Source: 0.0.0.0/0
Type: HTTPS         Port: 443    Source: 0.0.0.0/0
```

### Google Cloud GCE

1. 登录 [Google Cloud控制台](https://console.cloud.google.com/compute)
2. VPC网络 → 防火墙 → 创建防火墙规则
3. 创建规则：

```
名称: ygocard-mail-ports
方向: 入站
目标: 指定的目标标记
目标标记: ygocard-mail
协议和端口: TCP - 25,80,443,8080
```

## 🧪 端口测试

### 本地测试
```bash
# 测试SMTP端口
telnet localhost 25

# 测试Web端口
curl http://localhost:8080

# 查看监听端口
sudo netstat -tulpn | grep -E ':(25|8080|80|443)'
```

### 外部测试
```bash
# 从外部测试邮件端口
telnet [服务器IP] 25

# 测试Web端口
curl http://[服务器IP]:8080

# 使用在线工具
# https://www.yougetsignal.com/tools/open-ports/
# https://portchecker.co/
```

## 🛡️ 安全建议

### 最小权限原则
```bash
# 只对必要IP开放管理端口
sudo ufw delete allow 8080/tcp
sudo ufw allow from [你的固定IP] to any port 8080

# 禁用不必要的服务
sudo systemctl disable telnet
sudo systemctl disable ftp
```

### 端口伪装
```bash
# 将Web界面改为非标准端口
go run main.go ygocard.org 25 9527

# 对应防火墙规则
sudo ufw allow 9527/tcp
```

### 监控和日志
```bash
# 监控连接
sudo netstat -an | grep :25

# 查看防火墙日志
sudo tail -f /var/log/ufw.log

# 实时监控端口
sudo ss -tulpn | grep -E ':(25|8080)'
```

## 🚨 故障排除

### 端口被占用
```bash
# 查找占用端口的进程
sudo lsof -i :25
sudo lsof -i :8080

# 杀死进程
sudo kill -9 [PID]
```

### 防火墙阻拦
```bash
# 临时关闭防火墙测试
sudo ufw disable
# 测试后记得重新启用
sudo ufw enable
```

### ISP封锁端口25
```bash
# 使用备用端口
go run main.go ygocard.org 587 8080

# 相应修改防火墙规则
sudo ufw allow 587/tcp
```

## ✅ 配置检查清单

- [ ] **端口25**: SMTP邮件接收
- [ ] **端口8080**: Web管理界面
- [ ] **端口80**: HTTP (可选)
- [ ] **端口443**: HTTPS (可选)
- [ ] **云服务商安全组**: 已配置
- [ ] **本地防火墙**: 已配置
- [ ] **端口测试**: 外部可访问
- [ ] **服务启动**: 邮箱服务正常运行

完成配置后，你的 goposter 邮箱服务器将能够安全地接收和管理邮件！