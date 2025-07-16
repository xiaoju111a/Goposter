package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// SMTPRelayManager SMTP中继配置管理器
type SMTPRelayManager struct {
	configFile string
	config     SMTPRelayConfig
	relay      *SMTPRelay
}

// NewSMTPRelayManager 创建SMTP中继管理器
func NewSMTPRelayManager(configFile string) *SMTPRelayManager {
	manager := &SMTPRelayManager{
		configFile: configFile,
	}
	
	// 加载配置
	if err := manager.LoadConfig(); err != nil {
		log.Printf("加载SMTP中继配置失败，使用默认配置: %v", err)
		manager.config = TencentCloudSESConfig
	}
	
	// 创建中继实例
	manager.relay = NewSMTPRelay(manager.config)
	
	return manager
}

// LoadConfig 加载配置
func (m *SMTPRelayManager) LoadConfig() error {
	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat(m.configFile); os.IsNotExist(err) {
		return m.SaveConfig()
	}
	
	data, err := ioutil.ReadFile(m.configFile)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}
	
	err = json.Unmarshal(data, &m.config)
	if err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}
	
	log.Printf("SMTP中继配置加载成功: %s:%d (启用: %v)", m.config.Host, m.config.Port, m.config.Enabled)
	return nil
}

// SaveConfig 保存配置
func (m *SMTPRelayManager) SaveConfig() error {
	// 确保目录存在
	dir := filepath.Dir(m.configFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}
	
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}
	
	err = ioutil.WriteFile(m.configFile, data, 0644)
	if err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}
	
	log.Printf("SMTP中继配置保存成功: %s", m.configFile)
	return nil
}

// GetConfig 获取当前配置
func (m *SMTPRelayManager) GetConfig() SMTPRelayConfig {
	return m.config
}

// UpdateConfig 更新配置
func (m *SMTPRelayManager) UpdateConfig(config SMTPRelayConfig) error {
	// 验证配置
	testRelay := NewSMTPRelay(config)
	if err := testRelay.ValidateConfig(); err != nil {
		return fmt.Errorf("配置验证失败: %v", err)
	}
	
	// 更新配置
	m.config = config
	m.relay = NewSMTPRelay(m.config)
	
	// 保存配置
	return m.SaveConfig()
}

// GetRelay 获取SMTP中继实例
func (m *SMTPRelayManager) GetRelay() *SMTPRelay {
	return m.relay
}

// EnableRelay 启用中继
func (m *SMTPRelayManager) EnableRelay() error {
	if m.config.Username == "" || m.config.Password == "" {
		return fmt.Errorf("用户名和密码不能为空")
	}
	
	m.config.Enabled = true
	m.relay = NewSMTPRelay(m.config)
	return m.SaveConfig()
}

// DisableRelay 禁用中继
func (m *SMTPRelayManager) DisableRelay() error {
	m.config.Enabled = false
	m.relay = NewSMTPRelay(m.config)
	return m.SaveConfig()
}

// TestConnection 测试连接
func (m *SMTPRelayManager) TestConnection() error {
	if !m.config.Enabled {
		return fmt.Errorf("SMTP中继未启用")
	}
	return m.relay.TestConnection()
}

// SetPresetProvider 设置预设提供商
func (m *SMTPRelayManager) SetPresetProvider(provider, username, password string) error {
	preset, exists := GetPresetConfig(provider)
	if !exists {
		return fmt.Errorf("不支持的提供商: %s", provider)
	}
	
	// 复制预设配置并设置认证信息
	m.config = preset
	m.config.Username = username
	m.config.Password = password
	
	// 创建新的中继实例
	m.relay = NewSMTPRelay(m.config)
	
	return m.SaveConfig()
}

// GetAvailableProviders 获取可用的提供商列表
func (m *SMTPRelayManager) GetAvailableProviders() map[string]SMTPRelayConfig {
	return PresetConfigs
}

// GetStatus 获取中继状态
func (m *SMTPRelayManager) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"enabled":    m.config.Enabled,
		"host":       m.config.Host,
		"port":       m.config.Port,
		"username":   m.config.Username,
		"use_tls":    m.config.UseTLS,
		"has_password": m.config.Password != "",
	}
	
	// 如果启用了，测试连接状态
	if m.config.Enabled {
		err := m.TestConnection()
		status["connection_ok"] = err == nil
		if err != nil {
			status["connection_error"] = err.Error()
		}
	} else {
		status["connection_ok"] = false
		status["connection_error"] = "中继未启用"
	}
	
	return status
}