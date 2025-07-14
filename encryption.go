package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// EncryptionManager 加密管理器
type EncryptionManager struct {
	masterKey []byte
	keyDerivationIterations int
}

// EmailEncryption 邮件加密结构
type EmailEncryption struct {
	EncryptedSubject string `json:"encrypted_subject"`
	EncryptedBody    string `json:"encrypted_body"`
	EncryptedHeaders string `json:"encrypted_headers"`
	SubjectSalt      string `json:"subject_salt"`
	BodySalt         string `json:"body_salt"`
	HeadersSalt      string `json:"headers_salt"`
	SubjectNonce     string `json:"subject_nonce"`
	BodyNonce        string `json:"body_nonce"`
	HeadersNonce     string `json:"headers_nonce"`
	Algorithm        string `json:"algorithm"`
	KeyVersion       int    `json:"key_version"`
	CreatedAt        int64  `json:"created_at"`
}

// AttachmentEncryption 附件加密结构
type AttachmentEncryption struct {
	EncryptedContent string `json:"encrypted_content"`
	OriginalName     string `json:"original_name"`
	ContentType      string `json:"content_type"`
	Size             int64  `json:"size"`
	Salt             string `json:"salt"`
	Nonce            string `json:"nonce"`
	Checksum         string `json:"checksum"`
}

// NewEncryptionManager 创建加密管理器
func NewEncryptionManager(masterKey string) *EncryptionManager {
	key := sha256.Sum256([]byte(masterKey))
	
	return &EncryptionManager{
		masterKey: key[:],
		keyDerivationIterations: 10000,
	}
}

// EncryptEmail 加密邮件
func (em *EncryptionManager) EncryptEmail(subject, body, headers string) (*EmailEncryption, error) {
	// 加密主题
	encSubject, subjectSalt, subjectNonce, err := em.encryptText(subject)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt subject: %v", err)
	}
	
	// 加密正文
	encBody, bodySalt, bodyNonce, err := em.encryptText(body)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt body: %v", err)
	}
	
	// 加密头部
	encHeaders, headersSalt, headersNonce, err := em.encryptText(headers)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt headers: %v", err)
	}
	
	return &EmailEncryption{
		EncryptedSubject: encSubject,
		EncryptedBody:    encBody,
		EncryptedHeaders: encHeaders,
		SubjectSalt:      subjectSalt,
		BodySalt:         bodySalt,
		HeadersSalt:      headersSalt,
		SubjectNonce:     subjectNonce,
		BodyNonce:        bodyNonce,
		HeadersNonce:     headersNonce,
		Algorithm:        "AES-256-GCM",
		KeyVersion:       1,
		CreatedAt:        time.Now().Unix(),
	}, nil
}

// DecryptEmail 解密邮件
func (em *EncryptionManager) DecryptEmail(encEmail *EmailEncryption) (subject, body, headers string, err error) {
	// 解密主题
	subject, err = em.decryptText(encEmail.EncryptedSubject, encEmail.SubjectSalt, encEmail.SubjectNonce)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to decrypt subject: %v", err)
	}
	
	// 解密正文
	body, err = em.decryptText(encEmail.EncryptedBody, encEmail.BodySalt, encEmail.BodyNonce)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to decrypt body: %v", err)
	}
	
	// 解密头部
	headers, err = em.decryptText(encEmail.EncryptedHeaders, encEmail.HeadersSalt, encEmail.HeadersNonce)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to decrypt headers: %v", err)
	}
	
	return subject, body, headers, nil
}

// EncryptAttachment 加密附件
func (em *EncryptionManager) EncryptAttachment(content []byte, filename, contentType string) (*AttachmentEncryption, error) {
	// 生成盐和nonce
	salt := make([]byte, 16)
	nonce := make([]byte, 12)
	
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	
	// 派生密钥
	key := pbkdf2.Key(em.masterKey, salt, em.keyDerivationIterations, 32, sha256.New)
	
	// 创建AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	// 加密内容
	ciphertext := gcm.Seal(nil, nonce, content, nil)
	
	// 计算校验和
	checksum := sha256.Sum256(content)
	
	return &AttachmentEncryption{
		EncryptedContent: base64.StdEncoding.EncodeToString(ciphertext),
		OriginalName:     filename,
		ContentType:      contentType,
		Size:             int64(len(content)),
		Salt:             hex.EncodeToString(salt),
		Nonce:            hex.EncodeToString(nonce),
		Checksum:         hex.EncodeToString(checksum[:]),
	}, nil
}

// DecryptAttachment 解密附件
func (em *EncryptionManager) DecryptAttachment(encAttachment *AttachmentEncryption) ([]byte, error) {
	// 解码
	ciphertext, err := base64.StdEncoding.DecodeString(encAttachment.EncryptedContent)
	if err != nil {
		return nil, err
	}
	
	salt, err := hex.DecodeString(encAttachment.Salt)
	if err != nil {
		return nil, err
	}
	
	nonce, err := hex.DecodeString(encAttachment.Nonce)
	if err != nil {
		return nil, err
	}
	
	// 派生密钥
	key := pbkdf2.Key(em.masterKey, salt, em.keyDerivationIterations, 32, sha256.New)
	
	// 创建AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	
	// 验证校验和
	checksum := sha256.Sum256(plaintext)
	expectedChecksum, err := hex.DecodeString(encAttachment.Checksum)
	if err != nil {
		return nil, err
	}
	
	if !compareBytes(checksum[:], expectedChecksum) {
		return nil, fmt.Errorf("checksum verification failed")
	}
	
	return plaintext, nil
}

// encryptText 加密文本
func (em *EncryptionManager) encryptText(plaintext string) (encrypted, salt, nonce string, err error) {
	// 生成随机盐和nonce
	saltBytes := make([]byte, 16)
	nonceBytes := make([]byte, 12)
	
	if _, err := rand.Read(saltBytes); err != nil {
		return "", "", "", err
	}
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", "", "", err
	}
	
	// 派生密钥
	key := pbkdf2.Key(em.masterKey, saltBytes, em.keyDerivationIterations, 32, sha256.New)
	
	// 创建AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", "", err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", "", err
	}
	
	// 加密
	ciphertext := gcm.Seal(nil, nonceBytes, []byte(plaintext), nil)
	
	return base64.StdEncoding.EncodeToString(ciphertext),
		hex.EncodeToString(saltBytes),
		hex.EncodeToString(nonceBytes),
		nil
}

// decryptText 解密文本
func (em *EncryptionManager) decryptText(encrypted, salt, nonce string) (string, error) {
	// 解码
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	
	saltBytes, err := hex.DecodeString(salt)
	if err != nil {
		return "", err
	}
	
	nonceBytes, err := hex.DecodeString(nonce)
	if err != nil {
		return "", err
	}
	
	// 派生密钥
	key := pbkdf2.Key(em.masterKey, saltBytes, em.keyDerivationIterations, 32, sha256.New)
	
	// 创建AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	
	// 解密
	plaintext, err := gcm.Open(nil, nonceBytes, ciphertext, nil)
	if err != nil {
		return "", err
	}
	
	return string(plaintext), nil
}

// EncryptPassword 加密密码
func (em *EncryptionManager) EncryptPassword(password string) (string, string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", "", err
	}
	
	// 使用PBKDF2进行密码哈希
	hash := pbkdf2.Key([]byte(password), salt, 10000, 32, sha256.New)
	
	return hex.EncodeToString(hash), hex.EncodeToString(salt), nil
}

// VerifyPassword 验证密码
func (em *EncryptionManager) VerifyPassword(password, hashedPassword, salt string) bool {
	saltBytes, err := hex.DecodeString(salt)
	if err != nil {
		return false
	}
	
	hash := pbkdf2.Key([]byte(password), saltBytes, 10000, 32, sha256.New)
	expectedHash, err := hex.DecodeString(hashedPassword)
	if err != nil {
		return false
	}
	
	return compareBytes(hash, expectedHash)
}

// SecureDelete 安全删除敏感数据
func (em *EncryptionManager) SecureDelete(data []byte) {
	// 多次覆写内存
	for i := 0; i < 3; i++ {
		for j := range data {
			data[j] = 0
		}
		// 用随机数据覆写
		rand.Read(data)
	}
	
	// 最后一次置零
	for j := range data {
		data[j] = 0
	}
}

// GenerateEncryptionKey 生成加密密钥
func (em *EncryptionManager) GenerateEncryptionKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	
	return hex.EncodeToString(key), nil
}

// RotateEncryptionKey 轮换加密密钥
func (em *EncryptionManager) RotateEncryptionKey(newKey string) error {
	// 验证新密钥格式
	if len(newKey) != 64 {
		return fmt.Errorf("invalid key length")
	}
	
	keyBytes, err := hex.DecodeString(newKey)
	if err != nil {
		return fmt.Errorf("invalid key format")
	}
	
	// 安全删除旧密钥
	em.SecureDelete(em.masterKey)
	
	// 更新密钥
	em.masterKey = keyBytes
	
	return nil
}

// GetEncryptionInfo 获取加密信息
func (em *EncryptionManager) GetEncryptionInfo() map[string]interface{} {
	return map[string]interface{}{
		"algorithm":       "AES-256-GCM",
		"key_derivation":  "PBKDF2",
		"iterations":      em.keyDerivationIterations,
		"key_size":        256,
		"salt_size":       128,
		"nonce_size":      96,
		"secure_delete":   true,
		"key_rotation":    true,
	}
}

// EncryptSearchIndex 加密搜索索引
func (em *EncryptionManager) EncryptSearchIndex(text string) (string, error) {
	// 创建可搜索的加密索引
	words := strings.Fields(strings.ToLower(text))
	var encryptedWords []string
	
	for _, word := range words {
		if len(word) < 3 {
			continue // 跳过太短的词
		}
		
		// 使用确定性加密创建搜索索引
		hash := sha256.Sum256(append(em.masterKey, []byte(word)...))
		encryptedWords = append(encryptedWords, hex.EncodeToString(hash[:8]))
	}
	
	return strings.Join(encryptedWords, " "), nil
}

// SearchEncryptedContent 搜索加密内容
func (em *EncryptionManager) SearchEncryptedContent(searchTerm string, encryptedIndex string) bool {
	// 对搜索词进行同样的加密处理
	hash := sha256.Sum256(append(em.masterKey, []byte(strings.ToLower(searchTerm))...))
	encryptedTerm := hex.EncodeToString(hash[:8])
	
	// 在加密索引中查找
	return strings.Contains(encryptedIndex, encryptedTerm)
}

// ValidateEncryptedData 验证加密数据完整性
func (em *EncryptionManager) ValidateEncryptedData(encEmail *EmailEncryption) error {
	// 验证必要字段
	if encEmail.EncryptedSubject == "" || encEmail.EncryptedBody == "" {
		return fmt.Errorf("missing required encrypted fields")
	}
	
	// 验证盐和nonce长度
	if len(encEmail.SubjectSalt) != 32 || len(encEmail.BodySalt) != 32 {
		return fmt.Errorf("invalid salt length")
	}
	
	if len(encEmail.SubjectNonce) != 24 || len(encEmail.BodyNonce) != 24 {
		return fmt.Errorf("invalid nonce length")
	}
	
	// 验证算法和版本
	if encEmail.Algorithm != "AES-256-GCM" {
		return fmt.Errorf("unsupported encryption algorithm")
	}
	
	if encEmail.KeyVersion != 1 {
		return fmt.Errorf("unsupported key version")
	}
	
	return nil
}

// compareBytes 安全比较字节数组
func compareBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	
	return result == 0
}

// GenerateSecureRandom 生成安全随机数
func GenerateSecureRandom(length int) ([]byte, error) {
	bytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}