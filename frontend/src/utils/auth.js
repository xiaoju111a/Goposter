// 认证状态管理工具 - 升级为JWT令牌系统，支持2FA和高级安全功能

export const auth = {
  // 检查是否已登录
  isAuthenticated() {
    const accessToken = localStorage.getItem('access_token');
    const userEmail = localStorage.getItem('userEmail');
    const expiresAt = localStorage.getItem('token_expires_at');
    
    if (!accessToken || !userEmail) return false;
    
    // 检查token是否过期
    if (expiresAt) {
      const now = Date.now();
      if (now >= parseInt(expiresAt)) {
        console.warn('Token has expired');
        this.clearAuth();
        return false;
      }
    }
    
    // 验证JWT token格式
    try {
      // JWT应该有3个部分，用.分隔
      const parts = accessToken.split('.');
      if (parts.length === 3) {
        // 尝试解码JWT payload验证用户邮箱
        const payload = JSON.parse(atob(parts[1].replace(/-/g, '+').replace(/_/g, '/')));
        if (payload.email === userEmail) {
          return true;
        }
      }
      // 如果不是JWT格式，尝试简单的base64解码（兼容旧token）
      const decoded = atob(accessToken);
      if (decoded.includes(userEmail)) {
        return true;
      }
    } catch (error) {
      console.warn('Invalid token format:', error);
      this.clearAuth();
      return false;
    }
    
    return true;
  },

  // 解码JWT载荷
  decodeJWTPayload(token) {
    const parts = token.split('.');
    if (parts.length !== 3) throw new Error('Invalid JWT format');
    
    const payload = parts[1];
    const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(decoded);
  },

  // 获取当前用户信息
  getCurrentUser() {
    if (!this.isAuthenticated()) return null;
    
    const accessToken = localStorage.getItem('access_token');
    const userEmail = localStorage.getItem('userEmail');
    const expiresAt = localStorage.getItem('token_expires_at');
    
    return {
      accessToken,
      email: userEmail,
      username: userEmail?.split('@')[0] || '',
      isAdmin: userEmail === 'admin@freeagent.live', // 简单的管理员判断
      exp: expiresAt ? parseInt(expiresAt) / 1000 : null,
      iat: null
    };
  },

  // 获取访问令牌用于API调用
  getAccessToken() {
    return localStorage.getItem('access_token');
  },
  
  // 获取刷新令牌
  getRefreshToken() {
    return localStorage.getItem('refresh_token');
  },

  // 设置认证信息（JWT令牌）
  setAuth(tokenData, userEmail) {
    if (tokenData.access_token) {
      localStorage.setItem('access_token', tokenData.access_token);
    }
    if (tokenData.refresh_token) {
      localStorage.setItem('refresh_token', tokenData.refresh_token);
    }
    if (userEmail) {
      localStorage.setItem('userEmail', userEmail);
    }
    
    // 设置token过期时间
    if (tokenData.expires_in) {
      const expiresAt = Date.now() + (tokenData.expires_in * 1000);
      localStorage.setItem('token_expires_at', expiresAt.toString());
    }
  },

  // 清除认证信息（登出）
  clearAuth() {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('userEmail');
    localStorage.removeItem('token_expires_at');
    // 兼容旧版本
    localStorage.removeItem('sessionId');
  },
  
  // 检查token是否即将过期（5分钟内）
  isTokenExpiringSoon() {
    const expiresAt = localStorage.getItem('token_expires_at');
    if (!expiresAt) return false;
    
    const now = Date.now();
    const expires = parseInt(expiresAt);
    const fiveMinutes = 5 * 60 * 1000;
    
    return (expires - now) < fiveMinutes;
  },

  // 自动刷新令牌
  async refreshToken() {
    const refreshToken = this.getRefreshToken();
    if (!refreshToken) {
      throw new Error('No refresh token available');
    }

    try {
      const response = await fetch('/api/auth/refresh', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ refresh_token: refreshToken })
      });

      if (!response.ok) {
        throw new Error('Token refresh failed');
      }

      const data = await response.json();
      this.setAuth(data, this.getCurrentUser()?.email);
      return data;
    } catch (error) {
      console.error('Token refresh failed:', error);
      this.clearAuth();
      throw error;
    }
  },

  // 登出API调用
  async logout() {
    const accessToken = this.getAccessToken();
    
    if (accessToken) {
      try {
        await fetch('/api/auth/logout', {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${accessToken}`,
            'Content-Type': 'application/json'
          }
        });
      } catch (error) {
        console.error('登出API调用失败:', error);
      }
    }
    
    this.clearAuth();
  },

  // 获取认证头信息
  getAuthHeaders() {
    const accessToken = this.getAccessToken();
    if (!accessToken) return {};

    return {
      'Authorization': `Bearer ${accessToken}`
    };
  }
};

// 认证状态变化监听器
export class AuthListener {
  constructor() {
    this.listeners = [];
  }

  // 添加监听器
  addListener(callback) {
    this.listeners.push(callback);
    return () => {
      this.listeners = this.listeners.filter(l => l !== callback);
    };
  }

  // 触发状态变化
  notifyListeners(isAuthenticated, user) {
    this.listeners.forEach(callback => {
      try {
        callback(isAuthenticated, user);
      } catch (error) {
        console.error('认证状态监听器错误:', error);
      }
    });
  }
}

// 全局认证监听器实例
export const authListener = new AuthListener();

// 登录函数 - 支持2FA和JWT
export const login = async (username, password, twoFactorCode = null) => {
  // 如果username不包含@，自动添加域名
  const email = username.includes('@') ? username : `${username}@freeagent.live`;
  
  try {
    const requestBody = {
      email: email,
      password: password
    };
    
    // 如果提供了2FA代码，包含在请求中
    if (twoFactorCode) {
      requestBody.two_factor_code = twoFactorCode;
    }
    
    const response = await fetch('/api/auth/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(requestBody)
    });

    const data = await response.json();

    if (!response.ok) {
      throw new Error(data.error || data.message || '登录失败');
    }

    // 保存JWT令牌信息
    auth.setAuth(data, email);
    
    // 通知所有监听器
    const userData = { 
      email, 
      accessToken: data.access_token,
      isAdmin: data.is_admin || false
    };
    authListener.notifyListeners(true, userData);
    
    return data;
  } catch (error) {
    console.error('登录失败:', error);
    throw error;
  }
};

// 登出函数
export const logout = async () => {
  await auth.logout();
  authListener.notifyListeners(false, null);
};

// 自动令牌管理
let tokenRefreshInterval = null;

// 启动自动令牌刷新
export const startTokenRefresh = () => {
  if (tokenRefreshInterval) return;
  
  tokenRefreshInterval = setInterval(async () => {
    if (auth.isAuthenticated() && auth.isTokenExpiringSoon()) {
      try {
        await auth.refreshToken();
        console.log('Token refreshed successfully');
      } catch (error) {
        console.error('Token refresh failed:', error);
        // 刷新失败，可能需要重新登录
        authListener.notifyListeners(false, null);
      }
    }
  }, 60000); // 每分钟检查一次
};

// 停止自动令牌刷新
export const stopTokenRefresh = () => {
  if (tokenRefreshInterval) {
    clearInterval(tokenRefreshInterval);
    tokenRefreshInterval = null;
  }
};

// 2FA管理功能
export const twoFactorAuth = {
  // 启用2FA
  async enable() {
    const accessToken = auth.getAccessToken();
    const userEmail = localStorage.getItem('userEmail') || 'admin@freeagent.live';
    
    // 优先使用JWT token，如果没有则使用查询参数
    const useToken = accessToken && accessToken.length > 50;
    
    const url = useToken 
      ? '/api/auth/2fa/enable'
      : `/api/auth/2fa/enable?email=${encodeURIComponent(userEmail)}`;
    
    const headers = {
      'Content-Type': 'application/json'
    };
    
    if (useToken) {
      headers['Authorization'] = `Bearer ${accessToken}`;
    }

    const response = await fetch(url, {
      method: 'POST',
      headers
    });

    if (!response.ok) {
      let errorMessage = '启用2FA失败';
      try {
        const responseText = await response.text();
        if (responseText) {
          try {
            const error = JSON.parse(responseText);
            errorMessage = error.message || error.error || errorMessage;
          } catch (e) {
            errorMessage = responseText;
          }
        }
      } catch (e) {
        console.error('Error reading response:', e);
      }
      throw new Error(errorMessage);
    }

    const data = await response.json();
    
    // 如果没有真正的QR码，生成一个
    if (!data.qr_code || data.qr_code.startsWith('Generate QR code')) {
      const issuer = 'FreeAgent Mail';
      const accountName = auth.getCurrentUser()?.email || 'user@freeagent.live';
      const otpUrl = `otpauth://totp/${issuer}:${accountName}?secret=${data.secret}&issuer=${issuer}`;
      
      // 使用在线QR码生成服务
      data.qr_code = `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(otpUrl)}`;
    }
    
    return data;
  },

  // 禁用2FA
  async disable() {
    const accessToken = auth.getAccessToken();
    const userEmail = localStorage.getItem('userEmail') || 'admin@freeagent.live';
    
    // 优先使用JWT token，如果没有则使用查询参数
    const useToken = accessToken && accessToken.length > 50;
    
    const url = useToken 
      ? '/api/auth/2fa/disable'
      : `/api/auth/2fa/disable?email=${encodeURIComponent(userEmail)}`;
    
    const headers = {
      'Content-Type': 'application/json'
    };
    
    if (useToken) {
      headers['Authorization'] = `Bearer ${accessToken}`;
    }

    const response = await fetch(url, {
      method: 'POST',
      headers
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || '禁用2FA失败');
    }

    return await response.json();
  },

  // 验证2FA代码
  async verify(email, code) {
    const response = await fetch('/api/auth/2fa/verify', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ email, code })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || '验证失败');
    }

    return await response.json();
  },

  // 获取用户2FA状态
  async getStatus() {
    const accessToken = auth.getAccessToken();
    const userEmail = localStorage.getItem('userEmail') || 'admin@freeagent.live';
    
    // 优先使用JWT token，如果没有则使用查询参数
    const useToken = accessToken && accessToken.length > 50; // JWT token应该比较长
    
    const url = useToken 
      ? '/api/auth/2fa/status'
      : `/api/auth/2fa/status?email=${encodeURIComponent(userEmail)}`;
    
    const headers = {
      'Content-Type': 'application/json'
    };
    
    if (useToken) {
      headers['Authorization'] = `Bearer ${accessToken}`;
    }
    
    const response = await fetch(url, {
      method: 'GET',
      headers
    });

    if (!response.ok) {
      // 如果API不存在，尝试从用户信息推断
      if (response.status === 404) {
        return { enabled: false };
      }
      const error = await response.json();
      throw new Error(error.message || '获取2FA状态失败');
    }

    return await response.json();
  }
};

// 安全日志查看
export const securityLogs = {
  // 获取安全统计
  async getStats() {
    const accessToken = auth.getAccessToken();
    if (!accessToken) throw new Error('Not authenticated');

    const response = await fetch('/api/security/stats', {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    });

    if (!response.ok) {
      // 如果API不存在，返回默认值
      if (response.status === 404) {
        return {
          encryption_enabled: true,
          audit_logging: true,
          redis_caching: false,
          encrypted_emails: 0,
          audit_logs: 0
        };
      }
      throw new Error('获取安全统计失败');
    }

    return await response.json();
  },

  // 获取审计日志
  async getAuditLogs(userEmail = null, limit = 100) {
    const accessToken = auth.getAccessToken();
    if (!accessToken) throw new Error('Not authenticated');

    const params = new URLSearchParams({ limit: limit.toString() });
    if (userEmail) params.append('user_email', userEmail);

    const response = await fetch(`/api/security/audit-logs?${params}`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    });

    if (!response.ok) {
      // 如果API不存在，返回空数组
      if (response.status === 404) {
        return [];
      }
      throw new Error('获取审计日志失败');
    }

    return await response.json();
  }
};

// 在页面加载时启动令牌管理
if (typeof window !== 'undefined') {
  // 页面加载时启动
  startTokenRefresh();
  
  // 页面卸载时停止
  window.addEventListener('beforeunload', stopTokenRefresh);
}

// 检查认证状态的Hook（用于React组件）
export function useAuth() {
  return {
    isAuthenticated: auth.isAuthenticated(),
    user: auth.getCurrentUser(),
    login,
    logout,
    twoFactorAuth,
    securityLogs
  };
}