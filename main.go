package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/mail"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)


// Mailbox 邮箱结构

// AttachmentInfo 附件信息
type AttachmentInfo struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Content     []byte `json:"content,omitempty"`
	CID         string `json:"cid,omitempty"`
	Disposition string `json:"disposition"`
}

// EmailContent 邮件内容结构
type EmailContent struct {
	Subject     string
	Body        string
	HTMLBody    string
	Date        string
	From        string
	To          []string
	CC          []string
	BCC         []string
	Attachments []AttachmentInfo
	Signature   string
	IsAutoReply bool
	Charset     string
	Headers     map[string]string
}

type Email struct {
	From        string                 `json:"from"`
	To          string                 `json:"to"`
	Subject     string                 `json:"subject"`
	Body        string                 `json:"body"`
	HTMLBody    string                 `json:"html_body,omitempty"`
	Date        string                 `json:"date"`
	ID          string                 `json:"id"`
	Attachments []AttachmentInfo       `json:"attachments,omitempty"`
	Signature   string                 `json:"signature,omitempty"`
	IsAutoReply bool                   `json:"is_auto_reply"`
	Charset     string                 `json:"charset,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Embedded    map[string][]string    `json:"embedded,omitempty"`
}

// 临时添加缺失的类型定义
type SearchRequest struct {
	Query      string            `json:"query"`
	Mailbox    string            `json:"mailbox,omitempty"`
	From       string            `json:"from,omitempty"`
	To         string            `json:"to,omitempty"`
	Subject    string            `json:"subject,omitempty"`
	DateStart  string            `json:"date_start,omitempty"`
	DateEnd    string            `json:"date_end,omitempty"`
	HasAttachment *bool          `json:"has_attachment,omitempty"`
	IsRead     *bool             `json:"is_read,omitempty"`
	Priority   string            `json:"priority,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	Size       int               `json:"size"`
	From_      int               `json:"from"`
	Sort       string            `json:"sort,omitempty"`
	Highlight  bool              `json:"highlight"`
}

type SearchResult struct {
	Total    int64           `json:"total"`
	Emails   []EmailDocument `json:"emails"`
	Took     int             `json:"took"`
	TimedOut bool            `json:"timed_out"`
}

type EmailDocument struct {
	ID          string    `json:"id"`
	Mailbox     string    `json:"mailbox"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Subject     string    `json:"subject"`
	Body        string    `json:"body"`
	Timestamp   time.Time `json:"timestamp"`
	HasAttachment bool    `json:"has_attachment"`
	IsRead      bool      `json:"is_read"`
	Priority    string    `json:"priority"`
	Size        int64     `json:"size"`
	Tags        []string  `json:"tags"`
	Headers     map[string]string `json:"headers"`
}

type MailServer struct {
	domain         string
	hostname       string
	storage        *EmailStorage
	mailboxManager *MailboxManager
	userAuth       *UserAuth
	database       *Database
	smtpSender     *SMTPSender
	emailAuth      *EmailAuth
	imapServer     *IMAPServer
	relayManager   *SMTPRelayManager
	// esClient       *ElasticsearchClient // 临时禁用
}

func NewMailServer(domain, hostname string) *MailServer {
	// 创建数据目录
	dataDir := "./data"
	os.MkdirAll(dataDir, 0755)
	
	// 初始化SQLite数据库
	database, err := NewDatabase("./data/mailserver.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	
	// 初始化各个组件
	storage := NewEmailStorage(dataDir)
	mailboxManager := NewMailboxManager(domain, "./data/mailboxes.json")
	userAuth := NewUserAuth(database)
	emailAuth := NewEmailAuth(domain)
	smtpSender := NewSMTPSender(domain, hostname, emailAuth)
	relayManager := NewSMTPRelayManager("./data/smtp_relay.json")
	
	ms := &MailServer{
		domain:         domain,
		hostname:       hostname,
		storage:        storage,
		mailboxManager: mailboxManager,
		userAuth:       userAuth,
		database:       database,
		smtpSender:     smtpSender,
		emailAuth:      emailAuth,
		relayManager:   relayManager,
	}
	
	// 同步JSON邮箱数据到SQLite数据库
	ms.syncMailboxesToDatabase()
	
	// 设置SMTP发送器的中继
	ms.smtpSender.SetRelay(ms.relayManager.GetRelay())
	
	// 创建IMAP服务器
	ms.imapServer = NewIMAPServer(ms)
	
	// 初始化ElasticSearch客户端 (临时禁用)
	// ms.esClient = NewElasticsearchClient()
	
	return ms
}

func (ms *MailServer) AddEmail(to string, email Email) {
	// 检查邮箱是否存在
	if !ms.mailboxManager.IsValidMailbox(to) {
		log.Printf("邮件被拒绝，邮箱不存在: %s", to)
		return
	}
	
	// 存储邮件
	ms.storage.AddEmail(to, email)
	
	// 索引到ElasticSearch (临时禁用)
	// ms.indexEmailToElastic(to, email)
	
	// 记录处理日志
	log.Printf("邮件已接收: %s -> %s", email.From, to)
}

func (ms *MailServer) GetEmails(mailbox string) []Email {
	return ms.storage.GetEmails(mailbox)
}

func (ms *MailServer) GetAllMailboxes() []string {
	mailboxes, err := ms.database.GetAllMailboxes()
	if err != nil {
		log.Printf("Error getting mailboxes: %v", err)
		return []string{}
	}
	
	var mailboxNames []string
	for _, mailbox := range mailboxes {
		mailboxNames = append(mailboxNames, mailbox.Email)
	}
	return mailboxNames
}

// 同步JSON邮箱数据到SQLite数据库
func (ms *MailServer) syncMailboxesToDatabase() {
	// 获取JSON中的所有邮箱
	jsonMailboxes := ms.mailboxManager.GetAllMailboxes()
	
	log.Printf("开始同步邮箱数据到数据库，JSON中有 %d 个邮箱", len(jsonMailboxes))
	
	for _, mailbox := range jsonMailboxes {
		// 检查数据库中是否已存在该邮箱
		_, err := ms.database.GetMailbox(mailbox.Email)
		if err != nil {
			// 邮箱不存在，需要创建
			log.Printf("正在同步邮箱: %s", mailbox.Email)
			
			// 创建邮箱到数据库 (JSON中的密码是明文，CreateMailbox会自动哈希)
			err = ms.database.CreateMailbox(mailbox.Email, mailbox.Password, mailbox.Description, mailbox.Owner)
			if err != nil {
				log.Printf("同步邮箱 %s 失败: %v", mailbox.Email, err)
			} else {
				log.Printf("成功同步邮箱: %s", mailbox.Email)
			}
		} else {
			// 邮箱已存在，但可能需要更新转发设置
			log.Printf("邮箱 %s 已存在，跳过同步", mailbox.Email)
		}
	}
	
	log.Printf("邮箱数据同步完成")
}

func (ms *MailServer) HandleSMTP(conn net.Conn) {
	defer conn.Close()
	
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	
	writer.WriteString("220 " + ms.domain + " ESMTP\r\n")
	writer.Flush()
	
	var from, to string
	var dataMode bool
	var emailData []string
	
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		
		line = strings.TrimSpace(line)
		cmd := strings.ToUpper(line)
		
		if dataMode {
			if line == "." {
				dataMode = false
				
				// 解析邮件内容
				rawContent := strings.Join(emailData, "\n")
				
				// 调试：保存原始邮件到文件
				if err := os.WriteFile(fmt.Sprintf("./data/raw_email_%d.eml", time.Now().Unix()), []byte(rawContent), 0644); err != nil {
					log.Printf("Warning: Failed to save raw email: %v", err)
				}
				
				// 使用增强邮件解析器
				emailContent := parseEmailContentEnhanced(rawContent)
				
				email := Email{
					From:        from,
					To:          to,
					Subject:     emailContent.Subject,
					Body:        emailContent.Body,
					HTMLBody:    emailContent.HTMLBody,
					Date:        emailContent.Date,
					ID:          generateEmailID(to),
					Attachments: emailContent.Attachments,
					Signature:   emailContent.Signature,
					IsAutoReply: emailContent.IsAutoReply,
					Charset:     emailContent.Charset,
					Headers:     emailContent.Headers,
					Embedded:    extractEmbeddedContent(emailContent),
				}
				
				// 检查是否需要保留原邮件
				mailboxInfo, _ := ms.mailboxManager.GetMailbox(to)
				keepOriginal := true
				if mailboxInfo != nil && mailboxInfo.ForwardEnabled {
					keepOriginal = mailboxInfo.KeepOriginal
				}
				
				// 只有在需要保留原邮件时才存储
				if keepOriginal {
					ms.AddEmail(to, email)
				}
				
				// 处理邮件转发
				go ms.processEmailForwarding(to, rawContent)
				
				writer.WriteString("250 OK: Message accepted\r\n")
				writer.Flush()
				
				emailData = nil
			} else {
				emailData = append(emailData, line)
			}
			continue
		}
		
		switch {
		case strings.HasPrefix(cmd, "HELO") || strings.HasPrefix(cmd, "EHLO"):
			writer.WriteString("250 " + ms.domain + "\r\n")
		case strings.HasPrefix(cmd, "MAIL FROM:"):
			from = extractEmail(line[10:])
			writer.WriteString("250 OK\r\n")
		case strings.HasPrefix(cmd, "RCPT TO:"):
			to = extractEmail(line[8:])
			writer.WriteString("250 OK\r\n")
		case cmd == "DATA":
			dataMode = true
			writer.WriteString("354 End data with <CR><LF>.<CR><LF>\r\n")
		case cmd == "QUIT":
			writer.WriteString("221 Bye\r\n")
			writer.Flush()
			return
		default:
			writer.WriteString("502 Command not implemented\r\n")
		}
		writer.Flush()
	}
}

func extractEmail(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">") {
		s = s[1 : len(s)-1]
	}
	return strings.ToLower(s)
}

func (ms *MailServer) StartSMTPServer(port string) {
	// 监听所有网络接口，允许外部连接
	listener, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		log.Fatal("Failed to start SMTP server:", err)
	}
	defer listener.Close()
	
	log.Printf("SMTP server listening on 0.0.0.0:%s (accepting external connections)", port)
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Failed to accept connection:", err)
			continue
		}
		
		// 记录连接信息
		log.Printf("New SMTP connection from: %s", conn.RemoteAddr())
		go ms.HandleSMTP(conn)
	}
}

func (ms *MailServer) StartWebServer(port string) {
	// 静态文件和主页 - 直接使用React版本
	http.HandleFunc("/", ms.reactHandler)
	http.HandleFunc("/debug", ms.debugHandler)
	
	// API路由
	http.HandleFunc("/api/mailboxes", ms.apiMailboxes)
	http.HandleFunc("/api/emails/", ms.apiEmails)
	http.HandleFunc("/api/emails/delete/", ms.apiDeleteEmail)
	http.HandleFunc("/api/send", ms.apiSendEmail)
	http.HandleFunc("/api/mailboxes/create", ms.apiCreateMailbox)
	http.HandleFunc("/api/mailboxes/manage", ms.apiManageMailboxes)
	http.HandleFunc("/api/login", ms.apiLogin)
	http.HandleFunc("/api/logout", ms.apiLogout)
	http.HandleFunc("/api/stats", ms.apiStats)
	http.HandleFunc("/api/dns/config", ms.apiDNSConfig)
	
	// JWT认证API路由
	http.HandleFunc("/api/auth/login", ms.apiAuthLogin)
	http.HandleFunc("/api/auth/logout", ms.apiAuthLogout)
	http.HandleFunc("/api/auth/refresh", ms.apiAuthRefresh)
	http.HandleFunc("/api/auth/2fa/enable", ms.apiAuth2FAEnable)
	http.HandleFunc("/api/auth/2fa/disable", ms.apiAuth2FADisable)
	http.HandleFunc("/api/auth/2fa/verify", ms.apiAuth2FAVerify)
	http.HandleFunc("/api/auth/2fa/status", ms.apiAuth2FAStatus)
	
	// SMTP中继API
	http.HandleFunc("/api/relay/config", ms.apiRelayConfig)
	http.HandleFunc("/api/relay/providers", ms.apiRelayProviders)
	http.HandleFunc("/api/relay/test", ms.apiRelayTest)
	http.HandleFunc("/api/relay/status", ms.apiRelayStatus)
	
	// 搜索API
	http.HandleFunc("/api/search", ms.apiSearch)
	http.HandleFunc("/api/search/suggest", ms.apiSearchSuggestions)
	
	// 附件处理API
	http.HandleFunc("/api/attachments/", ms.apiAttachments)
	http.HandleFunc("/api/attachments/inline/", ms.apiInlineAttachments)
	
	// 用户管理API
	http.HandleFunc("/api/admin/users", ms.apiAdminUsers)
	http.HandleFunc("/api/admin/users/create", ms.apiAdminCreateUser)
	// http.HandleFunc("/api/admin/users/delete", ms.apiAdminDeleteUser)
	http.HandleFunc("/api/admin/mailboxes", ms.apiAdminMailboxes)
	http.HandleFunc("/api/admin/mailboxes/create", ms.apiAdminCreateMailbox)
	// http.HandleFunc("/api/admin/mailboxes/delete", ms.apiAdminDeleteMailbox)
	// http.HandleFunc("/api/admin/mailboxes/assign", ms.apiAdminAssignMailbox)
	
	// 邮件转发API
	http.HandleFunc("/api/forwarding/settings", ms.apiForwardingSettings)
	http.HandleFunc("/api/forwarding/update", ms.apiForwardingUpdate)
	
	// 用户邮箱访问API
	http.HandleFunc("/api/user/mailboxes", ms.apiUserMailboxes)
	http.HandleFunc("/api/user/emails/", ms.apiUserEmails)
	
	log.Printf("Web server listening on 0.0.0.0:%s (accepting external connections)", port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}

// API处理方法
func (ms *MailServer) apiSendEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		From    string `json:"from"`
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// 在后台发送邮件，避免阻塞Web界面
	go func() {
		err := ms.smtpSender.SendEmail(req.From, req.To, req.Subject, req.Body)
		if err != nil {
			log.Printf("后台邮件发送失败: %v", err)
		} else {
			log.Printf("后台邮件发送成功: %s -> %s", req.From, req.To)
		}
	}()
	
	// 立即返回成功响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "queued", 
		"message": "邮件已加入发送队列，正在后台处理"})
}

func (ms *MailServer) apiCreateMailbox(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		Description string `json:"description"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if req.Username == "" || req.Password == "" {
		http.Error(w, "用户名和密码不能为空", http.StatusBadRequest)
		return
	}
	
	// 获取当前用户作为邮箱所有者
	currentUser := ms.getUserFromRequest(r)
	if currentUser == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	// 构建完整邮箱地址
	fullEmail := req.Username + "@" + ms.domain
	
	err := ms.database.CreateMailbox(fullEmail, req.Password, req.Description, currentUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	json.NewEncoder(w).Encode(map[string]string{
		"status": "created",
		"email":  req.Username + "@" + ms.domain,
	})
}

func (ms *MailServer) apiManageMailboxes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	switch r.Method {
	case "GET":
		mailboxes := ms.mailboxManager.GetAllMailboxes()
		var result []map[string]interface{}
		
		for _, mailbox := range mailboxes {
			result = append(result, map[string]interface{}{
				"email":       mailbox.Email,
				"username":    mailbox.Username,
				"description": mailbox.Description,
				"created_at":  mailbox.CreatedAt,
				"is_active":   mailbox.IsActive,
			})
		}
		
		json.NewEncoder(w).Encode(result)
		
	case "DELETE":
		email := r.URL.Query().Get("email")
		if email == "" {
			http.Error(w, "邮箱地址不能为空", http.StatusBadRequest)
			return
		}
		
		err := ms.mailboxManager.DeleteMailbox(email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
		
	case "PUT":
		var req struct {
			Email       string `json:"email"`
			Password    string `json:"password"`
			Description string `json:"description"`
			IsActive    bool   `json:"is_active"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		err := ms.mailboxManager.UpdateMailbox(req.Email, req.Password, req.Description, req.IsActive)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}
}

func (ms *MailServer) apiLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// 使用邮箱认证
	if ms.database.ValidateMailboxCredentials(req.Email, req.Password) {
		// 检查是否是管理员用户
		isAdmin := false
		if user, err := ms.database.GetUser(req.Email); err == nil {
			isAdmin = user.IsAdmin
		}
		
		// 生成JWT令牌
		tokens, err := ms.userAuth.GenerateJWTTokenWithAdmin(req.Email, isAdmin)
		if err != nil {
			log.Printf("JWT generation failed for %s: %v", req.Email, err)
			// JWT失败，使用备用方案
			sessionID, _ := ms.userAuth.CreateSession(req.Email)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "backup_" + sessionID,
				"token_type":   "Bearer",
				"expires_in":   86400,
				"is_admin":     isAdmin,
				"user_email":   req.Email,
			})
			return
		}
		
		// JWT成功
		response := map[string]interface{}{
			"user_email": req.Email,
			"is_admin":   isAdmin,
		}
		for k, v := range tokens {
			response[k] = v
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
	}
}

func (ms *MailServer) apiLogout(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("Session-ID")
	ms.database.DeleteSession(sessionID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "logged out"})
}

func (ms *MailServer) apiStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	stats := ms.storage.GetStorageStats()
	json.NewEncoder(w).Encode(stats)
}

func (ms *MailServer) apiDNSConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// 获取DNS记录配置
	dnsRecords := ms.emailAuth.GetDNSRecords()
	
	// 格式化输出
	result := map[string]interface{}{
		"domain": ms.domain,
		"hostname": ms.hostname,
		"dns_records": dnsRecords,
		"instructions": map[string]string{
			"spf": "在域名DNS中添加TXT记录",
			"dkim": "在域名DNS中添加TXT记录",
			"dmarc": "在域名DNS中添加TXT记录",
		},
	}
	
	json.NewEncoder(w).Encode(result)
}

// SMTP中继API处理方法
func (ms *MailServer) apiRelayConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	switch r.Method {
	case "GET":
		// 获取当前配置
		config := ms.relayManager.GetConfig()
		// 隐藏密码
		safeConfig := struct {
			Enabled     bool   `json:"enabled"`
			Host        string `json:"host"`
			Port        int    `json:"port"`
			Username    string `json:"username"`
			UseTLS      bool   `json:"use_tls"`
			HasPassword bool   `json:"has_password"`
		}{
			Enabled:     config.Enabled,
			Host:        config.Host,
			Port:        config.Port,
			Username:    config.Username,
			UseTLS:      config.UseTLS,
			HasPassword: config.Password != "",
		}
		json.NewEncoder(w).Encode(safeConfig)
		
	case "POST":
		// 更新配置
		var req SMTPRelayConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		err := ms.relayManager.UpdateConfig(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		// 更新SMTPSender的中继
		ms.smtpSender.SetRelay(ms.relayManager.GetRelay())
		
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
		
	case "DELETE":
		// 禁用中继
		err := ms.relayManager.DisableRelay()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		// 更新SMTPSender的中继
		ms.smtpSender.SetRelay(ms.relayManager.GetRelay())
		
		json.NewEncoder(w).Encode(map[string]string{"status": "disabled"})
	}
}

func (ms *MailServer) apiRelayProviders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	if r.Method == "GET" {
		providers := ms.relayManager.GetAvailableProviders()
		json.NewEncoder(w).Encode(providers)
	} else if r.Method == "POST" {
		// 设置预设提供商
		var req struct {
			Provider string `json:"provider"`
			Username string `json:"username"`
			Password string `json:"password"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		err := ms.relayManager.SetPresetProvider(req.Provider, req.Username, req.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		// 更新SMTPSender的中继
		ms.smtpSender.SetRelay(ms.relayManager.GetRelay())
		
		json.NewEncoder(w).Encode(map[string]string{"status": "configured"})
	}
}

func (ms *MailServer) apiRelayTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	err := ms.relayManager.TestConnection()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "SMTP中继连接测试成功",
	})
}

func (ms *MailServer) apiRelayStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	status := ms.relayManager.GetStatus()
	json.NewEncoder(w).Encode(status)
}

func (ms *MailServer) reactHandler(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>FreeAgent 邮箱管理系统</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; 
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            color: #333;
        }
        
        /* 登录模态框样式 */
        .login-modal {
            display: none;
            position: fixed;
            z-index: 1000;
            left: 0;
            top: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0,0,0,0.5);
            backdrop-filter: blur(5px);
        }
        
        .login-content {
            background: rgba(255,255,255,0.95);
            margin: 15% auto;
            padding: 30px;
            border-radius: 20px;
            width: 400px;
            max-width: 90%;
            box-shadow: 0 8px 32px rgba(0,0,0,0.2);
            text-align: center;
        }
        
        .login-content h2 {
            color: #2c3e50;
            margin-bottom: 20px;
        }
        
        .login-form input {
            width: 100%;
            padding: 12px;
            margin: 10px 0;
            border: 1px solid #ddd;
            border-radius: 8px;
            font-size: 16px;
        }
        
        .login-btn {
            background: linear-gradient(45deg, #3498db, #2980b9);
            color: white;
            padding: 12px 30px;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            font-size: 16px;
            width: 100%;
            margin-top: 10px;
        }
        
        .login-btn:hover {
            background: linear-gradient(45deg, #2980b9, #3498db);
        }
        
        /* 用户状态栏 */
        .user-bar {
            background: rgba(255,255,255,0.1);
            backdrop-filter: blur(10px);
            padding: 10px 20px;
            display: flex;
            justify-content: space-between;
            align-items: center;
            color: white;
            margin-bottom: 20px;
            border-radius: 10px;
        }
        
        .logout-btn {
            background: rgba(255,255,255,0.2);
            color: white;
            border: none;
            padding: 8px 16px;
            border-radius: 6px;
            cursor: pointer;
        }
        
        .logout-btn:hover {
            background: rgba(255,255,255,0.3);
        }
        
        /* 搜索栏样式 */
        .search-section {
            background: rgba(255,255,255,0.95);
            backdrop-filter: blur(10px);
            border-radius: 15px;
            padding: 20px;
            margin-bottom: 20px;
            box-shadow: 0 4px 16px rgba(0,0,0,0.1);
        }
        
        .search-controls {
            display: flex;
            gap: 10px;
            align-items: center;
            flex-wrap: wrap;
        }
        
        .search-input {
            flex: 1;
            min-width: 200px;
            padding: 12px;
            border: 1px solid #ddd;
            border-radius: 8px;
            font-size: 16px;
        }
        
        .search-select {
            padding: 12px;
            border: 1px solid #ddd;
            border-radius: 8px;
            font-size: 16px;
            background: white;
        }
        
        .clear-search-btn {
            background: #e74c3c;
            color: white;
            border: none;
            padding: 12px 20px;
            border-radius: 8px;
            cursor: pointer;
        }
        
        /* 邮件操作按钮 */
        .email-actions {
            display: flex;
            gap: 10px;
            margin-top: 10px;
            padding-top: 10px;
            border-top: 1px solid #eee;
        }
        
        .email-action-btn {
            padding: 6px 12px;
            border: none;
            border-radius: 6px;
            cursor: pointer;
            font-size: 12px;
        }
        
        .delete-btn {
            background: #e74c3c;
            color: white;
        }
        
        .reply-btn {
            background: #3498db;
            color: white;
        }
        
        .archive-btn {
            background: #95a5a6;
            color: white;
        }
        
        /* 邮件预览改进 */
        .email {
            background: white;
            margin: 10px 0;
            padding: 15px;
            border-radius: 10px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            border-left: 4px solid #3498db;
            transition: all 0.3s ease;
            cursor: pointer;
        }
        
        .email:hover {
            box-shadow: 0 4px 16px rgba(0,0,0,0.15);
            transform: translateY(-2px);
        }
        
        .email.expanded {
            border-left-color: #2ecc71;
        }
        
        .email-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
        }
        
        .email-preview {
            max-height: 60px;
            overflow: hidden;
            color: #666;
            line-height: 1.4;
        }
        
        .email-full {
            display: none;
            margin-top: 15px;
            padding-top: 15px;
            border-top: 1px solid #eee;
        }
        
        .email-time {
            font-size: 12px;
            color: #999;
        }
        
        /* 加载状态 */
        .loading {
            text-align: center;
            padding: 20px;
            color: #666;
        }
        
        .loading-spinner {
            display: inline-block;
            width: 20px;
            height: 20px;
            border: 3px solid #f3f3f3;
            border-top: 3px solid #3498db;
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }
        
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        
        /* 主题切换 */
        .theme-toggle {
            background: rgba(255,255,255,0.2);
            color: white;
            border: none;
            padding: 8px 16px;
            border-radius: 6px;
            cursor: pointer;
            margin-right: 10px;
        }
        
        /* 暗黑主题 */
        body.dark-theme {
            background: linear-gradient(135deg, #2c3e50 0%, #34495e 100%);
        }
        
        body.dark-theme .header,
        body.dark-theme .search-section,
        body.dark-theme .section {
            background: rgba(52, 73, 94, 0.95);
            color: #ecf0f1;
        }
        
        body.dark-theme .email {
            background: rgba(44, 62, 80, 0.8);
            color: #ecf0f1;
        }
        .container { 
            max-width: 1200px; 
            margin: 0 auto; 
            padding: 20px;
        }
        .header {
            background: rgba(255,255,255,0.95);
            backdrop-filter: blur(10px);
            border-radius: 20px;
            padding: 30px;
            margin-bottom: 30px;
            box-shadow: 0 8px 32px rgba(0,0,0,0.1);
            text-align: center;
        }
        .header h1 {
            color: #2c3e50;
            font-size: 2.5em;
            margin-bottom: 10px;
            font-weight: 700;
        }
        .header .subtitle {
            color: #7f8c8d;
            font-size: 1.1em;
            margin-bottom: 20px;
        }
        .stats {
            display: flex;
            justify-content: center;
            gap: 30px;
            margin-top: 20px;
        }
        .stat-item {
            text-align: center;
        }
        .stat-number {
            font-size: 2em;
            font-weight: bold;
            color: #3498db;
        }
        .stat-label {
            color: #7f8c8d;
            font-size: 0.9em;
        }
        .controls {
            background: rgba(255,255,255,0.95);
            backdrop-filter: blur(10px);
            border-radius: 15px;
            padding: 25px;
            margin-bottom: 30px;
            box-shadow: 0 8px 32px rgba(0,0,0,0.1);
        }
        .controls h3 {
            margin-bottom: 15px;
            color: #2c3e50;
        }
        .email-form {
            display: flex;
            gap: 10px;
            margin-bottom: 15px;
            flex-wrap: wrap;
        }
        .email-form input {
            flex: 1;
            min-width: 200px;
            padding: 12px 15px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 14px;
            transition: border-color 0.3s;
        }
        .email-form input:focus {
            outline: none;
            border-color: #3498db;
        }
        .btn {
            padding: 12px 20px;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            font-weight: 600;
            transition: all 0.3s;
            text-decoration: none;
            display: inline-block;
        }
        .btn-primary {
            background: linear-gradient(45deg, #3498db, #2980b9);
            color: white;
        }
        .btn-primary:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(52,152,219,0.4);
        }
        .btn-success {
            background: linear-gradient(45deg, #27ae60, #2ecc71);
            color: white;
        }
        .btn-danger {
            background: linear-gradient(45deg, #e74c3c, #c0392b);
            color: white;
            font-size: 12px;
            padding: 8px 12px;
        }
        .mailboxes-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
            gap: 20px;
        }
        .mailbox {
            background: rgba(255,255,255,0.95);
            backdrop-filter: blur(10px);
            border-radius: 15px;
            padding: 20px;
            box-shadow: 0 8px 32px rgba(0,0,0,0.1);
            transition: transform 0.3s;
        }
        .mailbox:hover {
            transform: translateY(-5px);
        }
        .mailbox-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 15px;
            padding-bottom: 10px;
            border-bottom: 2px solid #ecf0f1;
        }
        .mailbox-title {
            font-size: 1.2em;
            font-weight: 600;
            color: #2c3e50;
            word-break: break-all;
        }
        .email-count {
            background: #3498db;
            color: white;
            padding: 4px 8px;
            border-radius: 12px;
            font-size: 0.8em;
            font-weight: bold;
        }
        .email {
            margin: 10px 0;
            padding: 15px;
            background: #f8f9fa;
            border-radius: 10px;
            border-left: 4px solid #3498db;
            transition: all 0.3s;
        }
        .email:hover {
            background: #e3f2fd;
            transform: translateX(5px);
        }
        .email-subject {
            font-weight: 600;
            color: #2c3e50;
            margin-bottom: 8px;
            font-size: 1.1em;
        }
        .email-meta {
            color: #7f8c8d;
            font-size: 0.9em;
            margin-bottom: 8px;
        }
        .email-body {
            color: #34495e;
            line-height: 1.5;
            margin-top: 10px;
            word-wrap: break-word;
        }
        .no-emails {
            text-align: center;
            color: #7f8c8d;
            font-style: italic;
            padding: 20px;
        }
        .refresh-btn {
            position: fixed;
            bottom: 30px;
            right: 30px;
            width: 60px;
            height: 60px;
            border-radius: 50%;
            background: linear-gradient(45deg, #3498db, #2980b9);
            color: white;
            border: none;
            cursor: pointer;
            font-size: 20px;
            box-shadow: 0 5px 15px rgba(52,152,219,0.4);
            transition: all 0.3s;
        }
        .refresh-btn:hover {
            transform: scale(1.1) rotate(180deg);
        }
        @media (max-width: 768px) {
            .container { padding: 10px; }
            .header h1 { font-size: 2em; }
            .stats { flex-direction: column; gap: 15px; }
            .email-form { flex-direction: column; }
            .mailboxes-grid { grid-template-columns: 1fr; }
        }
    </style>
    <script>
        let mailboxStats = { total: 0, emails: 0 };
        let currentUser = null;
        let filteredEmails = {};
        
        // 检查登录状态
        function checkLogin() {
            const accessToken = localStorage.getItem('access_token');
            const userEmail = localStorage.getItem('userEmail');
            const expiresAt = localStorage.getItem('token_expires_at');
            
            if (!accessToken || !userEmail) {
                showLoginModal();
                return false;
            }
            
            // 检查token是否过期
            if (expiresAt) {
                const now = Date.now();
                if (now >= parseInt(expiresAt)) {
                    console.warn('Token has expired');
                    clearAuth();
                    showLoginModal();
                    return false;
                }
            }
            
            currentUser = userEmail;
            updateUserBar();
            return true;
        }
        
        // 清除认证信息
        function clearAuth() {
            localStorage.removeItem('access_token');
            localStorage.removeItem('refresh_token');
            localStorage.removeItem('userEmail');
            localStorage.removeItem('token_expires_at');
            localStorage.removeItem('currentUser'); // 兼容旧版本
        }
        
        // 获取认证头
        function getAuthHeaders() {
            const accessToken = localStorage.getItem('access_token');
            const headers = {'Content-Type': 'application/json'};
            if (accessToken) {
                headers['Authorization'] = 'Bearer ' + accessToken;
            }
            return headers;
        }
        
        // 带认证的fetch
        async function authenticatedFetch(url, options = {}) {
            const headers = getAuthHeaders();
            const mergedOptions = {
                ...options,
                headers: {
                    ...headers,
                    ...(options.headers || {})
                }
            };
            
            console.log('Making authenticated request to:', url);
            console.log('Headers:', mergedOptions.headers);
            
            const response = await fetch(url, mergedOptions);
            
            console.log('Response status:', response.status);
            
            // 如果返回401，清除认证并重新登录
            if (response.status === 401) {
                console.log('401 Unauthorized - clearing auth and redirecting to login');
                clearAuth();
                checkLogin();
                throw new Error('Authentication required');
            }
            
            return response;
        }
        
        // 显示登录模态框
        function showLoginModal() {
            document.getElementById('loginModal').style.display = 'block';
        }
        
        // 隐藏登录模态框
        function hideLoginModal() {
            document.getElementById('loginModal').style.display = 'none';
        }
        
        // 登录处理
        async function login() {
            const email = document.getElementById('loginEmail').value;
            const password = document.getElementById('loginPassword').value;
            
            if (!email || !password) {
                alert('请输入邮箱和密码');
                return;
            }
            
            try {
                const response = await fetch('/api/login', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({email, password})
                });
                
                if (response.ok) {
                    const result = await response.json();
                    
                    // 保存JWT令牌信息
                    if (result.access_token) {
                        localStorage.setItem('access_token', result.access_token);
                    }
                    if (result.refresh_token) {
                        localStorage.setItem('refresh_token', result.refresh_token);
                    }
                    localStorage.setItem('userEmail', email);
                    localStorage.setItem('currentUser', email); // 兼容性
                    
                    // 设置token过期时间
                    if (result.expires_in) {
                        const expiresAt = Date.now() + (result.expires_in * 1000);
                        localStorage.setItem('token_expires_at', expiresAt.toString());
                    }
                    
                    currentUser = email;
                    hideLoginModal();
                    updateUserBar();
                    init(); // 重新初始化页面
                } else {
                    const error = await response.text();
                    alert('登录失败: ' + error);
                }
            } catch (error) {
                alert('登录失败: ' + error.message);
            }
        }
        
        // 登出处理
        async function logout() {
            try {
                const accessToken = localStorage.getItem('access_token');
                if (accessToken) {
                    await fetch('/api/auth/logout', {
                        method: 'POST',
                        headers: {
                            'Authorization': 'Bearer ' + accessToken,
                            'Content-Type': 'application/json'
                        }
                    });
                }
                clearAuth();
                currentUser = null;
                checkLogin();
            } catch (error) {
                console.error('登出失败:', error);
                clearAuth();
                currentUser = null;
                checkLogin();
            }
        }
        
        // 更新用户状态栏
        function updateUserBar() {
            const userBar = document.getElementById('userBar');
            if (currentUser) {
                userBar.style.display = 'flex';
                document.getElementById('currentUser').textContent = currentUser;
            } else {
                userBar.style.display = 'none';
            }
        }
        
        // 主题切换
        function toggleTheme() {
            document.body.classList.toggle('dark-theme');
            const isDark = document.body.classList.contains('dark-theme');
            localStorage.setItem('theme', isDark ? 'dark' : 'light');
            
            const button = document.getElementById('themeToggle');
            button.textContent = isDark ? '🌞' : '🌙';
        }
        
        // 搜索邮件
        function searchEmails() {
            const query = document.getElementById('searchInput').value.toLowerCase();
            const type = document.getElementById('searchType').value;
            
            if (!query.trim()) {
                clearSearch();
                return;
            }
            
            const mailboxes = document.querySelectorAll('.mailbox');
            filteredEmails = {};
            
            mailboxes.forEach(mailbox => {
                const mailboxName = mailbox.dataset.mailbox;
                const emails = mailbox.querySelectorAll('.email');
                const container = mailbox.querySelector('.emails');
                
                filteredEmails[mailboxName] = [];
                
                emails.forEach(email => {
                    const subject = email.querySelector('.email-subject').textContent.toLowerCase();
                    const from = email.querySelector('.email-meta').textContent.toLowerCase();
                    const body = email.querySelector('.email-body').textContent.toLowerCase();
                    
                    let match = false;
                    switch(type) {
                        case 'from':
                            match = from.includes(query);
                            break;
                        case 'subject':
                            match = subject.includes(query);
                            break;
                        case 'body':
                            match = body.includes(query);
                            break;
                        default:
                            match = subject.includes(query) || from.includes(query) || body.includes(query);
                    }
                    
                    if (match) {
                        email.style.display = 'block';
                        filteredEmails[mailboxName].push(email);
                    } else {
                        email.style.display = 'none';
                    }
                });
                
                // 显示搜索结果统计
                const resultCount = filteredEmails[mailboxName].length;
                let resultInfo = container.querySelector('.search-result-info');
                if (!resultInfo) {
                    resultInfo = document.createElement('div');
                    resultInfo.className = 'search-result-info';
                    resultInfo.style.cssText = 'padding: 10px; background: #e8f4f8; border-radius: 5px; margin: 10px 0; color: #2c3e50;';
                    container.insertBefore(resultInfo, container.firstChild);
                }
                resultInfo.textContent = '搜索结果: ' + resultCount + ' 封邮件';
            });
        }
        
        // 清除搜索
        function clearSearch() {
            document.getElementById('searchInput').value = '';
            const emails = document.querySelectorAll('.email');
            emails.forEach(email => {
                email.style.display = 'block';
            });
            
            // 移除搜索结果信息
            const resultInfos = document.querySelectorAll('.search-result-info');
            resultInfos.forEach(info => info.remove());
            
            filteredEmails = {};
        }
        
        // 展开/收起邮件
        function toggleEmail(emailElement) {
            const preview = emailElement.querySelector('.email-preview');
            const full = emailElement.querySelector('.email-full');
            
            if (full.style.display === 'none' || !full.style.display) {
                full.style.display = 'block';
                preview.style.display = 'none';
                emailElement.classList.add('expanded');
            } else {
                full.style.display = 'none';
                preview.style.display = 'block';
                emailElement.classList.remove('expanded');
            }
        }
        
        // 删除邮件 (模拟功能)
        function deleteEmail(emailElement, emailId) {
            if (confirm('确定要删除这封邮件吗？')) {
                emailElement.style.transition = 'opacity 0.3s ease';
                emailElement.style.opacity = '0';
                setTimeout(() => {
                    emailElement.remove();
                    // 这里应该调用删除API
                    console.log('删除邮件:', emailId);
                }, 300);
            }
        }
        
        // 回复邮件
        function replyEmail(from, subject) {
            document.getElementById('sendTo').value = from;
            document.getElementById('sendSubject').value = 'Re: ' + subject;
            document.getElementById('sendFrom').value = currentUser || '';
            
            // 滚动到发送区域
            document.querySelector('.send-section').scrollIntoView({behavior: 'smooth'});
        }
        
        async function loadMailboxes() {
            try {
                const response = await authenticatedFetch('/api/mailboxes');
                const mailboxes = await response.json();
                const container = document.getElementById('mailboxes');
                container.innerHTML = '';
                
                mailboxStats.total = mailboxes.length;
                mailboxStats.emails = 0;
                
                for (const mailbox of mailboxes) {
                    const div = document.createElement('div');
                    div.className = 'mailbox';
                    
                    // 先创建邮箱结构
                    div.innerHTML = 
                        '<div class="mailbox-header">' +
                            '<div class="mailbox-title">' + mailbox + '</div>' +
                            '<div class="email-count">-</div>' +
                        '</div>' +
                        '<div id="emails-' + mailbox.replace(/[@.]/g, '_') + '">' +
                            '<div class="no-emails">正在加载邮件...</div>' +
                        '</div>';
                    container.appendChild(div);
                    
                    // 然后加载邮件
                    const emails = await loadEmails(mailbox);
                    mailboxStats.emails += emails.length;
                    
                    // 更新邮件数量显示
                    const countElement = div.querySelector('.email-count');
                    if (countElement) {
                        countElement.textContent = emails.length;
                    }
                }
                
                updateStats();
            } catch (error) {
                console.error('加载邮箱失败:', error);
            }
        }
        
        // 邮件缓存
        const emailCache = new Map();
        
        // Base64解码函数
        function decodeBase64IfNeeded(text) {
            if (!text || typeof text !== 'string') return text;
            
            // 检查是否看起来像Base64编码
            const base64Regex = /^[A-Za-z0-9+/]+=*$/;
            if (text.length >= 8 && text.length % 4 === 0 && base64Regex.test(text)) {
                try {
                    const decoded = atob(text);
                    // 检查解码结果是否为有效UTF-8字符串
                    if (decoded && decoded.length > 0) {
                        return decoded;
                    }
                } catch (e) {
                    // 解码失败，返回原文
                }
            }
            return text;
        }
        
        async function loadEmails(mailbox) {
            try {
                // 检查缓存
                const cacheKey = mailbox;
                if (emailCache.has(cacheKey)) {
                    const cached = emailCache.get(cacheKey);
                    if (Date.now() - cached.timestamp < 30000) { // 30秒缓存
                        renderEmails(mailbox, cached.emails);
                        return cached.emails;
                    }
                }
                
                const response = await authenticatedFetch('/api/emails/' + encodeURIComponent(mailbox));
                const emails = await response.json();
                
                // 自动解码Base64内容
                emails.forEach(email => {
                    email.Body = decodeBase64IfNeeded(email.Body);
                    email.Subject = decodeBase64IfNeeded(email.Subject);
                });
                
                // 更新缓存
                emailCache.set(cacheKey, {
                    emails: emails,
                    timestamp: Date.now()
                });
                
                renderEmails(mailbox, emails);
                return emails;
            } catch (error) {
                console.error('加载邮件失败:', error);
                const container = document.getElementById('emails-' + mailbox.replace(/[@.]/g, '_'));
                if (container) {
                    container.innerHTML = '<div class="no-emails">加载邮件失败</div>';
                }
                return [];
            }
        }
        
        function renderEmails(mailbox, emails) {
            const container = document.getElementById('emails-' + mailbox.replace(/[@.]/g, '_'));
            
            if (!container) {
                console.error('找不到邮件容器:', 'emails-' + mailbox.replace(/[@.]/g, '_'));
                return;
            }
            
            container.innerHTML = '';
            
            if (emails.length === 0) {
                container.innerHTML = '<div class="no-emails">暂无邮件</div>';
            } else {
                for (const email of emails.reverse()) {
                    const div = document.createElement('div');
                    div.className = 'email';
                    div.onclick = () => toggleEmail(div);
                    
                    const emailId = email.id || Date.now().toString();
                    const bodyPreview = email.Body.length > 100 ? email.Body.substring(0, 100) + '...' : email.Body;
                        
                    div.innerHTML = 
                        '<div class="email-header">' +
                            '<div class="email-subject">' + (email.Subject || '无主题') + '</div>' +
                                '<div class="email-time">' + new Date(email.Date || email.timestamp).toLocaleString() + '</div>' +
                            '</div>' +
                            '<div class="email-meta">' +
                                '<strong>发件人:</strong> ' + email.From + ' → <strong>收件人:</strong> ' + email.To +
                            '</div>' +
                            '<div class="email-preview">' + bodyPreview.replace(/\n/g, '<br>') + '</div>' +
                            '<div class="email-full">' + email.Body.replace(/\n/g, '<br>') + 
                                '<div class="email-actions">' +
                                    '<button class="email-action-btn reply-btn" onclick="event.stopPropagation(); replyEmail(\'' + email.From + '\', \'' + (email.Subject || '') + '\')">回复</button>' +
                                    '<button class="email-action-btn delete-btn" onclick="event.stopPropagation(); deleteEmail(this.closest(\'.email\'), \'' + emailId + '\')">删除</button>' +
                                    '<button class="email-action-btn archive-btn" onclick="event.stopPropagation(); console.log(\'归档邮件\', \'' + emailId + '\')">归档</button>' +
                                '</div>' +
                            '</div>';
                        container.appendChild(div);
                    }
                }
                
                return emails;
            } catch (error) {
                console.error('加载邮件失败:', error);
                return [];
            }
        }
        
        function updateStats() {
            document.getElementById('total-mailboxes').textContent = mailboxStats.total;
            document.getElementById('total-emails').textContent = mailboxStats.emails;
        }
        
        async function sendEmail() {
            const from = document.getElementById('sendFrom').value;
            const to = document.getElementById('sendTo').value;
            const subject = document.getElementById('sendSubject').value;
            const body = document.getElementById('sendBody').value;
            
            if (!from || !to || !subject) {
                alert('请填写发件人、收件人和主题');
                return;
            }
            
            try {
                const response = await fetch('/api/send', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({from, to, subject, body})
                });
                
                if (response.ok) {
                    alert('邮件发送成功！');
                    // 清空表单
                    document.getElementById('sendFrom').value = '';
                    document.getElementById('sendTo').value = '';
                    document.getElementById('sendSubject').value = '';
                    document.getElementById('sendBody').value = '';
                } else {
                    const error = await response.json();
                    alert('发送失败: ' + (error.error || '未知错误'));
                }
            } catch (error) {
                alert('发送失败: ' + error.message);
            }
        }
        
        async function createMailbox() {
            const username = document.getElementById('newUsername').value;
            const password = document.getElementById('newPassword').value;
            const description = document.getElementById('newDescription').value;
            
            if (!username || !password) {
                alert('请填写用户名和密码');
                return;
            }
            
            // 验证用户名格式
            if (!/^[a-zA-Z0-9._]+$/.test(username) || username.length < 3 || username.length > 20) {
                alert('用户名格式错误：3-20位，只能包含字母、数字、点号、下划线');
                return;
            }
            
            try {
                const response = await fetch('/api/mailboxes/create', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({username, password, description})
                });
                
                if (response.ok) {
                    const result = await response.json();
                    alert('邮箱创建成功：' + result.email);
                    document.getElementById('newUsername').value = '';
                    document.getElementById('newPassword').value = '';
                    document.getElementById('newDescription').value = '';
                    loadMailboxesList();
                    loadMailboxes(); // 刷新主邮箱列表
                } else {
                    const error = await response.text();
                    alert('创建失败: ' + error);
                }
            } catch (error) {
                alert('创建失败: ' + error.message);
            }
        }
        
        async function loadMailboxesList() {
            try {
                const response = await authenticatedFetch('/api/mailboxes/manage');
                const mailboxes = await response.json();
                const container = document.getElementById('mailboxesList');
                
                container.innerHTML = '';
                for (const mailbox of mailboxes) {
                    const div = document.createElement('div');
                    div.style.cssText = 'background: #f8f9fa; padding: 10px; margin: 5px 0; border-radius: 5px; border-left: 4px solid #3498db;';
                    div.innerHTML = 
                        '<div style=\"font-weight: bold; color: #2c3e50;\">' + mailbox.email + '</div>' +
                        '<div style=\"color: #7f8c8d; font-size: 0.9em;\">' + (mailbox.description || '无描述') + '</div>' +
                        '<div style=\"color: #27ae60; font-size: 0.8em;\">状态: ' + (mailbox.is_active ? '激活' : '停用') + '</div>';
                    container.appendChild(div);
                }
                
                if (mailboxes.length === 0) {
                    container.innerHTML = '<div style=\"text-align: center; color: #7f8c8d; padding: 20px;\">暂无自定义邮箱</div>';
                }
            } catch (error) {
                console.error('加载邮箱列表失败:', error);
            }
        }
        
        function refreshData() {
            // 清空邮件缓存
            emailCache.clear();
            loadMailboxes();
        }
        
        // SMTP中继相关函数
        async function loadRelayConfig() {
            try {
                const response = await fetch('/api/relay/config');
                const config = await response.json();
                
                document.getElementById('relayHost').value = config.host || '';
                document.getElementById('relayPort').value = config.port || 587;
                document.getElementById('relayUsername').value = config.username || '';
                document.getElementById('relayUseTLS').checked = config.use_tls;
                document.getElementById('relayEnabled').checked = config.enabled;
                
                // 不自动填充密码，但显示是否已设置
                if (config.has_password) {
                    document.getElementById('relayPassword').placeholder = '密码已设置，留空表示不更改';
                }
                
                loadRelayStatus();
            } catch (error) {
                console.error('加载中继配置失败:', error);
            }
        }
        
        async function loadRelayStatus() {
            try {
                const response = await fetch('/api/relay/status');
                const status = await response.json();
                const statusDiv = document.getElementById('relayStatus');
                
                let statusHtml = '<div style="display: flex; align-items: center; gap: 10px;">';
                statusHtml += '<span><strong>状态:</strong> ' + (status.enabled ? '✅ 已启用' : '❌ 未启用') + '</span>';
                
                if (status.enabled) {
                    statusHtml += '<span><strong>连接:</strong> ' + (status.connection_ok ? '✅ 正常' : '❌ 异常') + '</span>';
                    if (!status.connection_ok && status.connection_error) {
                        statusHtml += '<span style="color: #e74c3c;"><strong>错误:</strong> ' + status.connection_error + '</span>';
                    }
                }
                statusHtml += '</div>';
                
                statusDiv.innerHTML = statusHtml;
                statusDiv.style.background = status.enabled ? (status.connection_ok ? '#d4edda' : '#f8d7da') : '#e2e3e5';
                statusDiv.style.color = status.enabled ? (status.connection_ok ? '#155724' : '#721c24') : '#6c757d';
                statusDiv.style.border = '1px solid ' + (status.enabled ? (status.connection_ok ? '#c3e6cb' : '#f1b0b7') : '#ced4da');
                
            } catch (error) {
                console.error('加载中继状态失败:', error);
            }
        }
        
        function setupRelayProviderChange() {
            document.getElementById('relayProvider').addEventListener('change', async function(e) {
                const provider = e.target.value;
                if (provider) {
                    try {
                        const response = await fetch('/api/relay/providers');
                        const providers = await response.json();
                        const config = providers[provider];
                        
                        if (config) {
                            document.getElementById('relayHost').value = config.host;
                            document.getElementById('relayPort').value = config.port;
                            document.getElementById('relayUseTLS').checked = config.use_tls;
                        }
                    } catch (error) {
                        console.error('加载提供商配置失败:', error);
                    }
                }
            });
        }
        
        async function saveRelayConfig() {
            const config = {
                enabled: document.getElementById('relayEnabled').checked,
                host: document.getElementById('relayHost').value,
                port: parseInt(document.getElementById('relayPort').value),
                username: document.getElementById('relayUsername').value,
                password: document.getElementById('relayPassword').value,
                use_tls: document.getElementById('relayUseTLS').checked
            };
            
            if (!config.host || !config.username) {
                alert('请填写SMTP主机和用户名');
                return;
            }
            
            try {
                const response = await fetch('/api/relay/config', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(config)
                });
                
                if (response.ok) {
                    alert('SMTP中继配置保存成功！');
                    loadRelayConfig();
                } else {
                    const error = await response.text();
                    alert('保存失败: ' + error);
                }
            } catch (error) {
                alert('保存失败: ' + error.message);
            }
        }
        
        async function testRelayConnection() {
            const resultDiv = document.getElementById('relayTestResult');
            resultDiv.innerHTML = '<div style="color: #666;">正在测试连接...</div>';
            
            try {
                const response = await fetch('/api/relay/test', {
                    method: 'POST'
                });
                
                const result = await response.json();
                
                if (result.success) {
                    resultDiv.innerHTML = '<div style="color: #27ae60; font-weight: bold;">✅ ' + result.message + '</div>';
                } else {
                    resultDiv.innerHTML = '<div style="color: #e74c3c; font-weight: bold;">❌ 连接失败: ' + result.error + '</div>';
                }
                
                // 3秒后清除结果
                setTimeout(() => {
                    resultDiv.innerHTML = '';
                }, 3000);
                
            } catch (error) {
                resultDiv.innerHTML = '<div style="color: #e74c3c; font-weight: bold;">❌ 测试失败: ' + error.message + '</div>';
            }
        }
        
        async function disableRelay() {
            if (!confirm('确定要禁用SMTP中继吗？')) {
                return;
            }
            
            try {
                const response = await fetch('/api/relay/config', {
                    method: 'DELETE'
                });
                
                if (response.ok) {
                    alert('SMTP中继已禁用');
                    loadRelayConfig();
                } else {
                    const error = await response.text();
                    alert('禁用失败: ' + error);
                }
            } catch (error) {
                alert('禁用失败: ' + error.message);
            }
        }
        
        // 初始化函数
        function init() {
            if (!checkLogin()) return;
            
            // 初始化主题
            const savedTheme = localStorage.getItem('theme');
            if (savedTheme === 'dark') {
                document.body.classList.add('dark-theme');
                document.getElementById('themeToggle').textContent = '🌞';
            }
            
            loadMailboxes();
            loadMailboxesList();
            loadRelayConfig();
            setupRelayProviderChange();
        }
        
        window.onload = function() {
            init();
        };
        setInterval(loadMailboxes, 10000);
        setInterval(loadMailboxesList, 30000);
        setInterval(loadRelayStatus, 30000);
    </script>
</head>
<body>
    <!-- 登录模态框 -->
    <div id="loginModal" class="login-modal">
        <div class="login-content">
            <h2>🔐 登录邮箱管理系统</h2>
            <div class="login-form">
                <input type="email" id="loginEmail" placeholder="管理员邮箱" autocomplete="username">
                <input type="password" id="loginPassword" placeholder="密码" autocomplete="current-password">
                <button class="login-btn" onclick="login()">登录</button>
            </div>
        </div>
    </div>

    <div class="container">
        <!-- 用户状态栏 -->
        <div id="userBar" class="user-bar" style="display: none;">
            <div>
                <span>👤 当前用户: <strong id="currentUser"></strong></span>
            </div>
            <div>
                <button id="themeToggle" class="theme-toggle" onclick="toggleTheme()">🌙</button>
                <button class="logout-btn" onclick="logout()">登出</button>
            </div>
        </div>
        
        <div class="header">
            <h1>🎮 FreeAgent 邮箱管理系统</h1>
            <div class="subtitle">基于 ` + ms.domain + ` 域名的专业邮箱服务</div>
            <div class="stats">
                <div class="stat-item">
                    <div class="stat-number" id="total-mailboxes">0</div>
                    <div class="stat-label">活跃邮箱</div>
                </div>
                <div class="stat-item">
                    <div class="stat-number" id="total-emails">0</div>
                    <div class="stat-label">总邮件数</div>
                </div>
                <div class="stat-item">
                    <div class="stat-number">∞</div>
                    <div class="stat-label">可创建别名</div>
                </div>
            </div>
        </div>
        
        <div class="controls">
            <h3>📋 系统信息</h3>
            <p><strong>SMTP服务器:</strong> ` + ms.hostname + `:25</p>
            <p><strong>IMAP服务器:</strong> ` + ms.hostname + `:143</p>
            <p><strong>域名:</strong> ` + ms.domain + `</p>
            <p><strong>别名支持:</strong> 任何 @` + ms.domain + ` 的邮件都会被自动接收</p>
            <p><strong>特性:</strong> 无限邮箱别名，邮件发送，IMAP支持，别名管理</p>
            
            <div style="margin-top: 20px;">
                <h4>📤 发送邮件</h4>
                <div class="email-form">
                    <input type="email" id="sendFrom" placeholder="发件人邮箱" style="min-width: 200px;">
                    <input type="email" id="sendTo" placeholder="收件人邮箱" style="min-width: 200px;">
                </div>
                <div class="email-form">
                    <input type="text" id="sendSubject" placeholder="邮件主题" style="min-width: 400px;">
                    <button class="btn btn-primary" onclick="sendEmail()">发送邮件</button>
                </div>
                <div style="margin-top: 10px;">
                    <textarea id="sendBody" placeholder="邮件内容" style="width: 100%; height: 100px; padding: 10px; border: 2px solid #e0e0e0; border-radius: 8px;"></textarea>
                </div>
            </div>
            
            <div style="margin-top: 20px;">
                <h4>📮 创建邮箱</h4>
                <div class="email-form">
                    <input type="text" id="newUsername" placeholder="用户名" style="min-width: 150px;" maxlength="20">
                    <span style="display: flex; align-items: center; color: #666;">@` + ms.domain + `</span>
                    <input type="password" id="newPassword" placeholder="密码" style="min-width: 150px;">
                    <button class="btn btn-success" onclick="createMailbox()">创建邮箱</button>
                </div>
                <div style="margin-top: 10px;">
                    <input type="text" id="newDescription" placeholder="描述信息（可选）" style="width: 100%; padding: 10px; border: 2px solid #e0e0e0; border-radius: 8px;">
                </div>
                <div id="mailboxesList" style="margin-top: 15px; max-height: 200px; overflow-y: auto;"></div>
            </div>
            
            <div style="margin-top: 20px;">
                <h4>🔗 SMTP中继配置</h4>
                <div id="relayStatus" style="margin-bottom: 15px; padding: 10px; border-radius: 8px;"></div>
                
                <div style="margin-bottom: 15px;">
                    <label style="display: block; margin-bottom: 5px; font-weight: bold;">快速配置:</label>
                    <select id="relayProvider" style="width: 250px; padding: 8px; border: 2px solid #e0e0e0; border-radius: 8px; margin-right: 10px;">
                        <option value="">选择提供商</option>
                        <optgroup label="亚马逊SES">
                            <option value="amazon_ses_us_east_1">美国东部 (us-east-1)</option>
                            <option value="amazon_ses_us_west_2">美国西部 (us-west-2)</option>
                            <option value="amazon_ses_eu_west_1">欧洲西部 (eu-west-1)</option>
                            <option value="amazon_ses_ap_southeast_1">亚太东南 (ap-southeast-1)</option>
                        </optgroup>
                        <optgroup label="腾讯云">
                            <option value="tencent_ses">腾讯云邮件推送</option>
                            <option value="tencent_exmail">腾讯企业邮箱</option>
                        </optgroup>
                        <optgroup label="其他邮箱">
                            <option value="qq">QQ邮箱</option>
                            <option value="163">网易163邮箱</option>
                            <option value="126">网易126邮箱</option>
                            <option value="gmail">Gmail</option>
                        </optgroup>
                    </select>
                </div>
                
                <div class="email-form">
                    <input type="text" id="relayHost" placeholder="SMTP主机" style="min-width: 200px;">
                    <input type="number" id="relayPort" placeholder="端口" style="min-width: 100px;" value="587">
                    <input type="text" id="relayUsername" placeholder="用户名" style="min-width: 200px;">
                    <input type="password" id="relayPassword" placeholder="密码" style="min-width: 200px;">
                </div>
                
                <div style="margin: 10px 0;">
                    <label style="display: flex; align-items: center;">
                        <input type="checkbox" id="relayUseTLS" checked style="margin-right: 5px;">
                        使用TLS加密
                    </label>
                    <label style="display: flex; align-items: center; margin-top: 5px;">
                        <input type="checkbox" id="relayEnabled" style="margin-right: 5px;">
                        启用SMTP中继
                    </label>
                </div>
                
                <div class="email-form">
                    <button class="btn btn-primary" onclick="saveRelayConfig()">保存配置</button>
                    <button class="btn btn-primary" onclick="testRelayConnection()">测试连接</button>
                    <button class="btn btn-danger" onclick="disableRelay()">禁用中继</button>
                </div>
                
                <div id="relayTestResult" style="margin-top: 10px;"></div>
            </div>
        </div>
        
        <!-- 搜索栏 -->
        <div class="search-section">
            <h3>🔍 邮件搜索</h3>
            <div class="search-controls">
                <input type="text" id="searchInput" class="search-input" placeholder="搜索邮件..." oninput="searchEmails()">
                <select id="searchType" class="search-select" onchange="searchEmails()">
                    <option value="all">全部内容</option>
                    <option value="from">发件人</option>
                    <option value="subject">主题</option>
                    <option value="body">正文</option>
                </select>
                <button class="clear-search-btn" onclick="clearSearch()">清除</button>
            </div>
        </div>
        
        <div class="mailboxes-grid" id="mailboxes">
            <div style="text-align: center; padding: 50px; color: #7f8c8d;">
                正在加载邮箱数据...
            </div>
        </div>
    </div>
    
    <button class="refresh-btn" onclick="refreshData()" title="刷新数据">
        🔄
    </button>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}


func (ms *MailServer) debugHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `<!DOCTYPE html>
<html>
<head>
    <title>邮件显示调试</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .mailbox { border: 1px solid #ccc; margin: 10px 0; padding: 10px; }
        .email { background: #f5f5f5; margin: 5px 0; padding: 10px; border-radius: 5px; }
        .email-subject { font-weight: bold; margin-bottom: 5px; }
        .email-meta { color: #666; font-size: 0.9em; margin-bottom: 5px; }
        .email-body { margin-top: 5px; }
    </style>
</head>
<body>
    <h1>邮件显示调试</h1>
    <div id="debug-output"></div>
    
    <script>
        async function debugEmailDisplay() {
            const output = document.getElementById('debug-output');
            
            try {
                output.innerHTML += '<h3>1. 测试API连接</h3>';
                
                const mailboxResponse = await fetch('/api/mailboxes');
                const mailboxes = await mailboxResponse.json();
                output.innerHTML += '<p>邮箱列表: ' + JSON.stringify(mailboxes) + '</p>';
                
                for (const mailbox of mailboxes) {
                    output.innerHTML += '<h3>2. 测试邮箱: ' + mailbox + '</h3>';
                    
                    const emailResponse = await fetch('/api/emails/' + encodeURIComponent(mailbox));
                    const emails = await emailResponse.json();
                    output.innerHTML += '<p>邮件数据: ' + JSON.stringify(emails) + '</p>';
                    
                    if (emails.length > 0) {
                        output.innerHTML += '<h3>3. 渲染邮件</h3>';
                        
                        const mailboxDiv = document.createElement('div');
                        mailboxDiv.className = 'mailbox';
                        mailboxDiv.innerHTML = '<h4>邮箱: ' + mailbox + ' (' + emails.length + ' 封邮件)</h4>';
                        
                        for (const email of emails) {
                            const emailDiv = document.createElement('div');
                            emailDiv.className = 'email';
                            emailDiv.innerHTML = 
                                '<div class="email-subject">主题: ' + (email.Subject || '无主题') + '</div>' +
                                '<div class="email-meta">发件人: ' + email.From + ' | 收件人: ' + email.To + '</div>' +
                                '<div class="email-body">内容: ' + email.Body.replace(/\\n/g, '<br>') + '</div>';
                            mailboxDiv.appendChild(emailDiv);
                        }
                        
                        output.appendChild(mailboxDiv);
                    }
                }
                
            } catch (error) {
                output.innerHTML += '<p style="color: red;">错误: ' + error.message + '</p>';
                console.error('调试错误:', error);
            }
        }
        
        window.onload = debugEmailDisplay;
    </script>
</body>
</html>`
	w.Write([]byte(html))
}

func (ms *MailServer) apiMailboxes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// 获取当前用户
	userEmail := ms.getUserFromRequest(r)
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// 根据用户权限获取邮箱列表
	var mailboxes []string
	if ms.isAdminRequest(r) {
		// 管理员可以看到所有邮箱
		mailboxes = ms.GetAllMailboxes()
	} else {
		// 普通用户只能看到自己的邮箱
		userMailboxes := ms.getUserMailboxes(userEmail)
		for _, mailbox := range userMailboxes {
			mailboxes = append(mailboxes, mailbox.Email)
		}
		// 如果用户没有关联的邮箱，至少显示自己的邮箱
		if len(mailboxes) == 0 {
			mailboxes = append(mailboxes, userEmail)
		}
	}
	
	response := "["
	for i, mailbox := range mailboxes {
		if i > 0 {
			response += ","
		}
		response += `"` + mailbox + `"`
	}
	response += "]"
	
	w.Write([]byte(response))
}

func (ms *MailServer) apiEmails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	mailbox := strings.TrimPrefix(r.URL.Path, "/api/emails/")
	emails := ms.GetEmails(mailbox)
	
	// 转换为可JSON序列化的结构，并解码邮件内容
	var apiEmails []map[string]interface{}
	for _, email := range emails {
		// 解码邮件正文（如果是base64编码）
		decodedBody := decodeEmailBodyIfNeeded(email.Body)
		decodedSubject := decodeEmailBodyIfNeeded(email.Subject)
		
		apiEmail := map[string]interface{}{
			"From":        email.From,
			"To":          email.To,
			"Subject":     decodedSubject,
			"Body":        decodedBody,
			"HTMLBody":    email.HTMLBody,
			"Date":        email.Date,
			"ID":          email.ID,
			"Attachments": email.Attachments,
			"Signature":   email.Signature,
			"IsAutoReply": email.IsAutoReply,
			"Charset":     email.Charset,
			"Headers":     email.Headers,
			"Embedded":    email.Embedded,
		}
		
		apiEmails = append(apiEmails, apiEmail)
	}
	
	// 使用标准JSON编码
	jsonData, err := json.Marshal(apiEmails)
	if err != nil {
		http.Error(w, "JSON编码失败", http.StatusInternalServerError)
		return
	}
	
	w.Write(jsonData)
}

func (ms *MailServer) apiDeleteEmail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	// 处理OPTIONS预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// 解析URL: /api/emails/delete/mailbox/emailID
	path := strings.TrimPrefix(r.URL.Path, "/api/emails/delete/")
	parts := strings.SplitN(path, "/", 2)
	
	if len(parts) != 2 {
		http.Error(w, "Invalid URL format. Expected: /api/emails/delete/mailbox/emailID", http.StatusBadRequest)
		return
	}
	
	mailbox := parts[0]
	emailID := parts[1]
	
	// 删除邮件
	success := ms.DeleteEmail(mailbox, emailID)
	
	if success {
		response := map[string]interface{}{
			"success": true,
			"message": "Email deleted successfully",
		}
		jsonData, _ := json.Marshal(response)
		w.Write(jsonData)
	} else {
		http.Error(w, "Email not found or deletion failed", http.StatusNotFound)
	}
}

// decodeEmailBodyIfNeeded 检查并解码邮件正文（如果是base64编码或RFC 2047编码）
func decodeEmailBodyIfNeeded(body string) string {
	trimmedBody := strings.TrimSpace(body)
	
	// 处理RFC 2047编码格式: =?charset?encoding?encoded-text?=
	if strings.Contains(trimmedBody, "=?") && strings.Contains(trimmedBody, "?=") {
		return decodeRFC2047(trimmedBody)
	}
	
	// 使用email_parser.go中的isLikelyBase64函数检查
	if isLikelyBase64(trimmedBody) {
		if decoded, err := base64.StdEncoding.DecodeString(trimmedBody); err == nil {
			if utf8.Valid(decoded) {
				return string(decoded)
			}
		}
	}
	
	return body
}

// decodeRFC2047 解码RFC 2047格式的邮件头部
func decodeRFC2047(text string) string {
	// 匹配 =?charset?encoding?encoded-text?= 格式
	parts := strings.Split(text, "=?")
	result := parts[0] // 第一部分保持原样
	
	for i := 1; i < len(parts); i++ {
		part := parts[i]
		endIndex := strings.Index(part, "?=")
		if endIndex == -1 {
			result += "=?" + part
			continue
		}
		
		encodedPart := part[:endIndex]
		remainingPart := part[endIndex+2:]
		
		// 解析编码部分: charset?encoding?encoded-text
		sections := strings.Split(encodedPart, "?")
		if len(sections) >= 3 {
			// charset := sections[0] // 暂时不需要使用charset
			encoding := strings.ToLower(sections[1])
			encodedText := sections[2]
			
			var decoded string
			if encoding == "b" {
				// Base64编码
				if decodedBytes, err := base64.StdEncoding.DecodeString(encodedText); err == nil {
					if utf8.Valid(decodedBytes) {
						decoded = string(decodedBytes)
					} else {
						decoded = encodedPart // 解码失败，保持原样
					}
				} else {
					decoded = encodedPart
				}
			} else if encoding == "q" {
				// Quoted-printable编码
				decoded = decodeQuotedPrintableRFC2047(encodedText)
			} else {
				decoded = encodedPart // 未知编码，保持原样
			}
			
			result += decoded + remainingPart
		} else {
			result += "=?" + part
		}
	}
	
	return result
}

// decodeQuotedPrintableRFC2047 解码RFC 2047的Quoted-Printable编码
func decodeQuotedPrintableRFC2047(text string) string {
	// 替换下划线为空格（RFC 2047特定）
	text = strings.ReplaceAll(text, "_", " ")
	
	// 处理=XX十六进制编码
	result := ""
	for i := 0; i < len(text); i++ {
		if text[i] == '=' && i+2 < len(text) {
			hex := text[i+1 : i+3]
			if len(hex) == 2 {
				var b byte
				if _, err := fmt.Sscanf(hex, "%02X", &b); err == nil {
					result += string(b)
					i += 2
					continue
				}
			}
		}
		result += string(text[i])
	}
	
	return result
}

// generateEmailID 生成唯一的邮件ID
func generateEmailID(mailbox string) string {
	// 使用当前时间戳和随机数生成ID
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	
	return fmt.Sprintf("%s_%d_%x", 
		strings.ReplaceAll(mailbox, "@", "_at_"), 
		timestamp, 
		randomBytes)
}

// DeleteEmail 删除指定邮件
func (ms *MailServer) DeleteEmail(mailbox, emailID string) bool {
	return ms.storage.DeleteEmail(mailbox, emailID)
}

func main() {
	domain := "freeagent.live"
	hostname := "localhost"
	if len(os.Args) > 1 {
		domain = os.Args[1]
	}
	if len(os.Args) > 2 {
		hostname = os.Args[2]
	}
	
	// 获取服务器监听地址
	smtpPort := "2525"
	imapPort := "1143"
	webPort := "8080"
	if len(os.Args) > 3 {
		smtpPort = os.Args[3]
	}
	if len(os.Args) > 4 {
		imapPort = os.Args[4]
	}
	if len(os.Args) > 5 {
		webPort = os.Args[5]
	}
	
	// 检查是否为生产环境（端口25）
	isProduction := smtpPort == "25"
	
	mailServer := NewMailServer(domain, hostname)
	
	// 启动SMTP服务器
	go mailServer.StartSMTPServer(smtpPort)
	
	// 启动IMAP服务器
	go mailServer.imapServer.StartIMAPServer(imapPort)
	
	fmt.Printf("===============================================\n")
	fmt.Printf("🎮 YgoCard 全功能邮箱服务器启动完成!\n")
	fmt.Printf("===============================================\n")
	fmt.Printf("域名: %s\n", domain)
	fmt.Printf("主机名: %s\n", hostname)
	fmt.Printf("SMTP服务器: 0.0.0.0:%s (接收邮件)\n", smtpPort)
	fmt.Printf("IMAP服务器: 0.0.0.0:%s (客户端访问)\n", imapPort)
	fmt.Printf("Web界面: http://0.0.0.0:%s\n", webPort)
	fmt.Printf("支持功能: SMTP接收/发送、IMAP访问、邮箱管理、自定义用户名\n")
	fmt.Printf("默认管理员: admin@%s / 密码: admin123\n", domain)
	fmt.Printf("===============================================\n")
	
	if isProduction {
		fmt.Printf("⚠️  生产环境模式 (端口25)\n")
		fmt.Printf("请确保已配置以下DNS记录:\n")
		fmt.Printf("1. A记录: mail.%s -> 服务器IP\n", domain)
		fmt.Printf("2. MX记录: %s -> mail.%s (优先级10)\n", domain, domain)
		fmt.Printf("3. TXT记录: %s -> \"v=spf1 a mx ~all\"\n", domain)
		fmt.Printf("防火墙端口: 25(SMTP), 143(IMAP), %s(Web)\n", webPort)
		fmt.Printf("===============================================\n")
	} else {
		fmt.Printf("🧪 开发/测试模式 (端口%s)\n", smtpPort)
		fmt.Printf("要启用真实域名邮件接收，请:\n")
		fmt.Printf("1. 使用 sudo 权限: sudo go run main.go %s %s 25 143 %s\n", domain, hostname, webPort)
		fmt.Printf("2. 配置防火墙开放端口: 25(SMTP), 143(IMAP), %s(Web)\n", webPort)
		fmt.Printf("3. 配置DNS MX记录指向此服务器\n")
		fmt.Printf("邮件客户端配置:\n")
		fmt.Printf("  IMAP: %s:%s, 用户名: 任意@%s, 密码: 任意\n", hostname, imapPort, domain)
		fmt.Printf("  SMTP: %s:%s\n", hostname, smtpPort)
		fmt.Printf("===============================================\n")
	}
	
	// 启动Web服务器
	mailServer.StartWebServer(webPort)
}

// apiSearch 处理邮件搜索请求
func (ms *MailServer) apiSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	if r.Method == "OPTIONS" {
		return
	}
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// 解析搜索请求
	var searchReq SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// 设置默认值
	if searchReq.Size == 0 {
		searchReq.Size = 20
	}
	if searchReq.Size > 100 {
		searchReq.Size = 100
	}
	
	// 检查ElasticSearch是否可用 (临时禁用)
	// if !ms.esClient.IsEnabled() {
		// 回退到简单搜索
		result := ms.fallbackSearch(searchReq)
		jsonData, _ := json.Marshal(result)
		w.Write(jsonData)
		return
	// }
	
	// 使用ElasticSearch搜索 (临时禁用)
	// result, err := ms.esClient.SearchEmails(searchReq)
	// if err != nil {
	//	log.Printf("搜索失败: %v", err)
	//	// 回退到简单搜索
	//	result = ms.fallbackSearch(searchReq)
	// }
	
	jsonData, err := json.Marshal(result)
	if err != nil {
		log.Printf("序列化搜索结果失败: %v", err)
		http.Error(w, "内部错误", http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)
}

// apiSearchSuggestions 处理搜索建议请求
func (ms *MailServer) apiSearchSuggestions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == "OPTIONS" {
		return
	}
	
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	query := r.URL.Query().Get("q")
	if query == "" {
		jsonData, _ := json.Marshal(map[string][]string{"suggestions": {}})
		w.Write(jsonData)
		return
	}
	
	// 生成搜索建议
	suggestions := ms.generateSearchSuggestions(query)
	
	response := map[string]interface{}{
		"suggestions": suggestions,
		"query":       query,
	}
	
	jsonData, _ := json.Marshal(response)
	w.Write(jsonData)
}

// fallbackSearch 简单搜索实现（当ElasticSearch不可用时）
func (ms *MailServer) fallbackSearch(searchReq SearchRequest) *SearchResult {
	result := &SearchResult{
		Emails: []EmailDocument{},
		Total:  0,
		Took:   0,
	}
	
	// 获取所有邮箱
	mailboxes := ms.GetAllMailboxes()
	
	// 如果指定了邮箱，只搜索该邮箱
	if searchReq.Mailbox != "" {
		mailboxes = []string{searchReq.Mailbox}
	}
	
	startTime := time.Now()
	var allEmails []EmailDocument
	
	for _, mailbox := range mailboxes {
		emails := ms.GetEmails(mailbox)
		for _, email := range emails {
			// 简单的文本匹配
			if ms.matchesSearch(email, searchReq) {
				doc := EmailDocument{
					ID:        email.ID,
					Mailbox:   mailbox,
					From:      email.From,
					To:        email.To,
					Subject:   email.Subject,
					Body:      email.Body,
					Timestamp: parseEmailDate(email.Date),
				}
				allEmails = append(allEmails, doc)
			}
		}
	}
	
	// 分页
	start := searchReq.From_
	end := start + searchReq.Size
	if start > len(allEmails) {
		start = len(allEmails)
	}
	if end > len(allEmails) {
		end = len(allEmails)
	}
	
	result.Total = int64(len(allEmails))
	result.Emails = allEmails[start:end]
	result.Took = int(time.Since(startTime).Milliseconds())
	
	return result
}

// matchesSearch 检查邮件是否匹配搜索条件
func (ms *MailServer) matchesSearch(email Email, searchReq SearchRequest) bool {
	// 主搜索查询
	if searchReq.Query != "" {
		query := strings.ToLower(searchReq.Query)
		content := strings.ToLower(email.Subject + " " + email.Body + " " + email.From + " " + email.To)
		if !strings.Contains(content, query) {
			return false
		}
	}
	
	// 发件人筛选
	if searchReq.From != "" {
		if !strings.Contains(strings.ToLower(email.From), strings.ToLower(searchReq.From)) {
			return false
		}
	}
	
	// 收件人筛选
	if searchReq.To != "" {
		if !strings.Contains(strings.ToLower(email.To), strings.ToLower(searchReq.To)) {
			return false
		}
	}
	
	// 主题筛选
	if searchReq.Subject != "" {
		if !strings.Contains(strings.ToLower(email.Subject), strings.ToLower(searchReq.Subject)) {
			return false
		}
	}
	
	// 日期筛选
	if searchReq.DateStart != "" || searchReq.DateEnd != "" {
		emailDate := parseEmailDate(email.Date)
		
		if searchReq.DateStart != "" {
			startDate, _ := time.Parse("2006-01-02", searchReq.DateStart)
			if emailDate.Before(startDate) {
				return false
			}
		}
		
		if searchReq.DateEnd != "" {
			endDate, _ := time.Parse("2006-01-02", searchReq.DateEnd)
			if emailDate.After(endDate.AddDate(0, 0, 1)) {
				return false
			}
		}
	}
	
	return true
}

// generateSearchSuggestions 生成搜索建议
func (ms *MailServer) generateSearchSuggestions(query string) []string {
	suggestions := []string{}
	query = strings.ToLower(query)
	
	// 常用搜索建议
	commonSuggestions := []string{
		"from:",
		"to:",
		"subject:",
		"has:attachment",
		"is:unread",
		"is:read",
		"priority:high",
		"size:>1MB",
		"date:today",
		"date:yesterday",
		"date:week",
		"date:month",
	}
	
	// 匹配常用建议
	for _, suggestion := range commonSuggestions {
		if strings.HasPrefix(suggestion, query) {
			suggestions = append(suggestions, suggestion)
		}
	}
	
	// 限制建议数量
	if len(suggestions) > 10 {
		suggestions = suggestions[:10]
	}
	
	return suggestions
}

// parseEmailDate 解析邮件日期
func parseEmailDate(dateStr string) time.Time {
	// 尝试多种日期格式
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}
	
	return time.Now()
}

// indexEmailToElastic 将邮件索引到ElasticSearch (临时禁用)
func (ms *MailServer) indexEmailToElastic(mailbox string, email Email) {
	// if !ms.esClient.IsEnabled() {
		return
	// }
	
	_ = EmailDocument{
		ID:        email.ID,
		Mailbox:   mailbox,
		From:      email.From,
		To:        email.To,
		Subject:   email.Subject,
		Body:      email.Body,
		Timestamp: parseEmailDate(email.Date),
		// 这些字段需要从邮件内容中提取
		HasAttachment: strings.Contains(email.Body, "Content-Disposition: attachment"),
		IsRead:        false, // 新邮件默认未读
		Priority:      "normal",
		Size:          int64(len(email.Body)),
		Tags:          []string{},
		Headers:       map[string]string{},
	}
	
	// if err := ms.esClient.IndexEmail(doc); err != nil {
	//	log.Printf("索引邮件到ElasticSearch失败: %v", err)
	// }
}

// JWT认证API处理函数
func (ms *MailServer) apiAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	var req struct {
		Email          string `json:"email"`
		Password       string `json:"password"`
		TwoFactorCode  string `json:"two_factor_code"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// 使用数据库验证邮箱凭据
	if !ms.database.ValidateMailboxCredentials(req.Email, req.Password) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid credentials"})
		return
	}
	
	// 检查用户是否存在，如果不存在则创建
	_, err := ms.database.GetUser(req.Email)
	if err != nil {
		// 用户不存在，创建默认用户记录
		isAdmin := req.Email == "admin@"+ms.domain
		err = ms.database.CreateUser(req.Email, req.Password, isAdmin)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to create user"})
			return
		}
	}
	
	// 获取用户信息
	user, err := ms.database.GetUser(req.Email)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to get user info"})
		return
	}
	
	// 检查是否启用了2FA
	if user.TwoFactorEnabled {
		// 如果启用了2FA但没有提供2FA代码，要求提供2FA代码
		if req.TwoFactorCode == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "2fa_required",
				"message": "Two-factor authentication code required",
				"requires_2fa": true,
			})
			return
		}
		
		// 验证2FA代码
		if !ms.userAuth.Verify2FA(req.Email, req.TwoFactorCode) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid 2FA code"})
			return
		}
	}
	
	// 生成令牌 - 确保万无一失
	var response map[string]interface{}
	
	// 尝试生成JWT令牌
	if tokens, err := ms.userAuth.GenerateJWTTokenWithAdmin(req.Email, user.IsAdmin); err == nil {
		// JWT成功
		response = map[string]interface{}{
			"token_type":   "Bearer",
			"user_email":   req.Email,
		}
		for k, v := range tokens {
			response[k] = v
		}
	} else {
		// JWT失败，使用base64备用方案
		log.Printf("JWT生成失败，使用备用方案 for %s: %v", req.Email, err)
		accessToken := base64.StdEncoding.EncodeToString([]byte(req.Email + ":" + time.Now().Format(time.RFC3339)))
		response = map[string]interface{}{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"expires_in":   86400,
			"user_email":   req.Email,
		}
	}
	
	json.NewEncoder(w).Encode(response)
}

func (ms *MailServer) apiAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	// 从Header中获取Authorization token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "No token provided", http.StatusUnauthorized)
		return
	}
	
	// 解析Bearer token
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		http.Error(w, "Invalid token format", http.StatusUnauthorized)
		return
	}
	
	// 将token加入黑名单
	err := ms.userAuth.RevokeJWTToken(tokenString)
	if err != nil {
		http.Error(w, "Token revocation failed", http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
}

func (ms *MailServer) apiAuthRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// 刷新JWT令牌
	tokens, err := ms.userAuth.RefreshJWTToken(req.RefreshToken)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Token refresh failed"})
		return
	}
	
	json.NewEncoder(w).Encode(tokens)
}

func (ms *MailServer) apiAuth2FAEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	// 获取用户邮箱地址
	email := ms.getUserFromRequest(r)
	if email == "" {
		// 如果无法从token获取用户，尝试从查询参数获取（兼容性）
		email = r.URL.Query().Get("email")
		if email == "" {
			http.Error(w, "Invalid token or missing email", http.StatusUnauthorized)
			return
		}
	}
	
	log.Printf("Attempting to enable 2FA for user: %s", email)
	
	// 启用2FA
	secret, err := ms.userAuth.Enable2FA(email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// 同步到数据库
	err = ms.database.Update2FA(email, true, secret)
	if err != nil {
		log.Printf("Failed to update 2FA status in database: %v", err)
		// 不返回错误，因为内存中已经更新了
	}
	
	json.NewEncoder(w).Encode(map[string]string{
		"secret":  secret,
		"qr_code": "Generate QR code with secret: " + secret,
	})
}

func (ms *MailServer) apiAuth2FADisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	// 获取用户邮箱地址
	email := ms.getUserFromRequest(r)
	if email == "" {
		// 如果无法从token获取用户，尝试从查询参数获取（兼容性）
		email = r.URL.Query().Get("email")
		if email == "" {
			http.Error(w, "Invalid token or missing email", http.StatusUnauthorized)
			return
		}
	}
	
	// 禁用2FA
	err := ms.userAuth.Disable2FA(email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// 同步到数据库
	err = ms.database.Update2FA(email, false, "")
	if err != nil {
		log.Printf("Failed to update 2FA status in database: %v", err)
		// 不返回错误，因为内存中已经更新了
	}
	
	json.NewEncoder(w).Encode(map[string]string{"message": "2FA disabled successfully"})
}

func (ms *MailServer) apiAuth2FAVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// 验证2FA代码
	valid := ms.userAuth.Verify2FA(req.Email, req.Code)
	
	json.NewEncoder(w).Encode(map[string]bool{"valid": valid})
}

// apiAuth2FAStatus 获取用户2FA状态
func (ms *MailServer) apiAuth2FAStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	// 获取用户邮箱地址
	userEmail := ms.getUserFromRequest(r)
	if userEmail == "" {
		// 如果无法从token获取用户，尝试从查询参数获取（兼容性）
		userEmail = r.URL.Query().Get("email")
		if userEmail == "" {
			http.Error(w, "Invalid token or missing email", http.StatusUnauthorized)
			return
		}
	}
	
	// 从数据库获取用户2FA状态
	user, err := ms.database.GetUser(userEmail)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	
	// 返回2FA状态
	response := map[string]bool{
		"enabled": user.TwoFactorEnabled,
	}
	
	json.NewEncoder(w).Encode(response)
}

// parseEmailContentEnhanced 增强版邮件解析器
func parseEmailContentEnhanced(rawContent string) *EmailContent {
	// 使用Go标准库解析邮件
	msg, err := mail.ReadMessage(strings.NewReader(rawContent))
	if err != nil {
		// 如果标准库解析失败，使用备用解析器
		return parseEmailFallbackEnhanced(rawContent)
	}
	
	emailContent := &EmailContent{
		Headers: make(map[string]string),
	}
	
	// 解析邮件头部
	parseEmailHeaders(msg.Header, emailContent)
	
	// 解析邮件正文和附件
	parseEmailBodyEnhanced(msg, emailContent)
	
	// 检测签名和自动回复
	detectSignatureAndAutoReply(emailContent)
	
	return emailContent
}

// parseEmailHeaders 解析邮件头部
func parseEmailHeaders(header mail.Header, emailContent *EmailContent) {
	// 解析主题
	if subjectHeader := header.Get("Subject"); subjectHeader != "" {
		emailContent.Subject = subjectHeader
	}
	
	// 解析日期
	if dateHeader := header.Get("Date"); dateHeader != "" {
		emailContent.Date = dateHeader
	} else {
		emailContent.Date = time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")
	}
	
	// 解析发件人
	if fromHeader := header.Get("From"); fromHeader != "" {
		emailContent.From = fromHeader
	}
	
	// 解析收件人
	if toHeader := header.Get("To"); toHeader != "" {
		emailContent.To = []string{toHeader}
	}
	
	// 解析字符编码
	if contentType := header.Get("Content-Type"); contentType != "" {
		emailContent.Charset = extractCharset(contentType)
	}
	
	// 存储所有头部
	for key, values := range header {
		if len(values) > 0 {
			emailContent.Headers[key] = values[0]
		}
	}
}

// parseEmailBodyEnhanced 增强版邮件正文解析
func parseEmailBodyEnhanced(msg *mail.Message, emailContent *EmailContent) {
	// 读取原始正文
	bodyBytes := make([]byte, 0, 1024*1024) // 1MB缓冲
	buffer := make([]byte, 1024)
	
	for {
		n, err := msg.Body.Read(buffer)
		if n > 0 {
			bodyBytes = append(bodyBytes, buffer[:n]...)
		}
		if err != nil {
			break
		}
	}
	
	bodyContent := string(bodyBytes)
	
	// 检查Content-Type头部
	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		emailContent.Body = strings.TrimSpace(bodyContent)
		return
	}
	
	// 处理multipart邮件
	if strings.Contains(strings.ToLower(contentType), "multipart") {
		parseMultipartContent(bodyContent, contentType, emailContent)
		return
	}
	
	// 处理单一内容类型
	if strings.Contains(strings.ToLower(contentType), "text/html") {
		emailContent.HTMLBody = bodyContent
		emailContent.Body = extractTextFromHTML(bodyContent)
	} else {
		emailContent.Body = bodyContent
	}
}

// parseEmailFallbackEnhanced 增强版备用邮件解析器
func parseEmailFallbackEnhanced(rawContent string) *EmailContent {
	emailContent := &EmailContent{
		Headers: make(map[string]string),
		Charset: "utf-8",
	}
	
	lines := strings.Split(rawContent, "\n")
	headerSection := true
	var bodyLines []string
	
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		
		if headerSection {
			// 空行表示头部结束，正文开始
			if strings.TrimSpace(line) == "" {
				headerSection = false
				continue
			}
			
			// 解析邮件头
			if strings.HasPrefix(strings.ToLower(line), "subject:") {
				emailContent.Subject = strings.TrimSpace(line[8:])
			} else if strings.HasPrefix(strings.ToLower(line), "date:") {
				emailContent.Date = strings.TrimSpace(line[5:])
			} else if strings.HasPrefix(strings.ToLower(line), "from:") {
				emailContent.From = strings.TrimSpace(line[5:])
			} else if strings.HasPrefix(strings.ToLower(line), "to:") {
				emailContent.To = []string{strings.TrimSpace(line[3:])}
			} else if strings.HasPrefix(strings.ToLower(line), "content-type:") {
				emailContent.Charset = extractCharset(strings.TrimSpace(line[13:]))
			}
			
			// 存储所有头部
			if colonIndex := strings.Index(line, ":"); colonIndex > 0 {
				key := strings.TrimSpace(line[:colonIndex])
				value := strings.TrimSpace(line[colonIndex+1:])
				emailContent.Headers[key] = value
			}
		} else {
			// 处理邮件正文
			bodyLines = append(bodyLines, line)
		}
	}
	
	// 处理multipart邮件
	body := strings.Join(bodyLines, "\n")
	emailContent.Body = extractTextFromMultipart(body)
	
	// 如果没有解析到日期，使用当前时间
	if emailContent.Date == "" {
		emailContent.Date = time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")
	}
	
	// 检测签名和自动回复
	detectSignatureAndAutoReply(emailContent)
	
	return emailContent
}

// extractCharset 从 Content-Type 中提取字符编码
func extractCharset(contentType string) string {
	parts := strings.Split(contentType, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "charset=") {
			charset := strings.TrimPrefix(strings.ToLower(part), "charset=")
			charset = strings.Trim(charset, `"'`)
			return charset
		}
	}
	return "utf-8"
}


// extractTextFromHTML 从HTML中提取纯文本
func extractTextFromHTML(htmlContent string) string {
	if htmlContent == "" {
		return ""
	}
	
	// 移除HTML标签
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(htmlContent, "")
	
	// 清理多余的空白字符
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)
	
	return text
}

// detectSignatureAndAutoReply 检测邮件签名和自动回复
func detectSignatureAndAutoReply(emailContent *EmailContent) {
	// 检测自动回复
	emailContent.IsAutoReply = isAutoReply(emailContent)
	
	// 检测并提取签名
	emailContent.Signature = extractSignature(emailContent.Body)
}

// isAutoReply 检查是否为自动回复邮件
func isAutoReply(emailContent *EmailContent) bool {
	// 检查主题中的自动回复关键词
	subject := strings.ToLower(emailContent.Subject)
	autoReplyKeywords := []string{
		"automatic reply", "auto reply", "auto-reply", "out of office",
		"vacation", "holiday", "absent", "away", "unavailable",
		"自动回复", "外出", "休假", "不在", "离开", "无法接收",
		"vacation message", "out-of-office", "autoreply",
	}
	
	for _, keyword := range autoReplyKeywords {
		if strings.Contains(subject, keyword) {
			return true
		}
	}
	
	return false
}

// extractSignature 提取邮件签名
func extractSignature(body string) string {
	lines := strings.Split(body, "\n")
	
	// 寻找签名分隔符
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// 检查标准签名分隔符
		if trimmed == "-- " || trimmed == "--" {
			if i+1 < len(lines) {
				signature := strings.Join(lines[i+1:], "\n")
				return strings.TrimSpace(signature)
			}
		}
	}
	
	return ""
}

// extractEmbeddedContent 提取嵌入式内容（图片、链接等）
func extractEmbeddedContent(emailContent *EmailContent) map[string][]string {
	embeddedContent := make(map[string][]string)
	
	// 从HTML正文中提取
	if emailContent.HTMLBody != "" {
		embeddedContent["images"] = extractImages(emailContent.HTMLBody)
		embeddedContent["links"] = extractLinks(emailContent.HTMLBody)
	}
	
	// 从纯文本正文中提取链接
	if emailContent.Body != "" {
		textLinks := extractLinksFromText(emailContent.Body)
		embeddedContent["links"] = append(embeddedContent["links"], textLinks...)
	}
	
	return embeddedContent
}

// extractImages 从HTML中提取图片
func extractImages(htmlContent string) []string {
	var images []string
	
	// 提取img标签的src属性
	re := regexp.MustCompile(`(?i)<img[^>]+src\s*=\s*["']([^"']+)["']`)
	matches := re.FindAllStringSubmatch(htmlContent, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			images = append(images, match[1])
		}
	}
	
	return images
}

// extractLinks 从HTML中提取链接
func extractLinks(htmlContent string) []string {
	var links []string
	
	// 提取a标签的href属性
	re := regexp.MustCompile(`(?i)<a[^>]+href\s*=\s*["']([^"']+)["']`)
	matches := re.FindAllStringSubmatch(htmlContent, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			links = append(links, match[1])
		}
	}
	
	return links
}

// extractLinksFromText 从纯文本中提取链接
func extractLinksFromText(text string) []string {
	var links []string
	
	// 提取HTTP/HTTPS链接
	re := regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`)
	matches := re.FindAllString(text, -1)
	links = append(links, matches...)
	
	// 提取邮件地址
	re = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	matches = re.FindAllString(text, -1)
	for _, match := range matches {
		links = append(links, "mailto:"+match)
	}
	
	return links
}

// parseMultipartContent 解析多部分MIME内容
func parseMultipartContent(bodyContent, contentType string, emailContent *EmailContent) {
	// 提取boundary
	boundary := extractBoundary(contentType)
	if boundary == "" {
		// 如果没有boundary，使用原有的解析方法
		emailContent.Body = extractTextFromMultipart(bodyContent)
		return
	}
	
	// 按boundary分割内容
	parts := strings.Split(bodyContent, "--"+boundary)
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "--" {
			continue
		}
		
		// 分离头部和正文
		headerEnd := strings.Index(part, "\n\n")
		if headerEnd == -1 {
			headerEnd = strings.Index(part, "\r\n\r\n")
		}
		if headerEnd == -1 {
			continue
		}
		
		headerSection := part[:headerEnd]
		bodySection := part[headerEnd+2:]
		
		// 解析头部
		headers := parsePartHeaders(headerSection)
		contentType := headers["Content-Type"]
		contentDisposition := headers["Content-Disposition"]
		contentTransferEncoding := headers["Content-Transfer-Encoding"]
		
		// 解码内容
		decodedBody := decodeTransferEncoding(bodySection, contentTransferEncoding)
		
		// 处理不同的内容类型
		if strings.Contains(strings.ToLower(contentType), "text/plain") {
			if emailContent.Body == "" {
				emailContent.Body = decodedBody
			}
		} else if strings.Contains(strings.ToLower(contentType), "text/html") {
			if emailContent.HTMLBody == "" {
				emailContent.HTMLBody = decodedBody
			}
			if emailContent.Body == "" {
				emailContent.Body = extractTextFromHTML(decodedBody)
			}
		} else if isAttachment(contentDisposition) || hasFilename(contentType, contentDisposition) {
			// 处理附件
			attachment := AttachmentInfo{
				Filename:    extractFilename(contentType, contentDisposition),
				ContentType: contentType,
				Size:        int64(len(decodedBody)),
				Content:     []byte(decodedBody),
				CID:         extractContentID(headers["Content-ID"]),
				Disposition: extractDisposition(contentDisposition),
			}
			emailContent.Attachments = append(emailContent.Attachments, attachment)
		}
	}
}

// parsePartHeaders 解析部分头部
func parsePartHeaders(headerSection string) map[string]string {
	headers := make(map[string]string)
	lines := strings.Split(headerSection, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		colonIndex := strings.Index(line, ":")
		if colonIndex > 0 {
			key := strings.TrimSpace(line[:colonIndex])
			value := strings.TrimSpace(line[colonIndex+1:])
			headers[key] = value
		}
	}
	
	return headers
}

// extractBoundary 提取MIME边界
func extractBoundary(contentType string) string {
	parts := strings.Split(contentType, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "boundary=") {
			boundary := strings.TrimPrefix(part, "boundary=")
			boundary = strings.Trim(boundary, `"'`)
			return boundary
		}
	}
	return ""
}


// isAttachment 检查是否为附件
func isAttachment(contentDisposition string) bool {
	return strings.Contains(strings.ToLower(contentDisposition), "attachment")
}

// hasFilename 检查是否有文件名
func hasFilename(contentType, contentDisposition string) bool {
	return extractFilename(contentType, contentDisposition) != ""
}

// extractFilename 提取文件名
func extractFilename(contentType, contentDisposition string) string {
	// 首先从Content-Disposition中提取
	if contentDisposition != "" {
		if filename := extractFilenameFromHeader(contentDisposition); filename != "" {
			return filename
		}
	}
	
	// 然后从Content-Type中提取
	if contentType != "" {
		if filename := extractFilenameFromHeader(contentType); filename != "" {
			return filename
		}
	}
	
	return ""
}

// extractFilenameFromHeader 从邮件头中提取文件名
func extractFilenameFromHeader(header string) string {
	// 处理 filename= 格式
	patterns := []string{
		`filename="([^"]+)"`,
		`filename=([^;\s]+)`,
		`name="([^"]+)"`,
		`name=([^;\s]+)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		if matches := re.FindStringSubmatch(header); len(matches) > 1 {
			return matches[1]
		}
	}
	
	return ""
}

// extractContentID 提取Content-ID
func extractContentID(contentID string) string {
	// 移除< >包围符号
	contentID = strings.Trim(contentID, "<>")
	return contentID
}

// extractDisposition 提取内容处理方式
func extractDisposition(contentDisposition string) string {
	if contentDisposition == "" {
		return "attachment"
	}
	
	parts := strings.Split(contentDisposition, ";")
	if len(parts) > 0 {
		disposition := strings.TrimSpace(strings.ToLower(parts[0]))
		if disposition == "inline" || disposition == "attachment" {
			return disposition
		}
	}
	
	return "attachment"
}

// apiAttachments 处理附件下载请求
func (ms *MailServer) apiAttachments(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	// 处理OPTIONS预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// 解析URL: /api/attachments/mailbox/emailID/attachmentIndex
	path := strings.TrimPrefix(r.URL.Path, "/api/attachments/")
	parts := strings.Split(path, "/")
	
	if len(parts) < 3 {
		http.Error(w, "Invalid URL format. Expected: /api/attachments/mailbox/emailID/attachmentIndex", http.StatusBadRequest)
		return
	}
	
	mailbox := parts[0]
	emailID := parts[1]
	attachmentIndex := parts[2]
	
	// 获取邮件
	emails := ms.GetEmails(mailbox)
	var targetEmail *Email
	
	for _, email := range emails {
		if email.ID == emailID {
			targetEmail = &email
			break
		}
	}
	
	if targetEmail == nil {
		http.Error(w, "Email not found", http.StatusNotFound)
		return
	}
	
	// 解析附件索引
	index := 0
	if attachmentIndex != "" {
		fmt.Sscanf(attachmentIndex, "%d", &index)
	}
	
	if index < 0 || index >= len(targetEmail.Attachments) {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}
	
	attachment := targetEmail.Attachments[index]
	
	// 设置下载头
	w.Header().Set("Content-Type", attachment.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.Filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(attachment.Content)))
	
	// 输出附件内容
	w.Write(attachment.Content)
}

// apiInlineAttachments 处理内联附件（图片）显示请求
func (ms *MailServer) apiInlineAttachments(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	// 处理OPTIONS预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// 解析URL: /api/attachments/inline/mailbox/emailID/cid
	path := strings.TrimPrefix(r.URL.Path, "/api/attachments/inline/")
	parts := strings.Split(path, "/")
	
	if len(parts) < 3 {
		http.Error(w, "Invalid URL format. Expected: /api/attachments/inline/mailbox/emailID/cid", http.StatusBadRequest)
		return
	}
	
	mailbox := parts[0]
	emailID := parts[1]
	cid := parts[2]
	
	// 获取邮件
	emails := ms.GetEmails(mailbox)
	var targetEmail *Email
	
	for _, email := range emails {
		if email.ID == emailID {
			targetEmail = &email
			break
		}
	}
	
	if targetEmail == nil {
		http.Error(w, "Email not found", http.StatusNotFound)
		return
	}
	
	// 查找对应CID的内联附件
	var targetAttachment *AttachmentInfo
	for _, attachment := range targetEmail.Attachments {
		if attachment.CID == cid && attachment.Disposition == "inline" {
			targetAttachment = &attachment
			break
		}
	}
	
	if targetAttachment == nil {
		http.Error(w, "Inline attachment not found", http.StatusNotFound)
		return
	}
	
	// 设置图片显示头
	w.Header().Set("Content-Type", targetAttachment.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(targetAttachment.Content)))
	w.Header().Set("Cache-Control", "public, max-age=3600") // 缓存1小时
	
	// 输出图片内容
	w.Write(targetAttachment.Content)
}

// apiAdminUsers 管理员用户管理API
func (ms *MailServer) apiAdminUsers(w http.ResponseWriter, r *http.Request) {
	// 验证管理员权限
	if !ms.isAdminRequest(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// 获取所有用户
	users := ms.getAllUsers()
	
	// 过滤敏感信息
	var publicUsers []map[string]interface{}
	for _, user := range users {
		publicUser := map[string]interface{}{
			"email":              user.Email,
			"is_admin":           user.IsAdmin,
			"created_at":         user.CreatedAt,
			"last_login":         user.LastLogin,
			"two_factor_enabled": user.TwoFactorEnabled,
			"assigned_mailboxes": []string{},
		}
		publicUsers = append(publicUsers, publicUser)
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": publicUsers,
	})
}

// apiAdminCreateUser 创建用户API
func (ms *MailServer) apiAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	if !ms.isAdminRequest(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		IsAdmin  bool   `json:"is_admin"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// 验证输入
	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}
	
	// 创建用户
	if err := ms.createUser(req.Email, req.Password, req.IsAdmin); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "User created successfully",
	})
}

// apiAdminMailboxes 管理员邮箱管理API
func (ms *MailServer) apiAdminMailboxes(w http.ResponseWriter, r *http.Request) {
	if !ms.isAdminRequest(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// 获取所有邮箱
	mailboxes := ms.getAllMailboxes()
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"mailboxes": mailboxes,
	})
}

// apiAdminCreateMailbox 创建邮箱API
func (ms *MailServer) apiAdminCreateMailbox(w http.ResponseWriter, r *http.Request) {
	if !ms.isAdminRequest(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		Description string `json:"description"`
		Owner       string `json:"owner"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// 验证输入
	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}
	
	// 创建邮箱
	if err := ms.createMailbox(req.Email, req.Password, req.Description, req.Owner); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Mailbox created successfully",
	})
}

// apiForwardingSettings 获取转发设置API
func (ms *MailServer) apiForwardingSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// 获取当前登录用户邮箱
	userEmail := ms.getUserFromRequest(r)
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	// 获取转发设置
	settings := ms.getForwardingSettings(userEmail)
	
	json.NewEncoder(w).Encode(settings)
}

// apiForwardingUpdate 更新转发设置API
func (ms *MailServer) apiForwardingUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		ForwardTo      string `json:"forward_to"`
		ForwardEnabled bool   `json:"forward_enabled"`
		KeepOriginal   bool   `json:"keep_original"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// 获取当前登录用户邮箱
	userEmail := ms.getUserFromRequest(r)
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	// 更新转发设置
	if err := ms.updateForwardingSettings(userEmail, req.ForwardTo, req.ForwardEnabled, req.KeepOriginal); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Forwarding settings updated successfully",
	})
}

// apiUserMailboxes 用户邮箱列表API
func (ms *MailServer) apiUserMailboxes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// 获取用户的邮箱列表
	userEmail := ms.getUserFromRequest(r)
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	mailboxes := ms.getUserMailboxes(userEmail)
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"mailboxes": mailboxes,
	})
}

// apiUserEmails 用户邮件API（权限受限）
func (ms *MailServer) apiUserEmails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	mailbox := strings.TrimPrefix(r.URL.Path, "/api/user/emails/")
	
	// 验证用户权限
	if !ms.hasMailboxAccess(r, mailbox) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	// 获取邮件（复用现有逻辑）
	emails := ms.GetEmails(mailbox)
	
	var apiEmails []map[string]interface{}
	for _, email := range emails {
		decodedBody := decodeEmailBodyIfNeeded(email.Body)
		decodedSubject := decodeEmailBodyIfNeeded(email.Subject)
		
		apiEmail := map[string]interface{}{
			"From":        email.From,
			"To":          email.To,
			"Subject":     decodedSubject,
			"Body":        decodedBody,
			"HTMLBody":    email.HTMLBody,
			"Date":        email.Date,
			"ID":          email.ID,
			"Attachments": email.Attachments,
			"Signature":   email.Signature,
			"IsAutoReply": email.IsAutoReply,
			"Charset":     email.Charset,
			"Headers":     email.Headers,
			"Embedded":    email.Embedded,
		}
		
		apiEmails = append(apiEmails, apiEmail)
	}
	
	jsonData, err := json.Marshal(apiEmails)
	if err != nil {
		http.Error(w, "JSON编码失败", http.StatusInternalServerError)
		return
	}
	
	w.Write(jsonData)
}

// 权限验证和用户管理辅助函数

// isAdminRequest 验证请求是否来自管理员
func (ms *MailServer) isAdminRequest(r *http.Request) bool {
	// 获取用户邮箱
	userEmail := ms.getUserFromRequest(r)
	if userEmail == "" {
		return false
	}
	
	// 检查用户是否是管理员
	user, err := ms.database.GetUser(userEmail)
	if err != nil {
		return false
	}
	
	return user.IsAdmin
}

// hasMailboxAccess 验证用户是否有邮箱访问权限
func (ms *MailServer) hasMailboxAccess(r *http.Request, mailbox string) bool {
	// 获取用户邮箱
	userEmail := ms.getUserFromRequest(r)
	if userEmail == "" {
		return false
	}
	
	// 管理员可以访问所有邮箱
	if ms.isAdminRequest(r) {
		return true
	}
	
	// 用户只能访问自己的邮箱
	return userEmail == mailbox
}

// getUserFromRequest 从请求中获取用户邮箱
func (ms *MailServer) getUserFromRequest(r *http.Request) string {
	// 尝试从 Authorization header 获取 Bearer token
	token := r.Header.Get("Authorization")
	if token != "" {
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
		
		// 先尝试JWT验证
		if claims, err := ms.userAuth.ValidateJWTToken(token); err == nil {
			return claims.Email
		}
		
		// 如果JWT验证失败，尝试简单的base64解码
		if decoded, err := base64.StdEncoding.DecodeString(token); err == nil {
			parts := strings.Split(string(decoded), ":")
			if len(parts) >= 2 {
				email := parts[0]
				// 简单验证：检查用户是否存在于数据库中
				if _, err := ms.database.GetUser(email); err == nil {
					return email
				}
				// 如果用户不存在，也检查邮箱
				if _, err := ms.database.GetMailbox(email); err == nil {
					return email
				}
			}
		}
	}
	
	// 尝试从 Session-ID header 获取会话
	sessionID := r.Header.Get("Session-ID")
	if sessionID != "" {
		session, err := ms.database.GetSession(sessionID)
		if err == nil {
			return session.Email
		}
	}
	
	return ""
}

// getAllUsers 获取所有用户
func (ms *MailServer) getAllUsers() []*UserDB {
	return ms.userAuth.GetAllUsers()
}

// createUser 创建新用户
func (ms *MailServer) createUser(email, password string, isAdmin bool) error {
	return ms.userAuth.CreateUser(email, password, isAdmin)
}

// getAllMailboxes 获取所有邮箱
func (ms *MailServer) getAllMailboxes() []MailboxDB {
	mailboxes, err := ms.database.GetAllMailboxes()
	if err != nil {
		log.Printf("Error getting mailboxes: %v", err)
		return []MailboxDB{}
	}
	return mailboxes
}

// createMailbox 创建新邮箱
func (ms *MailServer) createMailbox(email, password, description, owner string) error {
	return ms.database.CreateMailbox(email, password, description, owner)
}

// getUserMailboxes 获取用户的邮箱列表
func (ms *MailServer) getUserMailboxes(userEmail string) []MailboxDB {
	user, err := ms.database.GetUser(userEmail)
	if err != nil {
		return []MailboxDB{}
	}
	
	// 管理员可以看到所有邮箱
	if user.IsAdmin {
		return ms.getAllMailboxes()
	}
	
	// 普通用户只能看到自己的邮箱
	mailboxes, err := ms.database.GetMailboxesByOwner(userEmail)
	if err != nil {
		log.Printf("Error getting mailboxes for user %s: %v", userEmail, err)
		return []MailboxDB{}
	}
	
	return mailboxes
}

// getForwardingSettings 获取转发设置
func (ms *MailServer) getForwardingSettings(mailbox string) map[string]interface{} {
	mailboxInfo, err := ms.database.GetMailbox(mailbox)
	if err != nil {
		return map[string]interface{}{
			"forward_enabled": false,
			"forward_to":      "",
			"keep_original":   true,
		}
	}
	
	return map[string]interface{}{
		"forward_enabled": mailboxInfo.ForwardEnabled,
		"forward_to":      mailboxInfo.ForwardTo,
		"keep_original":   mailboxInfo.KeepOriginal,
		"mailbox":         mailbox,
	}
}

// updateForwardingSettings 更新转发设置
func (ms *MailServer) updateForwardingSettings(mailbox, forwardTo string, forwardEnabled, keepOriginal bool) error {
	return ms.database.UpdateMailboxForwarding(mailbox, forwardTo, forwardEnabled, keepOriginal)
}

// processEmailForwarding 处理邮件转发
func (ms *MailServer) processEmailForwarding(recipientEmail, rawContent string) {
	// 获取收件人邮箱设置
	mailboxInfo, err := ms.database.GetMailbox(recipientEmail)
	if err != nil || !mailboxInfo.ForwardEnabled || mailboxInfo.ForwardTo == "" {
		return
	}
	
	// 解析邮件内容
	emailContent, _, _ := ParseEmailContent(rawContent)
	if emailContent == "" {
		return
	}
	
	// 构建转发邮件
	forwardedContent := fmt.Sprintf("Forwarded from %s:\n\n%s", recipientEmail, emailContent)
	
	// 发送转发邮件
	err = ms.smtpSender.SendEmail(recipientEmail, mailboxInfo.ForwardTo, "Forwarded Email", forwardedContent)
	if err != nil {
		log.Printf("Failed to forward email from %s to %s: %v", recipientEmail, mailboxInfo.ForwardTo, err)
	} else {
		log.Printf("Email forwarded from %s to %s", recipientEmail, mailboxInfo.ForwardTo)
	}
}

// buildForwardedEmail 构建转发邮件内容
func (ms *MailServer) buildForwardedEmail(original *EmailContent, originalRecipient, forwardTo string) string {
	subject := fmt.Sprintf("Fwd: %s", original.Subject)
	
	var body strings.Builder
	body.WriteString(fmt.Sprintf("---------- Forwarded message ----------\n"))
	body.WriteString(fmt.Sprintf("From: %s\n", original.From))
	body.WriteString(fmt.Sprintf("Date: %s\n", original.Date))
	body.WriteString(fmt.Sprintf("Subject: %s\n", original.Subject))
	body.WriteString(fmt.Sprintf("To: %s\n", originalRecipient))
	if len(original.CC) > 0 {
		body.WriteString(fmt.Sprintf("CC: %s\n", strings.Join(original.CC, ", ")))
	}
	body.WriteString("\n")
	body.WriteString(original.Body)
	
	// 构建完整的邮件格式
	var email strings.Builder
	email.WriteString(fmt.Sprintf("From: %s\n", originalRecipient))
	email.WriteString(fmt.Sprintf("To: %s\n", forwardTo))
	email.WriteString(fmt.Sprintf("Subject: %s\n", subject))
	email.WriteString(fmt.Sprintf("Date: %s\n", time.Now().Format(time.RFC1123Z)))
	email.WriteString("Content-Type: text/plain; charset=utf-8\n")
	email.WriteString("\n")
	email.WriteString(body.String())
	
	return email.String()
}