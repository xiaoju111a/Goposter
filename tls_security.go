package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"time"
)

// TLSManager TLS管理器
type TLSManager struct {
	config     *TLSConfig
	certPath   string
	keyPath    string
	caPath     string
	tlsConfig  *tls.Config
	autoRenew  bool
	renewDays  int
}

// TLSConfig TLS配置
type TLSConfig struct {
	MinVersion     uint16   `json:"min_version"`
	MaxVersion     uint16   `json:"max_version"`
	CipherSuites   []uint16 `json:"cipher_suites"`
	CurvePreferences []tls.CurveID `json:"curve_preferences"`
	PreferServerCipherSuites bool `json:"prefer_server_cipher_suites"`
	SessionTicketsDisabled   bool `json:"session_tickets_disabled"`
	ClientAuth             tls.ClientAuthType `json:"client_auth"`
	InsecureSkipVerify     bool `json:"insecure_skip_verify"`
	ServerName             string `json:"server_name"`
	NextProtos             []string `json:"next_protos"`
	
	// 证书配置
	CertFile      string `json:"cert_file"`
	KeyFile       string `json:"key_file"`
	CAFile        string `json:"ca_file"`
	AutoGenerate  bool   `json:"auto_generate"`
	
	// 自动续期
	AutoRenew     bool `json:"auto_renew"`
	RenewDays     int  `json:"renew_days"`
	
	// OCSP
	OCSPStapling  bool `json:"ocsp_stapling"`
	
	// HSTS
	HSTSMaxAge    int  `json:"hsts_max_age"`
	HSTSIncludeSubdomains bool `json:"hsts_include_subdomains"`
	HSTSPreload   bool `json:"hsts_preload"`
}

// CertificateInfo 证书信息
type CertificateInfo struct {
	Subject        string    `json:"subject"`
	Issuer         string    `json:"issuer"`
	SerialNumber   string    `json:"serial_number"`
	NotBefore      time.Time `json:"not_before"`
	NotAfter       time.Time `json:"not_after"`
	DNSNames       []string  `json:"dns_names"`
	IPAddresses    []net.IP  `json:"ip_addresses"`
	KeyUsage       x509.KeyUsage `json:"key_usage"`
	ExtKeyUsage    []x509.ExtKeyUsage `json:"ext_key_usage"`
	IsCA           bool      `json:"is_ca"`
	Version        int       `json:"version"`
	SignatureAlgorithm string `json:"signature_algorithm"`
	PublicKeyAlgorithm string `json:"public_key_algorithm"`
	DaysUntilExpiry int      `json:"days_until_expiry"`
}

// NewTLSManager 创建TLS管理器
func NewTLSManager(config *TLSConfig) (*TLSManager, error) {
	if config == nil {
		config = getDefaultTLSConfig()
	}
	
	manager := &TLSManager{
		config:    config,
		certPath:  config.CertFile,
		keyPath:   config.KeyFile,
		caPath:    config.CAFile,
		autoRenew: config.AutoRenew,
		renewDays: config.RenewDays,
	}
	
	// 如果需要自动生成证书
	if config.AutoGenerate {
		if err := manager.generateSelfSignedCertificate(); err != nil {
			return nil, fmt.Errorf("failed to generate certificate: %v", err)
		}
	}
	
	// 加载TLS配置
	if err := manager.loadTLSConfig(); err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %v", err)
	}
	
	return manager, nil
}

// getDefaultTLSConfig 获取默认TLS配置
func getDefaultTLSConfig() *TLSConfig {
	return &TLSConfig{
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		},
		PreferServerCipherSuites: true,
		SessionTicketsDisabled:   false,
		ClientAuth:              tls.NoClientCert,
		InsecureSkipVerify:      false,
		NextProtos:              []string{"h2", "http/1.1"},
		
		CertFile:     "certs/server.crt",
		KeyFile:      "certs/server.key",
		CAFile:       "certs/ca.crt",
		AutoGenerate: true,
		
		AutoRenew:    true,
		RenewDays:    30,
		
		OCSPStapling: false,
		
		HSTSMaxAge:               31536000, // 1年
		HSTSIncludeSubdomains:   true,
		HSTSPreload:             false,
	}
}

// loadTLSConfig 加载TLS配置
func (tm *TLSManager) loadTLSConfig() error {
	cert, err := tls.LoadX509KeyPair(tm.certPath, tm.keyPath)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %v", err)
	}
	
	tlsConfig := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		MinVersion:               tm.config.MinVersion,
		MaxVersion:               tm.config.MaxVersion,
		CipherSuites:            tm.config.CipherSuites,
		CurvePreferences:        tm.config.CurvePreferences,
		PreferServerCipherSuites: tm.config.PreferServerCipherSuites,
		SessionTicketsDisabled:   tm.config.SessionTicketsDisabled,
		ClientAuth:              tm.config.ClientAuth,
		InsecureSkipVerify:      tm.config.InsecureSkipVerify,
		ServerName:              tm.config.ServerName,
		NextProtos:              tm.config.NextProtos,
	}
	
	// 如果有CA文件，设置客户端验证
	if tm.caPath != "" && tm.config.ClientAuth != tls.NoClientCert {
		caCert, err := os.ReadFile(tm.caPath)
		if err != nil {
			return fmt.Errorf("failed to read CA certificate: %v", err)
		}
		
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.ClientCAs = caCertPool
	}
	
	tm.tlsConfig = tlsConfig
	return nil
}

// generateSelfSignedCertificate 生成自签名证书
func (tm *TLSManager) generateSelfSignedCertificate() error {
	// 创建证书目录
	if err := os.MkdirAll("certs", 0755); err != nil {
		return fmt.Errorf("failed to create certs directory: %v", err)
	}
	
	// 生成私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %v", err)
	}
	
	// 创建证书模板
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"FreeAgent Mail Server"},
			Country:       []string{"US"},
			Province:      []string{"CA"},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
			CommonName:    "freeagent.live",
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour), // 1年有效期
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:     []string{"localhost", "freeagent.live", "*.freeagent.live"},
	}
	
	// 生成证书
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %v", err)
	}
	
	// 保存证书
	certOut, err := os.Create(tm.certPath)
	if err != nil {
		return fmt.Errorf("failed to create certificate file: %v", err)
	}
	defer certOut.Close()
	
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("failed to encode certificate: %v", err)
	}
	
	// 保存私钥
	keyOut, err := os.Create(tm.keyPath)
	if err != nil {
		return fmt.Errorf("failed to create key file: %v", err)
	}
	defer keyOut.Close()
	
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %v", err)
	}
	
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyDER}); err != nil {
		return fmt.Errorf("failed to encode private key: %v", err)
	}
	
	log.Printf("Generated self-signed certificate: %s", tm.certPath)
	return nil
}

// GetTLSConfig 获取TLS配置
func (tm *TLSManager) GetTLSConfig() *tls.Config {
	return tm.tlsConfig
}

// GetCertificateInfo 获取证书信息
func (tm *TLSManager) GetCertificateInfo() (*CertificateInfo, error) {
	certPEM, err := os.ReadFile(tm.certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %v", err)
	}
	
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}
	
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}
	
	daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)
	
	return &CertificateInfo{
		Subject:            cert.Subject.String(),
		Issuer:             cert.Issuer.String(),
		SerialNumber:       cert.SerialNumber.String(),
		NotBefore:          cert.NotBefore,
		NotAfter:           cert.NotAfter,
		DNSNames:           cert.DNSNames,
		IPAddresses:        cert.IPAddresses,
		KeyUsage:           cert.KeyUsage,
		ExtKeyUsage:        cert.ExtKeyUsage,
		IsCA:               cert.IsCA,
		Version:            cert.Version,
		SignatureAlgorithm: cert.SignatureAlgorithm.String(),
		PublicKeyAlgorithm: cert.PublicKeyAlgorithm.String(),
		DaysUntilExpiry:    daysUntilExpiry,
	}, nil
}

// CheckCertificateExpiry 检查证书过期时间
func (tm *TLSManager) CheckCertificateExpiry() (bool, int, error) {
	info, err := tm.GetCertificateInfo()
	if err != nil {
		return false, 0, err
	}
	
	needsRenewal := info.DaysUntilExpiry <= tm.renewDays
	return needsRenewal, info.DaysUntilExpiry, nil
}

// RenewCertificate 续期证书
func (tm *TLSManager) RenewCertificate() error {
	log.Println("Starting certificate renewal...")
	
	// 备份现有证书
	if err := tm.backupCertificate(); err != nil {
		return fmt.Errorf("failed to backup certificate: %v", err)
	}
	
	// 生成新证书
	if err := tm.generateSelfSignedCertificate(); err != nil {
		return fmt.Errorf("failed to generate new certificate: %v", err)
	}
	
	// 重新加载TLS配置
	if err := tm.loadTLSConfig(); err != nil {
		return fmt.Errorf("failed to reload TLS config: %v", err)
	}
	
	log.Println("Certificate renewal completed successfully")
	return nil
}

// backupCertificate 备份证书
func (tm *TLSManager) backupCertificate() error {
	timestamp := time.Now().Format("20060102_150405")
	backupCertPath := fmt.Sprintf("%s.backup_%s", tm.certPath, timestamp)
	backupKeyPath := fmt.Sprintf("%s.backup_%s", tm.keyPath, timestamp)
	
	// 备份证书文件
	if err := tm.copyFile(tm.certPath, backupCertPath); err != nil {
		return fmt.Errorf("failed to backup certificate: %v", err)
	}
	
	// 备份密钥文件
	if err := tm.copyFile(tm.keyPath, backupKeyPath); err != nil {
		return fmt.Errorf("failed to backup key: %v", err)
	}
	
	log.Printf("Certificate backed up to %s and %s", backupCertPath, backupKeyPath)
	return nil
}

// copyFile 复制文件
func (tm *TLSManager) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	
	return os.WriteFile(dst, data, 0644)
}

// ValidateCertificate 验证证书
func (tm *TLSManager) ValidateCertificate() error {
	info, err := tm.GetCertificateInfo()
	if err != nil {
		return fmt.Errorf("failed to get certificate info: %v", err)
	}
	
	// 检查证书是否过期
	if time.Now().After(info.NotAfter) {
		return fmt.Errorf("certificate has expired on %s", info.NotAfter.Format("2006-01-02"))
	}
	
	// 检查证书是否还未生效
	if time.Now().Before(info.NotBefore) {
		return fmt.Errorf("certificate is not yet valid (valid from %s)", info.NotBefore.Format("2006-01-02"))
	}
	
	// 检查是否需要续期
	if info.DaysUntilExpiry <= tm.renewDays {
		log.Printf("Certificate will expire in %d days, consider renewal", info.DaysUntilExpiry)
	}
	
	return nil
}

// GetTLSStats 获取TLS统计信息
func (tm *TLSManager) GetTLSStats() map[string]interface{} {
	info, err := tm.GetCertificateInfo()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}
	
	needsRenewal, daysUntilExpiry, _ := tm.CheckCertificateExpiry()
	
	return map[string]interface{}{
		"certificate": map[string]interface{}{
			"subject":              info.Subject,
			"issuer":               info.Issuer,
			"serial_number":        info.SerialNumber,
			"not_before":           info.NotBefore,
			"not_after":            info.NotAfter,
			"dns_names":            info.DNSNames,
			"days_until_expiry":    daysUntilExpiry,
			"needs_renewal":        needsRenewal,
			"signature_algorithm":  info.SignatureAlgorithm,
			"public_key_algorithm": info.PublicKeyAlgorithm,
		},
		"tls_config": map[string]interface{}{
			"min_version":      tm.config.MinVersion,
			"max_version":      tm.config.MaxVersion,
			"cipher_suites":    len(tm.config.CipherSuites),
			"curve_preferences": len(tm.config.CurvePreferences),
			"client_auth":      tm.config.ClientAuth,
			"next_protos":      tm.config.NextProtos,
		},
		"security": map[string]interface{}{
			"hsts_max_age":             tm.config.HSTSMaxAge,
			"hsts_include_subdomains":  tm.config.HSTSIncludeSubdomains,
			"hsts_preload":             tm.config.HSTSPreload,
			"ocsp_stapling":            tm.config.OCSPStapling,
			"session_tickets_disabled": tm.config.SessionTicketsDisabled,
		},
	}
}

// GetSecurityHeaders 获取安全头部
func (tm *TLSManager) GetSecurityHeaders() map[string]string {
	headers := make(map[string]string)
	
	// HSTS头部
	hstsValue := fmt.Sprintf("max-age=%d", tm.config.HSTSMaxAge)
	if tm.config.HSTSIncludeSubdomains {
		hstsValue += "; includeSubDomains"
	}
	if tm.config.HSTSPreload {
		hstsValue += "; preload"
	}
	headers["Strict-Transport-Security"] = hstsValue
	
	// 其他安全头部
	headers["X-Content-Type-Options"] = "nosniff"
	headers["X-Frame-Options"] = "DENY"
	headers["X-XSS-Protection"] = "1; mode=block"
	headers["Referrer-Policy"] = "strict-origin-when-cross-origin"
	headers["Content-Security-Policy"] = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'"
	
	return headers
}

// StartAutoRenewal 启动自动续期
func (tm *TLSManager) StartAutoRenewal() {
	if !tm.autoRenew {
		return
	}
	
	go func() {
		ticker := time.NewTicker(24 * time.Hour) // 每天检查一次
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				needsRenewal, days, err := tm.CheckCertificateExpiry()
				if err != nil {
					log.Printf("Error checking certificate expiry: %v", err)
					continue
				}
				
				if needsRenewal {
					log.Printf("Certificate expires in %d days, starting renewal...", days)
					if err := tm.RenewCertificate(); err != nil {
						log.Printf("Failed to renew certificate: %v", err)
					}
				}
			}
		}
	}()
	
	log.Println("TLS auto-renewal started")
}