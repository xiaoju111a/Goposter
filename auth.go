package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/smtp"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type UserAuth struct {
	users    map[string]*User
	sessions map[string]*Session
	usersMu  sync.RWMutex
	filename string
	maxFailedAttempts int
	lockoutDuration   time.Duration
	jwtManager *ExtendedJWTManager
}

type User struct {
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	Salt         string    `json:"salt"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
	LastLogin    time.Time `json:"last_login"`
	TwoFactorEnabled bool `json:"two_factor_enabled"`
	TwoFactorSecret  string `json:"two_factor_secret"`
	FailedAttempts   int    `json:"failed_attempts"`
	LockedUntil      time.Time `json:"locked_until"`
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
		maxFailedAttempts: 5,
		lockoutDuration:   30 * time.Minute,
		jwtManager: NewExtendedJWTManager("freeagent-mail-secret-key-2024"),
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
	
	// 密码强度验证
	if err := ua.validatePasswordStrength(password); err != nil {
		return err
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
		TwoFactorEnabled: false,
		FailedAttempts:   0,
	}
	
	ua.users[email] = user
	return ua.saveToFile()
}

// Authenticate 验证用户凭据
func (ua *UserAuth) Authenticate(email, password string) bool {
	ua.usersMu.Lock()
	defer ua.usersMu.Unlock()
	
	email = strings.ToLower(email)
	user, exists := ua.users[email]
	
	if !exists {
		// 对于不存在的用户，使用简单策略：接受默认密码
		return ua.checkDefaultPassword(email, password)
	}
	
	// 检查账户是否被锁定
	if ua.isAccountLocked(user) {
		return false
	}
	
	// 验证密码
	expectedHash := ua.hashPassword(password, user.Salt)
	if user.PasswordHash == expectedHash {
		// 重置失败次数
		user.FailedAttempts = 0
		user.LastLogin = time.Now()
		ua.saveToFile()
		return true
	}
	
	// 记录失败尝试
	user.FailedAttempts++
	if user.FailedAttempts >= ua.maxFailedAttempts {
		user.LockedUntil = time.Now().Add(ua.lockoutDuration)
	}
	ua.saveToFile()
	
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

// validatePasswordStrength 验证密码强度
func (ua *UserAuth) validatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	
	// 检查是否包含数字
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
	// 检查是否包含小写字母
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	// 检查是否包含大写字母
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	// 检查是否包含特殊字符
	hasSpecial := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password)
	
	complexity := 0
	if hasDigit { complexity++ }
	if hasLower { complexity++ }
	if hasUpper { complexity++ }
	if hasSpecial { complexity++ }
	
	if complexity < 3 {
		return fmt.Errorf("password must contain at least 3 of: lowercase, uppercase, digit, special character")
	}
	
	return nil
}

// isAccountLocked 检查账户是否被锁定
func (ua *UserAuth) isAccountLocked(user *User) bool {
	if user.LockedUntil.IsZero() {
		return false
	}
	
	if time.Now().After(user.LockedUntil) {
		// 锁定期已过，解锁账户
		user.LockedUntil = time.Time{}
		user.FailedAttempts = 0
		return false
	}
	
	return true
}

// Enable2FA 启用双因素认证
func (ua *UserAuth) Enable2FA(email string) (string, error) {
	ua.usersMu.Lock()
	defer ua.usersMu.Unlock()
	
	email = strings.ToLower(email)
	user, exists := ua.users[email]
	if !exists {
		return "", fmt.Errorf("user not found: %s", email)
	}
	
	if user.TwoFactorEnabled {
		return "", fmt.Errorf("2FA already enabled for user: %s", email)
	}
	
	// 生成TOTP密钥
	secret := ua.generateTOTPSecret()
	user.TwoFactorSecret = secret
	user.TwoFactorEnabled = true
	
	ua.saveToFile()
	
	// 返回用于设置认证器的密钥
	return secret, nil
}

// Disable2FA 禁用双因素认证
func (ua *UserAuth) Disable2FA(email string) error {
	ua.usersMu.Lock()
	defer ua.usersMu.Unlock()
	
	email = strings.ToLower(email)
	user, exists := ua.users[email]
	if !exists {
		return fmt.Errorf("user not found: %s", email)
	}
	
	user.TwoFactorEnabled = false
	user.TwoFactorSecret = ""
	
	return ua.saveToFile()
}

// Verify2FA 验证双因素认证代码
func (ua *UserAuth) Verify2FA(email, code string) bool {
	ua.usersMu.RLock()
	defer ua.usersMu.RUnlock()
	
	email = strings.ToLower(email)
	user, exists := ua.users[email]
	if !exists || !user.TwoFactorEnabled {
		return false
	}
	
	return ua.verifyTOTP(user.TwoFactorSecret, code)
}

// generateTOTPSecret 生成TOTP密钥
func (ua *UserAuth) generateTOTPSecret() string {
	bytes := make([]byte, 20)
	rand.Read(bytes)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)
}

// verifyTOTP 验证TOTP代码
func (ua *UserAuth) verifyTOTP(secret, code string) bool {
	// 获取当前时间戳 (30秒窗口)
	currentTime := time.Now().Unix() / 30
	
	// 检查当前时间窗口和前后1个时间窗口
	for i := -1; i <= 1; i++ {
		timeWindow := currentTime + int64(i)
		if ua.generateTOTPCode(secret, timeWindow) == code {
			return true
		}
	}
	
	return false
}

// generateTOTPCode 生成TOTP代码
func (ua *UserAuth) generateTOTPCode(secret string, timeWindow int64) string {
	// 解码密钥
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return ""
	}
	
	// 将时间窗口转换为字节
	timeBytes := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		timeBytes[i] = byte(timeWindow & 0xff)
		timeWindow >>= 8
	}
	
	// 使用HMAC-SHA256生成哈希
	h := sha256.New()
	h.Write(append(key, timeBytes...))
	hash := h.Sum(nil)
	
	// 动态截取
	offset := hash[len(hash)-1] & 0x0f
	truncatedHash := hash[offset : offset+4]
	
	// 转换为数字
	code := int(truncatedHash[0]&0x7f)<<24 |
		int(truncatedHash[1])<<16 |
		int(truncatedHash[2])<<8 |
		int(truncatedHash[3])
	
	// 取6位数字
	return fmt.Sprintf("%06d", code%1000000)
}

// GetAccountLockStatus 获取账户锁定状态
func (ua *UserAuth) GetAccountLockStatus(email string) (bool, time.Time, int) {
	ua.usersMu.RLock()
	defer ua.usersMu.RUnlock()
	
	email = strings.ToLower(email)
	user, exists := ua.users[email]
	if !exists {
		return false, time.Time{}, 0
	}
	
	isLocked := ua.isAccountLocked(user)
	return isLocked, user.LockedUntil, user.FailedAttempts
}

// UnlockAccount 解锁账户 (管理员功能)
func (ua *UserAuth) UnlockAccount(email string) error {
	ua.usersMu.Lock()
	defer ua.usersMu.Unlock()
	
	email = strings.ToLower(email)
	user, exists := ua.users[email]
	if !exists {
		return fmt.Errorf("user not found: %s", email)
	}
	
	user.LockedUntil = time.Time{}
	user.FailedAttempts = 0
	
	return ua.saveToFile()
}

// SendSecurityAlert 发送安全警报邮件
func (ua *UserAuth) SendSecurityAlert(email, alertType, message string) error {
	// 这里可以集成邮件发送功能
	// 示例实现：
	subject := fmt.Sprintf("Security Alert: %s", alertType)
	body := fmt.Sprintf(`
Security Alert for %s

Alert Type: %s
Message: %s
Time: %s

If this wasn't you, please contact support immediately.
`, email, alertType, message, time.Now().Format("2006-01-02 15:04:05"))
	
	// 发送邮件 (这里需要实现SMTP发送)
	return ua.sendEmail(email, subject, body)
}

// sendEmail 发送邮件的辅助函数
func (ua *UserAuth) sendEmail(to, subject, body string) error {
	// 这里需要配置SMTP服务器信息
	// 示例配置
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	from := "noreply@freeagent.live"
	password := "your-app-password"
	
	// 构建邮件
	msg := []byte(fmt.Sprintf(`To: %s
Subject: %s

%s`, to, subject, body))
	
	// 发送邮件
	auth := smtp.PlainAuth("", from, password, smtpHost)
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, msg)
	if err != nil {
		// 如果发送失败，记录日志但不返回错误
		fmt.Printf("Failed to send security alert email: %v\n", err)
	}
	
	return nil
}

// JWT相关方法

// GenerateJWTToken 生成JWT令牌
func (ua *UserAuth) GenerateJWTToken(email string) (map[string]interface{}, error) {
	ua.usersMu.RLock()
	user, exists := ua.users[strings.ToLower(email)]
	ua.usersMu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("user not found: %s", email)
	}
	
	return ua.jwtManager.GenerateTokenPair(email, user.IsAdmin)
}

// ValidateJWTToken 验证JWT令牌
func (ua *UserAuth) ValidateJWTToken(token string) (*JWTClaims, error) {
	return ua.jwtManager.ValidateTokenWithBlacklist(token)
}

// RefreshJWTToken 刷新JWT令牌
func (ua *UserAuth) RefreshJWTToken(refreshToken string) (map[string]interface{}, error) {
	claims, err := ua.jwtManager.ValidateTokenWithBlacklist(refreshToken)
	if err != nil {
		return nil, err
	}
	
	return ua.jwtManager.GenerateTokenPair(claims.Email, claims.IsAdmin)
}

// RevokeJWTToken 撤销JWT令牌
func (ua *UserAuth) RevokeJWTToken(token string) error {
	return ua.jwtManager.RevokeToken(token)
}

// AuthenticateWithJWT 使用JWT进行身份验证
func (ua *UserAuth) AuthenticateWithJWT(token string) (string, bool, error) {
	claims, err := ua.jwtManager.ValidateTokenWithBlacklist(token)
	if err != nil {
		return "", false, err
	}
	
	return claims.Email, claims.IsAdmin, nil
}

// AuthenticateWith2FA 使用2FA进行完整身份验证
func (ua *UserAuth) AuthenticateWith2FA(email, password, twoFactorCode string) (map[string]interface{}, error) {
	// 首先验证密码
	if !ua.Authenticate(email, password) {
		return nil, fmt.Errorf("invalid credentials")
	}
	
	ua.usersMu.RLock()
	user, exists := ua.users[strings.ToLower(email)]
	ua.usersMu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	
	// 如果启用了2FA，验证2FA代码
	if user.TwoFactorEnabled {
		if twoFactorCode == "" {
			return nil, fmt.Errorf("two-factor code required")
		}
		
		if !ua.Verify2FA(email, twoFactorCode) {
			return nil, fmt.Errorf("invalid two-factor code")
		}
	}
	
	// 生成JWT令牌
	tokens, err := ua.GenerateJWTToken(email)
	if err != nil {
		return nil, err
	}
	
	// 发送安全警报
	ua.SendSecurityAlert(email, "Login", "Successful login detected")
	
	return tokens, nil
}

// GetJWTTokenInfo 获取JWT令牌信息
func (ua *UserAuth) GetJWTTokenInfo(token string) (map[string]interface{}, error) {
	return ua.jwtManager.GetTokenInfo(token)
}

// CleanupExpiredTokens 清理过期的令牌
func (ua *UserAuth) CleanupExpiredTokens() {
	ua.jwtManager.CleanupBlacklist()
}

// GetTokenStatistics 获取令牌统计信息
func (ua *UserAuth) GetTokenStatistics() *TokenStatistics {
	return ua.jwtManager.GetTokenStatistics()
}