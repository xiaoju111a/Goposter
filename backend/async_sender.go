package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"runtime"
	"sync"
	"time"
)

// AsyncEmailSender 异步邮件发送器
type AsyncEmailSender struct {
	workerPool    *WorkerPool
	smtpConfig    *SMTPConfig
	queue         *EmailQueue
	rateLimiter   *RateLimiter
	circuitBreaker *CircuitBreaker
	metrics       *SenderMetrics
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// SMTPConfig SMTP配置
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	UseTLS   bool
	Timeout  time.Duration
}

// WorkerPool 工作池
type WorkerPool struct {
	workers    int
	taskChan   chan *EmailTask
	resultChan chan *EmailResult
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// EmailResult 邮件发送结果
type EmailResult struct {
	TaskID    string
	Success   bool
	Error     error
	Duration  time.Duration
	Timestamp time.Time
}

// RateLimiter 速率限制器
type RateLimiter struct {
	limit    int
	window   time.Duration
	requests map[string][]time.Time
	mu       sync.RWMutex
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	maxFailures int
	timeout     time.Duration
	failures    int
	lastFailure time.Time
	state       string // "closed", "open", "half-open"
	mu          sync.RWMutex
}

// SenderMetrics 发送器指标
type SenderMetrics struct {
	TotalSent      int64         `json:"total_sent"`
	TotalFailed    int64         `json:"total_failed"`
	AverageLatency time.Duration `json:"average_latency"`
	CurrentRate    float64       `json:"current_rate"`
	LastSent       time.Time     `json:"last_sent"`
	ErrorRate      float64       `json:"error_rate"`
	mu             sync.RWMutex
}

// NewAsyncEmailSender 创建异步邮件发送器
func NewAsyncEmailSender(config *SMTPConfig, queue *EmailQueue, workers int) *AsyncEmailSender {
	if workers <= 0 {
		workers = runtime.NumCPU() * 2
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	sender := &AsyncEmailSender{
		smtpConfig: config,
		queue:      queue,
		ctx:        ctx,
		cancel:     cancel,
		rateLimiter: &RateLimiter{
			limit:    100, // 每分钟100封邮件
			window:   time.Minute,
			requests: make(map[string][]time.Time),
		},
		circuitBreaker: &CircuitBreaker{
			maxFailures: 5,
			timeout:     30 * time.Second,
			state:       "closed",
		},
		metrics: &SenderMetrics{},
	}
	
	sender.workerPool = &WorkerPool{
		workers:    workers,
		taskChan:   make(chan *EmailTask, 1000),
		resultChan: make(chan *EmailResult, 1000),
		ctx:        ctx,
		cancel:     cancel,
	}
	
	return sender
}

// Start 启动异步发送器
func (aes *AsyncEmailSender) Start() {
	log.Printf("Starting async email sender with %d workers", aes.workerPool.workers)
	
	// 启动工作池
	aes.startWorkerPool()
	
	// 启动结果处理器
	aes.wg.Add(1)
	go aes.resultHandler()
	
	// 启动指标收集器
	aes.wg.Add(1)
	go aes.metricsCollector()
	
	// 启动熔断器监控
	aes.wg.Add(1)
	go aes.circuitBreakerMonitor()
	
	log.Println("Async email sender started successfully")
}

// Stop 停止异步发送器
func (aes *AsyncEmailSender) Stop() {
	log.Println("Stopping async email sender...")
	
	aes.cancel()
	aes.workerPool.cancel()
	
	// 等待所有工作者完成
	aes.workerPool.wg.Wait()
	aes.wg.Wait()
	
	log.Println("Async email sender stopped")
}

// startWorkerPool 启动工作池
func (aes *AsyncEmailSender) startWorkerPool() {
	for i := 0; i < aes.workerPool.workers; i++ {
		aes.workerPool.wg.Add(1)
		go aes.worker(i)
	}
}

// worker 工作者
func (aes *AsyncEmailSender) worker(id int) {
	defer aes.workerPool.wg.Done()
	
	log.Printf("Email sender worker %d started", id)
	
	for {
		select {
		case <-aes.workerPool.ctx.Done():
			log.Printf("Email sender worker %d stopping", id)
			return
		case task := <-aes.workerPool.taskChan:
			result := aes.processEmailTask(task)
			
			select {
			case aes.workerPool.resultChan <- result:
			case <-aes.workerPool.ctx.Done():
				return
			}
		}
	}
}

// processEmailTask 处理邮件任务
func (aes *AsyncEmailSender) processEmailTask(task *EmailTask) *EmailResult {
	start := time.Now()
	
	result := &EmailResult{
		TaskID:    task.ID,
		Timestamp: start,
	}
	
	// 检查熔断器状态
	if !aes.circuitBreaker.canExecute() {
		result.Success = false
		result.Error = fmt.Errorf("circuit breaker is open")
		result.Duration = time.Since(start)
		return result
	}
	
	// 检查速率限制
	if !aes.rateLimiter.allow(task.From) {
		result.Success = false
		result.Error = fmt.Errorf("rate limit exceeded for %s", task.From)
		result.Duration = time.Since(start)
		return result
	}
	
	// 发送邮件
	err := aes.sendEmail(task)
	
	result.Duration = time.Since(start)
	
	if err != nil {
		result.Success = false
		result.Error = err
		aes.circuitBreaker.recordFailure()
	} else {
		result.Success = true
		aes.circuitBreaker.recordSuccess()
	}
	
	return result
}

// sendEmail 发送邮件
func (aes *AsyncEmailSender) sendEmail(task *EmailTask) error {
	// 构建邮件地址
	addr := fmt.Sprintf("%s:%d", aes.smtpConfig.Host, aes.smtpConfig.Port)
	
	// 创建TLS配置
	tlsConfig := &tls.Config{
		ServerName: aes.smtpConfig.Host,
		MinVersion: tls.VersionTLS12,
	}
	
	// 建立连接
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %v", err)
	}
	defer conn.Close()
	
	// 创建SMTP客户端
	client, err := smtp.NewClient(conn, aes.smtpConfig.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %v", err)
	}
	defer client.Quit()
	
	// 认证
	if aes.smtpConfig.Username != "" && aes.smtpConfig.Password != "" {
		auth := smtp.PlainAuth("", aes.smtpConfig.Username, aes.smtpConfig.Password, aes.smtpConfig.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %v", err)
		}
	}
	
	// 设置发件人
	if err := client.Mail(task.From); err != nil {
		return fmt.Errorf("failed to set sender: %v", err)
	}
	
	// 设置收件人
	for _, to := range task.To {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("failed to set recipient %s: %v", to, err)
		}
	}
	
	// 发送邮件内容
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to start data transmission: %v", err)
	}
	
	// 构建邮件内容
	message := aes.buildEmailMessage(task)
	
	if _, err := writer.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to write email data: %v", err)
	}
	
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close email data: %v", err)
	}
	
	return nil
}

// buildEmailMessage 构建邮件消息
func (aes *AsyncEmailSender) buildEmailMessage(task *EmailTask) string {
	message := fmt.Sprintf("From: %s\r\n", task.From)
	message += fmt.Sprintf("To: %s\r\n", task.To[0])
	
	if len(task.To) > 1 {
		message += fmt.Sprintf("Cc: %s\r\n", task.To[1:])
	}
	
	message += fmt.Sprintf("Subject: %s\r\n", task.Subject)
	message += fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	message += fmt.Sprintf("Message-ID: <%s@%s>\r\n", task.ID, aes.smtpConfig.Host)
	
	// 添加自定义头部
	for key, value := range task.Headers {
		message += fmt.Sprintf("%s: %s\r\n", key, value)
	}
	
	message += "Content-Type: text/plain; charset=UTF-8\r\n"
	message += "\r\n"
	message += task.Body
	
	return message
}

// resultHandler 结果处理器
func (aes *AsyncEmailSender) resultHandler() {
	defer aes.wg.Done()
	
	for {
		select {
		case <-aes.ctx.Done():
			return
		case result := <-aes.workerPool.resultChan:
			aes.handleResult(result)
		}
	}
}

// handleResult 处理结果
func (aes *AsyncEmailSender) handleResult(result *EmailResult) {
	aes.metrics.mu.Lock()
	defer aes.metrics.mu.Unlock()
	
	if result.Success {
		aes.metrics.TotalSent++
		aes.metrics.LastSent = result.Timestamp
		log.Printf("Email sent successfully: %s (took %v)", result.TaskID, result.Duration)
	} else {
		aes.metrics.TotalFailed++
		log.Printf("Email failed: %s - %v (took %v)", result.TaskID, result.Error, result.Duration)
	}
	
	// 更新平均延迟
	total := aes.metrics.TotalSent + aes.metrics.TotalFailed
	if total > 0 {
		aes.metrics.AverageLatency = (aes.metrics.AverageLatency*time.Duration(total-1) + result.Duration) / time.Duration(total)
		aes.metrics.ErrorRate = float64(aes.metrics.TotalFailed) / float64(total) * 100
	}
}

// metricsCollector 指标收集器
func (aes *AsyncEmailSender) metricsCollector() {
	defer aes.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	var lastSent int64
	
	for {
		select {
		case <-aes.ctx.Done():
			return
		case <-ticker.C:
			aes.metrics.mu.Lock()
			currentSent := aes.metrics.TotalSent
			aes.metrics.CurrentRate = float64(currentSent-lastSent) / 30.0 // 每秒发送率
			lastSent = currentSent
			aes.metrics.mu.Unlock()
		}
	}
}

// circuitBreakerMonitor 熔断器监控
func (aes *AsyncEmailSender) circuitBreakerMonitor() {
	defer aes.wg.Done()
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-aes.ctx.Done():
			return
		case <-ticker.C:
			aes.circuitBreaker.mu.Lock()
			if aes.circuitBreaker.state == "open" && 
			   time.Since(aes.circuitBreaker.lastFailure) > aes.circuitBreaker.timeout {
				aes.circuitBreaker.state = "half-open"
				aes.circuitBreaker.failures = 0
				log.Println("Circuit breaker switched to half-open state")
			}
			aes.circuitBreaker.mu.Unlock()
		}
	}
}

// SendEmailAsync 异步发送邮件
func (aes *AsyncEmailSender) SendEmailAsync(task *EmailTask) error {
	select {
	case aes.workerPool.taskChan <- task:
		return nil
	case <-aes.ctx.Done():
		return fmt.Errorf("sender is shutting down")
	default:
		return fmt.Errorf("task queue is full")
	}
}

// GetMetrics 获取发送器指标
func (aes *AsyncEmailSender) GetMetrics() *SenderMetrics {
	aes.metrics.mu.RLock()
	defer aes.metrics.mu.RUnlock()
	
	return &SenderMetrics{
		TotalSent:      aes.metrics.TotalSent,
		TotalFailed:    aes.metrics.TotalFailed,
		AverageLatency: aes.metrics.AverageLatency,
		CurrentRate:    aes.metrics.CurrentRate,
		LastSent:       aes.metrics.LastSent,
		ErrorRate:      aes.metrics.ErrorRate,
	}
}

// GetStatus 获取发送器状态
func (aes *AsyncEmailSender) GetStatus() map[string]interface{} {
	metrics := aes.GetMetrics()
	
	aes.circuitBreaker.mu.RLock()
	cbState := aes.circuitBreaker.state
	cbFailures := aes.circuitBreaker.failures
	aes.circuitBreaker.mu.RUnlock()
	
	return map[string]interface{}{
		"workers":         aes.workerPool.workers,
		"queue_size":      len(aes.workerPool.taskChan),
		"metrics":         metrics,
		"circuit_breaker": map[string]interface{}{
			"state":    cbState,
			"failures": cbFailures,
		},
		"rate_limiter": map[string]interface{}{
			"limit":  aes.rateLimiter.limit,
			"window": aes.rateLimiter.window,
		},
	}
}

// RateLimiter 方法

// allow 检查是否允许发送
func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	
	// 清理过期记录
	if requests, exists := rl.requests[key]; exists {
		var validRequests []time.Time
		for _, req := range requests {
			if now.Sub(req) < rl.window {
				validRequests = append(validRequests, req)
			}
		}
		rl.requests[key] = validRequests
	}
	
	// 检查是否超过限制
	if len(rl.requests[key]) >= rl.limit {
		return false
	}
	
	// 添加当前请求
	rl.requests[key] = append(rl.requests[key], now)
	return true
}

// CircuitBreaker 方法

// canExecute 检查是否可以执行
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	return cb.state == "closed" || cb.state == "half-open"
}

// recordSuccess 记录成功
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.failures = 0
	if cb.state == "half-open" {
		cb.state = "closed"
		log.Println("Circuit breaker closed")
	}
}

// recordFailure 记录失败
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.failures++
	cb.lastFailure = time.Now()
	
	if cb.failures >= cb.maxFailures {
		cb.state = "open"
		log.Printf("Circuit breaker opened after %d failures", cb.failures)
	}
}