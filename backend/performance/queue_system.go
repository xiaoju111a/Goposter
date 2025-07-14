package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// EmailQueue 邮件队列系统
type EmailQueue struct {
	redis         *redis.Client
	workers       int
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	incomingQueue chan *EmailTask
	outgoingQueue chan *EmailTask
	processingQueue chan *EmailTask
	failedQueue   chan *EmailTask
	metrics       *QueueMetrics
	mu            sync.RWMutex
}

// EmailTask 邮件任务
type EmailTask struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "incoming", "outgoing", "processing"
	Priority    int                    `json:"priority"`
	Mailbox     string                 `json:"mailbox"`
	From        string                 `json:"from"`
	To          []string               `json:"to"`
	Subject     string                 `json:"subject"`
	Body        string                 `json:"body"`
	Headers     map[string]interface{} `json:"headers"`
	Attachments []AttachmentData       `json:"attachments"`
	CreatedAt   time.Time              `json:"created_at"`
	ProcessedAt time.Time              `json:"processed_at"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
	Status      string                 `json:"status"`
	Error       string                 `json:"error"`
}

// AttachmentData 附件数据
type AttachmentData struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Data        []byte `json:"data"`
}

// QueueMetrics 队列指标
type QueueMetrics struct {
	TotalProcessed    int64     `json:"total_processed"`
	TotalFailed       int64     `json:"total_failed"`
	CurrentIncoming   int64     `json:"current_incoming"`
	CurrentOutgoing   int64     `json:"current_outgoing"`
	CurrentProcessing int64     `json:"current_processing"`
	CurrentFailed     int64     `json:"current_failed"`
	AverageProcessTime float64  `json:"average_process_time"`
	LastProcessed     time.Time `json:"last_processed"`
	WorkerStats       map[int]*WorkerStats `json:"worker_stats"`
}

// WorkerStats 工作者统计
type WorkerStats struct {
	WorkerID       int       `json:"worker_id"`
	TasksProcessed int64     `json:"tasks_processed"`
	TasksFailed    int64     `json:"tasks_failed"`
	LastActive     time.Time `json:"last_active"`
	CurrentTask    string    `json:"current_task"`
}

// NewEmailQueue 创建邮件队列
func NewEmailQueue(redisAddr string, workers int) (*EmailQueue, error) {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       1, // 使用DB1用于队列
		PoolSize: 20,
	})
	
	// 测试连接
	ctx := context.Background()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %v", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	eq := &EmailQueue{
		redis:         rdb,
		workers:       workers,
		ctx:           ctx,
		cancel:        cancel,
		incomingQueue: make(chan *EmailTask, 1000),
		outgoingQueue: make(chan *EmailTask, 1000),
		processingQueue: make(chan *EmailTask, 500),
		failedQueue:   make(chan *EmailTask, 100),
		metrics: &QueueMetrics{
			WorkerStats: make(map[int]*WorkerStats),
		},
	}
	
	// 初始化工作者统计
	for i := 0; i < workers; i++ {
		eq.metrics.WorkerStats[i] = &WorkerStats{
			WorkerID: i,
		}
	}
	
	return eq, nil
}

// Start 启动队列系统
func (eq *EmailQueue) Start() {
	log.Printf("Starting email queue system with %d workers", eq.workers)
	
	// 启动工作者
	for i := 0; i < eq.workers; i++ {
		eq.wg.Add(1)
		go eq.worker(i)
	}
	
	// 启动队列处理器
	eq.wg.Add(3)
	go eq.processIncomingQueue()
	go eq.processOutgoingQueue()
	go eq.processFailedQueue()
	
	// 启动Redis队列监听
	eq.wg.Add(1)
	go eq.listenRedisQueue()
	
	// 启动指标收集器
	eq.wg.Add(1)
	go eq.metricsCollector()
	
	log.Println("Email queue system started successfully")
}

// Stop 停止队列系统
func (eq *EmailQueue) Stop() {
	log.Println("Stopping email queue system...")
	
	eq.cancel()
	
	// 关闭通道
	close(eq.incomingQueue)
	close(eq.outgoingQueue)
	close(eq.processingQueue)
	close(eq.failedQueue)
	
	// 等待所有工作者完成
	eq.wg.Wait()
	
	// 关闭Redis连接
	eq.redis.Close()
	
	log.Println("Email queue system stopped")
}

// worker 工作者
func (eq *EmailQueue) worker(id int) {
	defer eq.wg.Done()
	
	log.Printf("Worker %d started", id)
	
	for {
		select {
		case <-eq.ctx.Done():
			log.Printf("Worker %d stopping", id)
			return
		case task := <-eq.processingQueue:
			eq.processTask(id, task)
		}
	}
}

// processTask 处理任务
func (eq *EmailQueue) processTask(workerID int, task *EmailTask) {
	start := time.Now()
	
	// 更新工作者状态
	eq.mu.Lock()
	eq.metrics.WorkerStats[workerID].CurrentTask = task.ID
	eq.metrics.WorkerStats[workerID].LastActive = time.Now()
	eq.mu.Unlock()
	
	defer func() {
		eq.mu.Lock()
		eq.metrics.WorkerStats[workerID].CurrentTask = ""
		eq.metrics.WorkerStats[workerID].LastActive = time.Now()
		eq.mu.Unlock()
	}()
	
	var err error
	
	switch task.Type {
	case "incoming":
		err = eq.processIncomingEmail(task)
	case "outgoing":
		err = eq.processOutgoingEmail(task)
	default:
		err = fmt.Errorf("unknown task type: %s", task.Type)
	}
	
	processingTime := time.Since(start)
	
	if err != nil {
		log.Printf("Worker %d failed to process task %s: %v", workerID, task.ID, err)
		eq.handleTaskFailure(task, err)
		
		eq.mu.Lock()
		eq.metrics.WorkerStats[workerID].TasksFailed++
		eq.metrics.TotalFailed++
		eq.mu.Unlock()
	} else {
		log.Printf("Worker %d successfully processed task %s in %v", workerID, task.ID, processingTime)
		eq.handleTaskSuccess(task)
		
		eq.mu.Lock()
		eq.metrics.WorkerStats[workerID].TasksProcessed++
		eq.metrics.TotalProcessed++
		eq.metrics.LastProcessed = time.Now()
		
		// 更新平均处理时间
		if eq.metrics.AverageProcessTime == 0 {
			eq.metrics.AverageProcessTime = processingTime.Seconds()
		} else {
			eq.metrics.AverageProcessTime = (eq.metrics.AverageProcessTime + processingTime.Seconds()) / 2
		}
		eq.mu.Unlock()
	}
}

// processIncomingEmail 处理接收邮件
func (eq *EmailQueue) processIncomingEmail(task *EmailTask) error {
	// 模拟邮件处理
	time.Sleep(100 * time.Millisecond)
	
	// 这里应该调用实际的邮件处理逻辑
	// 例如：存储到数据库、病毒扫描、垃圾邮件过滤等
	
	log.Printf("Processing incoming email: %s -> %s", task.From, task.Mailbox)
	
	// 存储邮件数据
	_ = map[string]interface{}{
		"from":    task.From,
		"to":      task.To,
		"subject": task.Subject,
		"body":    task.Body,
		"headers": task.Headers,
	}
	
	// 这里应该调用数据库存储
	// return database.StoreEncryptedEmail(task.Mailbox, emailData)
	
	return nil
}

// processOutgoingEmail 处理发送邮件
func (eq *EmailQueue) processOutgoingEmail(task *EmailTask) error {
	// 模拟邮件发送
	time.Sleep(200 * time.Millisecond)
	
	log.Printf("Processing outgoing email: %s -> %v", task.From, task.To)
	
	// 这里应该调用实际的邮件发送逻辑
	// 例如：SMTP发送、中继处理等
	
	return nil
}

// processIncomingQueue 处理接收队列
func (eq *EmailQueue) processIncomingQueue() {
	defer eq.wg.Done()
	
	for {
		select {
		case <-eq.ctx.Done():
			return
		case task := <-eq.incomingQueue:
			eq.processingQueue <- task
		}
	}
}

// processOutgoingQueue 处理发送队列
func (eq *EmailQueue) processOutgoingQueue() {
	defer eq.wg.Done()
	
	for {
		select {
		case <-eq.ctx.Done():
			return
		case task := <-eq.outgoingQueue:
			eq.processingQueue <- task
		}
	}
}

// processFailedQueue 处理失败队列
func (eq *EmailQueue) processFailedQueue() {
	defer eq.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-eq.ctx.Done():
			return
		case <-ticker.C:
			eq.retryFailedTasks()
		case task := <-eq.failedQueue:
			eq.storeFailedTask(task)
		}
	}
}

// listenRedisQueue 监听Redis队列
func (eq *EmailQueue) listenRedisQueue() {
	defer eq.wg.Done()
	
	for {
		select {
		case <-eq.ctx.Done():
			return
		default:
			// 监听Redis队列
			result, err := eq.redis.BLPop(eq.ctx, 1*time.Second, "email_queue:incoming", "email_queue:outgoing").Result()
			if err != nil {
				if err != redis.Nil {
					log.Printf("Redis queue listen error: %v", err)
				}
				continue
			}
			
			if len(result) == 2 {
				queueName := result[0]
				taskData := result[1]
				
				var task EmailTask
				if err := json.Unmarshal([]byte(taskData), &task); err != nil {
					log.Printf("Failed to unmarshal task: %v", err)
					continue
				}
				
				switch queueName {
				case "email_queue:incoming":
					select {
					case eq.incomingQueue <- &task:
					case <-eq.ctx.Done():
						return
					}
				case "email_queue:outgoing":
					select {
					case eq.outgoingQueue <- &task:
					case <-eq.ctx.Done():
						return
					}
				}
			}
		}
	}
}

// metricsCollector 指标收集器
func (eq *EmailQueue) metricsCollector() {
	defer eq.wg.Done()
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-eq.ctx.Done():
			return
		case <-ticker.C:
			eq.updateMetrics()
		}
	}
}

// updateMetrics 更新指标
func (eq *EmailQueue) updateMetrics() {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	
	eq.metrics.CurrentIncoming = int64(len(eq.incomingQueue))
	eq.metrics.CurrentOutgoing = int64(len(eq.outgoingQueue))
	eq.metrics.CurrentProcessing = int64(len(eq.processingQueue))
	eq.metrics.CurrentFailed = int64(len(eq.failedQueue))
	
	// 存储指标到Redis
	metricsData, _ := json.Marshal(eq.metrics)
	eq.redis.Set(eq.ctx, "email_queue:metrics", metricsData, 1*time.Hour)
}

// AddIncomingEmail 添加接收邮件到队列
func (eq *EmailQueue) AddIncomingEmail(from, to, subject, body string, headers map[string]interface{}) error {
	task := &EmailTask{
		ID:        fmt.Sprintf("incoming_%d", time.Now().UnixNano()),
		Type:      "incoming",
		Priority:  1,
		Mailbox:   to,
		From:      from,
		To:        []string{to},
		Subject:   subject,
		Body:      body,
		Headers:   headers,
		CreatedAt: time.Now(),
		Status:    "pending",
		MaxRetries: 3,
	}
	
	taskData, err := json.Marshal(task)
	if err != nil {
		return err
	}
	
	// 添加到Redis队列
	return eq.redis.RPush(eq.ctx, "email_queue:incoming", taskData).Err()
}

// AddOutgoingEmail 添加发送邮件到队列
func (eq *EmailQueue) AddOutgoingEmail(from string, to []string, subject, body string, headers map[string]interface{}) error {
	task := &EmailTask{
		ID:        fmt.Sprintf("outgoing_%d", time.Now().UnixNano()),
		Type:      "outgoing",
		Priority:  1,
		From:      from,
		To:        to,
		Subject:   subject,
		Body:      body,
		Headers:   headers,
		CreatedAt: time.Now(),
		Status:    "pending",
		MaxRetries: 3,
	}
	
	taskData, err := json.Marshal(task)
	if err != nil {
		return err
	}
	
	// 添加到Redis队列
	return eq.redis.RPush(eq.ctx, "email_queue:outgoing", taskData).Err()
}

// handleTaskFailure 处理任务失败
func (eq *EmailQueue) handleTaskFailure(task *EmailTask, err error) {
	task.RetryCount++
	task.Error = err.Error()
	task.Status = "failed"
	
	if task.RetryCount < task.MaxRetries {
		// 重新排队
		task.Status = "retry"
		select {
		case eq.failedQueue <- task:
		default:
			log.Printf("Failed queue is full, dropping task %s", task.ID)
		}
	} else {
		// 永久失败
		log.Printf("Task %s permanently failed after %d retries", task.ID, task.RetryCount)
		eq.storeFailedTask(task)
	}
}

// handleTaskSuccess 处理任务成功
func (eq *EmailQueue) handleTaskSuccess(task *EmailTask) {
	task.Status = "completed"
	task.ProcessedAt = time.Now()
	
	// 可以选择存储成功的任务记录
	taskData, _ := json.Marshal(task)
	eq.redis.Set(eq.ctx, fmt.Sprintf("email_queue:completed:%s", task.ID), taskData, 24*time.Hour)
}

// storeFailedTask 存储失败任务
func (eq *EmailQueue) storeFailedTask(task *EmailTask) {
	taskData, _ := json.Marshal(task)
	eq.redis.Set(eq.ctx, fmt.Sprintf("email_queue:failed:%s", task.ID), taskData, 7*24*time.Hour)
}

// retryFailedTasks 重试失败任务
func (eq *EmailQueue) retryFailedTasks() {
	keys, err := eq.redis.Keys(eq.ctx, "email_queue:failed:*").Result()
	if err != nil {
		return
	}
	
	for _, key := range keys {
		taskData, err := eq.redis.Get(eq.ctx, key).Result()
		if err != nil {
			continue
		}
		
		var task EmailTask
		if err := json.Unmarshal([]byte(taskData), &task); err != nil {
			continue
		}
		
		if task.Status == "retry" && time.Since(task.ProcessedAt) > 5*time.Minute {
			// 重新添加到队列
			newTaskData, _ := json.Marshal(&task)
			
			switch task.Type {
			case "incoming":
				eq.redis.RPush(eq.ctx, "email_queue:incoming", newTaskData)
			case "outgoing":
				eq.redis.RPush(eq.ctx, "email_queue:outgoing", newTaskData)
			}
			
			// 删除失败记录
			eq.redis.Del(eq.ctx, key)
		}
	}
}

// GetMetrics 获取队列指标
func (eq *EmailQueue) GetMetrics() *QueueMetrics {
	eq.mu.RLock()
	defer eq.mu.RUnlock()
	
	// 创建副本
	metrics := &QueueMetrics{
		TotalProcessed:    eq.metrics.TotalProcessed,
		TotalFailed:       eq.metrics.TotalFailed,
		CurrentIncoming:   eq.metrics.CurrentIncoming,
		CurrentOutgoing:   eq.metrics.CurrentOutgoing,
		CurrentProcessing: eq.metrics.CurrentProcessing,
		CurrentFailed:     eq.metrics.CurrentFailed,
		AverageProcessTime: eq.metrics.AverageProcessTime,
		LastProcessed:     eq.metrics.LastProcessed,
		WorkerStats:       make(map[int]*WorkerStats),
	}
	
	for k, v := range eq.metrics.WorkerStats {
		metrics.WorkerStats[k] = &WorkerStats{
			WorkerID:       v.WorkerID,
			TasksProcessed: v.TasksProcessed,
			TasksFailed:    v.TasksFailed,
			LastActive:     v.LastActive,
			CurrentTask:    v.CurrentTask,
		}
	}
	
	return metrics
}

// GetQueueStatus 获取队列状态
func (eq *EmailQueue) GetQueueStatus() map[string]interface{} {
	metrics := eq.GetMetrics()
	
	return map[string]interface{}{
		"workers": eq.workers,
		"queues": map[string]interface{}{
			"incoming":   metrics.CurrentIncoming,
			"outgoing":   metrics.CurrentOutgoing,
			"processing": metrics.CurrentProcessing,
			"failed":     metrics.CurrentFailed,
		},
		"stats": map[string]interface{}{
			"total_processed":     metrics.TotalProcessed,
			"total_failed":        metrics.TotalFailed,
			"average_process_time": metrics.AverageProcessTime,
			"last_processed":      metrics.LastProcessed,
		},
		"workers_stats": metrics.WorkerStats,
	}
}