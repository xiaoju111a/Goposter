package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type JWTManager struct {
	secretKey []byte
}

type JWTClaims struct {
	Email     string `json:"email"`
	IsAdmin   bool   `json:"is_admin"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	NotBefore int64  `json:"nbf"`
	Subject   string `json:"sub"`
	Issuer    string `json:"iss"`
}

type JWTHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

func NewJWTManager(secretKey string) *JWTManager {
	if secretKey == "" {
		// 生成随机密钥
		key := make([]byte, 32)
		rand.Read(key)
		secretKey = hex.EncodeToString(key)
	}
	
	return &JWTManager{
		secretKey: []byte(secretKey),
	}
}

// GenerateToken 生成JWT令牌
func (jm *JWTManager) GenerateToken(email string, isAdmin bool, expiration time.Duration) (string, error) {
	now := time.Now()
	
	// 创建JWT头部
	header := JWTHeader{
		Algorithm: "HS256",
		Type:      "JWT",
	}
	
	// 创建JWT声明
	claims := JWTClaims{
		Email:     email,
		IsAdmin:   isAdmin,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(expiration).Unix(),
		NotBefore: now.Unix(),
		Subject:   email,
		Issuer:    "freeagent.live",
	}
	
	// 编码头部
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerBytes)
	
	// 编码声明
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	claimsEncoded := base64.RawURLEncoding.EncodeToString(claimsBytes)
	
	// 创建签名
	message := headerEncoded + "." + claimsEncoded
	signature := jm.signHMAC(message)
	
	// 组合JWT
	token := message + "." + signature
	return token, nil
}

// ValidateToken 验证JWT令牌
func (jm *JWTManager) ValidateToken(token string) (*JWTClaims, error) {
	// 分割JWT
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}
	
	headerEncoded := parts[0]
	claimsEncoded := parts[1]
	signature := parts[2]
	
	// 验证签名
	message := headerEncoded + "." + claimsEncoded
	expectedSignature := jm.signHMAC(message)
	if signature != expectedSignature {
		return nil, fmt.Errorf("invalid JWT signature")
	}
	
	// 解码声明
	claimsBytes, err := base64.RawURLEncoding.DecodeString(claimsEncoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode claims: %v", err)
	}
	
	var claims JWTClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %v", err)
	}
	
	// 验证时间
	now := time.Now().Unix()
	if claims.ExpiresAt < now {
		return nil, fmt.Errorf("token expired")
	}
	
	if claims.NotBefore > now {
		return nil, fmt.Errorf("token not yet valid")
	}
	
	return &claims, nil
}

// RefreshToken 刷新JWT令牌
func (jm *JWTManager) RefreshToken(token string, newExpiration time.Duration) (string, error) {
	claims, err := jm.ValidateToken(token)
	if err != nil {
		return "", err
	}
	
	// 检查是否在刷新窗口内（例如，令牌过期前1小时内）
	refreshWindow := time.Hour
	expireTime := time.Unix(claims.ExpiresAt, 0)
	if time.Until(expireTime) > refreshWindow {
		return "", fmt.Errorf("token not eligible for refresh yet")
	}
	
	// 生成新令牌
	return jm.GenerateToken(claims.Email, claims.IsAdmin, newExpiration)
}

// signHMAC 使用HMAC-SHA256签名
func (jm *JWTManager) signHMAC(message string) string {
	h := hmac.New(sha256.New, jm.secretKey)
	h.Write([]byte(message))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// ExtractEmailFromToken 从令牌中提取邮箱
func (jm *JWTManager) ExtractEmailFromToken(token string) (string, error) {
	claims, err := jm.ValidateToken(token)
	if err != nil {
		return "", err
	}
	
	return claims.Email, nil
}

// IsTokenAdmin 检查令牌是否为管理员
func (jm *JWTManager) IsTokenAdmin(token string) (bool, error) {
	claims, err := jm.ValidateToken(token)
	if err != nil {
		return false, err
	}
	
	return claims.IsAdmin, nil
}

// GetTokenInfo 获取令牌信息
func (jm *JWTManager) GetTokenInfo(token string) (map[string]interface{}, error) {
	claims, err := jm.ValidateToken(token)
	if err != nil {
		return nil, err
	}
	
	info := map[string]interface{}{
		"email":      claims.Email,
		"is_admin":   claims.IsAdmin,
		"issued_at":  time.Unix(claims.IssuedAt, 0).Format("2006-01-02 15:04:05"),
		"expires_at": time.Unix(claims.ExpiresAt, 0).Format("2006-01-02 15:04:05"),
		"subject":    claims.Subject,
		"issuer":     claims.Issuer,
		"valid":      true,
	}
	
	return info, nil
}

// BlacklistedTokens 令牌黑名单管理
type BlacklistedTokens struct {
	tokens map[string]time.Time // token -> 过期时间
}

func NewBlacklistedTokens() *BlacklistedTokens {
	return &BlacklistedTokens{
		tokens: make(map[string]time.Time),
	}
}

// AddToken 添加令牌到黑名单
func (bt *BlacklistedTokens) AddToken(token string, expiration time.Time) {
	bt.tokens[token] = expiration
}

// IsBlacklisted 检查令牌是否在黑名单中
func (bt *BlacklistedTokens) IsBlacklisted(token string) bool {
	expiration, exists := bt.tokens[token]
	if !exists {
		return false
	}
	
	// 如果令牌已过期，从黑名单中移除
	if time.Now().After(expiration) {
		delete(bt.tokens, token)
		return false
	}
	
	return true
}

// CleanupExpired 清理过期的黑名单令牌
func (bt *BlacklistedTokens) CleanupExpired() {
	now := time.Now()
	for token, expiration := range bt.tokens {
		if now.After(expiration) {
			delete(bt.tokens, token)
		}
	}
}

// ExtendedJWTManager 扩展的JWT管理器，包含黑名单功能
type ExtendedJWTManager struct {
	*JWTManager
	blacklist *BlacklistedTokens
}

func NewExtendedJWTManager(secretKey string) *ExtendedJWTManager {
	return &ExtendedJWTManager{
		JWTManager: NewJWTManager(secretKey),
		blacklist:  NewBlacklistedTokens(),
	}
}

// ValidateTokenWithBlacklist 验证令牌（包含黑名单检查）
func (ejm *ExtendedJWTManager) ValidateTokenWithBlacklist(token string) (*JWTClaims, error) {
	// 检查黑名单
	if ejm.blacklist.IsBlacklisted(token) {
		return nil, fmt.Errorf("token is blacklisted")
	}
	
	// 验证令牌
	return ejm.ValidateToken(token)
}

// RevokeToken 撤销令牌
func (ejm *ExtendedJWTManager) RevokeToken(token string) error {
	// 验证令牌格式并获取过期时间
	claims, err := ejm.ValidateToken(token)
	if err != nil {
		return err
	}
	
	// 添加到黑名单
	expiration := time.Unix(claims.ExpiresAt, 0)
	ejm.blacklist.AddToken(token, expiration)
	
	return nil
}

// CleanupBlacklist 清理黑名单
func (ejm *ExtendedJWTManager) CleanupBlacklist() {
	ejm.blacklist.CleanupExpired()
}

// GenerateTokenPair 生成访问令牌和刷新令牌对
func (ejm *ExtendedJWTManager) GenerateTokenPair(email string, isAdmin bool) (map[string]interface{}, error) {
	// 生成访问令牌（短期）
	accessToken, err := ejm.GenerateToken(email, isAdmin, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	
	// 生成刷新令牌（长期）
	refreshToken, err := ejm.GenerateToken(email, isAdmin, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    86400, // 24小时
		"is_admin":      isAdmin,
	}, nil
}

// TokenStatistics 令牌统计
type TokenStatistics struct {
	TotalGenerated int `json:"total_generated"`
	TotalRevoked   int `json:"total_revoked"`
	ActiveTokens   int `json:"active_tokens"`
}

// GetTokenStatistics 获取令牌统计信息
func (ejm *ExtendedJWTManager) GetTokenStatistics() *TokenStatistics {
	return &TokenStatistics{
		TotalGenerated: 0, // 需要在实际使用中计数
		TotalRevoked:   len(ejm.blacklist.tokens),
		ActiveTokens:   0, // 需要在实际使用中计数
	}
}