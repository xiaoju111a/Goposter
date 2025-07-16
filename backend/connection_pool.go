package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// ConnectionPoolManager 连接池管理器
type ConnectionPoolManager struct {
	dbPool    *DatabasePool
	redisPool *RedisPool
	metrics   *PoolMetrics
	config    *PoolConfig
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// DatabasePool 数据库连接池
type DatabasePool struct {
	db          *sql.DB
	connections map[string]*PoolConnection
	available   chan *PoolConnection
	maxConns    int
	maxIdle     int
	maxLifetime time.Duration
	mu          sync.RWMutex
}

// RedisPool Redis连接池
type RedisPool struct {
	client   *redis.Client
	// pool     *redis.Pool // 临时禁用
	maxConns int
	maxIdle  int
	timeout  time.Duration
}

// PoolConnection 池连接
type PoolConnection struct {
	ID          string
	DB          *sql.DB
	CreatedAt   time.Time
	LastUsed    time.Time
	InUse       bool
	QueryCount  int64
	TotalTime   time.Duration
	mu          sync.RWMutex
}

// PoolConfig 连接池配置
type PoolConfig struct {
	Database struct {
		MaxOpenConns    int           `json:"max_open_conns"`
		MaxIdleConns    int           `json:"max_idle_conns"`
		ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
		ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
	} `json:"database"`
	Redis struct {
		MaxConns    int           `json:"max_conns"`
		MaxIdle     int           `json:"max_idle"`
		IdleTimeout time.Duration `json:"idle_timeout"`
		PoolSize    int           `json:"pool_size"`
	} `json:"redis"`
	Monitoring struct {
		MetricsInterval time.Duration `json:"metrics_interval"`
		HealthCheck     time.Duration `json:"health_check"`
	} `json:"monitoring"`
}

// PoolMetrics 连接池指标
type PoolMetrics struct {
	Database struct {
		OpenConnections  int           `json:"open_connections"`
		InUseConnections int           `json:"in_use_connections"`
		IdleConnections  int           `json:"idle_connections"`
		TotalQueries     int64         `json:"total_queries"`
		AverageQueryTime time.Duration `json:"average_query_time"`
		ConnectionErrors int64         `json:"connection_errors"`
	} `json:"database"`
	Redis struct {
		ActiveConnections int           `json:"active_connections"`
		IdleConnections   int           `json:"idle_connections"`
		TotalCommands     int64         `json:"total_commands"`
		AverageLatency    time.Duration `json:"average_latency"`
		Errors            int64         `json:"errors"`
	} `json:"redis"`
	System struct {
		MemoryUsage      uint64    `json:"memory_usage"`
		GoroutineCount   int       `json:"goroutine_count"`
		LastHealthCheck  time.Time `json:"last_health_check"`
		HealthyConnections int     `json:"healthy_connections"`
	} `json:"system"`
}

// NewConnectionPoolManager 创建连接池管理器
func NewConnectionPoolManager(dbPath string, redisAddr string, config *PoolConfig) (*ConnectionPoolManager, error) {
	if config == nil {
		config = getDefaultPoolConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &ConnectionPoolManager{
		config:  config,
		ctx:     ctx,
		cancel:  cancel,
		metrics: &PoolMetrics{},
	}
	
	// 初始化数据库连接池
	dbPool, err := newDatabasePool(dbPath, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %v", err)
	}
	manager.dbPool = dbPool
	
	// 初始化Redis连接池
	redisPool, err := newRedisPool(redisAddr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis pool: %v", err)
	}
	manager.redisPool = redisPool
	
	return manager, nil
}

// getDefaultPoolConfig 获取默认池配置
func getDefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		Database: struct {
			MaxOpenConns    int           `json:"max_open_conns"`
			MaxIdleConns    int           `json:"max_idle_conns"`
			ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
			ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
		}{
			MaxOpenConns:    50,
			MaxIdleConns:    10,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
		},
		Redis: struct {
			MaxConns    int           `json:"max_conns"`
			MaxIdle     int           `json:"max_idle"`
			IdleTimeout time.Duration `json:"idle_timeout"`
			PoolSize    int           `json:"pool_size"`
		}{
			MaxConns:    100,
			MaxIdle:     20,
			IdleTimeout: 5 * time.Minute,
			PoolSize:    20,
		},
		Monitoring: struct {
			MetricsInterval time.Duration `json:"metrics_interval"`
			HealthCheck     time.Duration `json:"health_check"`
		}{
			MetricsInterval: 30 * time.Second,
			HealthCheck:     1 * time.Minute,
		},
	}
}

// newDatabasePool 创建数据库连接池
func newDatabasePool(dbPath string, config *PoolConfig) (*DatabasePool, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, err
	}
	
	// 配置连接池
	db.SetMaxOpenConns(config.Database.MaxOpenConns)
	db.SetMaxIdleConns(config.Database.MaxIdleConns)
	db.SetConnMaxLifetime(config.Database.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.Database.ConnMaxIdleTime)
	
	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database ping failed: %v", err)
	}
	
	pool := &DatabasePool{
		db:          db,
		connections: make(map[string]*PoolConnection),
		available:   make(chan *PoolConnection, config.Database.MaxIdleConns),
		maxConns:    config.Database.MaxOpenConns,
		maxIdle:     config.Database.MaxIdleConns,
		maxLifetime: config.Database.ConnMaxLifetime,
	}
	
	return pool, nil
}

// newRedisPool 创建Redis连接池
func newRedisPool(addr string, config *PoolConfig) (*RedisPool, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     "",
		DB:           0,
		PoolSize:     config.Redis.PoolSize,
		MinIdleConns: config.Redis.MaxIdle,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		IdleTimeout:  config.Redis.IdleTimeout,
	})
	
	// 测试连接
	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %v", err)
	}
	
	pool := &RedisPool{
		client:   client,
		maxConns: config.Redis.MaxConns,
		maxIdle:  config.Redis.MaxIdle,
		timeout:  config.Redis.IdleTimeout,
	}
	
	return pool, nil
}

// Start 启动连接池管理器
func (cpm *ConnectionPoolManager) Start() {
	log.Println("Starting connection pool manager...")
	
	// 启动监控服务
	cpm.wg.Add(1)
	go cpm.metricsCollector()
	
	// 启动健康检查
	cpm.wg.Add(1)
	go cpm.healthChecker()
	
	// 启动连接清理器
	cpm.wg.Add(1)
	go cpm.connectionCleaner()
	
	log.Println("Connection pool manager started successfully")
}

// Stop 停止连接池管理器
func (cpm *ConnectionPoolManager) Stop() {
	log.Println("Stopping connection pool manager...")
	
	cpm.cancel()
	cpm.wg.Wait()
	
	// 关闭数据库连接
	if cpm.dbPool != nil && cpm.dbPool.db != nil {
		cpm.dbPool.db.Close()
	}
	
	// 关闭Redis连接
	if cpm.redisPool != nil && cpm.redisPool.client != nil {
		cpm.redisPool.client.Close()
	}
	
	log.Println("Connection pool manager stopped")
}

// GetDatabaseConnection 获取数据库连接
func (cpm *ConnectionPoolManager) GetDatabaseConnection() (*sql.DB, error) {
	return cpm.dbPool.db, nil
}

// GetRedisClient 获取Redis客户端
func (cpm *ConnectionPoolManager) GetRedisClient() *redis.Client {
	return cpm.redisPool.client
}

// metricsCollector 指标收集器
func (cpm *ConnectionPoolManager) metricsCollector() {
	defer cpm.wg.Done()
	
	ticker := time.NewTicker(cpm.config.Monitoring.MetricsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-cpm.ctx.Done():
			return
		case <-ticker.C:
			cpm.collectMetrics()
		}
	}
}

// collectMetrics 收集指标
func (cpm *ConnectionPoolManager) collectMetrics() {
	// 收集数据库指标
	if cpm.dbPool != nil && cpm.dbPool.db != nil {
		stats := cpm.dbPool.db.Stats()
		cpm.metrics.Database.OpenConnections = stats.OpenConnections
		cpm.metrics.Database.InUseConnections = stats.InUse
		cpm.metrics.Database.IdleConnections = stats.Idle
	}
	
	// 收集Redis指标
	if cpm.redisPool != nil && cpm.redisPool.client != nil {
		poolStats := cpm.redisPool.client.PoolStats()
		cpm.metrics.Redis.ActiveConnections = int(poolStats.TotalConns - poolStats.IdleConns)
		cpm.metrics.Redis.IdleConnections = int(poolStats.IdleConns)
	}
	
	// 收集系统指标
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	cpm.metrics.System.MemoryUsage = m.Alloc
	cpm.metrics.System.GoroutineCount = runtime.NumGoroutine()
	
	// 记录指标
	log.Printf("Pool Metrics - DB: %d open, %d in use, %d idle | Redis: %d active, %d idle | System: %d goroutines, %d KB memory",
		cpm.metrics.Database.OpenConnections,
		cpm.metrics.Database.InUseConnections,
		cpm.metrics.Database.IdleConnections,
		cpm.metrics.Redis.ActiveConnections,
		cpm.metrics.Redis.IdleConnections,
		cpm.metrics.System.GoroutineCount,
		cpm.metrics.System.MemoryUsage/1024,
	)
}

// healthChecker 健康检查器
func (cpm *ConnectionPoolManager) healthChecker() {
	defer cpm.wg.Done()
	
	ticker := time.NewTicker(cpm.config.Monitoring.HealthCheck)
	defer ticker.Stop()
	
	for {
		select {
		case <-cpm.ctx.Done():
			return
		case <-ticker.C:
			cpm.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (cpm *ConnectionPoolManager) performHealthCheck() {
	healthy := 0
	
	// 检查数据库连接健康
	if cpm.dbPool != nil && cpm.dbPool.db != nil {
		if err := cpm.dbPool.db.Ping(); err == nil {
			healthy++
		} else {
			log.Printf("Database health check failed: %v", err)
			cpm.metrics.Database.ConnectionErrors++
		}
	}
	
	// 检查Redis连接健康
	if cpm.redisPool != nil && cpm.redisPool.client != nil {
		ctx := context.Background()
		if _, err := cpm.redisPool.client.Ping(ctx).Result(); err == nil {
			healthy++
		} else {
			log.Printf("Redis health check failed: %v", err)
			cpm.metrics.Redis.Errors++
		}
	}
	
	cpm.metrics.System.HealthyConnections = healthy
	cpm.metrics.System.LastHealthCheck = time.Now()
}

// connectionCleaner 连接清理器
func (cpm *ConnectionPoolManager) connectionCleaner() {
	defer cpm.wg.Done()
	
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-cpm.ctx.Done():
			return
		case <-ticker.C:
			cpm.cleanupConnections()
		}
	}
}

// cleanupConnections 清理连接
func (cpm *ConnectionPoolManager) cleanupConnections() {
	log.Println("Cleaning up idle connections...")
	
	// 清理数据库连接
	if cpm.dbPool != nil {
		cpm.dbPool.mu.Lock()
		for id, conn := range cpm.dbPool.connections {
			if !conn.InUse && time.Since(conn.LastUsed) > cpm.config.Database.ConnMaxIdleTime {
				delete(cpm.dbPool.connections, id)
				log.Printf("Cleaned up idle database connection: %s", id)
			}
		}
		cpm.dbPool.mu.Unlock()
	}
	
	// 强制垃圾回收
	runtime.GC()
}

// OptimizeQuery 查询优化
func (cpm *ConnectionPoolManager) OptimizeQuery(query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	
	db, err := cpm.GetDatabaseConnection()
	if err != nil {
		return nil, err
	}
	
	// 记录查询
	defer func() {
		duration := time.Since(start)
		cpm.metrics.Database.TotalQueries++
		
		// 更新平均查询时间
		if cpm.metrics.Database.AverageQueryTime == 0 {
			cpm.metrics.Database.AverageQueryTime = duration
		} else {
			cpm.metrics.Database.AverageQueryTime = (cpm.metrics.Database.AverageQueryTime + duration) / 2
		}
		
		// 记录慢查询
		if duration > 1*time.Second {
			log.Printf("Slow query detected: %s (took %v)", query, duration)
		}
	}()
	
	// 执行查询
	rows, err := db.Query(query, args...)
	if err != nil {
		cpm.metrics.Database.ConnectionErrors++
		return nil, err
	}
	
	return rows, nil
}

// ExecuteWithRetry 带重试的执行
func (cpm *ConnectionPoolManager) ExecuteWithRetry(query string, args ...interface{}) error {
	const maxRetries = 3
	
	for i := 0; i < maxRetries; i++ {
		db, err := cpm.GetDatabaseConnection()
		if err != nil {
			if i == maxRetries-1 {
				return err
			}
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
		
		_, err = db.Exec(query, args...)
		if err == nil {
			return nil
		}
		
		log.Printf("Query execution failed (attempt %d/%d): %v", i+1, maxRetries, err)
		
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}
	
	return fmt.Errorf("query failed after %d retries", maxRetries)
}

// GetPoolStatus 获取连接池状态
func (cpm *ConnectionPoolManager) GetPoolStatus() map[string]interface{} {
	return map[string]interface{}{
		"database": map[string]interface{}{
			"max_open_conns":    cpm.config.Database.MaxOpenConns,
			"max_idle_conns":    cpm.config.Database.MaxIdleConns,
			"open_connections":  cpm.metrics.Database.OpenConnections,
			"in_use_connections": cpm.metrics.Database.InUseConnections,
			"idle_connections":  cpm.metrics.Database.IdleConnections,
			"total_queries":     cpm.metrics.Database.TotalQueries,
			"average_query_time": cpm.metrics.Database.AverageQueryTime,
			"connection_errors": cpm.metrics.Database.ConnectionErrors,
		},
		"redis": map[string]interface{}{
			"max_conns":          cpm.config.Redis.MaxConns,
			"pool_size":          cpm.config.Redis.PoolSize,
			"active_connections": cpm.metrics.Redis.ActiveConnections,
			"idle_connections":   cpm.metrics.Redis.IdleConnections,
			"total_commands":     cpm.metrics.Redis.TotalCommands,
			"average_latency":    cpm.metrics.Redis.AverageLatency,
			"errors":             cpm.metrics.Redis.Errors,
		},
		"system": map[string]interface{}{
			"memory_usage":       cpm.metrics.System.MemoryUsage,
			"goroutine_count":    cpm.metrics.System.GoroutineCount,
			"healthy_connections": cpm.metrics.System.HealthyConnections,
			"last_health_check":  cpm.metrics.System.LastHealthCheck,
		},
	}
}

// GetMetrics 获取指标
func (cpm *ConnectionPoolManager) GetMetrics() *PoolMetrics {
	return cpm.metrics
}

// UpdateConfiguration 更新配置
func (cpm *ConnectionPoolManager) UpdateConfiguration(config *PoolConfig) error {
	cpm.config = config
	
	// 更新数据库连接池配置
	if cpm.dbPool != nil && cpm.dbPool.db != nil {
		cpm.dbPool.db.SetMaxOpenConns(config.Database.MaxOpenConns)
		cpm.dbPool.db.SetMaxIdleConns(config.Database.MaxIdleConns)
		cpm.dbPool.db.SetConnMaxLifetime(config.Database.ConnMaxLifetime)
		cpm.dbPool.db.SetConnMaxIdleTime(config.Database.ConnMaxIdleTime)
	}
	
	log.Println("Connection pool configuration updated")
	return nil
}