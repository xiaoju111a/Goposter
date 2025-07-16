package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"log"
	"net/smtp"
	"regexp"
	"strings"
	"sync"
	"time"
)

type UserAuth struct {
	db       *Database
	usersMu  sync.RWMutex
	maxFailedAttempts int
	lockoutDuration   time.Duration
	jwtManager *ExtendedJWTManager
}

type User struct {
	Email           string    `json:"email"`
	PasswordHash    string    `json:"password_hash"`
	Salt            string    `json:"salt"`
	IsAdmin         bool      `json:"is_admin"`
	CreatedAt       time.Time `json:"created_at"`
	LastLogin       time.Time `json:"last_login"`
	TwoFactorEnabled bool     `json:"two_factor_enabled"`
	TwoFactorSecret  string    `json:"two_factor_secret"`
	FailedAttempts   int       `json:"failed_attempts"`
	LockedUntil      time.Time `json:"locked_until"`
	AssignedMailboxes []string `json:"assigned_mailboxes"` // 分配的邮箱列表
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

func NewUserAuth(db *Database) *UserAuth {
	ua := &UserAuth{
		db:                db,
		maxFailedAttempts: 5,
		lockoutDuration:   30 * time.Minute,
		jwtManager: NewExtendedJWTManager("freeagent-mail-secret-key-2024"),
	}
	ua.ensureDefaultAdmin()
	return ua
}

// CreateUser 创建新用户
func (ua *UserAuth) CreateUser(email, password string, isAdmin bool) error {
	ua.usersMu.Lock()
	defer ua.usersMu.Unlock()
	
	email = strings.ToLower(email)
	
	// 检查用户是否已存在
	if _, err := ua.db.GetUser(email); err == nil {
		return fmt.Errorf("user already exists: %s", email)
	}
	
	// 密码强度验证
	if err := ua.validatePasswordStrength(password); err != nil {
		return err
	}
	
	// 生成盐值和密码哈希
	salt := ua.generateSalt()
	passwordHash := ua.hashPassword(password, salt)
	
	user := &UserDB{
		Email:             email,
		PasswordHash:      passwordHash,
		Salt:              salt,
		IsAdmin:           isAdmin,
		CreatedAt:         time.Now(),
		TwoFactorEnabled:  false,
		TwoFactorSecret:   "",
		LastLogin:         time.Time{},
	}
	
	return ua.db.CreateUser(user.Email, password, user.IsAdmin)
}

// Authenticate 验证用户凭据
func (ua *UserAuth) Authenticate(email, password string) bool {
	ua.usersMu.Lock()
	defer ua.usersMu.Unlock()
	
	email = strings.ToLower(email)
	user, err := ua.db.GetUser(email)
	
	if err != nil {
		// 对于不存在的用户，使用简单策略：接受默认密码
		return ua.checkDefaultPassword(email, password)
	}
	
	// 验证密码
	expectedHash := ua.hashPassword(password, user.Salt)
	if user.PasswordHash == expectedHash {
		// 更新最后登录时间
		user.LastLogin = time.Now()
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
	
	session := &SessionDB{
		SessionID: sessionID,
		Email:     email,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24小时过期
	}
	
	return sessionID, ua.db.CreateSession(session)
}

// ValidateSession 验证会话
func (ua *UserAuth) ValidateSession(sessionID string) (string, bool) {
	ua.usersMu.RLock()
	defer ua.usersMu.RUnlock()
	
	session, err := ua.db.GetSession(sessionID)
	if err != nil {
		return "", false
	}
	
	// 检查是否过期
	if time.Now().After(session.ExpiresAt) {
		ua.db.DeleteSession(sessionID)
		return "", false
	}
	
	return session.Email, true
}

// DeleteSession 删除会话
func (ua *UserAuth) DeleteSession(sessionID string) {
	ua.usersMu.Lock()
	defer ua.usersMu.Unlock()
	
	ua.db.DeleteSession(sessionID)
}

// IsAdmin 检查用户是否为管理员
func (ua *UserAuth) IsAdmin(email string) bool {
	ua.usersMu.RLock()
	defer ua.usersMu.RUnlock()
	
	user, err := ua.db.GetUser(strings.ToLower(email))
	if err != nil {
		// 默认管理员账号
		return email == "admin" || strings.HasSuffix(email, "@admin")
	}
	
	return user.IsAdmin
}

// GetUser 获取用户信息
func (ua *UserAuth) GetUser(email string) (*UserDB, bool) {
	ua.usersMu.RLock()
	defer ua.usersMu.RUnlock()
	
	user, err := ua.db.GetUser(strings.ToLower(email))
	if err != nil {
		return nil, false
	}
	
	return user, true
}

// GetAllUsers 获取所有用户（仅管理员）
func (ua *UserAuth) GetAllUsers() []*UserDB {
	ua.usersMu.RLock()
	defer ua.usersMu.RUnlock()
	
	users, err := ua.db.GetAllUsers()
	if err != nil {
		log.Printf("Failed to get all users: %v", err)
		return []*UserDB{}
	}
	
	return users
}

// ValidateJWT 验证JWT token并返回用户邮箱
func (ua *UserAuth) ValidateJWT(token string) (string, error) {
	claims, err := ua.ValidateJWTToken(token)
	if err != nil {
		return "", err
	}
	return claims.Email, nil
}

// DeleteUser 删除用户
func (ua *UserAuth) DeleteUser(email string) error {
	ua.usersMu.Lock()
	defer ua.usersMu.Unlock()
	
	email = strings.ToLower(email)
	_, err := ua.db.GetUser(email)
	if err != nil {
		return fmt.Errorf("user not found: %s", email)
	}
	
	// 清理该用户的所有会话
	ua.db.DeleteUserSessions(email)
	
	return ua.db.DeleteUser(email)
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
	
	if _, err := ua.db.GetUser(adminEmail); err != nil {
		salt := ua.generateSalt()
		passwordHash := ua.hashPassword("admin123", salt)
		
		admin := &UserDB{
			Email:             adminEmail,
			PasswordHash:      passwordHash,
			Salt:              salt,
			IsAdmin:           true,
			CreatedAt:         time.Now(),
			TwoFactorEnabled:  false,
			TwoFactorSecret:   "",
			LastLogin:         time.Time{},
		}
		
		ua.db.CreateUser(admin.Email, "admin123", admin.IsAdmin)
	}
}

// 不再需要文件存储相关方法

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
	user, err := ua.db.GetUser(email)
	
	if err != nil {
		return "", fmt.Errorf("user not found: %s", email)
	}
	
	if user.TwoFactorEnabled {
		return "", fmt.Errorf("2FA already enabled for user: %s", email)
	}
	
	// 生成TOTP密钥
	secret := ua.generateTOTPSecret()
	user.TwoFactorSecret = secret
	user.TwoFactorEnabled = true
	
	ua.db.Update2FA(email, true, secret)
	
	// 返回用于设置认证器的密钥
	return secret, nil
}

// Disable2FA 禁用双因素认证
func (ua *UserAuth) Disable2FA(email string) error {
	ua.usersMu.Lock()
	defer ua.usersMu.Unlock()
	
	email = strings.ToLower(email)
	user, err := ua.db.GetUser(email)
	if err != nil {
		return fmt.Errorf("user not found: %s", email)
	}
	
	user.TwoFactorEnabled = false
	user.TwoFactorSecret = ""
	
	return ua.db.Update2FA(email, false, "")
}

// Verify2FA 验证双因素认证代码
func (ua *UserAuth) Verify2FA(email, code string) bool {
	ua.usersMu.RLock()
	defer ua.usersMu.RUnlock()
	
	email = strings.ToLower(email)
	user, err := ua.db.GetUser(email)
	if err != nil || !user.TwoFactorEnabled {
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
	
	// 使用HMAC-SHA1生成哈希 (TOTP标准)
	hash := hmac.New(sha1.New, key)
	hash.Write(timeBytes)
	hashBytes := hash.Sum(nil)
	
	// 提取4字节动态代码
	offset := hashBytes[len(hashBytes)-1] & 0x0f
	code := (int(hashBytes[offset]) & 0x7f) << 24 |
		(int(hashBytes[offset+1]) & 0xff) << 16 |
		(int(hashBytes[offset+2]) & 0xff) << 8 |
		(int(hashBytes[offset+3]) & 0xff)
	
	// 转换为6位数字
	return fmt.Sprintf("%06d", code%1000000)
}

// GetAccountLockStatus 获取账户锁定状态
func (ua *UserAuth) GetAccountLockStatus(email string) (bool, time.Time, int) {
	// TODO: 从数据库获取账户锁定状态
	return false, time.Time{}, 0
}

// UnlockAccount 解锁账户 (管理员功能)
func (ua *UserAuth) UnlockAccount(email string) error {
	// TODO: 在数据库中解锁账户
	return nil
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
	// 从数据库中查找用户
	ua.usersMu.RLock()
	user, err := ua.db.GetUser(strings.ToLower(email))
	ua.usersMu.RUnlock()
	
	var isAdmin bool
	if err == nil {
		isAdmin = user.IsAdmin
	} else {
		// 如果数据库中不存在，假设为普通用户
		isAdmin = false
	}
	
	return ua.jwtManager.GenerateTokenPair(email, isAdmin)
}

// GenerateJWTTokenWithAdmin 使用指定的admin状态生成JWT令牌
func (ua *UserAuth) GenerateJWTTokenWithAdmin(email string, isAdmin bool) (map[string]interface{}, error) {
	return ua.jwtManager.GenerateTokenPair(email, isAdmin)
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
	user, err := ua.db.GetUser(strings.ToLower(email))
	ua.usersMu.RUnlock()
	
	if err != nil {
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