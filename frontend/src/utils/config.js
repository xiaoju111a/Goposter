// 系统配置管理模块
class ConfigManager {
  constructor() {
    this.config = null;
    this.loaded = false;
  }

  // 获取系统配置
  async getConfig() {
    if (this.loaded && this.config) {
      return this.config;
    }

    try {
      const response = await fetch('/api/config');
      if (!response.ok) {
        throw new Error('Failed to fetch config');
      }
      
      this.config = await response.json();
      this.loaded = true;
      return this.config;
    } catch (error) {
      console.error('Error fetching config:', error);
      // 返回默认配置
      this.config = {
        domain: 'goposter.fun',
        hostname: 'localhost',
        admin_email: 'admin@goposter.fun',
        app_name: 'YGoCard Mail',
        version: '1.0.0'
      };
      return this.config;
    }
  }

  // 获取域名
  async getDomain() {
    const config = await this.getConfig();
    return config.domain;
  }

  // 获取管理员邮箱
  async getAdminEmail() {
    const config = await this.getConfig();
    return config.admin_email;
  }

  // 获取应用名称
  async getAppName() {
    const config = await this.getConfig();
    return config.app_name;
  }

  // 格式化邮箱地址（如果没有域名则自动添加）
  async formatEmailAddress(username) {
    const domain = await this.getDomain();
    return username.includes('@') ? username : `${username}@${domain}`;
  }

  // 检查是否为管理员邮箱
  async isAdminEmail(email) {
    const adminEmail = await this.getAdminEmail();
    return email === adminEmail;
  }

  // 清除缓存（用于测试或配置更新）
  clearCache() {
    this.config = null;
    this.loaded = false;
  }
}

// 创建全局配置管理实例
const configManager = new ConfigManager();

export default configManager;