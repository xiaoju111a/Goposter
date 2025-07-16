package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// IMAPTester IMAP性能测试器
type IMAPTester struct {
	Host           string
	Port           int
	TestResults    []TestResult
	ResultsMutex   sync.Mutex
	TotalTests     int
	SuccessfulTests int
	FailedTests    int
	TagCounter     int
}

// TestResult 测试结果
type TestResult struct {
	TestName     string
	Success      bool
	ResponseTime time.Duration
	Error        string
	Timestamp    time.Time
}

// NewIMAPTester 创建IMAP测试器
func NewIMAPTester(host string, port int) *IMAPTester {
	return &IMAPTester{
		Host:        host,
		Port:        port,
		TestResults: make([]TestResult, 0),
		TagCounter:  1,
	}
}

// RecordResult 记录测试结果
func (i *IMAPTester) RecordResult(testName string, success bool, responseTime time.Duration, err error) {
	i.ResultsMutex.Lock()
	defer i.ResultsMutex.Unlock()

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

	i.TestResults = append(i.TestResults, result)
	i.TotalTests++
	if success {
		i.SuccessfulTests++
	} else {
		i.FailedTests++
	}
}

// GetNextTag 获取下一个IMAP标签
func (i *IMAPTester) GetNextTag() string {
	tag := fmt.Sprintf("A%03d", i.TagCounter)
	i.TagCounter++
	return tag
}

// SendIMAPCommand 发送IMAP命令
func (i *IMAPTester) SendIMAPCommand(conn net.Conn, command string) (string, error) {
	tag := i.GetNextTag()
	fullCommand := fmt.Sprintf("%s %s\r\n", tag, command)
	
	_, err := conn.Write([]byte(fullCommand))
	if err != nil {
		return "", err
	}
	
	reader := bufio.NewReader(conn)
	var response strings.Builder
	
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		
		response.WriteString(line)
		
		// 检查是否是带标签的响应（命令完成）
		if strings.HasPrefix(line, tag+" ") {
			break
		}
	}
	
	return response.String(), nil
}

// TestIMAPConnection 测试IMAP连接
func (i *IMAPTester) TestIMAPConnection(iterations int) {
	fmt.Printf("\n📡 测试IMAP连接 (%d次)...\n", iterations)

	for iter := 0; iter < iterations; iter++ {
		startTime := time.Now()
		
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", i.Host, i.Port), 10*time.Second)
		responseTime := time.Since(startTime)
		
		if err != nil {
			i.RecordResult("IMAP Connection", false, responseTime, err)
			fmt.Printf("❌ 连接%d: %v - 错误: %v\n", iter+1, responseTime, err)
			continue
		}
		
		// 读取服务器欢迎消息
		reader := bufio.NewReader(conn)
		_, err = reader.ReadString('\n')
		if err != nil {
			i.RecordResult("IMAP Connection", false, responseTime, err)
			fmt.Printf("❌ 连接%d: %v - 读取欢迎消息失败: %v\n", iter+1, responseTime, err)
			conn.Close()
			continue
		}
		
		conn.Close()
		i.RecordResult("IMAP Connection", true, responseTime, nil)
		fmt.Printf("✅ 连接%d: %v - 连接成功\n", iter+1, responseTime)
		
		time.Sleep(100 * time.Millisecond)
	}
}

// TestIMAPLogin 测试IMAP登录
func (i *IMAPTester) TestIMAPLogin(iterations int) {
	fmt.Printf("\n🔐 测试IMAP登录 (%d次)...\n", iterations)

	for iter := 0; iter < iterations; iter++ {
		startTime := time.Now()
		
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", i.Host, i.Port), 10*time.Second)
		if err != nil {
			responseTime := time.Since(startTime)
			i.RecordResult("IMAP Login", false, responseTime, err)
			fmt.Printf("❌ 登录%d: %v - 连接失败: %v\n", iter+1, responseTime, err)
			continue
		}
		
		reader := bufio.NewReader(conn)
		// 读取欢迎消息
		_, err = reader.ReadString('\n')
		if err != nil {
			responseTime := time.Since(startTime)
			i.RecordResult("IMAP Login", false, responseTime, err)
			fmt.Printf("❌ 登录%d: %v - 读取欢迎消息失败: %v\n", iter+1, responseTime, err)
			conn.Close()
			continue
		}
		
		// 发送LOGIN命令
		loginCmd := `LOGIN "admin@ygocard.org" "admin123"`
		response, err := i.SendIMAPCommand(conn, loginCmd)
		responseTime := time.Since(startTime)
		
		if err != nil {
			i.RecordResult("IMAP Login", false, responseTime, err)
			fmt.Printf("❌ 登录%d: %v - 登录失败: %v\n", iter+1, responseTime, err)
		} else if strings.Contains(response, "OK") {
			i.RecordResult("IMAP Login", true, responseTime, nil)
			fmt.Printf("✅ 登录%d: %v - 登录成功\n", iter+1, responseTime)
		} else {
			i.RecordResult("IMAP Login", false, responseTime, fmt.Errorf("login failed: %s", response))
			fmt.Printf("❌ 登录%d: %v - 登录失败: %s\n", iter+1, responseTime, strings.TrimSpace(response))
		}
		
		conn.Close()
		time.Sleep(200 * time.Millisecond)
	}
}

// TestIMAPListFolders 测试IMAP列表文件夹
func (i *IMAPTester) TestIMAPListFolders(iterations int) {
	fmt.Printf("\n📁 测试IMAP列表文件夹 (%d次)...\n", iterations)

	for iter := 0; iter < iterations; iter++ {
		startTime := time.Now()
		
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", i.Host, i.Port), 10*time.Second)
		if err != nil {
			responseTime := time.Since(startTime)
			i.RecordResult("IMAP List Folders", false, responseTime, err)
			fmt.Printf("❌ 列表%d: %v - 连接失败: %v\n", iter+1, responseTime, err)
			continue
		}
		
		reader := bufio.NewReader(conn)
		// 读取欢迎消息
		_, err = reader.ReadString('\n')
		if err != nil {
			responseTime := time.Since(startTime)
			i.RecordResult("IMAP List Folders", false, responseTime, err)
			conn.Close()
			continue
		}
		
		// 登录
		loginCmd := `LOGIN "admin@ygocard.org" "admin123"`
		_, err = i.SendIMAPCommand(conn, loginCmd)
		if err != nil {
			responseTime := time.Since(startTime)
			i.RecordResult("IMAP List Folders", false, responseTime, err)
			conn.Close()
			continue
		}
		
		// 列出文件夹
		listCmd := `LIST "" "*"`
		response, err := i.SendIMAPCommand(conn, listCmd)
		responseTime := time.Since(startTime)
		
		if err != nil {
			i.RecordResult("IMAP List Folders", false, responseTime, err)
			fmt.Printf("❌ 列表%d: %v - 列表失败: %v\n", iter+1, responseTime, err)
		} else if strings.Contains(response, "OK") {
			i.RecordResult("IMAP List Folders", true, responseTime, nil)
			fmt.Printf("✅ 列表%d: %v - 列表成功\n", iter+1, responseTime)
		} else {
			i.RecordResult("IMAP List Folders", false, responseTime, fmt.Errorf("list failed: %s", response))
			fmt.Printf("❌ 列表%d: %v - 列表失败: %s\n", iter+1, responseTime, strings.TrimSpace(response))
		}
		
		conn.Close()
		time.Sleep(200 * time.Millisecond)
	}
}

// TestIMAPSelectInbox 测试IMAP选择收件箱
func (i *IMAPTester) TestIMAPSelectInbox(iterations int) {
	fmt.Printf("\n📮 测试IMAP选择收件箱 (%d次)...\n", iterations)

	for iter := 0; iter < iterations; iter++ {
		startTime := time.Now()
		
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", i.Host, i.Port), 10*time.Second)
		if err != nil {
			responseTime := time.Since(startTime)
			i.RecordResult("IMAP Select Inbox", false, responseTime, err)
			fmt.Printf("❌ 选择%d: %v - 连接失败: %v\n", iter+1, responseTime, err)
			continue
		}
		
		reader := bufio.NewReader(conn)
		// 读取欢迎消息
		_, err = reader.ReadString('\n')
		if err != nil {
			responseTime := time.Since(startTime)
			i.RecordResult("IMAP Select Inbox", false, responseTime, err)
			conn.Close()
			continue
		}
		
		// 登录
		loginCmd := `LOGIN "admin@ygocard.org" "admin123"`
		_, err = i.SendIMAPCommand(conn, loginCmd)
		if err != nil {
			responseTime := time.Since(startTime)
			i.RecordResult("IMAP Select Inbox", false, responseTime, err)
			conn.Close()
			continue
		}
		
		// 选择收件箱
		selectCmd := `SELECT INBOX`
		response, err := i.SendIMAPCommand(conn, selectCmd)
		responseTime := time.Since(startTime)
		
		if err != nil {
			i.RecordResult("IMAP Select Inbox", false, responseTime, err)
			fmt.Printf("❌ 选择%d: %v - 选择失败: %v\n", iter+1, responseTime, err)
		} else if strings.Contains(response, "OK") {
			i.RecordResult("IMAP Select Inbox", true, responseTime, nil)
			fmt.Printf("✅ 选择%d: %v - 选择成功\n", iter+1, responseTime)
		} else {
			i.RecordResult("IMAP Select Inbox", false, responseTime, fmt.Errorf("select failed: %s", response))
			fmt.Printf("❌ 选择%d: %v - 选择失败: %s\n", iter+1, responseTime, strings.TrimSpace(response))
		}
		
		conn.Close()
		time.Sleep(200 * time.Millisecond)
	}
}

// TestConcurrentConnections 测试并发连接
func (i *IMAPTester) TestConcurrentConnections(concurrency int) {
	fmt.Printf("\n🚀 并发连接测试 (%d个并发连接)...\n", concurrency)

	var wg sync.WaitGroup
	startTime := time.Now()

	for iter := 0; iter < concurrency; iter++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			connStart := time.Now()
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", i.Host, i.Port), 10*time.Second)
			responseTime := time.Since(connStart)
			
			testName := fmt.Sprintf("Concurrent IMAP Connection #%d", index+1)
			
			if err != nil {
				i.RecordResult(testName, false, responseTime, err)
				return
			}
			
			// 读取欢迎消息
			reader := bufio.NewReader(conn)
			_, err = reader.ReadString('\n')
			if err != nil {
				i.RecordResult(testName, false, responseTime, err)
				conn.Close()
				return
			}
			
			conn.Close()
			i.RecordResult(testName, true, responseTime, nil)
		}(iter)
	}

	wg.Wait()
	totalTime := time.Since(startTime)
	
	fmt.Printf("✅ 并发测试完成: 总耗时: %v\n", totalTime)
}

// GenerateReport 生成测试报告
func (i *IMAPTester) GenerateReport() {
	fmt.Println("\n📊 测试报告摘要:")
	fmt.Println("=====================================")
	fmt.Printf("总测试数: %d\n", i.TotalTests)
	fmt.Printf("成功测试: %d\n", i.SuccessfulTests)
	fmt.Printf("失败测试: %d\n", i.FailedTests)
	
	if i.TotalTests > 0 {
		successRate := float64(i.SuccessfulTests) / float64(i.TotalTests) * 100
		fmt.Printf("成功率: %.2f%%\n", successRate)
	}
	
	// 计算响应时间统计
	if len(i.TestResults) > 0 {
		var totalTime time.Duration
		minTime := i.TestResults[0].ResponseTime
		maxTime := i.TestResults[0].ResponseTime
		
		for _, result := range i.TestResults {
			totalTime += result.ResponseTime
			if result.ResponseTime < minTime {
				minTime = result.ResponseTime
			}
			if result.ResponseTime > maxTime {
				maxTime = result.ResponseTime
			}
		}
		
		avgTime := totalTime / time.Duration(len(i.TestResults))
		fmt.Printf("平均响应时间: %v\n", avgTime)
		fmt.Printf("最快响应时间: %v\n", minTime)
		fmt.Printf("最慢响应时间: %v\n", maxTime)
	}
	
	fmt.Println("=====================================")
	
	// 保存详细报告到文件
	reportFile := fmt.Sprintf("results/imap_test_%d.txt", time.Now().Unix())
	file, err := os.Create(reportFile)
	if err == nil {
		defer file.Close()
		
		fmt.Fprintf(file, "IMAP性能测试报告\n")
		fmt.Fprintf(file, "测试时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Fprintf(file, "目标服务器: %s:%d\n\n", i.Host, i.Port)
		
		fmt.Fprintf(file, "测试摘要:\n")
		fmt.Fprintf(file, "总测试数: %d\n", i.TotalTests)
		fmt.Fprintf(file, "成功测试: %d\n", i.SuccessfulTests)
		fmt.Fprintf(file, "失败测试: %d\n", i.FailedTests)
		
		fmt.Fprintf(file, "\n详细结果:\n")
		for _, result := range i.TestResults {
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
func (i *IMAPTester) RunAllTests() {
	fmt.Println("🧪 开始IMAP性能测试...")
	fmt.Printf("目标服务器: %s:%d\n", i.Host, i.Port)
	
	i.TestIMAPConnection(10)
	i.TestIMAPLogin(10)
	i.TestIMAPListFolders(5)
	i.TestIMAPSelectInbox(5)
	i.TestConcurrentConnections(10)
	
	i.GenerateReport()
	fmt.Println("\n✅ 所有测试完成！")
}

func main() {
	// 默认测试本地IMAP服务器
	host := "localhost"
	port := 143
	
	// 支持命令行参数
	if len(os.Args) > 1 {
		host = os.Args[1]
	}
	if len(os.Args) > 2 {
		if p, err := strconv.Atoi(os.Args[2]); err == nil {
			port = p
		}
	}
	
	tester := NewIMAPTester(host, port)
	tester.RunAllTests()
}