package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
)

type MailboxManager struct {
	mailboxes map[string]*Mailbox
	domain    string
	mu        sync.RWMutex
	filename  string
}

type Mailbox struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	CreatedAt   string `json:"created_at"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

type MailboxConfig struct {
	Mailboxes map[string]*Mailbox `json:"mailboxes"`
}

func NewMailboxManager(domain, filename string) *MailboxManager {
	mm := &MailboxManager{
		mailboxes: make(map[string]*Mailbox),
		domain:    domain,
		filename:  filename,
	}
	mm.loadFromFile()
	mm.ensureDefaultMailboxes()
	return mm
}

// CreateMailbox 创建新邮箱
func (mm *MailboxManager) CreateMailbox(username, password, description string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// 验证用户名格式
	if !mm.isValidUsername(username) {
		return fmt.Errorf("无效的用户名：只能包含字母、数字、点号和下划线")
	}

	email := username + "@" + mm.domain

	// 检查是否已存在
	if _, exists := mm.mailboxes[email]; exists {
		return fmt.Errorf("邮箱 %s 已存在", email)
	}

	// 创建邮箱
	mailbox := &Mailbox{
		Username:    username,
		Email:       email,
		Password:    password,
		CreatedAt:   getCurrentTime(),
		Description: description,
		IsActive:    true,
	}

	mm.mailboxes[email] = mailbox
	return mm.saveToFile()
}

// DeleteMailbox 删除邮箱
func (mm *MailboxManager) DeleteMailbox(email string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if _, exists := mm.mailboxes[email]; !exists {
		return fmt.Errorf("邮箱 %s 不存在", email)
	}

	delete(mm.mailboxes, email)
	return mm.saveToFile()
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

	return mm.saveToFile()
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

	return mailbox.Password == password
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
			mm.mailboxes[email] = &Mailbox{
				Username:    def.username,
				Email:       email,
				Password:    def.password,
				CreatedAt:   getCurrentTime(),
				Description: def.description,
				IsActive:    true,
			}
		}
	}

	mm.saveToFile()
}

func (mm *MailboxManager) loadFromFile() error {
	if mm.filename == "" {
		return nil
	}

	data, err := os.ReadFile(mm.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var config MailboxConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	mm.mailboxes = config.Mailboxes
	if mm.mailboxes == nil {
		mm.mailboxes = make(map[string]*Mailbox)
	}

	return nil
}

func (mm *MailboxManager) saveToFile() error {
	if mm.filename == "" {
		return nil
	}

	config := MailboxConfig{
		Mailboxes: mm.mailboxes,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(mm.filename, data, 0644)
}

func getCurrentTime() string {
	return fmt.Sprintf("%d", 1752322350) // 简化的时间戳
}