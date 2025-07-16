package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db               *sql.DB
	secureDB         *SecureDatabase
	encryptionManager *EncryptionManager
}

type UserDB struct {
	ID                int       `json:"id"`
	Email             string    `json:"email"`
	PasswordHash      string    `json:"password_hash"`
	Salt              string    `json:"salt"`
	IsAdmin           bool      `json:"is_admin"`
	TwoFactorEnabled  bool      `json:"two_factor_enabled"`
	TwoFactorSecret   string    `json:"two_factor_secret"`
	CreatedAt         time.Time `json:"created_at"`
	LastLogin         time.Time `json:"last_login"`
}

type SessionDB struct {
	ID        int       `json:"id"`
	SessionID string    `json:"session_id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type EmailDB struct {
	ID        int       `json:"id"`
	Mailbox   string    `json:"mailbox"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Headers   string    `json:"headers"`
	Received  time.Time `json:"received"`
}

type MailboxDB struct {
	ID             int       `json:"id"`
	Email          string    `json:"email"`
	PasswordHash   string    `json:"password_hash"`
	Salt           string    `json:"salt"`
	Description    string    `json:"description"`
	IsActive       bool      `json:"is_active"`
	Owner          string    `json:"owner"`
	ForwardTo      string    `json:"forward_to"`
	ForwardEnabled bool      `json:"forward_enabled"`
	KeepOriginal   bool      `json:"keep_original"`
	CreatedAt      time.Time `json:"created_at"`
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// 创建加密管理器
	encryptionManager := NewEncryptionManager("ygocard-mail-encryption-key-2024")
	
	// 创建安全数据库
	secureDB, err := NewSecureDatabase(dbPath, "ygocard-secure-key-2024", "localhost:6379")
	if err != nil {
		log.Printf("Failed to create secure database: %v", err)
		// 继续使用基础数据库
	}

	database := &Database{
		db:               db,
		secureDB:         secureDB,
		encryptionManager: encryptionManager,
	}
	
	if err := database.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	if err := database.ensureDefaultAdmin(); err != nil {
		log.Printf("Warning: failed to create default admin: %v", err)
	}

	return database, nil
}

func (d *Database) createTables() error {
	// 用户表
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		salt TEXT NOT NULL,
		is_admin BOOLEAN DEFAULT FALSE,
		two_factor_enabled BOOLEAN DEFAULT FALSE,
		two_factor_secret TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_login DATETIME
	);`

	// 会话表
	sessionTable := `
	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT UNIQUE NOT NULL,
		email TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		FOREIGN KEY (email) REFERENCES users(email)
	);`

	// 邮件表
	emailTable := `
	CREATE TABLE IF NOT EXISTS emails (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		mailbox TEXT NOT NULL,
		from_addr TEXT NOT NULL,
		to_addr TEXT NOT NULL,
		subject TEXT,
		body TEXT,
		headers TEXT,
		received DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// 邮箱表
	mailboxTable := `
	CREATE TABLE IF NOT EXISTS mailboxes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		salt TEXT NOT NULL,
		description TEXT,
		is_active BOOLEAN DEFAULT TRUE,
		owner TEXT,
		forward_to TEXT,
		forward_enabled BOOLEAN DEFAULT FALSE,
		keep_original BOOLEAN DEFAULT TRUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_emails_mailbox ON emails(mailbox);",
		"CREATE INDEX IF NOT EXISTS idx_emails_received ON emails(received);",
		"CREATE INDEX IF NOT EXISTS idx_sessions_session_id ON sessions(session_id);",
		"CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);",
		"CREATE INDEX IF NOT EXISTS idx_mailboxes_email ON mailboxes(email);",
		"CREATE INDEX IF NOT EXISTS idx_mailboxes_owner ON mailboxes(owner);",
	}

	tables := []string{userTable, sessionTable, emailTable, mailboxTable}
	for _, table := range tables {
		if _, err := d.db.Exec(table); err != nil {
			return err
		}
	}

	for _, index := range indexes {
		if _, err := d.db.Exec(index); err != nil {
			return err
		}
	}

	return nil
}

// 用户认证方法
func (d *Database) CreateUser(email, password string, isAdmin bool) error {
	email = strings.ToLower(email)

	// 检查用户是否已存在
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("user already exists: %s", email)
	}

	// 生成盐值和密码哈希
	salt := d.generateSalt()
	passwordHash := d.hashPassword(password, salt)

	query := `
		INSERT INTO users (email, password_hash, salt, is_admin, two_factor_enabled, two_factor_secret)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err = d.db.Exec(query, email, passwordHash, salt, isAdmin, false, "")
	return err
}

func (d *Database) Authenticate(email, password string) bool {
	email = strings.ToLower(email)

	var user UserDB
	query := `
		SELECT email, password_hash, salt
		FROM users 
		WHERE email = ?
	`
	err := d.db.QueryRow(query, email).Scan(&user.Email, &user.PasswordHash, &user.Salt)
	if err != nil {
		if err == sql.ErrNoRows {
			// 对于不存在的用户，使用默认密码策略
			return d.checkDefaultPassword(email, password)
		}
		return false
	}

	// 验证密码
	expectedHash := d.hashPassword(password, user.Salt)
	if user.PasswordHash == expectedHash {
		// 更新最后登录时间
		d.db.Exec("UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE email = ?", email)
		return true
	}

	return false
}

func (d *Database) CreateSession(session *SessionDB) error {
	query := `
		INSERT INTO sessions (session_id, email, expires_at)
		VALUES (?, ?, ?)
	`
	_, err := d.db.Exec(query, session.SessionID, session.Email, session.ExpiresAt)
	return err
}

func (d *Database) CreateSessionOld(email string) (string, error) {
	email = strings.ToLower(email)
	sessionID := d.generateSessionID()

	query := `
		INSERT INTO sessions (session_id, email, expires_at)
		VALUES (?, ?, ?)
	`
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err := d.db.Exec(query, sessionID, email, expiresAt)
	if err != nil {
		return "", err
	}

	return sessionID, nil
}

func (d *Database) ValidateSession(sessionID string) (string, bool) {
	var email string
	var expiresAt time.Time

	query := `
		SELECT email, expires_at 
		FROM sessions 
		WHERE session_id = ?
	`
	err := d.db.QueryRow(query, sessionID).Scan(&email, &expiresAt)
	if err != nil {
		return "", false
	}

	// 检查是否过期
	if time.Now().After(expiresAt) {
		d.DeleteSession(sessionID)
		return "", false
	}

	return email, true
}

func (d *Database) DeleteSession(sessionID string) {
	d.db.Exec("DELETE FROM sessions WHERE session_id = ?", sessionID)
}

func (d *Database) IsAdmin(email string) bool {
	email = strings.ToLower(email)

	var isAdmin bool
	err := d.db.QueryRow("SELECT is_admin FROM users WHERE email = ?", email).Scan(&isAdmin)
	if err != nil {
		// 默认管理员账号
		return email == "admin@ygocard.live" || email == "xiaoju@ygocard.live"
	}

	return isAdmin
}

// 邮件存储方法
func (d *Database) SaveEmail(mailbox, from, to, subject, body, headers string) error {
	query := `
		INSERT INTO emails (mailbox, from_addr, to_addr, subject, body, headers)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := d.db.Exec(query, mailbox, from, to, subject, body, headers)
	return err
}

func (d *Database) GetEmails(mailbox string) ([]EmailDB, error) {
	query := `
		SELECT id, mailbox, from_addr, to_addr, subject, body, headers, received
		FROM emails 
		WHERE mailbox = ?
		ORDER BY received DESC
	`
	rows, err := d.db.Query(query, mailbox)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []EmailDB
	for rows.Next() {
		var email EmailDB
		err := rows.Scan(
			&email.ID, &email.Mailbox, &email.From, &email.To,
			&email.Subject, &email.Body, &email.Headers, &email.Received,
		)
		if err != nil {
			continue
		}
		emails = append(emails, email)
	}

	return emails, nil
}

func (d *Database) GetAllMailboxNames() ([]string, error) {
	query := `
		SELECT DISTINCT mailbox 
		FROM emails 
		ORDER BY mailbox
	`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mailboxes []string
	for rows.Next() {
		var mailbox string
		if err := rows.Scan(&mailbox); err != nil {
			continue
		}
		mailboxes = append(mailboxes, mailbox)
	}

	return mailboxes, nil
}

func (d *Database) DeleteEmail(mailbox string, emailID int) error {
	query := "DELETE FROM emails WHERE mailbox = ? AND id = ?"
	_, err := d.db.Exec(query, mailbox, emailID)
	return err
}

// 工具方法
func (d *Database) checkDefaultPassword(email, password string) bool {
	defaultPasswords := []string{
		"123456",
		"password",
		"admin",
		"xiaoju123", // 为xiaoju用户添加默认密码
		email,
	}

	for _, defaultPwd := range defaultPasswords {
		if password == defaultPwd {
			return true
		}
	}

	return false
}

func (d *Database) generateSalt() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (d *Database) generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GetAllUsers 获取所有用户
func (d *Database) GetAllUsers() ([]*UserDB, error) {
	query := `
		SELECT id, email, password_hash, salt, is_admin, two_factor_enabled, two_factor_secret, created_at, last_login
		FROM users
		ORDER BY created_at DESC
	`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []*UserDB
	for rows.Next() {
		user := &UserDB{}
		err := rows.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Salt, &user.IsAdmin, 
			&user.TwoFactorEnabled, &user.TwoFactorSecret, &user.CreatedAt, &user.LastLogin)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	
	return users, nil
}

// DeleteUser 删除用户
func (d *Database) DeleteUser(email string) error {
	query := `DELETE FROM users WHERE email = ?`
	_, err := d.db.Exec(query, email)
	return err
}

// DeleteUserSessions 删除用户的所有会话
func (d *Database) DeleteUserSessions(email string) error {
	query := `DELETE FROM sessions WHERE email = ?`
	_, err := d.db.Exec(query, email)
	return err
}

func (d *Database) hashPassword(password, salt string) string {
	hash := sha256.Sum256([]byte(password + salt))
	return hex.EncodeToString(hash[:])
}

func (d *Database) ensureDefaultAdmin() error {
	adminEmail := "admin@ygocard.live"

	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", adminEmail).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		if err := d.CreateUser(adminEmail, "admin123", true); err != nil {
			return err
		}
		log.Printf("Created default admin user: %s (password: admin123)", adminEmail)
	}

	// 也为xiaoju用户创建账号
	xiaojuEmail := "xiaoju@ygocard.live"
	err = d.db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", xiaojuEmail).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		if err := d.CreateUser(xiaojuEmail, "xiaoju123", true); err != nil {
			return err
		}
		log.Printf("Created xiaoju user: %s (password: xiaoju123)", xiaojuEmail)
	}

	return nil
}

func (d *Database) Close() error {
	if d.secureDB != nil {
		d.secureDB.Close()
	}
	return d.db.Close()
}

// StoreEncryptedEmail 存储加密邮件
func (d *Database) StoreEncryptedEmail(mailbox string, emailData map[string]interface{}) error {
	if d.secureDB == nil || d.encryptionManager == nil {
		return fmt.Errorf("secure database not initialized")
	}
	
	// 提取邮件内容
	subject := getStringFromMap(emailData, "subject")
	body := getStringFromMap(emailData, "body")
	headers := getStringFromMap(emailData, "headers")
	
	// 加密邮件内容
	encryptedEmail, err := d.encryptionManager.EncryptEmail(subject, body, headers)
	if err != nil {
		return fmt.Errorf("failed to encrypt email: %v", err)
	}
	
	// 创建完整的加密邮件数据
	encryptedData := make(map[string]interface{})
	for k, v := range emailData {
		encryptedData[k] = v
	}
	
	// 替换敏感字段为加密版本
	encryptedData["encrypted_subject"] = encryptedEmail.EncryptedSubject
	encryptedData["encrypted_body"] = encryptedEmail.EncryptedBody
	encryptedData["encrypted_headers"] = encryptedEmail.EncryptedHeaders
	encryptedData["subject_salt"] = encryptedEmail.SubjectSalt
	encryptedData["body_salt"] = encryptedEmail.BodySalt
	encryptedData["headers_salt"] = encryptedEmail.HeadersSalt
	encryptedData["subject_nonce"] = encryptedEmail.SubjectNonce
	encryptedData["body_nonce"] = encryptedEmail.BodyNonce
	encryptedData["headers_nonce"] = encryptedEmail.HeadersNonce
	encryptedData["algorithm"] = encryptedEmail.Algorithm
	encryptedData["key_version"] = encryptedEmail.KeyVersion
	encryptedData["encrypted_at"] = encryptedEmail.CreatedAt
	
	// 创建搜索索引
	searchIndex, err := d.encryptionManager.EncryptSearchIndex(subject + " " + body)
	if err != nil {
		log.Printf("Failed to create search index: %v", err)
	} else {
		encryptedData["search_index"] = searchIndex
	}
	
	// 删除明文数据
	delete(encryptedData, "subject")
	delete(encryptedData, "body")
	delete(encryptedData, "headers")
	
	// 存储到安全数据库
	return d.secureDB.StoreEncryptedEmail(mailbox, encryptedData)
}

// GetDecryptedEmails 获取解密邮件
func (d *Database) GetDecryptedEmails(mailbox string) ([]map[string]interface{}, error) {
	if d.secureDB == nil || d.encryptionManager == nil {
		return nil, fmt.Errorf("secure database not initialized")
	}
	
	// 获取加密邮件
	encryptedEmails, err := d.secureDB.GetEncryptedEmails(mailbox)
	if err != nil {
		return nil, err
	}
	
	var decryptedEmails []map[string]interface{}
	
	for _, encEmail := range encryptedEmails {
		// 构造加密邮件结构
		emailEnc := &EmailEncryption{
			EncryptedSubject: getStringFromMap(encEmail, "encrypted_subject"),
			EncryptedBody:    getStringFromMap(encEmail, "encrypted_body"),
			EncryptedHeaders: getStringFromMap(encEmail, "encrypted_headers"),
			SubjectSalt:      getStringFromMap(encEmail, "subject_salt"),
			BodySalt:         getStringFromMap(encEmail, "body_salt"),
			HeadersSalt:      getStringFromMap(encEmail, "headers_salt"),
			SubjectNonce:     getStringFromMap(encEmail, "subject_nonce"),
			BodyNonce:        getStringFromMap(encEmail, "body_nonce"),
			HeadersNonce:     getStringFromMap(encEmail, "headers_nonce"),
			Algorithm:        getStringFromMap(encEmail, "algorithm"),
			KeyVersion:       getIntFromMap(encEmail, "key_version"),
			CreatedAt:        getInt64FromMap(encEmail, "encrypted_at"),
		}
		
		// 验证加密数据完整性
		if err := d.encryptionManager.ValidateEncryptedData(emailEnc); err != nil {
			log.Printf("Invalid encrypted data: %v", err)
			continue
		}
		
		// 解密邮件
		subject, body, headers, err := d.encryptionManager.DecryptEmail(emailEnc)
		if err != nil {
			log.Printf("Failed to decrypt email: %v", err)
			continue
		}
		
		// 创建解密后的邮件数据
		decryptedEmail := make(map[string]interface{})
		for k, v := range encEmail {
			// 跳过加密相关字段
			if strings.HasPrefix(k, "encrypted_") || strings.HasSuffix(k, "_salt") || 
			   strings.HasSuffix(k, "_nonce") || k == "algorithm" || k == "key_version" {
				continue
			}
			decryptedEmail[k] = v
		}
		
		// 添加解密后的内容
		decryptedEmail["subject"] = subject
		decryptedEmail["body"] = body
		decryptedEmail["headers"] = headers
		
		decryptedEmails = append(decryptedEmails, decryptedEmail)
	}
	
	return decryptedEmails, nil
}

// SearchEncryptedEmails 搜索加密邮件
func (d *Database) SearchEncryptedEmails(mailbox, searchTerm string) ([]map[string]interface{}, error) {
	if d.secureDB == nil || d.encryptionManager == nil {
		return nil, fmt.Errorf("secure database not initialized")
	}
	
	// 获取所有加密邮件
	encryptedEmails, err := d.secureDB.GetEncryptedEmails(mailbox)
	if err != nil {
		return nil, err
	}
	
	var matchedEmails []map[string]interface{}
	
	for _, encEmail := range encryptedEmails {
		// 检查搜索索引
		searchIndex := getStringFromMap(encEmail, "search_index")
		if searchIndex != "" {
			if d.encryptionManager.SearchEncryptedContent(searchTerm, searchIndex) {
				matchedEmails = append(matchedEmails, encEmail)
			}
		}
	}
	
	// 解密匹配的邮件
	return d.decryptEmailList(matchedEmails)
}

// decryptEmailList 解密邮件列表
func (d *Database) decryptEmailList(encryptedEmails []map[string]interface{}) ([]map[string]interface{}, error) {
	var decryptedEmails []map[string]interface{}
	
	for _, encEmail := range encryptedEmails {
		emailEnc := &EmailEncryption{
			EncryptedSubject: getStringFromMap(encEmail, "encrypted_subject"),
			EncryptedBody:    getStringFromMap(encEmail, "encrypted_body"),
			EncryptedHeaders: getStringFromMap(encEmail, "encrypted_headers"),
			SubjectSalt:      getStringFromMap(encEmail, "subject_salt"),
			BodySalt:         getStringFromMap(encEmail, "body_salt"),
			HeadersSalt:      getStringFromMap(encEmail, "headers_salt"),
			SubjectNonce:     getStringFromMap(encEmail, "subject_nonce"),
			BodyNonce:        getStringFromMap(encEmail, "body_nonce"),
			HeadersNonce:     getStringFromMap(encEmail, "headers_nonce"),
			Algorithm:        getStringFromMap(encEmail, "algorithm"),
			KeyVersion:       getIntFromMap(encEmail, "key_version"),
			CreatedAt:        getInt64FromMap(encEmail, "encrypted_at"),
		}
		
		subject, body, headers, err := d.encryptionManager.DecryptEmail(emailEnc)
		if err != nil {
			continue
		}
		
		decryptedEmail := make(map[string]interface{})
		for k, v := range encEmail {
			if !strings.HasPrefix(k, "encrypted_") && !strings.HasSuffix(k, "_salt") && 
			   !strings.HasSuffix(k, "_nonce") && k != "algorithm" && k != "key_version" {
				decryptedEmail[k] = v
			}
		}
		
		decryptedEmail["subject"] = subject
		decryptedEmail["body"] = body
		decryptedEmail["headers"] = headers
		
		decryptedEmails = append(decryptedEmails, decryptedEmail)
	}
	
	return decryptedEmails, nil
}

// GetSecurityStats 获取安全统计信息
func (d *Database) GetSecurityStats() map[string]interface{} {
	if d.secureDB == nil {
		return map[string]interface{}{
			"encryption_enabled": false,
			"secure_database":    false,
		}
	}
	
	stats := d.secureDB.GetSecurityStats()
	if d.encryptionManager != nil {
		encInfo := d.encryptionManager.GetEncryptionInfo()
		stats["encryption_info"] = encInfo
	}
	
	return stats
}

// 辅助函数
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getIntFromMap(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		if i, ok := val.(int); ok {
			return i
		}
		if f, ok := val.(float64); ok {
			return int(f)
		}
	}
	return 0
}

func getInt64FromMap(m map[string]interface{}, key string) int64 {
	if val, ok := m[key]; ok {
		if i, ok := val.(int64); ok {
			return i
		}
		if f, ok := val.(float64); ok {
			return int64(f)
		}
	}
	return 0
}

// 邮箱管理方法
func (d *Database) CreateMailbox(email, password, description, owner string) error {
	email = strings.ToLower(email)
	
	// 检查邮箱是否已存在
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM mailboxes WHERE email = ?", email).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("mailbox already exists: %s", email)
	}

	// 生成盐值和密码哈希
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	saltString := hex.EncodeToString(salt)
	
	passwordHash := sha256.Sum256([]byte(password + saltString))
	passwordHashString := hex.EncodeToString(passwordHash[:])

	// 插入邮箱
	_, err = d.db.Exec(`
		INSERT INTO mailboxes (email, password_hash, salt, description, is_active, owner, forward_to, forward_enabled, keep_original)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, email, passwordHashString, saltString, description, true, owner, "", false, true)
	
	return err
}

func (d *Database) GetMailbox(email string) (*MailboxDB, error) {
	email = strings.ToLower(email)
	
	var mailbox MailboxDB
	err := d.db.QueryRow(`
		SELECT id, email, password_hash, salt, description, is_active, owner, forward_to, forward_enabled, keep_original, created_at
		FROM mailboxes WHERE email = ?
	`, email).Scan(&mailbox.ID, &mailbox.Email, &mailbox.PasswordHash, &mailbox.Salt, &mailbox.Description, 
		&mailbox.IsActive, &mailbox.Owner, &mailbox.ForwardTo, &mailbox.ForwardEnabled, &mailbox.KeepOriginal, &mailbox.CreatedAt)
	
	if err != nil {
		return nil, err
	}
	
	return &mailbox, nil
}

func (d *Database) GetMailboxesByOwner(owner string) ([]MailboxDB, error) {
	rows, err := d.db.Query(`
		SELECT id, email, password_hash, salt, description, is_active, owner, forward_to, forward_enabled, keep_original, created_at
		FROM mailboxes WHERE owner = ?
	`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mailboxes []MailboxDB
	for rows.Next() {
		var mailbox MailboxDB
		err := rows.Scan(&mailbox.ID, &mailbox.Email, &mailbox.PasswordHash, &mailbox.Salt, &mailbox.Description, 
			&mailbox.IsActive, &mailbox.Owner, &mailbox.ForwardTo, &mailbox.ForwardEnabled, &mailbox.KeepOriginal, &mailbox.CreatedAt)
		if err != nil {
			continue
		}
		mailboxes = append(mailboxes, mailbox)
	}
	
	return mailboxes, nil
}

func (d *Database) GetAllMailboxes() ([]MailboxDB, error) {
	rows, err := d.db.Query(`
		SELECT id, email, password_hash, salt, description, is_active, owner, forward_to, forward_enabled, keep_original, created_at
		FROM mailboxes
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mailboxes []MailboxDB
	for rows.Next() {
		var mailbox MailboxDB
		err := rows.Scan(&mailbox.ID, &mailbox.Email, &mailbox.PasswordHash, &mailbox.Salt, &mailbox.Description, 
			&mailbox.IsActive, &mailbox.Owner, &mailbox.ForwardTo, &mailbox.ForwardEnabled, &mailbox.KeepOriginal, &mailbox.CreatedAt)
		if err != nil {
			continue
		}
		mailboxes = append(mailboxes, mailbox)
	}
	
	return mailboxes, nil
}

func (d *Database) ValidateMailboxCredentials(email, password string) bool {
	email = strings.ToLower(email)
	
	var passwordHash, salt string
	err := d.db.QueryRow("SELECT password_hash, salt FROM mailboxes WHERE email = ? AND is_active = true", email).Scan(&passwordHash, &salt)
	if err != nil {
		return false
	}
	
	// 验证密码
	expectedHash := sha256.Sum256([]byte(password + salt))
	expectedHashString := hex.EncodeToString(expectedHash[:])
	
	return passwordHash == expectedHashString
}

func (d *Database) UpdateMailboxForwarding(email, forwardTo string, forwardEnabled, keepOriginal bool) error {
	email = strings.ToLower(email)
	
	_, err := d.db.Exec(`
		UPDATE mailboxes SET forward_to = ?, forward_enabled = ?, keep_original = ?
		WHERE email = ?
	`, forwardTo, forwardEnabled, keepOriginal, email)
	
	return err
}

func (d *Database) DeleteMailbox(email string) error {
	email = strings.ToLower(email)
	
	_, err := d.db.Exec("DELETE FROM mailboxes WHERE email = ?", email)
	return err
}

// GetUser 获取用户信息
func (d *Database) GetUser(email string) (*UserDB, error) {
	email = strings.ToLower(email)
	
	var user UserDB
	var lastLogin *time.Time
	err := d.db.QueryRow(`
		SELECT id, email, password_hash, salt, is_admin, two_factor_enabled, two_factor_secret, created_at, last_login
		FROM users WHERE email = ?
	`, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Salt, &user.IsAdmin, &user.TwoFactorEnabled, &user.TwoFactorSecret, &user.CreatedAt, &lastLogin)
	
	if err != nil {
		return nil, err
	}
	
	// 处理可能为null的last_login字段
	if lastLogin != nil {
		user.LastLogin = *lastLogin
	}
	
	return &user, nil
}

// Update2FA 更新用户2FA设置
func (d *Database) Update2FA(email string, enabled bool, secret string) error {
	email = strings.ToLower(email)
	
	query := `UPDATE users SET two_factor_enabled = ?, two_factor_secret = ? WHERE email = ?`
	_, err := d.db.Exec(query, enabled, secret, email)
	return err
}

// GetSession 获取会话信息
func (d *Database) GetSession(sessionID string) (*SessionDB, error) {
	var session SessionDB
	err := d.db.QueryRow(`
		SELECT id, session_id, email, created_at, expires_at
		FROM sessions WHERE session_id = ? AND expires_at > datetime('now')
	`, sessionID).Scan(&session.ID, &session.SessionID, &session.Email, &session.CreatedAt, &session.ExpiresAt)
	
	if err != nil {
		return nil, err
	}
	
	return &session, nil
}