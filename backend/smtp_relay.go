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

// SMTPRelayConfig SMTP中继配置
type SMTPRelayConfig struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	UseTLS   bool   `json:"use_tls"`
}

// 腾讯云邮件推送默认配置
var TencentCloudSESConfig = SMTPRelayConfig{
	Enabled:  false,                    // 需要手动启用
	Host:     "smtp.qcloudmail.com",    // 腾讯云邮件推送SMTP服务器
	Port:     587,                      // 使用587端口避免25端口封锁
	Username: "",                       // SMTP用户名（在腾讯云控制台配置）
	Password: "",                       // SMTP密码（在腾讯云控制台获取）
	UseTLS:   true,
}

// 其他常用SMTP服务配置
var PresetConfigs = map[string]SMTPRelayConfig{
	"amazon_ses_us_east_1": {
		Host:   "email-smtp.us-east-1.amazonaws.com",
		Port:   587,
		UseTLS: true,
	},
	"amazon_ses_us_west_2": {
		Host:   "email-smtp.us-west-2.amazonaws.com",
		Port:   587,
		UseTLS: true,
	},
	"amazon_ses_eu_west_1": {
		Host:   "email-smtp.eu-west-1.amazonaws.com",
		Port:   587,
		UseTLS: true,
	},
	"amazon_ses_ap_southeast_1": {
		Host:   "email-smtp.ap-southeast-1.amazonaws.com",
		Port:   587,
		UseTLS: true,
	},
	"tencent_ses": {
		Host:   "smtp.qcloudmail.com",
		Port:   587,
		UseTLS: true,
	},
	"tencent_exmail": {
		Host:   "smtp.exmail.qq.com",
		Port:   587,
		UseTLS: true,
	},
	"qq": {
		Host:   "smtp.qq.com",
		Port:   587,
		UseTLS: true,
	},
	"163": {
		Host:   "smtp.163.com",
		Port:   587,
		UseTLS: true,
	},
	"126": {
		Host:   "smtp.126.com",
		Port:   587,
		UseTLS: true,
	},
	"gmail": {
		Host:   "smtp.gmail.com",
		Port:   587,
		UseTLS: true,
	},
}

// SMTPRelay SMTP中继发送器
type SMTPRelay struct {
	config SMTPRelayConfig
}

// NewSMTPRelay 创建SMTP中继发送器
func NewSMTPRelay(config SMTPRelayConfig) *SMTPRelay {
	return &SMTPRelay{
		config: config,
	}
}

// SendEmail 通过SMTP中继发送邮件
func (r *SMTPRelay) SendEmail(from, to, subject, body string) error {
	if !r.config.Enabled {
		log.Printf("[SMTP中继] 中继未启用，无法发送邮件: %s -> %s", from, to)
		return fmt.Errorf("SMTP中继未启用")
	}

	startTime := time.Now()
	log.Printf("[SMTP中继] 开始发送邮件: %s -> %s, 主题: %s", from, to, subject)

	// 构建邮件内容
	msg := r.buildMessage(from, to, subject, body)
	log.Printf("[SMTP中继] 邮件内容构建完成，大小: %d字节", len(msg))

	// 连接到SMTP服务器
	serverAddr := fmt.Sprintf("%s:%d", r.config.Host, r.config.Port)
	log.Printf("[SMTP中继] 连接目标服务器: %s (TLS: %v)", serverAddr, r.config.UseTLS)
	
	var client *smtp.Client
	var err error
	
	if r.config.UseTLS {
		client, err = r.connectWithTLS(serverAddr)
	} else {
		client, err = r.connectPlain(serverAddr)
	}
	
	if err != nil {
		log.Printf("[SMTP中继] 连接失败: %s, 耗时: %v, 错误: %v", serverAddr, time.Since(startTime), err)
		return fmt.Errorf("连接SMTP中继服务器失败: %v", err)
	}
	defer client.Quit()

	// 如果配置了用户名和密码，则进行认证
	if r.config.Username != "" && r.config.Password != "" {
		log.Printf("[SMTP中继] 开始认证，用户名: %s", r.config.Username)
		auth := smtp.PlainAuth("", r.config.Username, r.config.Password, r.config.Host)
		if err := client.Auth(auth); err != nil {
			log.Printf("[SMTP中继] 认证失败，用户名: %s, 错误: %v", r.config.Username, err)
			return fmt.Errorf("SMTP认证失败: %v", err)
		}
		log.Printf("[SMTP中继] 认证成功，用户名: %s", r.config.Username)
	} else {
		log.Printf("[SMTP中继] 跳过认证（未配置用户名密码）")
	}

	// 发送邮件
	err = r.sendMessage(client, from, to, msg)
	if err != nil {
		log.Printf("[SMTP中继] 邮件发送失败: %s -> %s, 总耗时: %v, 错误: %v", from, to, time.Since(startTime), err)
		return err
	}
	
	log.Printf("[SMTP中继] 邮件发送流程完成: %s -> %s, 总耗时: %v", from, to, time.Since(startTime))
	return nil
}

func (r *SMTPRelay) connectWithTLS(serverAddr string) (*smtp.Client, error) {
	log.Printf("[SMTP中继] 使用TLS连接: %s", serverAddr)
	
	startTime := time.Now()
	
	// 对于465端口，使用SSL直连
	if r.config.Port == 465 {
		return r.connectWithSSL(serverAddr, startTime)
	}
	
	// 对于其他端口，使用STARTTLS
	conn, err := net.DialTimeout("tcp", serverAddr, 10*time.Second)
	if err != nil {
		log.Printf("[SMTP中继] TCP连接失败: %s, 耗时: %v, 错误: %v", serverAddr, time.Since(startTime), err)
		return nil, fmt.Errorf("TCP连接失败: %v", err)
	}

	client, err := smtp.NewClient(conn, r.config.Host)
	if err != nil {
		conn.Close()
		log.Printf("[SMTP中继] SMTP客户端创建失败: %s, 耗时: %v, 错误: %v", serverAddr, time.Since(startTime), err)
		return nil, fmt.Errorf("SMTP客户端创建失败: %v", err)
	}

	// 尝试STARTTLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         r.config.Host,
	}

	if err = client.StartTLS(tlsConfig); err != nil {
		log.Printf("[SMTP中继] STARTTLS失败: %s, 耗时: %v, 错误: %v", serverAddr, time.Since(startTime), err)
		client.Quit()
		return nil, fmt.Errorf("STARTTLS失败: %v", err)
	}

	log.Printf("[SMTP中继] TLS连接建立成功: %s, 耗时: %v", serverAddr, time.Since(startTime))
	return client, nil
}

func (r *SMTPRelay) connectWithSSL(serverAddr string, startTime time.Time) (*smtp.Client, error) {
	log.Printf("[SMTP中继] 使用SSL直连: %s", serverAddr)
	
	// 使用SSL直连（适用于465端口）
	conn, err := tls.Dial("tcp", serverAddr, nil)
	if err != nil {
		log.Printf("[SMTP中继] SSL连接失败: %s, 耗时: %v, 错误: %v", serverAddr, time.Since(startTime), err)
		return nil, fmt.Errorf("SSL连接失败: %v", err)
	}

	host, _, _ := net.SplitHostPort(serverAddr)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		log.Printf("[SMTP中继] SMTP客户端创建失败: %s, 耗时: %v, 错误: %v", serverAddr, time.Since(startTime), err)
		return nil, fmt.Errorf("SMTP客户端创建失败: %v", err)
	}

	log.Printf("[SMTP中继] SSL连接建立成功: %s, 耗时: %v", serverAddr, time.Since(startTime))
	return client, nil
}

func (r *SMTPRelay) connectPlain(serverAddr string) (*smtp.Client, error) {
	log.Printf("[SMTP中继] 使用普通连接: %s", serverAddr)
	
	startTime := time.Now()
	
	conn, err := net.DialTimeout("tcp", serverAddr, 10*time.Second)
	if err != nil {
		log.Printf("[SMTP中继] TCP连接失败: %s, 耗时: %v, 错误: %v", serverAddr, time.Since(startTime), err)
		return nil, fmt.Errorf("TCP连接失败: %v", err)
	}

	client, err := smtp.NewClient(conn, r.config.Host)
	if err != nil {
		conn.Close()
		log.Printf("[SMTP中继] SMTP客户端创建失败: %s, 耗时: %v, 错误: %v", serverAddr, time.Since(startTime), err)
		return nil, fmt.Errorf("SMTP客户端创建失败: %v", err)
	}

	log.Printf("[SMTP中继] 普通连接建立成功: %s, 耗时: %v", serverAddr, time.Since(startTime))
	return client, nil
}

func (r *SMTPRelay) buildMessage(from, to, subject, body string) string {
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = to
	headers["Subject"] = subject
	headers["Date"] = time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")
	headers["Message-ID"] = fmt.Sprintf("<%d@relay>", time.Now().Unix())
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

func (r *SMTPRelay) sendMessage(client *smtp.Client, from, to, message string) error {
	startTime := time.Now()
	log.Printf("[SMTP中继] 开始发送邮件: %s -> %s", from, to)
	
	// 设置发件人
	if err := client.Mail(from); err != nil {
		log.Printf("[SMTP中继] 设置发件人失败: %s, 错误: %v", from, err)
		return fmt.Errorf("设置发件人失败: %v", err)
	}
	log.Printf("[SMTP中继] 发件人设置成功: %s", from)
	
	// 设置收件人
	if err := client.Rcpt(to); err != nil {
		log.Printf("[SMTP中继] 设置收件人失败: %s, 错误: %v", to, err)
		return fmt.Errorf("设置收件人失败: %v", err)
	}
	log.Printf("[SMTP中继] 收件人设置成功: %s", to)
	
	// 发送邮件内容
	writer, err := client.Data()
	if err != nil {
		log.Printf("[SMTP中继] 开始数据传输失败: %v", err)
		return fmt.Errorf("开始数据传输失败: %v", err)
	}
	defer writer.Close()
	
	messageSize := len(message)
	_, err = writer.Write([]byte(message))
	if err != nil {
		log.Printf("[SMTP中继] 写入邮件内容失败: 大小:%d字节, 错误: %v", messageSize, err)
		return fmt.Errorf("写入邮件内容失败: %v", err)
	}
	
	log.Printf("[SMTP中继] 邮件发送成功: %s -> %s, 大小: %d字节, 耗时: %v", from, to, messageSize, time.Since(startTime))
	return nil
}

// IsExternalDomain 检查是否为外部域名
func (r *SMTPRelay) IsExternalDomain(domain, localDomain string) bool {
	return domain != localDomain
}

// GetPresetConfig 获取预设配置
func GetPresetConfig(provider string) (SMTPRelayConfig, bool) {
	config, exists := PresetConfigs[provider]
	return config, exists
}

// ValidateConfig 验证SMTP中继配置
func (r *SMTPRelay) ValidateConfig() error {
	if r.config.Host == "" {
		return fmt.Errorf("SMTP主机不能为空")
	}
	if r.config.Port <= 0 || r.config.Port > 65535 {
		return fmt.Errorf("SMTP端口无效: %d", r.config.Port)
	}
	if r.config.Enabled && r.config.Username == "" {
		return fmt.Errorf("启用SMTP中继时用户名不能为空")
	}
	if r.config.Enabled && r.config.Password == "" {
		return fmt.Errorf("启用SMTP中继时密码不能为空")
	}
	return nil
}

// TestConnection 测试SMTP连接
func (r *SMTPRelay) TestConnection() error {
	if !r.config.Enabled {
		return fmt.Errorf("SMTP中继未启用")
	}

	err := r.ValidateConfig()
	if err != nil {
		return fmt.Errorf("配置验证失败: %v", err)
	}

	log.Printf("测试SMTP中继连接: %s:%d", r.config.Host, r.config.Port)

	serverAddr := fmt.Sprintf("%s:%d", r.config.Host, r.config.Port)
	
	var client *smtp.Client
	
	if r.config.UseTLS {
		client, err = r.connectWithTLS(serverAddr)
	} else {
		client, err = r.connectPlain(serverAddr)
	}
	
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	defer client.Quit()

	// 测试认证
	if r.config.Username != "" && r.config.Password != "" {
		auth := smtp.PlainAuth("", r.config.Username, r.config.Password, r.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("认证失败: %v", err)
		}
	}

	log.Printf("SMTP中继连接测试成功")
	return nil
}