package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type EmailStorage struct {
	dataDir string
	emails  map[string][]Email
	emailsMu sync.RWMutex
}

type StoredEmail struct {
	Email
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Flags     []string  `json:"flags"`
}

func NewEmailStorage(dataDir string) *EmailStorage {
	es := &EmailStorage{
		dataDir: dataDir,
		emails:  make(map[string][]Email),
	}
	
	// 创建数据目录
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("Warning: Failed to create data directory: %v\n", err)
	}
	
	// 加载现有邮件
	es.loadEmails()
	
	return es
}

// AddEmail 添加邮件到存储
func (es *EmailStorage) AddEmail(to string, email Email) error {
	es.emailsMu.Lock()
	defer es.emailsMu.Unlock()
	
	// 添加到内存
	to = strings.ToLower(to)
	es.emails[to] = append(es.emails[to], email)
	
	// 持久化到文件
	return es.saveEmailToFile(to, email)
}

// GetEmails 获取邮箱的所有邮件
func (es *EmailStorage) GetEmails(mailbox string) []Email {
	es.emailsMu.RLock()
	defer es.emailsMu.RUnlock()
	
	mailbox = strings.ToLower(mailbox)
	if emails, exists := es.emails[mailbox]; exists {
		// 返回副本，避免并发修改
		result := make([]Email, len(emails))
		copy(result, emails)
		return result
	}
	
	return []Email{}
}

// GetAllMailboxes 获取所有邮箱列表
func (es *EmailStorage) GetAllMailboxes() []string {
	es.emailsMu.RLock()
	defer es.emailsMu.RUnlock()
	
	var mailboxes []string
	for mailbox := range es.emails {
		mailboxes = append(mailboxes, mailbox)
	}
	return mailboxes
}

// DeleteEmail 删除指定ID的邮件
func (es *EmailStorage) DeleteEmail(mailbox string, emailID string) bool {
	es.emailsMu.Lock()
	defer es.emailsMu.Unlock()
	
	mailbox = strings.ToLower(mailbox)
	emails, exists := es.emails[mailbox]
	if !exists {
		return false
	}
	
	// 查找并删除指定ID的邮件
	for i, email := range emails {
		if email.ID == emailID {
			// 从内存中删除
			es.emails[mailbox] = append(emails[:i], emails[i+1:]...)
			
			// 重新保存整个邮箱
			err := es.saveMailboxToFile(mailbox)
			return err == nil
		}
	}
	
	return false // 未找到指定ID的邮件
}

// DeleteMailbox 删除整个邮箱
func (es *EmailStorage) DeleteMailbox(mailbox string) error {
	es.emailsMu.Lock()
	defer es.emailsMu.Unlock()
	
	mailbox = strings.ToLower(mailbox)
	
	// 从内存中删除
	delete(es.emails, mailbox)
	
	// 删除文件
	filename := es.getMailboxFilename(mailbox)
	return os.Remove(filename)
}

// GetEmailCount 获取邮箱邮件数量
func (es *EmailStorage) GetEmailCount(mailbox string) int {
	es.emailsMu.RLock()
	defer es.emailsMu.RUnlock()
	
	mailbox = strings.ToLower(mailbox)
	if emails, exists := es.emails[mailbox]; exists {
		return len(emails)
	}
	return 0
}

// GetTotalEmailCount 获取总邮件数量
func (es *EmailStorage) GetTotalEmailCount() int {
	es.emailsMu.RLock()
	defer es.emailsMu.RUnlock()
	
	total := 0
	for _, emails := range es.emails {
		total += len(emails)
	}
	return total
}

// GetStorageStats 获取存储统计信息
func (es *EmailStorage) GetStorageStats() map[string]interface{} {
	es.emailsMu.RLock()
	defer es.emailsMu.RUnlock()
	
	stats := map[string]interface{}{
		"total_mailboxes": len(es.emails),
		"total_emails":    0,
		"mailbox_stats":   make(map[string]int),
	}
	
	totalEmails := 0
	mailboxStats := make(map[string]int)
	
	for mailbox, emails := range es.emails {
		count := len(emails)
		totalEmails += count
		mailboxStats[mailbox] = count
	}
	
	stats["total_emails"] = totalEmails
	stats["mailbox_stats"] = mailboxStats
	
	return stats
}

// SearchEmails 搜索邮件
func (es *EmailStorage) SearchEmails(mailbox, keyword string) []Email {
	es.emailsMu.RLock()
	defer es.emailsMu.RUnlock()
	
	mailbox = strings.ToLower(mailbox)
	emails, exists := es.emails[mailbox]
	if !exists {
		return []Email{}
	}
	
	var results []Email
	keyword = strings.ToLower(keyword)
	
	for _, email := range emails {
		// 搜索主题、正文、发件人
		if strings.Contains(strings.ToLower(email.Subject), keyword) ||
		   strings.Contains(strings.ToLower(email.Body), keyword) ||
		   strings.Contains(strings.ToLower(email.From), keyword) {
			results = append(results, email)
		}
	}
	
	return results
}

// BackupData 备份数据
func (es *EmailStorage) BackupData(backupDir string) error {
	es.emailsMu.RLock()
	defer es.emailsMu.RUnlock()
	
	// 创建备份目录
	timestamp := time.Now().Format("20060102_150405")
	fullBackupDir := filepath.Join(backupDir, "backup_"+timestamp)
	
	if err := os.MkdirAll(fullBackupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}
	
	// 备份每个邮箱
	for mailbox := range es.emails {
		srcFile := es.getMailboxFilename(mailbox)
		dstFile := filepath.Join(fullBackupDir, filepath.Base(srcFile))
		
		if err := es.copyFile(srcFile, dstFile); err != nil {
			return fmt.Errorf("failed to backup mailbox %s: %v", mailbox, err)
		}
	}
	
	fmt.Printf("Data backed up to: %s\n", fullBackupDir)
	return nil
}

// saveEmailToFile 保存单个邮件到文件
func (es *EmailStorage) saveEmailToFile(mailbox string, email Email) error {
	return es.saveMailboxToFile(mailbox)
}

// saveMailboxToFile 保存整个邮箱到文件
func (es *EmailStorage) saveMailboxToFile(mailbox string) error {
	filename := es.getMailboxFilename(mailbox)
	
	emails := es.emails[mailbox]
	if emails == nil {
		emails = []Email{}
	}
	
	// 转换为存储格式
	storedEmails := make([]StoredEmail, len(emails))
	for i, email := range emails {
		storedEmails[i] = StoredEmail{
			Email:     email,
			ID:        fmt.Sprintf("%s_%d_%d", mailbox, i, time.Now().Unix()),
			Timestamp: time.Now(),
			Flags:     []string{},
		}
	}
	
	data, err := json.MarshalIndent(storedEmails, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal emails: %v", err)
	}
	
	return os.WriteFile(filename, data, 0644)
}

// loadEmails 从文件加载所有邮件
func (es *EmailStorage) loadEmails() {
	if es.dataDir == "" {
		return
	}
	
	// 读取数据目录中的所有邮箱文件
	files, err := filepath.Glob(filepath.Join(es.dataDir, "mailbox_*.json"))
	if err != nil {
		fmt.Printf("Warning: Failed to load emails: %v\n", err)
		return
	}
	
	for _, filename := range files {
		es.loadMailboxFromFile(filename)
	}
}

// loadMailboxFromFile 从文件加载邮箱
func (es *EmailStorage) loadMailboxFromFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	
	var storedEmails []StoredEmail
	if err := json.Unmarshal(data, &storedEmails); err != nil {
		fmt.Printf("Warning: Failed to parse mailbox file %s: %v\n", filename, err)
		return
	}
	
	// 从文件名提取邮箱名
	base := filepath.Base(filename)
	if !strings.HasPrefix(base, "mailbox_") || !strings.HasSuffix(base, ".json") {
		return
	}
	
	safeName := base[8 : len(base)-5] // 移除 "mailbox_" 前缀和 ".json" 后缀
	
	// 反向转换文件名为邮箱地址
	mailbox := strings.ReplaceAll(safeName, "_at_", "@")
	mailbox = strings.ReplaceAll(mailbox, "_dot_", ".")
	
	// 转换为普通邮件格式
	emails := make([]Email, len(storedEmails))
	for i, stored := range storedEmails {
		emails[i] = stored.Email
	}
	
	es.emails[mailbox] = emails
}

// getMailboxFilename 获取邮箱文件路径
func (es *EmailStorage) getMailboxFilename(mailbox string) string {
	// 替换特殊字符，确保文件名安全
	safeName := strings.ReplaceAll(mailbox, "@", "_at_")
	safeName = strings.ReplaceAll(safeName, ".", "_dot_")
	return filepath.Join(es.dataDir, "mailbox_"+safeName+".json")
}

// copyFile 复制文件
func (es *EmailStorage) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}