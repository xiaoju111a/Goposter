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
	"os"
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