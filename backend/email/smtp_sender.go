package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type SMTPSender struct {
	domain    string
	hostname  string
	emailAuth *EmailAuth
	relay     *SMTPRelay
}

func NewSMTPSender(domain, hostname string, emailAuth *EmailAuth) *SMTPSender {
	return &SMTPSender{
		domain:    domain,
		hostname:  hostname,
		emailAuth: emailAuth,
		relay:     nil, // 初始化为nil，稍后可配置
	}
}

// SetRelay 设置SMTP中继
func (s *SMTPSender) SetRelay(relay *SMTPRelay) {
	s.relay = relay
}

// SendEmail 发送邮件
func (s *SMTPSender) SendEmail(from, to, subject, body string) error {
	log.Printf("准备发送邮件: %s -> %s, 主题: %s", from, to, subject)
	
	// 构建邮件内容
	msg := s.buildMessage(from, to, subject, body)
	
	// 添加邮件验证头
	if s.emailAuth != nil {
		msg = s.emailAuth.AddAuthHeaders(msg)
	}
	
	// 解析收件人域名
	toDomain := s.extractDomain(to)
	if toDomain == "" {
		log.Printf("无效的收件人邮箱: %s", to)
		return fmt.Errorf("invalid recipient email: %s", to)
	}
	
	log.Printf("目标域名: %s", toDomain)
	
	// 检查是否为内部域名
	if toDomain == s.domain {
		log.Printf("检测到内部邮件，直接发送到本地服务器")
		return s.sendToHost("localhost:25", from, to, msg)
	}
	
	// 检查是否配置了SMTP中继且为外部域名
	if s.relay != nil && s.relay.config.Enabled && toDomain != s.domain {
		log.Printf("检测到外部邮件，使用SMTP中继发送")
		return s.relay.SendEmail(from, to, subject, body)
	}
	
	// 查找MX记录
	mxHosts, err := s.lookupMX(toDomain)
	if err != nil {
		log.Printf("查找MX记录失败: %v", err)
		return fmt.Errorf("failed to lookup MX records for %s: %v", toDomain, err)
	}
	
	// 尝试发送到MX主机
	var lastErr error
	for i, mxHost := range mxHosts {
		log.Printf("尝试第 %d 个MX主机: %s", i+1, mxHost)
		err := s.sendToHost(mxHost, from, to, msg)
		if err == nil {
			log.Printf("邮件发送成功通过: %s", mxHost)
			return nil // 发送成功
		}
		log.Printf("MX主机 %s 发送失败: %v", mxHost, err)
		lastErr = err
	}
	
	log.Printf("所有MX主机发送失败，最后错误: %v", lastErr)
	
	// 如果直接发送失败且配置了中继，尝试使用中继作为备用方案
	if s.relay != nil && s.relay.config.Enabled {
		log.Printf("直接发送失败，尝试使用SMTP中继作为备用方案")
		relayErr := s.relay.SendEmail(from, to, subject, body)
		if relayErr == nil {
			log.Printf("SMTP中继发送成功")
			return nil
		}
		log.Printf("SMTP中继也失败: %v", relayErr)
	}
	
	return fmt.Errorf("failed to send email to all MX hosts: %v", lastErr)
}

func (s *SMTPSender) buildMessage(from, to, subject, body string) string {
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = to
	headers["Subject"] = subject
	headers["Date"] = time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")
	headers["Message-ID"] = fmt.Sprintf("<%d@%s>", time.Now().Unix(), s.domain)
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=utf-8"
	
	var msg strings.Builder
	for key, value := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)
	
	return msg.String()
}

func (s *SMTPSender) extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

func (s *SMTPSender) lookupMX(domain string) ([]string, error) {
	log.Printf("查找 %s 的 MX 记录", domain)
	
	// 首先尝试真实的MX查询
	mxRecords, err := net.LookupMX(domain)
	if err == nil && len(mxRecords) > 0 {
		var hosts []string
		for _, mx := range mxRecords {
			// 去掉末尾的点号
			host := strings.TrimSuffix(mx.Host, ".")
			hosts = append(hosts, host+":25")
		}
		log.Printf("找到 MX 记录: %v", hosts)
		return hosts, nil
	}
	
	log.Printf("MX 查询失败: %v，使用备用方案", err)
	
	// 备用方案：常见邮件服务商配置 (优先使用25端口)
	mxMap := map[string][]string{
		"gmail.com":     {"gmail-smtp-in.l.google.com:25", "alt1.gmail-smtp-in.l.google.com:25"},
		"outlook.com":   {"outlook-com.olc.protection.outlook.com:25"},
		"hotmail.com":   {"hotmail-com.olc.protection.outlook.com:25"},
		"live.com":      {"hotmail-com.olc.protection.outlook.com:25"},
		"yahoo.com":     {"mta5.am0.yahoodns.net:25", "mta6.am0.yahoodns.net:25"},
		"qq.com":        {"mx1.qq.com:25", "mx2.qq.com:25"},
		"163.com":       {"163mx00.mxmail.netease.com:25", "163mx01.mxmail.netease.com:25"},
		"126.com":       {"126mx00.mxmail.netease.com:25", "126mx01.mxmail.netease.com:25"},
		"sina.com":      {"mx1.sina.com.cn:25", "mx2.sina.com.cn:25"},
		"ygocard.org":   {s.hostname + ":25"},
		"oylcorp.org":   {s.hostname + ":25"},
	}
	
	if hosts, exists := mxMap[domain]; exists {
		log.Printf("使用预配置的 MX: %v", hosts)
		return hosts, nil
	}
	
	// 最后尝试标准端口
	hosts := []string{domain + ":25", domain + ":587"}
	log.Printf("使用默认端口: %v", hosts)
	return hosts, nil
}

func (s *SMTPSender) sendToHost(hostPort, from, to, message string) error {
	parts := strings.Split(hostPort, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid host:port format: %s", hostPort)
	}
	
	host := parts[0]
	port := parts[1]
	
	log.Printf("尝试发送邮件到 %s:%s", host, port)
	
	// 根据端口选择连接方式
	if port == "587" || port == "465" {
		return s.sendWithTLS(host, port, from, to, message)
	} else {
		return s.sendPlain(host, port, from, to, message)
	}
}

func (s *SMTPSender) sendWithTLS(host, port, from, to, message string) error {
	log.Printf("使用TLS发送到 %s:%s", host, port)
	
	// 先尝试 STARTTLS (587端口)
	if port == "587" {
		return s.sendWithSTARTTLS(host, port, from, to, message)
	}
	
	// 直接TLS连接 (465端口)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}
	
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", host+":"+port, tlsConfig)
	if err != nil {
		log.Printf("TLS连接失败: %v", err)
		return fmt.Errorf("failed to connect to %s:%s: %v", host, port, err)
	}
	defer conn.Close()
	
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		log.Printf("创建SMTP客户端失败: %v", err)
		return fmt.Errorf("failed to create SMTP client: %v", err)
	}
	defer client.Quit()
	
	// 发送邮件 (不需要认证)
	return s.sendMessage(client, from, to, message)
}

func (s *SMTPSender) sendWithSTARTTLS(host, port, from, to, message string) error {
	log.Printf("使用STARTTLS发送到 %s:%s", host, port)
	
	// 设置超时时间
	conn, err := net.DialTimeout("tcp", host+":"+port, 10*time.Second)
	if err != nil {
		log.Printf("连接失败: %v", err)
		return fmt.Errorf("failed to connect to %s:%s: %v", host, port, err)
	}
	defer conn.Close()
	
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		log.Printf("创建SMTP客户端失败: %v", err)
		return fmt.Errorf("failed to create SMTP client: %v", err)
	}
	defer client.Quit()
	
	// 尝试STARTTLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}
	
	if err = client.StartTLS(tlsConfig); err != nil {
		log.Printf("STARTTLS失败，降级到普通连接: %v", err)
		// 降级到普通连接
		return s.sendMessage(client, from, to, message)
	}
	
	log.Printf("STARTTLS成功")
	return s.sendMessage(client, from, to, message)
}

func (s *SMTPSender) sendPlain(host, port, from, to, message string) error {
	log.Printf("使用普通连接发送到 %s:%s", host, port)
	
	// 设置超时时间
	conn, err := net.DialTimeout("tcp", host+":"+port, 10*time.Second)
	if err != nil {
		log.Printf("普通连接失败: %v", err)
		return fmt.Errorf("failed to connect to %s:%s: %v", host, port, err)
	}
	defer conn.Close()
	
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		log.Printf("创建SMTP客户端失败: %v", err)
		return fmt.Errorf("failed to create SMTP client: %v", err)
	}
	defer client.Quit()
	
	return s.sendMessage(client, from, to, message)
}

func (s *SMTPSender) sendMessage(client *smtp.Client, from, to, message string) error {
	log.Printf("开始发送邮件: %s -> %s", from, to)
	
	// 设置发件人
	if err := client.Mail(from); err != nil {
		log.Printf("设置发件人失败: %v", err)
		return fmt.Errorf("failed to set sender: %v", err)
	}
	log.Printf("发件人设置成功: %s", from)
	
	// 设置收件人
	if err := client.Rcpt(to); err != nil {
		log.Printf("设置收件人失败: %v", err)
		return fmt.Errorf("failed to set recipient: %v", err)
	}
	log.Printf("收件人设置成功: %s", to)
	
	// 发送邮件内容
	writer, err := client.Data()
	if err != nil {
		log.Printf("开始数据传输失败: %v", err)
		return fmt.Errorf("failed to start data transmission: %v", err)
	}
	defer writer.Close()
	
	_, err = writer.Write([]byte(message))
	if err != nil {
		log.Printf("写入邮件内容失败: %v", err)
		return fmt.Errorf("failed to write message: %v", err)
	}
	
	log.Printf("邮件发送成功: %s -> %s", from, to)
	return nil
}