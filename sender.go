package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

type SMTPClient struct {
	host string
	port string
	conn net.Conn
}

func NewSMTPClient(host, port string) *SMTPClient {
	return &SMTPClient{
		host: host,
		port: port,
	}
}

func (client *SMTPClient) Connect() error {
	conn, err := net.Dial("tcp", client.host+":"+client.port)
	if err != nil {
		return err
	}
	client.conn = conn
	
	reader := bufio.NewReader(conn)
	_, err = reader.ReadString('\n')
	if err != nil {
		return err
	}
	
	return nil
}

func (client *SMTPClient) SendCommand(command string) (string, error) {
	writer := bufio.NewWriter(client.conn)
	reader := bufio.NewReader(client.conn)
	
	_, err := writer.WriteString(command + "\r\n")
	if err != nil {
		return "", err
	}
	writer.Flush()
	
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(response), nil
}

func (client *SMTPClient) SendEmail(from, to, subject, body string) error {
	_, err := client.SendCommand("HELO client")
	if err != nil {
		return err
	}
	
	_, err = client.SendCommand("MAIL FROM:<" + from + ">")
	if err != nil {
		return err
	}
	
	_, err = client.SendCommand("RCPT TO:<" + to + ">")
	if err != nil {
		return err
	}
	
	_, err = client.SendCommand("DATA")
	if err != nil {
		return err
	}
	
	emailContent := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\n\r\n%s\r\n.",
		from, to, subject, time.Now().Format(time.RFC822), body)
	
	_, err = client.SendCommand(emailContent)
	if err != nil {
		return err
	}
	
	return nil
}

func (client *SMTPClient) Close() error {
	if client.conn != nil {
		client.SendCommand("QUIT")
		return client.conn.Close()
	}
	return nil
}

func SendTestEmail() {
	client := NewSMTPClient("localhost", "2525")
	
	err := client.Connect()
	if err != nil {
		fmt.Println("连接失败:", err)
		return
	}
	defer client.Close()
	
	testEmails := []struct {
		from    string
		to      string
		subject string
		body    string
	}{
		{"test@ygocard.org", "user@ygocard.org", "测试邮件1", "这是第一封测试邮件"},
		{"admin@ygocard.org", "support@ygocard.org", "测试邮件2", "这是发给支持邮箱的邮件"},
		{"noreply@ygocard.org", "info@ygocard.org", "测试邮件3", "这是发给信息邮箱的邮件"},
		{"sales@ygocard.org", "contact@ygocard.org", "测试邮件4", "这是发给联系邮箱的邮件"},
	}
	
	for _, email := range testEmails {
		err = client.SendEmail(email.from, email.to, email.subject, email.body)
		if err != nil {
			fmt.Printf("发送邮件失败 (%s -> %s): %v\n", email.from, email.to, err)
			continue
		}
		
		fmt.Printf("邮件发送成功: %s -> %s\n", email.from, email.to)
	}
}

// SendTestEmail 的主函数已移动到 main.go
// 如需单独使用发送功能，请调用 SendTestEmail() 函数