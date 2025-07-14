package main

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// MemoryMonitor 内存监控器
type MemoryMonitor struct {
	config     *MemoryConfig
	metrics    *MemoryMetrics
	alerts     *AlertManager
	optimizer  *MemoryOptimizer
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
}

// MemoryConfig 内存配置
type MemoryConfig struct {
	MaxMemoryUsage     uint64        `json:"max_memory_usage"`     // 最大内存使用量 (字节)
	WarningThreshold   float64       `json:"warning_threshold"`   // 警告阈值 (百分比)
	CriticalThreshold  float64       `json:"critical_threshold"`  // 严重阈值 (百分比)
	GCFrequency        time.Duration `json:"gc_frequency"`        // GC频率
	MonitorInterval    time.Duration `json:"monitor_interval"`    // 监控间隔
	OptimizationMode   string        `json:"optimization_mode"`   // 优化模式: "conservative", "aggressive", "balanced"
	EnableProfiling    bool          `json:"enable_profiling"`    // 启用性能分析
	EnableMemoryLimit  bool          `json:"enable_memory_limit"` // 启用内存限制
}

// MemoryMetrics 内存指标
type MemoryMetrics struct {
	Timestamp         time.Time `json:"timestamp"`
	AllocatedMemory   uint64    `json:"allocated_memory"`   // 已分配内存
	TotalAllocated    uint64    `json:"total_allocated"`    // 总分配内存
	SystemMemory      uint64    `json:"system_memory"`      // 系统内存
	HeapMemory        uint64    `json:"heap_memory"`        // 堆内存
	StackMemory       uint64    `json:"stack_memory"`       // 栈内存
	GCCount           uint32    `json:"gc_count"`           // GC次数
	GCPauseTime       uint64    `json:"gc_pause_time"`      // GC暂停时间
	GoroutineCount    int       `json:"goroutine_count"`    // 协程数量
	MemoryUsagePercent float64  `json:"memory_usage_percent"` // 内存使用百分比
	PeakMemoryUsage   uint64    `json:"peak_memory_usage"`  // 峰值内存使用
}

// AlertManager 告警管理器
type AlertManager struct {
	alerts    []MemoryAlert
	callbacks map[string]func(alert MemoryAlert)
	mu        sync.RWMutex
}

// MemoryAlert 内存告警
type MemoryAlert struct {
	Level     string    `json:"level"`     // "warning", "critical", "info"
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Metrics   MemoryMetrics `json:"metrics"`
	Resolved  bool      `json:"resolved"`
}

// MemoryOptimizer 内存优化器
type MemoryOptimizer struct {
	config          *MemoryConfig
	lastOptimization time.Time
	optimizationCount int64
	mu              sync.RWMutex
}

// MemoryCache 内存缓存管理
type MemoryCache struct {
	data      map[string]*CacheItem
	maxSize   int64
	currentSize int64
	mu        sync.RWMutex
}

// CacheItem 缓存项
type CacheItem struct {
	Key        string
	Value      interface{}
	Size       int64
	CreatedAt  time.Time
	LastAccessed time.Time
	AccessCount int64
}

// NewMemoryMonitor 创建内存监控器
func NewMemoryMonitor(config *MemoryConfig) *MemoryMonitor {
	if config == nil {
		config = getDefaultMemoryConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	monitor := &MemoryMonitor{
		config:  config,
		metrics: &MemoryMetrics{},
		alerts:  &AlertManager{
			alerts:    make([]MemoryAlert, 0),
			callbacks: make(map[string]func(alert MemoryAlert)),
		},
		optimizer: &MemoryOptimizer{
			config: config,
		},
		ctx:    ctx,
		cancel: cancel,
	}
	
	return monitor
}

// getDefaultMemoryConfig 获取默认内存配置
func getDefaultMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		MaxMemoryUsage:     1024 * 1024 * 1024, // 1GB
		WarningThreshold:   80.0,                // 80%
		CriticalThreshold:  90.0,                // 90%
		GCFrequency:        30 * time.Second,
		MonitorInterval:    10 * time.Second,
		OptimizationMode:   "balanced",
		EnableProfiling:    false,
		EnableMemoryLimit:  true,
	}
}

// Start 启动内存监控
func (mm *MemoryMonitor) Start() {
	log.Println("Starting memory monitor...")
	
	// 设置内存限制
	if mm.config.EnableMemoryLimit {
		debug.SetMemoryLimit(int64(mm.config.MaxMemoryUsage))
		log.Printf("Memory limit set to %d bytes", mm.config.MaxMemoryUsage)
	}
	
	// 启动内存监控
	mm.wg.Add(1)
	go mm.memoryMonitor()
	
	// 启动垃圾回收调度器
	mm.wg.Add(1)
	go mm.gcScheduler()
	
	// 启动内存优化器
	mm.wg.Add(1)
	go mm.memoryOptimizer()
	
	// 启动告警处理器
	mm.wg.Add(1)
	go mm.alertHandler()
	
	log.Println("Memory monitor started successfully")
}

// Stop 停止内存监控
func (mm *MemoryMonitor) Stop() {
	log.Println("Stopping memory monitor...")
	
	mm.cancel()
	mm.wg.Wait()
	
	log.Println("Memory monitor stopped")
}

// memoryMonitor 内存监控器
func (mm *MemoryMonitor) memoryMonitor() {
	defer mm.wg.Done()
	
	ticker := time.NewTicker(mm.config.MonitorInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-mm.ctx.Done():
			return
		case <-ticker.C:
			mm.collectMemoryMetrics()
			mm.analyzeMemoryUsage()
		}
	}
}

// collectMemoryMetrics 收集内存指标
func (mm *MemoryMonitor) collectMemoryMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	mm.metrics.Timestamp = time.Now()
	mm.metrics.AllocatedMemory = m.Alloc
	mm.metrics.TotalAllocated = m.TotalAlloc
	mm.metrics.SystemMemory = m.Sys
	mm.metrics.HeapMemory = m.HeapAlloc
	mm.metrics.StackMemory = m.StackSys
	mm.metrics.GCCount = m.NumGC
	mm.metrics.GCPauseTime = m.PauseNs[(m.NumGC+255)%256]
	mm.metrics.GoroutineCount = runtime.NumGoroutine()
	
	// 计算内存使用百分比
	if mm.config.MaxMemoryUsage > 0 {
		mm.metrics.MemoryUsagePercent = float64(m.Alloc) / float64(mm.config.MaxMemoryUsage) * 100
	}
	
	// 更新峰值内存使用
	if m.Alloc > mm.metrics.PeakMemoryUsage {
		mm.metrics.PeakMemoryUsage = m.Alloc
	}
}

// analyzeMemoryUsage 分析内存使用
func (mm *MemoryMonitor) analyzeMemoryUsage() {
	mm.mu.RLock()
	usagePercent := mm.metrics.MemoryUsagePercent
	metrics := mm.metrics
	mm.mu.RUnlock()
	
	// 检查警告阈值
	if usagePercent >= mm.config.WarningThreshold && usagePercent < mm.config.CriticalThreshold {
		alert := MemoryAlert{
			Level:     "warning",
			Message:   fmt.Sprintf("Memory usage is %.2f%%, exceeding warning threshold %.2f%%", usagePercent, mm.config.WarningThreshold),
			Timestamp: time.Now(),
			Metrics:   metrics,
			Resolved:  false,
		}
		mm.alerts.addAlert(alert)
	}
	
	// 检查严重阈值
	if usagePercent >= mm.config.CriticalThreshold {
		alert := MemoryAlert{
			Level:     "critical",
			Message:   fmt.Sprintf("Memory usage is %.2f%%, exceeding critical threshold %.2f%%", usagePercent, mm.config.CriticalThreshold),
			Timestamp: time.Now(),
			Metrics:   metrics,
			Resolved:  false,
		}
		mm.alerts.addAlert(alert)
		
		// 触发紧急优化
		go mm.optimizer.emergencyOptimization()
	}
}

// gcScheduler 垃圾回收调度器
func (mm *MemoryMonitor) gcScheduler() {
	defer mm.wg.Done()
	
	ticker := time.NewTicker(mm.config.GCFrequency)
	defer ticker.Stop()
	
	for {
		select {
		case <-mm.ctx.Done():
			return
		case <-ticker.C:
			mm.performGC()
		}
	}
}

// performGC 执行垃圾回收
func (mm *MemoryMonitor) performGC() {
	start := time.Now()
	runtime.GC()
	duration := time.Since(start)
	
	log.Printf("Manual GC completed in %v", duration)
	
	// 记录GC后的内存使用
	mm.collectMemoryMetrics()
}

// memoryOptimizer 内存优化器
func (mm *MemoryMonitor) memoryOptimizer() {
	defer mm.wg.Done()
	
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-mm.ctx.Done():
			return
		case <-ticker.C:
			mm.optimizer.optimize()
		}
	}
}

// optimize 内存优化
func (mo *MemoryOptimizer) optimize() {
	mo.mu.Lock()
	defer mo.mu.Unlock()
	
	switch mo.config.OptimizationMode {
	case "conservative":
		mo.conservativeOptimization()
	case "aggressive":
		mo.aggressiveOptimization()
	case "balanced":
		mo.balancedOptimization()
	}
	
	mo.lastOptimization = time.Now()
	mo.optimizationCount++
}

// conservativeOptimization 保守优化
func (mo *MemoryOptimizer) conservativeOptimization() {
	// 仅在内存使用超过阈值时进行优化
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	if float64(m.Alloc)/float64(mo.config.MaxMemoryUsage)*100 > mo.config.WarningThreshold {
		runtime.GC()
		debug.FreeOSMemory()
		log.Println("Conservative memory optimization performed")
	}
}

// aggressiveOptimization 激进优化
func (mo *MemoryOptimizer) aggressiveOptimization() {
	// 强制GC和内存释放
	runtime.GC()
	debug.FreeOSMemory()
	
	// 调整GC目标
	debug.SetGCPercent(50)
	
	log.Println("Aggressive memory optimization performed")
}

// balancedOptimization 平衡优化
func (mo *MemoryOptimizer) balancedOptimization() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	usagePercent := float64(m.Alloc) / float64(mo.config.MaxMemoryUsage) * 100
	
	if usagePercent > mo.config.WarningThreshold {
		runtime.GC()
		
		if usagePercent > mo.config.CriticalThreshold {
			debug.FreeOSMemory()
			debug.SetGCPercent(25)
		}
		
		log.Printf("Balanced memory optimization performed (usage: %.2f%%)", usagePercent)
	}
}

// emergencyOptimization 紧急优化
func (mo *MemoryOptimizer) emergencyOptimization() {
	log.Println("Emergency memory optimization triggered")
	
	// 强制多次GC
	for i := 0; i < 3; i++ {
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
	}
	
	// 释放OS内存
	debug.FreeOSMemory()
	
	// 设置更激进的GC目标
	debug.SetGCPercent(10)
	
	log.Println("Emergency memory optimization completed")
}

// alertHandler 告警处理器
func (mm *MemoryMonitor) alertHandler() {
	defer mm.wg.Done()
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-mm.ctx.Done():
			return
		case <-ticker.C:
			mm.alerts.processAlerts()
		}
	}
}

// addAlert 添加告警
func (am *AlertManager) addAlert(alert MemoryAlert) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	am.alerts = append(am.alerts, alert)
	
	// 触发回调
	for _, callback := range am.callbacks {
		go callback(alert)
	}
	
	log.Printf("Memory alert: %s - %s", alert.Level, alert.Message)
}

// processAlerts 处理告警
func (am *AlertManager) processAlerts() {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	// 清理旧告警
	now := time.Now()
	var activeAlerts []MemoryAlert
	
	for _, alert := range am.alerts {
		if now.Sub(alert.Timestamp) < 5*time.Minute {
			activeAlerts = append(activeAlerts, alert)
		}
	}
	
	am.alerts = activeAlerts
}

// RegisterAlertCallback 注册告警回调
func (mm *MemoryMonitor) RegisterAlertCallback(name string, callback func(alert MemoryAlert)) {
	mm.alerts.mu.Lock()
	defer mm.alerts.mu.Unlock()
	
	mm.alerts.callbacks[name] = callback
}

// GetMetrics 获取内存指标
func (mm *MemoryMonitor) GetMetrics() MemoryMetrics {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	return mm.metrics
}

// GetAlerts 获取告警
func (mm *MemoryMonitor) GetAlerts() []MemoryAlert {
	mm.alerts.mu.RLock()
	defer mm.alerts.mu.RUnlock()
	
	alerts := make([]MemoryAlert, len(mm.alerts.alerts))
	copy(alerts, mm.alerts.alerts)
	
	return alerts
}

// GetOptimizationStats 获取优化统计
func (mm *MemoryMonitor) GetOptimizationStats() map[string]interface{} {
	mm.optimizer.mu.RLock()
	defer mm.optimizer.mu.RUnlock()
	
	return map[string]interface{}{
		"optimization_count": mm.optimizer.optimizationCount,
		"last_optimization": mm.optimizer.lastOptimization,
		"optimization_mode":  mm.config.OptimizationMode,
	}
}

// GetMemoryStatus 获取内存状态
func (mm *MemoryMonitor) GetMemoryStatus() map[string]interface{} {
	metrics := mm.GetMetrics()
	alerts := mm.GetAlerts()
	optimizationStats := mm.GetOptimizationStats()
	
	return map[string]interface{}{
		"metrics":      metrics,
		"alerts":       alerts,
		"optimization": optimizationStats,
		"config": map[string]interface{}{
			"max_memory_usage":    mm.config.MaxMemoryUsage,
			"warning_threshold":   mm.config.WarningThreshold,
			"critical_threshold":  mm.config.CriticalThreshold,
			"gc_frequency":        mm.config.GCFrequency,
			"monitor_interval":    mm.config.MonitorInterval,
			"optimization_mode":   mm.config.OptimizationMode,
		},
	}
}

// UpdateConfig 更新配置
func (mm *MemoryMonitor) UpdateConfig(config *MemoryConfig) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	mm.config = config
	mm.optimizer.config = config
	
	// 更新内存限制
	if config.EnableMemoryLimit {
		debug.SetMemoryLimit(int64(config.MaxMemoryUsage))
	}
	
	log.Println("Memory monitor configuration updated")
}

// ForceOptimization 强制优化
func (mm *MemoryMonitor) ForceOptimization() {
	go mm.optimizer.optimize()
}

// ForceGC 强制垃圾回收
func (mm *MemoryMonitor) ForceGC() {
	mm.performGC()
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(maxSize int64) *MemoryCache {
	return &MemoryCache{
		data:    make(map[string]*CacheItem),
		maxSize: maxSize,
	}
}

// Set 设置缓存
func (mc *MemoryCache) Set(key string, value interface{}, size int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	// 检查是否需要清理
	if mc.currentSize+size > mc.maxSize {
		mc.evictLRU(size)
	}
	
	// 添加新项
	item := &CacheItem{
		Key:         key,
		Value:       value,
		Size:        size,
		CreatedAt:   time.Now(),
		LastAccessed: time.Now(),
		AccessCount: 1,
	}
	
	mc.data[key] = item
	mc.currentSize += size
}

// Get 获取缓存
func (mc *MemoryCache) Get(key string) (interface{}, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	item, exists := mc.data[key]
	if !exists {
		return nil, false
	}
	
	item.LastAccessed = time.Now()
	item.AccessCount++
	
	return item.Value, true
}

// evictLRU 清理最少使用的项
func (mc *MemoryCache) evictLRU(neededSize int64) {
	var oldestKey string
	var oldestTime time.Time
	
	for key, item := range mc.data {
		if oldestKey == "" || item.LastAccessed.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.LastAccessed
		}
	}
	
	if oldestKey != "" {
		item := mc.data[oldestKey]
		delete(mc.data, oldestKey)
		mc.currentSize -= item.Size
		
		// 如果还需要更多空间，继续清理
		if mc.currentSize+neededSize > mc.maxSize && len(mc.data) > 0 {
			mc.evictLRU(neededSize)
		}
	}
}

// GetCacheStats 获取缓存统计
func (mc *MemoryCache) GetCacheStats() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	return map[string]interface{}{
		"items":        len(mc.data),
		"current_size": mc.currentSize,
		"max_size":     mc.maxSize,
		"usage_percent": float64(mc.currentSize) / float64(mc.maxSize) * 100,
	}
}