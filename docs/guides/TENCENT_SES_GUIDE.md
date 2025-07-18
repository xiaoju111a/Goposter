# 腾讯云邮件推送(SES)配置指南

## 概述

腾讯云邮件推送(Simple Email Service, SES)是腾讯云提供的安全稳定、简单快捷、高可达的邮件推送服务。本指南将帮助您配置SMTP中继以使用腾讯云SES发送邮件。

## 前置条件

1. 拥有腾讯云账号
2. 已开通邮件推送服务
3. 已验证发送域名
4. 已创建SMTP用户

## 配置步骤

### 1. 登录腾讯云控制台

访问 [腾讯云邮件推送控制台](https://console.cloud.tencent.com/ses)

### 2. 验证发送域名

1. 在控制台左侧菜单中点击"发送域名"
2. 点击"新建"按钮
3. 输入您的域名（如：ygocard.org）
4. 根据提示添加DNS记录：
   - SPF记录：TXT类型，值为 `v=spf1 include:spf.mail.qq.com ~all`
   - MX记录：MX类型，值为 `mxbiz1.qq.com`，优先级10
   - CNAME记录：根据控制台提示添加
5. 等待域名验证通过

### 3. 创建SMTP用户

1. 在控制台左侧菜单中点击"SMTP"
2. 点击"新建SMTP用户"
3. 填写用户名（建议使用邮箱格式，如：noreply@ygocard.org）
4. 设置密码（建议使用强密码）
5. 选择已验证的发送域名
6. 点击"确定"创建

### 4. 获取SMTP配置信息

创建完成后，您将获得以下信息：
- **SMTP服务器**: smtp.qcloudmail.com
- **端口**: 587 (TLS)
- **用户名**: 您创建的SMTP用户名
- **密码**: 您设置的密码

### 5. 在邮箱服务器中配置

1. 访问您的邮箱管理界面
2. 找到"SMTP中继配置"部分
3. 选择"腾讯云邮件推送"预设配置，或手动填写：
   - **SMTP主机**: smtp.qcloudmail.com
   - **端口**: 587
   - **用户名**: 您的SMTP用户名
   - **密码**: 您的SMTP密码
   - **使用TLS**: 勾选
4. 点击"测试连接"验证配置
5. 点击"保存配置"并启用中继

## 发送限制

### 免费版限制
- 每日免费额度：1000封
- 发送频率：20封/秒

### 付费版限制
- 根据购买的套餐包确定
- 详细价格请参考腾讯云官网

## 常见问题

### Q1: 连接测试失败
**A1**: 检查以下项目：
- 确认SMTP用户名和密码正确
- 确认网络连接正常
- 确认防火墙允许587端口出站连接
- 确认域名已正确验证

### Q2: 邮件发送失败
**A2**: 可能的原因：
- 发件人邮箱不在验证域名内
- 超出发送限制
- 邮件内容被识别为垃圾邮件
- 收件人邮箱不存在或已满

### Q3: 邮件进入垃圾箱
**A3**: 改善措施：
- 完善DNS记录（SPF、DKIM、DMARC）
- 避免垃圾邮件关键词
- 建立良好的发送声誉
- 使用合适的发件人地址

## 安全建议

1. **密码安全**: 使用强密码，定期更换
2. **权限控制**: 仅授权必要的发送权限
3. **监控发送**: 定期查看发送日志和统计
4. **域名保护**: 确保DNS记录的安全性

## 技术支持

如遇问题，可通过以下方式获取支持：
- 腾讯云工单系统
- 邮件推送产品文档
- 技术交流群

## 参考链接

- [腾讯云邮件推送产品文档](https://cloud.tencent.com/document/product/1288)
- [SMTP接口说明](https://cloud.tencent.com/document/product/1288/51034)
- [域名验证指南](https://cloud.tencent.com/document/product/1288/51055)