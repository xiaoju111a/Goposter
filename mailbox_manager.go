package main

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

type MailboxManager struct {
	mailboxes map[string]*Mailbox
	domain    string
	mu        sync.RWMutex
	database  *Database
}

type Mailbox struct {
	Username        string `json:"username"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	CreatedAt       string `json:"created_at"`
	Description     string `json:"description"`
	IsActive        bool   `json:"is_active"`
	Owner           string `json:"owner"`           // 邮箱所有者
	ForwardTo       string `json:"forward_to"`      // 转发邮箱
	ForwardEnabled  bool   `json:"forward_enabled"` // 转发开关
	KeepOriginal    bool   `json:"keep_original"`   // 是否保留原邮件
}


func NewMailboxManager(domain string, database *Database) *MailboxManager {
	mm := &MailboxManager{
		mailboxes: make(map[string]*Mailbox),
		domain:    domain,
		database:  database,
	}
	mm.loadFromDatabase()
	mm.ensureDefaultMailboxes()
	return mm
}

// CreateMailbox 创建新邮箱
func (mm *MailboxManager) CreateMailbox(email, password, description, owner string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// 从email中提取用户名
	username := strings.Split(email, "@")[0]
	
	// 验证用户名格式
	if !mm.isValidUsername(username) {
		return fmt.Errorf("无效的用户名：只能包含字母、数字、点号和下划线")
	}

	// 检查是否已存在
	if _, exists := mm.mailboxes[email]; exists {
		return fmt.Errorf("邮箱 %s 已存在", email)
	}

	// 创建邮箱
	mailbox := &Mailbox{
		Username:        username,
		Email:           email,
		Password:        password,
		CreatedAt:       getCurrentTime(),
		Description:     description,
		IsActive:        true,
		Owner:           owner,
		ForwardTo:       "",
		ForwardEnabled:  false,
		KeepOriginal:    true,
	}

	mm.mailboxes[email] = mailbox
	return mm.saveToDatabase(email)
}

// DeleteMailbox 删除邮箱
func (mm *MailboxManager) DeleteMailbox(email string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if _, exists := mm.mailboxes[email]; !exists {
		return fmt.Errorf("邮箱 %s 不存在", email)
	}

	delete(mm.mailboxes, email)
	return mm.database.DeleteMailbox(email)
}

// UpdateMailbox 更新邮箱信息
func (mm *MailboxManager) UpdateMailbox(email, password, description string, isActive bool) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mailbox, exists := mm.mailboxes[email]
	if !exists {
		return fmt.Errorf("邮箱 %s 不存在", email)
	}

	if password != "" {
		mailbox.Password = password
	}
	mailbox.Description = description
	mailbox.IsActive = isActive

	return mm.saveToDatabase(email)
}

// GetMailbox 获取邮箱信息
func (mm *MailboxManager) GetMailbox(email string) (*Mailbox, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	mailbox, exists := mm.mailboxes[email]
	if !exists {
		return nil, false
	}

	// 返回副本
	result := *mailbox
	return &result, true
}

// GetAllMailboxes 获取所有邮箱
func (mm *MailboxManager) GetAllMailboxes() []*Mailbox {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	var result []*Mailbox
	for _, mailbox := range mm.mailboxes {
		mailboxCopy := *mailbox
		result = append(result, &mailboxCopy)
	}

	return result
}

// IsValidMailbox 检查邮箱是否存在且激活
func (mm *MailboxManager) IsValidMailbox(email string) bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	mailbox, exists := mm.mailboxes[email]
	return exists && mailbox.IsActive
}

// AuthenticateMailbox 验证邮箱密码
func (mm *MailboxManager) AuthenticateMailbox(email, password string) bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	mailbox, exists := mm.mailboxes[email]
	if !exists || !mailbox.IsActive {
		return false
	}

	// Use database validation for secure password check
	return mm.database.ValidateMailboxCredentials(email, password)
}

// GetMailboxesByDomain 获取指定域名的所有邮箱
func (mm *MailboxManager) GetMailboxesByDomain() []string {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	var emails []string
	for email, mailbox := range mm.mailboxes {
		if mailbox.IsActive {
			emails = append(emails, email)
		}
	}

	return emails
}

// isValidUsername 验证用户名格式
func (mm *MailboxManager) isValidUsername(username string) bool {
	// 用户名规则：3-20位，只能包含字母、数字、点号、下划线
	if len(username) < 3 || len(username) > 20 {
		return false
	}

	// 正则表达式验证
	matched, _ := regexp.MatchString("^[a-zA-Z0-9._]+$", username)
	if !matched {
		return false
	}

	// 不能以点号开头或结尾
	if strings.HasPrefix(username, ".") || strings.HasSuffix(username, ".") {
		return false
	}

	// 不能有连续的点号
	if strings.Contains(username, "..") {
		return false
	}

	return true
}

// ensureDefaultMailboxes 确保有默认邮箱
func (mm *MailboxManager) ensureDefaultMailboxes() {
	defaultMailboxes := []struct {
		username    string
		password    string
		description string
	}{
		{"admin", "admin123", "系统管理员邮箱"},
		{"support", "support123", "技术支持邮箱"},
		{"info", "info123", "信息咨询邮箱"},
		{"noreply", "noreply123", "系统邮件邮箱"},
	}

	for _, def := range defaultMailboxes {
		email := def.username + "@" + mm.domain
		if _, exists := mm.mailboxes[email]; !exists {
			// Create mailbox in memory
			mm.mailboxes[email] = &Mailbox{
				Username:    def.username,
				Email:       email,
				Password:    def.password,
				CreatedAt:   getCurrentTime(),
				Description: def.description,
				IsActive:    true,
			}
			// Create in database if it doesn't exist
			if err := mm.database.CreateMailbox(email, def.password, def.description, ""); err != nil {
				// If mailbox already exists in database, update memory to use hashed password
				mm.mailboxes[email].Password = "***hashed***"
			}
		}
	}

	// Default mailboxes are created only if they don't exist
	// No need to call saveToDatabase as they're handled individually
}

func (mm *MailboxManager) loadFromDatabase() error {
	mailboxes, err := mm.database.GetAllMailboxes()
	if err != nil {
		return err
	}

	for _, mailboxDB := range mailboxes {
		// For existing mailboxes, we use a placeholder password since we can't recover the original
		// Authentication will be handled by the database directly
		mailbox := &Mailbox{
			Username:        strings.Split(mailboxDB.Email, "@")[0],
			Email:           mailboxDB.Email,
			Password:        "***hashed***", // Placeholder - actual auth uses database
			CreatedAt:       mailboxDB.CreatedAt.Format("2006-01-02 15:04:05"),
			Description:     mailboxDB.Description,
			IsActive:        mailboxDB.IsActive,
			Owner:           mailboxDB.Owner,
			ForwardTo:       mailboxDB.ForwardTo,
			ForwardEnabled:  mailboxDB.ForwardEnabled,
			KeepOriginal:    mailboxDB.KeepOriginal,
		}
		mm.mailboxes[mailboxDB.Email] = mailbox
	}

	return nil
}

func (mm *MailboxManager) saveToDatabase(email string) error {
	if email == "" {
		// This should not be called for bulk operations
		return nil
	}

	// Save specific mailbox
	mailbox, exists := mm.mailboxes[email]
	if !exists {
		return fmt.Errorf("邮箱 %s 不存在", email)
	}

	// Only create in database if it's a new mailbox (not already hashed)
	if mailbox.Password != "***hashed***" {
		return mm.database.CreateMailbox(mailbox.Email, mailbox.Password, mailbox.Description, mailbox.Owner)
	}
	
	return nil
}

func getCurrentTime() string {
	return fmt.Sprintf("%d", 1752322350) // 简化的时间戳
}

// GetAllMailboxes 获取所有邮箱

// UpdateForwardingSettings 更新转发设置
func (mm *MailboxManager) UpdateForwardingSettings(email, forwardTo string, forwardEnabled, keepOriginal bool) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	mailbox, exists := mm.mailboxes[email]
	if !exists {
		return fmt.Errorf("邮箱 %s 不存在", email)
	}
	
	mailbox.ForwardTo = forwardTo
	mailbox.ForwardEnabled = forwardEnabled
	mailbox.KeepOriginal = keepOriginal
	
	return mm.database.UpdateMailboxForwarding(email, forwardTo, forwardEnabled, keepOriginal)
}