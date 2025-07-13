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
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

type Email struct {
	From    string
	To      string
	Subject string
	Body    string
	Date    string
	ID      string // 添加邮件ID字段
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
	userAuth := NewUserAuth("./data/users.json")
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
	
	// 设置SMTP发送器的中继
	ms.smtpSender.SetRelay(ms.relayManager.GetRelay())
	
	// 创建IMAP服务器
	ms.imapServer = NewIMAPServer(ms)
	
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
	
	// 记录处理日志
	log.Printf("邮件已接收: %s -> %s", email.From, to)
}

func (ms *MailServer) GetEmails(mailbox string) []Email {
	return ms.storage.GetEmails(mailbox)
}

func (ms *MailServer) GetAllMailboxes() []string {
	return ms.mailboxManager.GetMailboxesByDomain()
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
				subject, body, date := ParseEmailContent(rawContent)
				
				email := Email{
					From:    from,
					To:      to,
					Subject: subject,
					Body:    body,
					Date:    date,
					ID:      generateEmailID(to),
				}
				
				ms.AddEmail(to, email)
				
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
	
	// SMTP中继API
	http.HandleFunc("/api/relay/config", ms.apiRelayConfig)
	http.HandleFunc("/api/relay/providers", ms.apiRelayProviders)
	http.HandleFunc("/api/relay/test", ms.apiRelayTest)
	http.HandleFunc("/api/relay/status", ms.apiRelayStatus)
	
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
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
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
	
	err := ms.mailboxManager.CreateMailbox(req.Username, req.Password, req.Description)
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
	
	// 优先使用数据库认证
	if ms.database.Authenticate(req.Email, req.Password) {
		sessionID, _ := ms.database.CreateSession(req.Email)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":     "success",
			"session_id": sessionID,
			"email":      req.Email,
		})
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
            const user = localStorage.getItem('currentUser');
            if (!user) {
                showLoginModal();
                return false;
            }
            currentUser = user;
            updateUserBar();
            return true;
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
                    currentUser = email;
                    localStorage.setItem('currentUser', email);
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
                await fetch('/api/logout', {method: 'POST'});
                localStorage.removeItem('currentUser');
                currentUser = null;
                checkLogin();
            } catch (error) {
                console.error('登出失败:', error);
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
                const response = await fetch('/api/mailboxes');
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
                
                const response = await fetch('/api/emails/' + encodeURIComponent(mailbox));
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
                const response = await fetch('/api/mailboxes/manage');
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
	
	mailboxes := ms.GetAllMailboxes()
	
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
			"From":    email.From,
			"To":      email.To,
			"Subject": decodedSubject,
			"Body":    decodedBody,
			"Date":    email.Date,
			"ID":      email.ID,
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