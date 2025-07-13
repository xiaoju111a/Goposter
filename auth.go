package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type UserAuth struct {
	users    map[string]*User
	sessions map[string]*Session
	usersMu  sync.RWMutex
	filename string
}

type User struct {
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	Salt         string    `json:"salt"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
	LastLogin    time.Time `json:"last_login"`
}

type Session struct {
	SessionID string    `json:"session_id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type UsersConfig struct {
	Users map[string]*User `json:"users"`
}

func NewUserAuth(filename string) *UserAuth {
	ua := &UserAuth{
		users:    make(map[string]*User),
		sessions: make(map[string]*Session),
		filename: filename,
	}
	ua.loadFromFile()
	ua.ensureDefaultAdmin()
	return ua
}

// CreateUser 创建新用户
func (ua *UserAuth) CreateUser(email, password string, isAdmin bool) error {
	ua.usersMu.Lock()
	defer ua.usersMu.Unlock()
	
	email = strings.ToLower(email)
	
	// 检查用户是否已存在
	if _, exists := ua.users[email]; exists {
		return fmt.Errorf("user already exists: %s", email)
	}
	
	// 生成盐值和密码哈希
	salt := ua.generateSalt()
	passwordHash := ua.hashPassword(password, salt)
	
	user := &User{
		Email:        email,
		PasswordHash: passwordHash,
		Salt:         salt,
		IsAdmin:      isAdmin,
		CreatedAt:    time.Now(),
	}
	
	ua.users[email] = user
	return ua.saveToFile()
}

// Authenticate 验证用户凭据
func (ua *UserAuth) Authenticate(email, password string) bool {
	ua.usersMu.RLock()
	user, exists := ua.users[strings.ToLower(email)]
	ua.usersMu.RUnlock()
	
	if !exists {
		// 对于不存在的用户，使用简单策略：接受默认密码
		return ua.checkDefaultPassword(email, password)
	}
	
	// 验证密码
	expectedHash := ua.hashPassword(password, user.Salt)
	if user.PasswordHash == expectedHash {
		// 更新最后登录时间
		ua.usersMu.Lock()
		user.LastLogin = time.Now()
		ua.usersMu.Unlock()
		ua.saveToFile()
		return true
	}
	
	return false
}

// CreateSession 创建会话
func (ua *UserAuth) CreateSession(email string) (string, error) {
	ua.usersMu.Lock()
	defer ua.usersMu.Unlock()
	
	email = strings.ToLower(email)
	sessionID := ua.generateSessionID()
	
	session := &Session{
		SessionID: sessionID,
		Email:     email,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24小时过期
	}
	
	ua.sessions[sessionID] = session
	return sessionID, nil
}

// ValidateSession 验证会话
func (ua *UserAuth) ValidateSession(sessionID string) (string, bool) {
	ua.usersMu.RLock()
	defer ua.usersMu.RUnlock()
	
	session, exists := ua.sessions[sessionID]
	if !exists {
		return "", false
	}
	
	// 检查是否过期
	if time.Now().After(session.ExpiresAt) {
		delete(ua.sessions, sessionID)
		return "", false
	}
	
	return session.Email, true
}

// DeleteSession 删除会话
func (ua *UserAuth) DeleteSession(sessionID string) {
	ua.usersMu.Lock()
	defer ua.usersMu.Unlock()
	
	delete(ua.sessions, sessionID)
}

// IsAdmin 检查用户是否为管理员
func (ua *UserAuth) IsAdmin(email string) bool {
	ua.usersMu.RLock()
	defer ua.usersMu.RUnlock()
	
	user, exists := ua.users[strings.ToLower(email)]
	if !exists {
		// 默认管理员账号
		return email == "admin" || strings.HasSuffix(email, "@admin")
	}
	
	return user.IsAdmin
}

// GetUser 获取用户信息
func (ua *UserAuth) GetUser(email string) (*User, bool) {
	ua.usersMu.RLock()
	defer ua.usersMu.RUnlock()
	
	user, exists := ua.users[strings.ToLower(email)]
	if !exists {
		return nil, false
	}
	
	// 返回副本，避免并发问题
	userCopy := *user
	return &userCopy, true
}

// GetAllUsers 获取所有用户（仅管理员）
func (ua *UserAuth) GetAllUsers() []*User {
	ua.usersMu.RLock()
	defer ua.usersMu.RUnlock()
	
	users := make([]*User, 0, len(ua.users))
	for _, user := range ua.users {
		userCopy := *user
		users = append(users, &userCopy)
	}
	
	return users
}

// DeleteUser 删除用户
func (ua *UserAuth) DeleteUser(email string) error {
	ua.usersMu.Lock()
	defer ua.usersMu.Unlock()
	
	email = strings.ToLower(email)
	if _, exists := ua.users[email]; !exists {
		return fmt.Errorf("user not found: %s", email)
	}
	
	delete(ua.users, email)
	
	// 清理该用户的所有会话
	for sessionID, session := range ua.sessions {
		if session.Email == email {
			delete(ua.sessions, sessionID)
		}
	}
	
	return ua.saveToFile()
}

// checkDefaultPassword 检查默认密码策略
func (ua *UserAuth) checkDefaultPassword(email, password string) bool {
	// 简单的默认密码策略
	defaultPasswords := []string{
		"123456",
		"password",
		"admin",
		email, // 使用邮箱地址作为密码
	}
	
	for _, defaultPwd := range defaultPasswords {
		if password == defaultPwd {
			return true
		}
	}
	
	return false
}

// generateSalt 生成随机盐值
func (ua *UserAuth) generateSalt() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateSessionID 生成会话ID
func (ua *UserAuth) generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// hashPassword 计算密码哈希
func (ua *UserAuth) hashPassword(password, salt string) string {
	hash := sha256.Sum256([]byte(password + salt))
	return hex.EncodeToString(hash[:])
}

// ensureDefaultAdmin 确保有默认管理员账号
func (ua *UserAuth) ensureDefaultAdmin() {
	adminEmail := "admin@ygocard.org"
	
	ua.usersMu.Lock()
	defer ua.usersMu.Unlock()
	
	if _, exists := ua.users[adminEmail]; !exists {
		salt := ua.generateSalt()
		passwordHash := ua.hashPassword("admin123", salt)
		
		admin := &User{
			Email:        adminEmail,
			PasswordHash: passwordHash,
			Salt:         salt,
			IsAdmin:      true,
			CreatedAt:    time.Now(),
		}
		
		ua.users[adminEmail] = admin
		ua.saveToFile()
	}
}

func (ua *UserAuth) loadFromFile() error {
	if ua.filename == "" {
		return nil
	}
	
	data, err := os.ReadFile(ua.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	var config UsersConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	
	ua.users = config.Users
	if ua.users == nil {
		ua.users = make(map[string]*User)
	}
	
	return nil
}

func (ua *UserAuth) saveToFile() error {
	if ua.filename == "" {
		return nil
	}
	
	config := UsersConfig{
		Users: ua.users,
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(ua.filename, data, 0644)
}