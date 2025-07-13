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
	db *sql.DB
}

type UserDB struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	Salt         string    `json:"salt"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
	LastLogin    time.Time `json:"last_login"`
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

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	database := &Database{db: db}
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

	// 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_emails_mailbox ON emails(mailbox);",
		"CREATE INDEX IF NOT EXISTS idx_emails_received ON emails(received);",
		"CREATE INDEX IF NOT EXISTS idx_sessions_session_id ON sessions(session_id);",
		"CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);",
	}

	tables := []string{userTable, sessionTable, emailTable}
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
		INSERT INTO users (email, password_hash, salt, is_admin)
		VALUES (?, ?, ?, ?)
	`
	_, err = d.db.Exec(query, email, passwordHash, salt, isAdmin)
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

func (d *Database) CreateSession(email string) (string, error) {
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
		return email == "admin@freeagent.live" || email == "xiaoju@freeagent.live"
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

func (d *Database) GetAllMailboxes() ([]string, error) {
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

func (d *Database) hashPassword(password, salt string) string {
	hash := sha256.Sum256([]byte(password + salt))
	return hex.EncodeToString(hash[:])
}

func (d *Database) ensureDefaultAdmin() error {
	adminEmail := "admin@freeagent.live"

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
	xiaojuEmail := "xiaoju@freeagent.live"
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
	return d.db.Close()
}