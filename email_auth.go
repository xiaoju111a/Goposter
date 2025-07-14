package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
)

type EmailAuth struct {
	domain     string
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	selector   string
}

func NewEmailAuth(domain string) *EmailAuth {
	ea := &EmailAuth{
		domain:   domain,
		selector: "default",
	}
	
	// 加载或生成DKIM密钥对
	ea.loadOrGenerateKeys()
	
	return ea
}

func (ea *EmailAuth) loadOrGenerateKeys() {
	keyFile := "./data/dkim_private.pem"
	
	// 尝试加载现有密钥
	if data, err := os.ReadFile(keyFile); err == nil {
		block, _ := pem.Decode(data)
		if block != nil {
			if privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
				ea.privateKey = privateKey
				ea.publicKey = &privateKey.PublicKey
				log.Printf("已加载DKIM密钥")
				return
			}
		}
	}
	
	// 生成新密钥对
	log.Printf("生成新的DKIM密钥对...")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Printf("生成DKIM密钥失败: %v", err)
		return
	}
	
	ea.privateKey = privateKey
	ea.publicKey = &privateKey.PublicKey
	
	// 保存私钥
	os.MkdirAll("./data", 0755)
	keyData := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyData,
	}
	
	file, err := os.Create(keyFile)
	if err != nil {
		log.Printf("保存DKIM私钥失败: %v", err)
		return
	}
	defer file.Close()
	
	if err := pem.Encode(file, block); err != nil {
		log.Printf("编码DKIM私钥失败: %v", err)
		return
	}
	
	log.Printf("DKIM密钥对生成并保存成功")
}

func (ea *EmailAuth) GetDKIMPublicKey() string {
	if ea.publicKey == nil {
		return ""
	}
	
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(ea.publicKey)
	if err != nil {
		return ""
	}
	
	pubKeyB64 := base64.StdEncoding.EncodeToString(pubKeyBytes)
	return fmt.Sprintf("v=DKIM1;k=rsa;p=%s", pubKeyB64)
}

func (ea *EmailAuth) SignEmail(headers, body string) string {
	if ea.privateKey == nil {
		return ""
	}
	
	// 简化的DKIM签名实现
	timestamp := time.Now().Unix()
	
	// 构建DKIM签名头
	dkimHeader := fmt.Sprintf("v=1; a=rsa-sha256; c=relaxed/simple; d=%s; s=%s; t=%d; h=from:to:subject:date; b=",
		ea.domain, ea.selector, timestamp)
	
	// 创建签名数据
	signData := fmt.Sprintf("DKIM-Signature: %s\r\n%s\r\n\r\n%s", dkimHeader, headers, body)
	
	// 计算SHA256哈希
	hash := sha256.Sum256([]byte(signData))
	
	// RSA签名
	signature, err := rsa.SignPKCS1v15(rand.Reader, ea.privateKey, 0, hash[:])
	if err != nil {
		log.Printf("DKIM签名失败: %v", err)
		return ""
	}
	
	// Base64编码签名
	signatureB64 := base64.StdEncoding.EncodeToString(signature)
	
	return fmt.Sprintf("v=1; a=rsa-sha256; c=relaxed/simple; d=%s; s=%s; t=%d; h=from:to:subject:date; b=%s",
		ea.domain, ea.selector, timestamp, signatureB64)
}

func (ea *EmailAuth) AddAuthHeaders(message string) string {
	lines := strings.Split(message, "\r\n")
	headerEnd := -1
	
	// 找到头部结束位置
	for i, line := range lines {
		if line == "" {
			headerEnd = i
			break
		}
	}
	
	if headerEnd == -1 {
		return message
	}
	
	// 提取头部和正文
	headers := strings.Join(lines[:headerEnd], "\r\n")
	body := strings.Join(lines[headerEnd+1:], "\r\n")
	
	// 生成DKIM签名
	dkimSignature := ea.SignEmail(headers, body)
	
	// 添加认证头部
	var authHeaders []string
	
	// 添加DKIM签名
	if dkimSignature != "" {
		authHeaders = append(authHeaders, fmt.Sprintf("DKIM-Signature: %s", dkimSignature))
	}
	
	// 构建最终消息
	var result []string
	result = append(result, lines[:headerEnd]...)
	result = append(result, authHeaders...)
	result = append(result, "")
	result = append(result, lines[headerEnd+1:]...)
	
	return strings.Join(result, "\r\n")
}

func (ea *EmailAuth) GetDNSRecords() map[string]string {
	records := make(map[string]string)
	
	// SPF记录
	records["SPF"] = fmt.Sprintf("v=spf1 a mx ip4:[服务器IP] ~all")
	
	// DKIM记录
	if dkimKey := ea.GetDKIMPublicKey(); dkimKey != "" {
		records["DKIM"] = fmt.Sprintf("%s._domainkey.%s: %s", ea.selector, ea.domain, dkimKey)
	}
	
	// DMARC记录
	records["DMARC"] = fmt.Sprintf("_dmarc.%s: v=DMARC1; p=none; rua=mailto:dmarc@%s; ruf=mailto:dmarc@%s; fo=1", 
		ea.domain, ea.domain, ea.domain)
	
	return records
}

// VerifyDKIMSignature 验证DKIM签名
func (ea *EmailAuth) VerifyDKIMSignature(message string) (bool, error) {
	// 解析DKIM-Signature头
	dkimSignature := ea.extractDKIMSignature(message)
	if dkimSignature == "" {
		return false, fmt.Errorf("未找到DKIM签名")
	}
	
	// 解析DKIM签名参数
	params := ea.parseDKIMSignature(dkimSignature)
	if len(params) == 0 {
		return false, fmt.Errorf("DKIM签名格式无效")
	}
	
	// 获取签名域名和选择器
	domain := params["d"]
	selector := params["s"]
	if domain == "" || selector == "" {
		return false, fmt.Errorf("DKIM签名缺少必要参数")
	}
	
	// 查询DNS获取公钥
	publicKey, err := ea.getDKIMPublicKeyFromDNS(selector, domain)
	if err != nil {
		return false, fmt.Errorf("获取DKIM公钥失败: %v", err)
	}
	
	// 验证签名
	return ea.verifySignature(message, params, publicKey), nil
}

// extractDKIMSignature 提取DKIM签名头
func (ea *EmailAuth) extractDKIMSignature(message string) string {
	lines := strings.Split(message, "\r\n")
	var dkimSignature string
	
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "dkim-signature:") {
			dkimSignature = strings.TrimPrefix(line, "DKIM-Signature:")
			dkimSignature = strings.TrimPrefix(dkimSignature, "dkim-signature:")
			dkimSignature = strings.TrimSpace(dkimSignature)
			break
		}
	}
	
	return dkimSignature
}

// parseDKIMSignature 解析DKIM签名参数
func (ea *EmailAuth) parseDKIMSignature(signature string) map[string]string {
	params := make(map[string]string)
	
	// 移除空格和换行
	signature = strings.ReplaceAll(signature, " ", "")
	signature = strings.ReplaceAll(signature, "\t", "")
	signature = strings.ReplaceAll(signature, "\r\n", "")
	
	// 按分号分割参数
	pairs := strings.Split(signature, ";")
	for _, pair := range pairs {
		if strings.Contains(pair, "=") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				params[key] = value
			}
		}
	}
	
	return params
}

// getDKIMPublicKeyFromDNS 从DNS查询DKIM公钥
func (ea *EmailAuth) getDKIMPublicKeyFromDNS(selector, domain string) (*rsa.PublicKey, error) {
	// 构建DNS查询名称
	dnsName := fmt.Sprintf("%s._domainkey.%s", selector, domain)
	
	// 查询TXT记录
	txtRecords, err := net.LookupTXT(dnsName)
	if err != nil {
		return nil, fmt.Errorf("DNS查询失败: %v", err)
	}
	
	// 查找DKIM记录
	var dkimRecord string
	for _, record := range txtRecords {
		if strings.Contains(record, "v=DKIM1") {
			dkimRecord = record
			break
		}
	}
	
	if dkimRecord == "" {
		return nil, fmt.Errorf("未找到DKIM记录")
	}
	
	// 解析公钥
	return ea.parseDKIMPublicKey(dkimRecord)
}

// parseDKIMPublicKey 解析DKIM公钥
func (ea *EmailAuth) parseDKIMPublicKey(record string) (*rsa.PublicKey, error) {
	// 提取公钥部分
	re := regexp.MustCompile(`p=([A-Za-z0-9+/=]+)`)
	matches := re.FindStringSubmatch(record)
	if len(matches) < 2 {
		return nil, fmt.Errorf("DKIM记录中未找到公钥")
	}
	
	pubKeyB64 := matches[1]
	
	// Base64解码
	pubKeyBytes, err := base64.StdEncoding.DecodeString(pubKeyB64)
	if err != nil {
		return nil, fmt.Errorf("公钥解码失败: %v", err)
	}
	
	// 解析公钥
	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("公钥解析失败: %v", err)
	}
	
	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("不是RSA公钥")
	}
	
	return rsaPubKey, nil
}

// verifySignature 验证签名
func (ea *EmailAuth) verifySignature(message string, params map[string]string, publicKey *rsa.PublicKey) bool {
	// 获取签名
	signatureB64 := params["b"]
	if signatureB64 == "" {
		return false
	}
	
	signature, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		log.Printf("签名解码失败: %v", err)
		return false
	}
	
	// 重建要验证的数据
	signData := ea.buildSignatureData(message, params)
	
	// 计算哈希
	hash := sha256.Sum256([]byte(signData))
	
	// 验证签名
	err = rsa.VerifyPKCS1v15(publicKey, 0, hash[:], signature)
	return err == nil
}

// buildSignatureData 重建签名数据
func (ea *EmailAuth) buildSignatureData(message string, params map[string]string) string {
	lines := strings.Split(message, "\r\n")
	headerEnd := -1
	
	// 找到头部结束位置
	for i, line := range lines {
		if line == "" {
			headerEnd = i
			break
		}
	}
	
	if headerEnd == -1 {
		return ""
	}
	
	// 提取头部和正文
	headers := strings.Join(lines[:headerEnd], "\r\n")
	body := strings.Join(lines[headerEnd+1:], "\r\n")
	
	// 构建DKIM签名头（不包含b=部分）
	dkimHeader := fmt.Sprintf("v=%s; a=%s; c=%s; d=%s; s=%s; t=%s; h=%s; b=",
		params["v"], params["a"], params["c"], params["d"], 
		params["s"], params["t"], params["h"])
	
	return fmt.Sprintf("DKIM-Signature: %s\r\n%s\r\n\r\n%s", dkimHeader, headers, body)
}

// ValidateEmailAuthentication 验证邮件的各种认证信息
func (ea *EmailAuth) ValidateEmailAuthentication(message string) map[string]interface{} {
	result := map[string]interface{}{
		"dkim_valid":    false,
		"spf_result":    "none",
		"dmarc_result":  "none",
		"authenticated": false,
	}
	
	// 验证DKIM
	if valid, err := ea.VerifyDKIMSignature(message); err == nil && valid {
		result["dkim_valid"] = true
		result["authenticated"] = true
	} else if err != nil {
		result["dkim_error"] = err.Error()
	}
	
	// TODO: 添加SPF和DMARC验证
	
	return result
}