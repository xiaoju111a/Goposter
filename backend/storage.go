package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

type EmailStorage struct {
	dataDir string
	emails  map[string][]Email
	emailsMu sync.RWMutex
	database *Database
}


func NewEmailStorage(dataDir string, database *Database) *EmailStorage {
	es := &EmailStorage{
		dataDir: dataDir,
		emails:  make(map[string][]Email),
		database: database,
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
	
	// 持久化到数据库
	return es.database.SaveEmail(to, email.From, email.To, email.Subject, email.Body, "")
}

// GetEmails 获取邮箱的所有邮件
func (es *EmailStorage) GetEmails(mailbox string) []Email {
	mailbox = strings.ToLower(mailbox)
	
	// 从数据库获取邮件
	emailsDB, err := es.database.GetEmails(mailbox)
	if err != nil {
		fmt.Printf("Warning: Failed to get emails from database: %v\n", err)
		return []Email{}
	}
	
	// 转换为 Email 结构
	var emails []Email
	for _, emailDB := range emailsDB {
		email := Email{
			ID:      fmt.Sprintf("%d", emailDB.ID),
			From:    emailDB.From,
			To:      emailDB.To,
			Subject: emailDB.Subject,
			Body:    emailDB.Body,
			Date:    emailDB.Received.Format("2006-01-02 15:04:05"),
			Headers: make(map[string]string),
		}
		emails = append(emails, email)
	}
	
	return emails
}

// GetAllMailboxes 获取所有邮箱列表
func (es *EmailStorage) GetAllMailboxes() []string {
	mailboxes, err := es.database.GetAllMailboxNames()
	if err != nil {
		fmt.Printf("Warning: Failed to get mailboxes from database: %v\n", err)
		return []string{}
	}
	return mailboxes
}

// DeleteEmail 删除指定ID的邮件
func (es *EmailStorage) DeleteEmail(mailbox string, emailID string) bool {
	mailbox = strings.ToLower(mailbox)
	
	// 转换emailID为整数
	var id int
	if _, err := fmt.Sscanf(emailID, "%d", &id); err != nil {
		return false
	}
	
	// 从数据库删除
	err := es.database.DeleteEmail(mailbox, id)
	if err != nil {
		fmt.Printf("Warning: Failed to delete email from database: %v\n", err)
		return false
	}
	
	// 从内存中删除
	es.emailsMu.Lock()
	defer es.emailsMu.Unlock()
	
	emails, exists := es.emails[mailbox]
	if exists {
		for i, email := range emails {
			if email.ID == emailID {
				es.emails[mailbox] = append(emails[:i], emails[i+1:]...)
				break
			}
		}
	}
	
	return true
}

// DeleteMailbox 删除整个邮箱
func (es *EmailStorage) DeleteMailbox(mailbox string) error {
	es.emailsMu.Lock()
	defer es.emailsMu.Unlock()
	
	mailbox = strings.ToLower(mailbox)
	
	// 从内存中删除
	delete(es.emails, mailbox)
	
	// 这里可以添加从数据库删除所有邮件的逻辑，但应该由上层代码处理
	return nil
}

// GetEmailCount 获取邮箱邮件数量
func (es *EmailStorage) GetEmailCount(mailbox string) int {
	mailbox = strings.ToLower(mailbox)
	emails, err := es.database.GetEmails(mailbox)
	if err != nil {
		return 0
	}
	return len(emails)
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
	mailbox = strings.ToLower(mailbox)
	
	// 从数据库获取邮件
	emailsDB, err := es.database.GetEmails(mailbox)
	if err != nil {
		return []Email{}
	}
	
	var results []Email
	keyword = strings.ToLower(keyword)
	
	for _, emailDB := range emailsDB {
		// 搜索主题、正文、发件人
		if strings.Contains(strings.ToLower(emailDB.Subject), keyword) ||
		   strings.Contains(strings.ToLower(emailDB.Body), keyword) ||
		   strings.Contains(strings.ToLower(emailDB.From), keyword) {
			
			email := Email{
				ID:      fmt.Sprintf("%d", emailDB.ID),
				From:    emailDB.From,
				To:      emailDB.To,
				Subject: emailDB.Subject,
				Body:    emailDB.Body,
				Date:    emailDB.Received.Format("2006-01-02 15:04:05"),
				Headers: make(map[string]string),
			}
			results = append(results, email)
		}
	}
	
	return results
}

// BackupData 备份数据
func (es *EmailStorage) BackupData(backupDir string) error {
	// 备份功能应该直接备份SQLite数据库文件
	// 这里只做一个简单的占位实现
	fmt.Printf("Backup functionality should backup the SQLite database directly\n")
	return nil
}



// loadEmails 从数据库加载邮件到内存缓存
func (es *EmailStorage) loadEmails() {
	if es.database == nil {
		return
	}
	
	// 获取所有邮箱名称
	mailboxes, err := es.database.GetAllMailboxNames()
	if err != nil {
		fmt.Printf("Warning: Failed to load mailboxes: %v\n", err)
		return
	}
	
	// 为每个邮箱加载邮件到内存缓存
	for _, mailbox := range mailboxes {
		emails := es.GetEmails(mailbox)
		es.emails[mailbox] = emails
	}
}



