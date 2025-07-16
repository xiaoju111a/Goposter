package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SMTPTester SMTP性能测试器
type SMTPTester struct {
	Host           string
	Port           int
	TestResults    []TestResult
	ResultsMutex   sync.Mutex
	TotalTests     int
	SuccessfulTests int
	FailedTests    int
}

// TestResult 测试结果
type TestResult struct {
	TestName     string
	Success      bool
	ResponseTime time.Duration
	Error        string
	Timestamp    time.Time
}

// NewSMTPTester 创建SMTP测试器
func NewSMTPTester(host string, port int) *SMTPTester {
	return &SMTPTester{
		Host:        host,
		Port:        port,
		TestResults: make([]TestResult, 0),
	}
}

// RecordResult 记录测试结果
func (s *SMTPTester) RecordResult(testName string, success bool, responseTime time.Duration, err error) {
	s.ResultsMutex.Lock()
	defer s.ResultsMutex.Unlock()

	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	result := TestResult{
		TestName:     testName,
		Success:      success,
		ResponseTime: responseTime,
		Error:        errorMsg,
		Timestamp:    time.Now(),
	}

	s.TestResults = append(s.TestResults, result)
	s.TotalTests++
	if success {
		s.SuccessfulTests++
	} else {
		s.FailedTests++
	}
}

// TestSMTPConnection 测试SMTP连接
func (s *SMTPTester) TestSMTPConnection(iterations int) {
	fmt.Printf("\n📡 测试SMTP连接 (%d次)...\n", iterations)

	for i := 0; i < iterations; i++ {
		startTime := time.Now()
		
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", s.Host, s.Port), 10*time.Second)
		responseTime := time.Since(startTime)
		
		if err != nil {
			s.RecordResult("SMTP Connection", false, responseTime, err)
			fmt.Printf("❌ 连接%d: %v - 错误: %v\n", i+1, responseTime, err)
			continue
		}
		
		conn.Close()
		s.RecordResult("SMTP Connection", true, responseTime, nil)
		fmt.Printf("✅ 连接%d: %v - 连接成功\n", i+1, responseTime)
		
		time.Sleep(100 * time.Millisecond)
	}
}

// TestSMTPHandshake 测试SMTP握手
func (s *SMTPTester) TestSMTPHandshake(iterations int) {
	fmt.Printf("\n🤝 测试SMTP握手 (%d次)...\n", iterations)

	for i := 0; i < iterations; i++ {
		startTime := time.Now()
		
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", s.Host, s.Port), 10*time.Second)
		if err != nil {
			responseTime := time.Since(startTime)
			s.RecordResult("SMTP Handshake", false, responseTime, err)
			fmt.Printf("❌ 握手%d: %v - 连接失败: %v\n", i+1, responseTime, err)
			continue
		}
		
		reader := bufio.NewReader(conn)
		
		// 读取服务器欢迎消息
		_, err = reader.ReadString('\n')
		if err != nil {
			responseTime := time.Since(startTime)
			s.RecordResult("SMTP Handshake", false, responseTime, err)
			fmt.Printf("❌ 握手%d: %v - 读取欢迎消息失败: %v\n", i+1, responseTime, err)
			conn.Close()
			continue
		}
		
		// 发送EHLO命令
		_, err = conn.Write([]byte("EHLO localhost\r\n"))
		if err != nil {
			responseTime := time.Since(startTime)
			s.RecordResult("SMTP Handshake", false, responseTime, err)
			fmt.Printf("❌ 握手%d: %v - 发送EHLO失败: %v\n", i+1, responseTime, err)
			conn.Close()
			continue
		}
		
		// 读取EHLO响应
		response, err := reader.ReadString('\n')
		responseTime := time.Since(startTime)
		
		if err != nil {
			s.RecordResult("SMTP Handshake", false, responseTime, err)
			fmt.Printf("❌ 握手%d: %v - 读取EHLO响应失败: %v\n", i+1, responseTime, err)
		} else if strings.HasPrefix(response, "250") {
			s.RecordResult("SMTP Handshake", true, responseTime, nil)
			fmt.Printf("✅ 握手%d: %v - 握手成功\n", i+1, responseTime)
		} else {
			s.RecordResult("SMTP Handshake", false, responseTime, fmt.Errorf("unexpected response: %s", response))
			fmt.Printf("❌ 握手%d: %v - 意外响应: %s\n", i+1, responseTime, strings.TrimSpace(response))
		}
		
		conn.Close()
		time.Sleep(100 * time.Millisecond)
	}
}

// TestEmailSending 测试邮件发送
func (s *SMTPTester) TestEmailSending(iterations int) {
	fmt.Printf("\n📧 测试邮件发送 (%d次)...\n", iterations)

	for i := 0; i < iterations; i++ {
		startTime := time.Now()
		
		// 构造测试邮件
		from := "test@ygocard.org"
		to := []string{"recipient@example.com"}
		subject := fmt.Sprintf("SMTP性能测试邮件 %d", i+1)
		body := fmt.Sprintf("这是第%d封SMTP性能测试邮件\n发送时间: %s", i+1, time.Now().Format("2006-01-02 15:04:05"))
		
		msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, strings.Join(to, ","), subject, body)
		
		// 发送邮件
		err := smtp.SendMail(fmt.Sprintf("%s:%d", s.Host, s.Port), nil, from, to, []byte(msg))
		responseTime := time.Since(startTime)
		
		if err != nil {
			s.RecordResult("Email Sending", false, responseTime, err)
			fmt.Printf("❌ 邮件%d: %v - 发送失败: %v\n", i+1, responseTime, err)
		} else {
			s.RecordResult("Email Sending", true, responseTime, nil)
			fmt.Printf("✅ 邮件%d: %v - 发送成功\n", i+1, responseTime)
		}
		
		time.Sleep(500 * time.Millisecond)
	}
}

// TestConcurrentConnections 测试并发连接
func (s *SMTPTester) TestConcurrentConnections(concurrency int) {
	fmt.Printf("\n🚀 并发连接测试 (%d个并发连接)...\n", concurrency)

	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			connStart := time.Now()
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", s.Host, s.Port), 10*time.Second)
			responseTime := time.Since(connStart)
			
			testName := fmt.Sprintf("Concurrent Connection #%d", index+1)
			
			if err != nil {
				s.RecordResult(testName, false, responseTime, err)
				return
			}
			
			conn.Close()
			s.RecordResult(testName, true, responseTime, nil)
		}(i)
	}

	wg.Wait()
	totalTime := time.Since(startTime)
	
	fmt.Printf("✅ 并发测试完成: 总耗时: %v\n", totalTime)
}

// TestTLSConnection 测试TLS连接
func (s *SMTPTester) TestTLSConnection(iterations int) {
	fmt.Printf("\n🔒 测试TLS连接 (%d次)...\n", iterations)

	for i := 0; i < iterations; i++ {
		startTime := time.Now()
		
		config := &tls.Config{
			ServerName:         s.Host,
			InsecureSkipVerify: true, // 测试环境跳过证书验证
		}
		
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", fmt.Sprintf("%s:%d", s.Host, s.Port), config)
		responseTime := time.Since(startTime)
		
		if err != nil {
			s.RecordResult("TLS Connection", false, responseTime, err)
			fmt.Printf("❌ TLS连接%d: %v - 错误: %v\n", i+1, responseTime, err)
			continue
		}
		
		conn.Close()
		s.RecordResult("TLS Connection", true, responseTime, nil)
		fmt.Printf("✅ TLS连接%d: %v - 连接成功\n", i+1, responseTime)
		
		time.Sleep(100 * time.Millisecond)
	}
}

// GenerateReport 生成测试报告
func (s *SMTPTester) GenerateReport() {
	fmt.Println("\n📊 测试报告摘要:")
	fmt.Println("=====================================")
	fmt.Printf("总测试数: %d\n", s.TotalTests)
	fmt.Printf("成功测试: %d\n", s.SuccessfulTests)
	fmt.Printf("失败测试: %d\n", s.FailedTests)
	
	if s.TotalTests > 0 {
		successRate := float64(s.SuccessfulTests) / float64(s.TotalTests) * 100
		fmt.Printf("成功率: %.2f%%\n", successRate)
	}
	
	// 计算响应时间统计
	if len(s.TestResults) > 0 {
		var totalTime time.Duration
		minTime := s.TestResults[0].ResponseTime
		maxTime := s.TestResults[0].ResponseTime
		
		for _, result := range s.TestResults {
			totalTime += result.ResponseTime
			if result.ResponseTime < minTime {
				minTime = result.ResponseTime
			}
			if result.ResponseTime > maxTime {
				maxTime = result.ResponseTime
			}
		}
		
		avgTime := totalTime / time.Duration(len(s.TestResults))
		fmt.Printf("平均响应时间: %v\n", avgTime)
		fmt.Printf("最快响应时间: %v\n", minTime)
		fmt.Printf("最慢响应时间: %v\n", maxTime)
	}
	
	fmt.Println("=====================================")
	
	// 保存详细报告到文件
	reportFile := fmt.Sprintf("results/smtp_test_%d.txt", time.Now().Unix())
	file, err := os.Create(reportFile)
	if err == nil {
		defer file.Close()
		
		fmt.Fprintf(file, "SMTP性能测试报告\n")
		fmt.Fprintf(file, "测试时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Fprintf(file, "目标服务器: %s:%d\n\n", s.Host, s.Port)
		
		fmt.Fprintf(file, "测试摘要:\n")
		fmt.Fprintf(file, "总测试数: %d\n", s.TotalTests)
		fmt.Fprintf(file, "成功测试: %d\n", s.SuccessfulTests)
		fmt.Fprintf(file, "失败测试: %d\n", s.FailedTests)
		
		fmt.Fprintf(file, "\n详细结果:\n")
		for _, result := range s.TestResults {
			status := "成功"
			if !result.Success {
				status = "失败"
			}
			fmt.Fprintf(file, "%s - %s: %v (%s)\n", result.Timestamp.Format("15:04:05"), result.TestName, result.ResponseTime, status)
			if result.Error != "" {
				fmt.Fprintf(file, "  错误: %s\n", result.Error)
			}
		}
		
		fmt.Printf("详细报告已保存至: %s\n", reportFile)
	}
}

// RunAllTests 运行所有测试
func (s *SMTPTester) RunAllTests() {
	fmt.Println("🧪 开始SMTP性能测试...")
	fmt.Printf("目标服务器: %s:%d\n", s.Host, s.Port)
	
	s.TestSMTPConnection(10)
	s.TestSMTPHandshake(10)
	s.TestEmailSending(5)
	s.TestConcurrentConnections(10)
	// s.TestTLSConnection(5) // 如果服务器支持TLS
	
	s.GenerateReport()
	fmt.Println("\n✅ 所有测试完成！")
}

func main() {
	// 默认测试本地SMTP服务器
	host := "localhost"
	port := 25
	
	// 支持命令行参数
	if len(os.Args) > 1 {
		host = os.Args[1]
	}
	if len(os.Args) > 2 {
		if p, err := strconv.Atoi(os.Args[2]); err == nil {
			port = p
		}
	}
	
	tester := NewSMTPTester(host, port)
	tester.RunAllTests()
}