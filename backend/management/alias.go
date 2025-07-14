package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
)

type AliasManager struct {
	aliases   map[string]string // alias -> realEmail
	forwards  map[string][]string // realEmail -> []forwardTo
	aliasesMu sync.RWMutex
	filename  string
}

type AliasConfig struct {
	Aliases  map[string]string   `json:"aliases"`
	Forwards map[string][]string `json:"forwards"`
}

func NewAliasManager(filename string) *AliasManager {
	am := &AliasManager{
		aliases:  make(map[string]string),
		forwards: make(map[string][]string),
		filename: filename,
	}
	am.loadFromFile()
	return am
}

// AddAlias 添加邮箱别名
func (am *AliasManager) AddAlias(alias, realEmail string) error {
	am.aliasesMu.Lock()
	defer am.aliasesMu.Unlock()
	
	am.aliases[alias] = realEmail
	return am.saveToFile()
}

// RemoveAlias 删除邮箱别名
func (am *AliasManager) RemoveAlias(alias string) error {
	am.aliasesMu.Lock()
	defer am.aliasesMu.Unlock()
	
	delete(am.aliases, alias)
	return am.saveToFile()
}

// AddForward 添加邮件转发
func (am *AliasManager) AddForward(email, forwardTo string) error {
	am.aliasesMu.Lock()
	defer am.aliasesMu.Unlock()
	
	if am.forwards[email] == nil {
		am.forwards[email] = make([]string, 0)
	}
	
	// 检查是否已存在
	for _, existing := range am.forwards[email] {
		if existing == forwardTo {
			return nil // 已存在，不重复添加
		}
	}
	
	am.forwards[email] = append(am.forwards[email], forwardTo)
	return am.saveToFile()
}

// RemoveForward 删除邮件转发
func (am *AliasManager) RemoveForward(email, forwardTo string) error {
	am.aliasesMu.Lock()
	defer am.aliasesMu.Unlock()
	
	if am.forwards[email] == nil {
		return nil
	}
	
	// 移除指定的转发地址
	newForwards := make([]string, 0)
	for _, existing := range am.forwards[email] {
		if existing != forwardTo {
			newForwards = append(newForwards, existing)
		}
	}
	
	if len(newForwards) == 0 {
		delete(am.forwards, email)
	} else {
		am.forwards[email] = newForwards
	}
	
	return am.saveToFile()
}

// ResolveEmail 解析邮箱地址，返回真实邮箱地址
func (am *AliasManager) ResolveEmail(email string) string {
	am.aliasesMu.RLock()
	defer am.aliasesMu.RUnlock()
	
	if realEmail, exists := am.aliases[email]; exists {
		return realEmail
	}
	return email // 如果没有别名，返回原地址
}

// GetForwards 获取邮箱的转发地址列表
func (am *AliasManager) GetForwards(email string) []string {
	am.aliasesMu.RLock()
	defer am.aliasesMu.RUnlock()
	
	// 先解析别名
	resolvedEmail := am.ResolveEmail(email)
	
	if forwards, exists := am.forwards[resolvedEmail]; exists {
		return forwards
	}
	return nil
}

// GetAllAliases 获取所有别名
func (am *AliasManager) GetAllAliases() map[string]string {
	am.aliasesMu.RLock()
	defer am.aliasesMu.RUnlock()
	
	result := make(map[string]string)
	for alias, real := range am.aliases {
		result[alias] = real
	}
	return result
}

// GetAllForwards 获取所有转发规则
func (am *AliasManager) GetAllForwards() map[string][]string {
	am.aliasesMu.RLock()
	defer am.aliasesMu.RUnlock()
	
	result := make(map[string][]string)
	for email, forwards := range am.forwards {
		result[email] = make([]string, len(forwards))
		copy(result[email], forwards)
	}
	return result
}

// IsValidEmail 检查邮箱是否属于当前域
func (am *AliasManager) IsValidEmail(email, domain string) bool {
	if email == "" {
		return false
	}
	
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	
	return parts[1] == domain
}

// ProcessEmail 处理邮件，考虑别名和转发
func (am *AliasManager) ProcessEmail(email Email, sender *SMTPSender) (string, []string) {
	// 解析收件人
	realEmail := am.ResolveEmail(email.To)
	
	// 获取转发地址
	forwards := am.GetForwards(realEmail)
	
	// 如果有转发地址，发送转发邮件
	if len(forwards) > 0 && sender != nil {
		for _, forwardTo := range forwards {
			go func(to string) {
				err := sender.SendEmail(email.From, to, 
					"[转发] "+email.Subject, 
					"此邮件从 "+email.To+" 转发\n\n原邮件内容:\n"+email.Body)
				if err != nil {
					log.Printf("转发邮件失败 %s -> %s: %v", email.To, to, err)
				} else {
					log.Printf("邮件转发成功 %s -> %s", email.To, to)
				}
			}(forwardTo)
		}
	}
	
	return realEmail, forwards
}

func (am *AliasManager) loadFromFile() error {
	if am.filename == "" {
		return nil
	}
	
	data, err := os.ReadFile(am.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，使用默认配置
		}
		return err
	}
	
	var config AliasConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	
	am.aliases = config.Aliases
	am.forwards = config.Forwards
	
	if am.aliases == nil {
		am.aliases = make(map[string]string)
	}
	if am.forwards == nil {
		am.forwards = make(map[string][]string)
	}
	
	return nil
}

func (am *AliasManager) saveToFile() error {
	if am.filename == "" {
		return nil
	}
	
	config := AliasConfig{
		Aliases:  am.aliases,
		Forwards: am.forwards,
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(am.filename, data, 0644)
}