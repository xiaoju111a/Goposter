package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/smtp"
	"os"
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

func main() {
	fmt.Println("=== 腾讯云 SMTP 中继连接测试 ===")
	
	// 测试腾讯云配置
	fmt.Println("\n🔍 测试腾讯云SMTP配置...")
	testConfig("smtp_relay.json")
	
	// 测试AWS SES配置
	fmt.Println("\n🔍 测试Amazon SES配置...")
	testConfig("smtp_realy1.json")
	
	fmt.Println("\n🎉 测试完成！")
}

func testConfig(filename string) {
	fmt.Printf("\n📁 读取配置文件: %s\n", filename)
	
	// 检查文件是否存在
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Printf("❌ 配置文件不存在: %s\n", filename)
		return
	}
	
	// 读取配置
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("❌ 读取配置文件失败: %v\n", err)
		return
	}
	
	var config SMTPRelayConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("❌ 解析配置文件失败: %v\n", err)
		return
	}
	
	// 显示配置信息
	fmt.Printf("📋 配置信息:\n")
	fmt.Printf("  主机: %s\n", config.Host)
	fmt.Printf("  端口: %d\n", config.Port)
	fmt.Printf("  用户名: %s\n", config.Username)
	fmt.Printf("  密码: %s\n", maskPassword(config.Password))
	fmt.Printf("  启用状态: %v\n", config.Enabled)
	fmt.Printf("  使用TLS: %v\n", config.UseTLS)
	
	// 验证配置
	if err := validateConfig(config); err != nil {
		fmt.Printf("❌ 配置验证失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 配置验证通过\n")
	
	// 测试连接
	fmt.Printf("🔗 测试SMTP连接...\n")
	if err := testConnection(config); err != nil {
		fmt.Printf("❌ 连接测试失败: %v\n", err)
		fmt.Printf("💡 建议检查:\n")
		fmt.Printf("  - 网络连接是否正常\n")
		fmt.Printf("  - SMTP服务器地址和端口是否正确\n")
		fmt.Printf("  - 用户名和密码是否正确\n")
		fmt.Printf("  - 防火墙是否阻止连接\n")
		return
	}
	fmt.Printf("✅ SMTP连接测试成功！\n")
}

func validateConfig(config SMTPRelayConfig) error {
	if config.Host == "" {
		return fmt.Errorf("SMTP主机不能为空")
	}
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("SMTP端口无效: %d", config.Port)
	}
	if config.Enabled && config.Username == "" {
		return fmt.Errorf("启用SMTP中继时用户名不能为空")
	}
	if config.Enabled && config.Password == "" {
		return fmt.Errorf("启用SMTP中继时密码不能为空")
	}
	return nil
}

func testConnection(config SMTPRelayConfig) error {
	if !config.Enabled {
		return fmt.Errorf("SMTP中继未启用")
	}

	serverAddr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	fmt.Printf("  连接到: %s (TLS: %v)\n", serverAddr, config.UseTLS)
	
	var client *smtp.Client
	var err error
	
	startTime := time.Now()
	
	if config.UseTLS {
		client, err = connectWithTLS(serverAddr, config.Host)
	} else {
		client, err = connectPlain(serverAddr, config.Host)
	}
	
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	defer client.Quit()
	
	fmt.Printf("  连接耗时: %v\n", time.Since(startTime))
	
	// 测试认证
	if config.Username != "" && config.Password != "" {
		fmt.Printf("  测试SMTP认证...\n")
		auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("认证失败: %v", err)
		}
		fmt.Printf("  认证成功\n")
	}
	
	return nil
}

func connectWithTLS(serverAddr, hostname string) (*smtp.Client, error) {
	// 对于465端口，使用SSL直连
	if serverAddr[len(serverAddr)-3:] == "465" {
		return connectWithSSL(serverAddr, hostname)
	}
	
	// 首先建立TCP连接
	conn, err := net.DialTimeout("tcp", serverAddr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("TCP连接失败: %v", err)
	}

	client, err := smtp.NewClient(conn, hostname)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("SMTP客户端创建失败: %v", err)
	}

	// 尝试STARTTLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         hostname,
	}

	if err = client.StartTLS(tlsConfig); err != nil {
		client.Quit()
		return nil, fmt.Errorf("STARTTLS失败: %v", err)
	}

	return client, nil
}

func connectWithSSL(serverAddr, hostname string) (*smtp.Client, error) {
	// 按照腾讯云官方示例使用TLS直连
	conn, err := tls.Dial("tcp", serverAddr, nil)
	if err != nil {
		return nil, fmt.Errorf("TLS连接失败: %v", err)
	}

	host, _, _ := net.SplitHostPort(serverAddr)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("SMTP客户端创建失败: %v", err)
	}

	return client, nil
}

func connectPlain(serverAddr, hostname string) (*smtp.Client, error) {
	conn, err := net.DialTimeout("tcp", serverAddr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("TCP连接失败: %v", err)
	}

	client, err := smtp.NewClient(conn, hostname)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("SMTP客户端创建失败: %v", err)
	}

	return client, nil
}

func maskPassword(password string) string {
	if len(password) == 0 {
		return "(未设置)"
	}
	if len(password) <= 4 {
		return "****"
	}
	return password[:2] + "****" + password[len(password)-2:]
}