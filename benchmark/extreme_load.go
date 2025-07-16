package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ExtremeConcurrencyTester 极限并发测试器
type ExtremeConcurrencyTester struct {
	Host               string
	Port               int
	Results            []ConnectionResult
	ResultsMutex       sync.Mutex
	TotalConnections   int64
	ActiveConnections  int64
	SuccessConnections int64
	FailedConnections  int64
	MaxActiveConn      int64
}

// ConnectionResult 连接测试结果
type ConnectionResult struct {
	ConnectionID   int
	Success        bool
	ResponseTime   time.Duration
	Error          string
	Timestamp      time.Time
	ConcurrencyLevel int
}

// NewExtremeTester 创建极限测试器
func NewExtremeTester(host string, port int) *ExtremeConcurrencyTester {
	return &ExtremeConcurrencyTester{
		Host:    host,
		Port:    port,
		Results: make([]ConnectionResult, 0),
	}
}

// TestExtremeConnections 测试极限并发连接
func (e *ExtremeConcurrencyTester) TestExtremeConnections(concurrency int, duration time.Duration) {
	fmt.Printf("\n🚀 极限并发连接测试 (%d并发, 持续%v)\n", concurrency, duration)
	fmt.Printf("目标服务器: %s:%d\n", e.Host, e.Port)
	
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	
	var wg sync.WaitGroup
	startTime := time.Now()
	
	// 启动并发连接workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			e.extremeConnectionWorker(ctx, workerID, concurrency)
		}(i)
	}
	
	// 启动监控goroutine
	go e.monitorConnections(ctx, startTime)
	
	wg.Wait()
	totalTime := time.Since(startTime)
	
	e.printExtremeResults(concurrency, totalTime)
}

// extremeConnectionWorker 极限连接工作器
func (e *ExtremeConcurrencyTester) extremeConnectionWorker(ctx context.Context, workerID int, concurrency int) {
	connectionCount := 0
	
	for {
		select {
		case <-ctx.Done():
			return
		default:
			connectionCount++
			atomic.AddInt64(&e.TotalConnections, 1)
			atomic.AddInt64(&e.ActiveConnections, 1)
			
			// 更新最大活跃连接数
			currentActive := atomic.LoadInt64(&e.ActiveConnections)
			for {
				maxActive := atomic.LoadInt64(&e.MaxActiveConn)
				if currentActive <= maxActive || atomic.CompareAndSwapInt64(&e.MaxActiveConn, maxActive, currentActive) {
					break
				}
			}
			
			startTime := time.Now()
			success := e.testSingleConnection(workerID, connectionCount)
			responseTime := time.Since(startTime)
			
			atomic.AddInt64(&e.ActiveConnections, -1)
			
			if success {
				atomic.AddInt64(&e.SuccessConnections, 1)
			} else {
				atomic.AddInt64(&e.FailedConnections, 1)
			}
			
			// 记录结果
			result := ConnectionResult{
				ConnectionID:     connectionCount,
				Success:          success,
				ResponseTime:     responseTime,
				Timestamp:        time.Now(),
				ConcurrencyLevel: concurrency,
			}
			
			e.ResultsMutex.Lock()
			e.Results = append(e.Results, result)
			e.ResultsMutex.Unlock()
			
			// 动态调整连接间隔
			interval := time.Duration(10+connectionCount%50) * time.Millisecond
			time.Sleep(interval)
		}
	}
}

// testSingleConnection 测试单个连接
func (e *ExtremeConcurrencyTester) testSingleConnection(workerID, connectionID int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", e.Host, e.Port), 5*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	
	// 设置读写超时
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	
	// 尝试SMTP握手
	buffer := make([]byte, 1024)
	_, err = conn.Read(buffer) // 读取欢迎消息
	if err != nil {
		return false
	}
	
	// 发送EHLO命令
	_, err = conn.Write([]byte("EHLO extreme-test\r\n"))
	if err != nil {
		return false
	}
	
	// 读取响应
	_, err = conn.Read(buffer)
	if err != nil {
		return false
	}
	
	// 发送QUIT命令
	conn.Write([]byte("QUIT\r\n"))
	
	return true
}

// monitorConnections 监控连接状态
func (e *ExtremeConcurrencyTester) monitorConnections(ctx context.Context, startTime time.Time) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := time.Since(startTime)
			total := atomic.LoadInt64(&e.TotalConnections)
			active := atomic.LoadInt64(&e.ActiveConnections)
			success := atomic.LoadInt64(&e.SuccessConnections)
			failed := atomic.LoadInt64(&e.FailedConnections)
			
			var successRate float64
			if total > 0 {
				successRate = float64(success) / float64(total) * 100
			}
			
			fmt.Printf("[%v] 总连接: %d, 活跃: %d, 成功: %d, 失败: %d, 成功率: %.1f%%\n",
				elapsed.Truncate(time.Second), total, active, success, failed, successRate)
		}
	}
}

// TestStairCaseLoad 阶梯式负载测试
func (e *ExtremeConcurrencyTester) TestStairCaseLoad() {
	fmt.Println("\n📈 阶梯式负载测试")
	fmt.Println("逐步增加并发连接数，观察性能表现")
	
	concurrencyLevels := []int{10, 25, 50, 75, 100, 150, 200}
	stepDuration := 30 * time.Second
	
	for _, concurrency := range concurrencyLevels {
		fmt.Printf("\n--- 测试阶段: %d并发 ---\n", concurrency)
		
		// 重置计数器
		atomic.StoreInt64(&e.TotalConnections, 0)
		atomic.StoreInt64(&e.ActiveConnections, 0)
		atomic.StoreInt64(&e.SuccessConnections, 0)
		atomic.StoreInt64(&e.FailedConnections, 0)
		atomic.StoreInt64(&e.MaxActiveConn, 0)
		
		e.TestExtremeConnections(concurrency, stepDuration)
		
		// 休息5秒后进入下一阶段
		fmt.Println("⏸️ 休息5秒...")
		time.Sleep(5 * time.Second)
	}
}

// TestBurstLoad 突发负载测试
func (e *ExtremeConcurrencyTester) TestBurstLoad() {
	fmt.Println("\n💥 突发负载测试")
	fmt.Println("短时间内产生大量并发连接")
	
	burstSizes := []int{50, 100, 200, 500}
	
	for _, burstSize := range burstSizes {
		fmt.Printf("\n--- 突发测试: %d个连接 ---\n", burstSize)
		
		startTime := time.Now()
		var wg sync.WaitGroup
		successCount := int64(0)
		
		// 同时发起大量连接
		for i := 0; i < burstSize; i++ {
			wg.Add(1)
			go func(connID int) {
				defer wg.Done()
				
				startConn := time.Now()
				success := e.testSingleConnection(0, connID)
				responseTime := time.Since(startConn)
				
				if success {
					atomic.AddInt64(&successCount, 1)
				}
				
				result := ConnectionResult{
					ConnectionID:     connID,
					Success:          success,
					ResponseTime:     responseTime,
					Timestamp:        time.Now(),
					ConcurrencyLevel: burstSize,
				}
				
				e.ResultsMutex.Lock()
				e.Results = append(e.Results, result)
				e.ResultsMutex.Unlock()
			}(i)
		}
		
		wg.Wait()
		totalTime := time.Since(startTime)
		successRate := float64(successCount) / float64(burstSize) * 100
		
		fmt.Printf("✅ 突发测试完成: %d/%d成功 (%.1f%%), 耗时: %v\n",
			successCount, burstSize, successRate, totalTime)
		
		// 休息3秒
		time.Sleep(3 * time.Second)
	}
}

// printExtremeResults 打印极限测试结果
func (e *ExtremeConcurrencyTester) printExtremeResults(concurrency int, duration time.Duration) {
	total := atomic.LoadInt64(&e.TotalConnections)
	success := atomic.LoadInt64(&e.SuccessConnections)
	failed := atomic.LoadInt64(&e.FailedConnections)
	maxActive := atomic.LoadInt64(&e.MaxActiveConn)
	
	var successRate float64
	if total > 0 {
		successRate = float64(success) / float64(total) * 100
	}
	
	throughput := float64(total) / duration.Seconds()
	
	fmt.Printf("\n📊 测试结果摘要 (%d并发):\n", concurrency)
	fmt.Printf("总连接数: %d\n", total)
	fmt.Printf("成功连接: %d\n", success)
	fmt.Printf("失败连接: %d\n", failed)
	fmt.Printf("成功率: %.2f%%\n", successRate)
	fmt.Printf("最大同时活跃连接: %d\n", maxActive)
	fmt.Printf("平均吞吐量: %.2f conn/s\n", throughput)
	fmt.Printf("测试持续时间: %v\n", duration)
	
	// 计算响应时间统计
	if len(e.Results) > 0 {
		e.ResultsMutex.Lock()
		responseTimes := make([]time.Duration, len(e.Results))
		for i, result := range e.Results {
			responseTimes[i] = result.ResponseTime
		}
		e.ResultsMutex.Unlock()
		
		sort.Slice(responseTimes, func(i, j int) bool {
			return responseTimes[i] < responseTimes[j]
		})
		
		count := len(responseTimes)
		avg := time.Duration(0)
		for _, rt := range responseTimes {
			avg += rt
		}
		avg /= time.Duration(count)
		
		p50 := responseTimes[count*50/100]
		p95 := responseTimes[count*95/100]
		p99 := responseTimes[count*99/100]
		
		fmt.Printf("\n⏱️ 响应时间统计:\n")
		fmt.Printf("平均响应时间: %v\n", avg)
		fmt.Printf("最快响应时间: %v\n", responseTimes[0])
		fmt.Printf("最慢响应时间: %v\n", responseTimes[count-1])
		fmt.Printf("50%% 响应时间: %v\n", p50)
		fmt.Printf("95%% 响应时间: %v\n", p95)
		fmt.Printf("99%% 响应时间: %v\n", p99)
	}
}

// GenerateReport 生成详细报告
func (e *ExtremeConcurrencyTester) GenerateReport() {
	timestamp := time.Now().Format("20060102_150405")
	reportFile := fmt.Sprintf("results/extreme_load_test_%s.txt", timestamp)
	
	file, err := os.Create(reportFile)
	if err != nil {
		log.Printf("创建报告文件失败: %v", err)
		return
	}
	defer file.Close()
	
	fmt.Fprintf(file, "极限并发负载测试报告\n")
	fmt.Fprintf(file, "=====================\n\n")
	fmt.Fprintf(file, "测试时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "目标服务器: %s:%d\n\n", e.Host, e.Port)
	
	// 按并发级别分组统计
	concurrencyStats := make(map[int][]ConnectionResult)
	e.ResultsMutex.Lock()
	for _, result := range e.Results {
		level := result.ConcurrencyLevel
		concurrencyStats[level] = append(concurrencyStats[level], result)
	}
	e.ResultsMutex.Unlock()
	
	fmt.Fprintf(file, "并发级别性能统计:\n")
	fmt.Fprintf(file, "-----------------\n")
	
	for concurrency, results := range concurrencyStats {
		if len(results) == 0 {
			continue
		}
		
		success := 0
		var totalTime time.Duration
		for _, result := range results {
			if result.Success {
				success++
			}
			totalTime += result.ResponseTime
		}
		
		successRate := float64(success) / float64(len(results)) * 100
		avgResponseTime := totalTime / time.Duration(len(results))
		
		fmt.Fprintf(file, "\n%d并发级别:\n", concurrency)
		fmt.Fprintf(file, "  总连接数: %d\n", len(results))
		fmt.Fprintf(file, "  成功连接: %d\n", success)
		fmt.Fprintf(file, "  成功率: %.2f%%\n", successRate)
		fmt.Fprintf(file, "  平均响应时间: %v\n", avgResponseTime)
	}
	
	fmt.Printf("\n✅ 详细报告已保存: %s\n", reportFile)
}

func main() {
	// 确保results目录存在
	os.MkdirAll("results", 0755)
	
	// 默认测试本地SMTP服务器
	host := "localhost"
	port := 25
	
	// 支持命令行参数
	if len(os.Args) > 1 {
		host = os.Args[1]
	}
	if len(os.Args) > 2 {
		fmt.Sscanf(os.Args[2], "%d", &port)
	}
	
	tester := NewExtremeTester(host, port)
	
	fmt.Println("🔥 极限并发负载测试开始")
	fmt.Printf("目标服务器: %s:%d\n", host, port)
	fmt.Println("============================================")
	
	// 执行阶梯式负载测试
	tester.TestStairCaseLoad()
	
	// 执行突发负载测试
	tester.TestBurstLoad()
	
	// 生成报告
	tester.GenerateReport()
	
	fmt.Println("\n🎉 极限并发测试完成！")
}