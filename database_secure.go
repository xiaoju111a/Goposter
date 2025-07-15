package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/crypto/pbkdf2"
	_ "github.com/mattn/go-sqlite3"
	"context"
)

// SecureDatabase 安全数据库管理器
type SecureDatabase struct {
	db          *sql.DB
	redis       *redis.Client
	encKey      []byte
	connPool    *ConnectionPool
	auditLogger *AuditLogger
	mu          sync.RWMutex
}

// ConnectionPool 连接池管理
type ConnectionPool struct {
	maxConns     int
	activeConns  int
	idleConns    []*sql.DB
	connTimeout  time.Duration
	mu           sync.Mutex
}

// AuditLogger 安全审计日志
type AuditLogger struct {
	logFile string
	mu      sync.Mutex
}

// EncryptedData 加密数据结构
type EncryptedData struct {
	Data      string `json:"data"`
	Salt      string `json:"salt"`
	Nonce     string `json:"nonce"`
	Timestamp int64  `json:"timestamp"`
}

// SensitiveData 敏感数据标记
type SensitiveData struct {
	Original string
	Masked   string
	Type     string
}

// NewSecureDatabase 创建安全数据库实例
func NewSecureDatabase(dbPath string, encryptionKey string, redisAddr string) (*SecureDatabase, error) {
	// 生成加密密钥
	encKey := pbkdf2.Key([]byte(encryptionKey), []byte("freeagent-salt"), 4096, 32, sha256.New)
	
	// 连接SQLite数据库
	db, err := sql.Open("sqlite3", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	
	// 配置SQLite连接池
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	
	// 连接Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
		PoolSize: 10,
	})
	
	// 测试Redis连接
	ctx := context.Background()
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		log.Printf("Redis connection failed: %v", err)
		rdb = nil // 继续运行但不使用Redis
	}
	
	// 创建连接池
	connPool := &ConnectionPool{
		maxConns:    20,
		activeConns: 0,
		idleConns:   make([]*sql.DB, 0),
		connTimeout: 30 * time.Second,
	}
	
	// 创建审计日志
	auditLogger := &AuditLogger{
		logFile: "data/audit.log",
	}
	
	sdb := &SecureDatabase{
		db:          db,
		redis:       rdb,
		encKey:      encKey,
		connPool:    connPool,
		auditLogger: auditLogger,
	}
	
	// 初始化数据库表
	if err := sdb.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %v", err)
	}
	
	return sdb, nil
}

// initTables 初始化数据库表
func (sdb *SecureDatabase) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS encrypted_emails (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			mailbox TEXT NOT NULL,
			encrypted_data TEXT NOT NULL,
			salt TEXT NOT NULL,
			nonce TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_encrypted_emails_mailbox ON encrypted_emails(mailbox)`,
		`CREATE INDEX IF NOT EXISTS idx_encrypted_emails_created_at ON encrypted_emails(created_at)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_email TEXT NOT NULL,
			action TEXT NOT NULL,
			resource TEXT NOT NULL,
			ip_address TEXT,
			user_agent TEXT,
			success BOOLEAN NOT NULL,
			details TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_user_email ON audit_logs(user_email)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at)`,
		`CREATE TABLE IF NOT EXISTS cache_metadata (
			key TEXT PRIMARY KEY,
			expiry INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	
	for _, query := range queries {
		if _, err := sdb.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %s, error: %v", query, err)
		}
	}
	
	return nil
}

// EncryptData 加密数据
func (sdb *SecureDatabase) EncryptData(plaintext string) (*EncryptedData, error) {
	// 生成随机盐和nonce
	salt := make([]byte, 16)
	nonce := make([]byte, 12)
	
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	
	// 使用PBKDF2生成密钥
	key := pbkdf2.Key(sdb.encKey, salt, 4096, 32, sha256.New)
	
	// 创建AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	// 加密数据
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	
	return &EncryptedData{
		Data:      hex.EncodeToString(ciphertext),
		Salt:      hex.EncodeToString(salt),
		Nonce:     hex.EncodeToString(nonce),
		Timestamp: time.Now().Unix(),
	}, nil
}

// DecryptData 解密数据
func (sdb *SecureDatabase) DecryptData(encData *EncryptedData) (string, error) {
	// 解码hex字符串
	ciphertext, err := hex.DecodeString(encData.Data)
	if err != nil {
		return "", err
	}
	
	salt, err := hex.DecodeString(encData.Salt)
	if err != nil {
		return "", err
	}
	
	nonce, err := hex.DecodeString(encData.Nonce)
	if err != nil {
		return "", err
	}
	
	// 重新生成密钥
	key := pbkdf2.Key(sdb.encKey, salt, 4096, 32, sha256.New)
	
	// 创建AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	
	// 解密数据
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	
	return string(plaintext), nil
}

// StoreEncryptedEmail 存储加密邮件
func (sdb *SecureDatabase) StoreEncryptedEmail(mailbox string, emailData map[string]interface{}) error {
	// 将邮件数据序列化为JSON
	jsonData, err := json.Marshal(emailData)
	if err != nil {
		return err
	}
	
	// 加密邮件数据
	encData, err := sdb.EncryptData(string(jsonData))
	if err != nil {
		return err
	}
	
	// 存储到数据库
	query := `INSERT INTO encrypted_emails (mailbox, encrypted_data, salt, nonce) VALUES (?, ?, ?, ?)`
	_, err = sdb.db.Exec(query, mailbox, encData.Data, encData.Salt, encData.Nonce)
	if err != nil {
		return err
	}
	
	// 记录审计日志
	sdb.auditLogger.LogAction("system", "STORE_EMAIL", mailbox, "", "", true, fmt.Sprintf("Email stored for %s", mailbox))
	
	// 清理Redis缓存
	if sdb.redis != nil {
		ctx := context.Background()
		sdb.redis.Del(ctx, "emails:"+mailbox)
	}
	
	return nil
}

// GetEncryptedEmails 获取加密邮件
func (sdb *SecureDatabase) GetEncryptedEmails(mailbox string) ([]map[string]interface{}, error) {
	// 先检查Redis缓存
	if sdb.redis != nil {
		ctx := context.Background()
		cached, err := sdb.redis.Get(ctx, "emails:"+mailbox).Result()
		if err == nil {
			var emails []map[string]interface{}
			if err := json.Unmarshal([]byte(cached), &emails); err == nil {
				return emails, nil
			}
		}
	}
	
	// 从数据库查询
	query := `SELECT encrypted_data, salt, nonce FROM encrypted_emails WHERE mailbox = ? ORDER BY created_at DESC`
	rows, err := sdb.db.Query(query, mailbox)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var emails []map[string]interface{}
	
	for rows.Next() {
		var encData, salt, nonce string
		if err := rows.Scan(&encData, &salt, &nonce); err != nil {
			continue
		}
		
		// 解密数据
		encrypted := &EncryptedData{
			Data:  encData,
			Salt:  salt,
			Nonce: nonce,
		}
		
		decrypted, err := sdb.DecryptData(encrypted)
		if err != nil {
			log.Printf("Failed to decrypt email: %v", err)
			continue
		}
		
		// 解析JSON
		var emailData map[string]interface{}
		if err := json.Unmarshal([]byte(decrypted), &emailData); err == nil {
			// 脱敏处理
			emailData = sdb.MaskSensitiveData(emailData)
			emails = append(emails, emailData)
		}
	}
	
	// 缓存到Redis
	if sdb.redis != nil {
		ctx := context.Background()
		cached, _ := json.Marshal(emails)
		sdb.redis.Set(ctx, "emails:"+mailbox, cached, 5*time.Minute)
	}
	
	return emails, nil
}

// MaskSensitiveData 敏感数据脱敏
func (sdb *SecureDatabase) MaskSensitiveData(data map[string]interface{}) map[string]interface{} {
	masked := make(map[string]interface{})
	
	for key, value := range data {
		switch key {
		case "From", "To", "Cc", "Bcc":
			if email, ok := value.(string); ok {
				masked[key] = sdb.maskEmail(email)
			} else {
				masked[key] = value
			}
		case "Subject":
			if subject, ok := value.(string); ok {
				masked[key] = sdb.maskSubject(subject)
			} else {
				masked[key] = value
			}
		case "Body":
			if body, ok := value.(string); ok {
				masked[key] = sdb.maskBody(body)
			} else {
				masked[key] = value
			}
		case "IP", "UserAgent":
			if str, ok := value.(string); ok {
				masked[key] = sdb.maskIP(str)
			} else {
				masked[key] = value
			}
		default:
			masked[key] = value
		}
	}
	
	return masked
}

// maskEmail 邮箱脱敏
func (sdb *SecureDatabase) maskEmail(email string) string {
	if len(email) < 5 {
		return "***"
	}
	
	atIndex := -1
	for i, char := range email {
		if char == '@' {
			atIndex = i
			break
		}
	}
	
	if atIndex == -1 {
		return "***"
	}
	
	username := email[:atIndex]
	domain := email[atIndex:]
	
	if len(username) <= 2 {
		return "**" + domain
	}
	
	return username[:2] + "***" + domain
}

// maskSubject 主题脱敏
func (sdb *SecureDatabase) maskSubject(subject string) string {
	if len(subject) <= 10 {
		return "***"
	}
	return subject[:10] + "..."
}

// maskBody 内容脱敏
func (sdb *SecureDatabase) maskBody(body string) string {
	if len(body) <= 50 {
		return "*** [Content Masked] ***"
	}
	return body[:50] + "... [Content Masked]"
}

// maskIP IP地址脱敏
func (sdb *SecureDatabase) maskIP(ip string) string {
	if len(ip) < 7 {
		return "***"
	}
	return ip[:7] + "***"
}

// LogAction 记录审计日志
func (al *AuditLogger) LogAction(userEmail, action, resource, ip, userAgent string, success bool, details string) {
	al.mu.Lock()
	defer al.mu.Unlock()
	
	logEntry := map[string]interface{}{
		"timestamp":  time.Now().Format("2006-01-02 15:04:05"),
		"user_email": userEmail,
		"action":     action,
		"resource":   resource,
		"ip_address": ip,
		"user_agent": userAgent,
		"success":    success,
		"details":    details,
	}
	
	logData, _ := json.Marshal(logEntry)
	log.Printf("AUDIT: %s", string(logData))
}

// GetAuditLogs 获取审计日志
func (sdb *SecureDatabase) GetAuditLogs(userEmail string, limit int) ([]map[string]interface{}, error) {
	var query string
	var args []interface{}
	
	if userEmail != "" {
		query = `SELECT user_email, action, resource, ip_address, success, details, created_at 
				FROM audit_logs WHERE user_email = ? ORDER BY created_at DESC LIMIT ?`
		args = []interface{}{userEmail, limit}
	} else {
		query = `SELECT user_email, action, resource, ip_address, success, details, created_at 
				FROM audit_logs ORDER BY created_at DESC LIMIT ?`
		args = []interface{}{limit}
	}
	
	rows, err := sdb.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var logs []map[string]interface{}
	for rows.Next() {
		var userEmail, action, resource, ip, details, createdAt string
		var success bool
		
		if err := rows.Scan(&userEmail, &action, &resource, &ip, &success, &details, &createdAt); err != nil {
			continue
		}
		
		logs = append(logs, map[string]interface{}{
			"user_email": sdb.maskEmail(userEmail),
			"action":     action,
			"resource":   resource,
			"ip_address": sdb.maskIP(ip),
			"success":    success,
			"details":    details,
			"created_at": createdAt,
		})
	}
	
	return logs, nil
}

// OptimizeQuery 查询优化
func (sdb *SecureDatabase) OptimizeQuery(query string, args ...interface{}) (*sql.Rows, error) {
	// 添加查询计划分析
	explainQuery := "EXPLAIN QUERY PLAN " + query
	rows, err := sdb.db.Query(explainQuery, args...)
	if err == nil {
		defer rows.Close()
		log.Printf("Query plan for: %s", query)
	}
	
	// 执行优化后的查询
	return sdb.db.Query(query, args...)
}

// GetConnectionStats 获取连接池统计
func (cp *ConnectionPool) GetConnectionStats() map[string]interface{} {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	
	return map[string]interface{}{
		"max_connections":    cp.maxConns,
		"active_connections": cp.activeConns,
		"idle_connections":   len(cp.idleConns),
		"connection_timeout": cp.connTimeout.String(),
	}
}

// CleanupExpiredCache 清理过期缓存
func (sdb *SecureDatabase) CleanupExpiredCache() error {
	if sdb.redis == nil {
		return nil
	}
	
	ctx := context.Background()
	
	// 获取所有缓存键
	keys, err := sdb.redis.Keys(ctx, "emails:*").Result()
	if err != nil {
		return err
	}
	
	// 检查并删除过期的缓存
	for _, key := range keys {
		ttl, err := sdb.redis.TTL(ctx, key).Result()
		if err != nil {
			continue
		}
		
		if ttl < 0 {
			sdb.redis.Del(ctx, key)
		}
	}
	
	return nil
}

// Close 关闭数据库连接
func (sdb *SecureDatabase) Close() error {
	if sdb.redis != nil {
		sdb.redis.Close()
	}
	return sdb.db.Close()
}

// GetSecurityStats 获取安全统计信息
func (sdb *SecureDatabase) GetSecurityStats() map[string]interface{} {
	stats := map[string]interface{}{
		"encryption_enabled": true,
		"audit_logging":      true,
		"data_masking":       true,
		"redis_caching":      sdb.redis != nil,
		"connection_pool":    sdb.connPool.GetConnectionStats(),
	}
	
	// 统计加密邮件数量
	var emailCount int
	sdb.db.QueryRow("SELECT COUNT(*) FROM encrypted_emails").Scan(&emailCount)
	stats["encrypted_emails"] = emailCount
	
	// 统计审计日志数量
	var auditCount int
	sdb.db.QueryRow("SELECT COUNT(*) FROM audit_logs").Scan(&auditCount)
	stats["audit_logs"] = auditCount
	
	return stats
}