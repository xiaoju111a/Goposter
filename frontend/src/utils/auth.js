// 认证状态管理工具

export const auth = {
  // 检查是否已登录
  isAuthenticated() {
    const sessionId = localStorage.getItem('sessionId');
    const userEmail = localStorage.getItem('userEmail');
    return !!(sessionId && userEmail);
  },

  // 获取当前用户信息
  getCurrentUser() {
    if (!this.isAuthenticated()) return null;
    
    return {
      sessionId: localStorage.getItem('sessionId'),
      email: localStorage.getItem('userEmail'),
      username: localStorage.getItem('userEmail')?.split('@')[0] || ''
    };
  },

  // 获取session ID用于API调用
  getSessionId() {
    return localStorage.getItem('sessionId');
  },

  // 设置认证信息
  setAuth(sessionId, userEmail) {
    localStorage.setItem('sessionId', sessionId);
    localStorage.setItem('userEmail', userEmail);
  },

  // 清除认证信息（登出）
  clearAuth() {
    localStorage.removeItem('sessionId');
    localStorage.removeItem('userEmail');
  },

  // 登出API调用
  async logout() {
    const sessionId = this.getSessionId();
    if (!sessionId) return;

    try {
      await fetch('/api/logout', {
        method: 'POST',
        headers: {
          'Session-ID': sessionId,
          'Content-Type': 'application/json'
        }
      });
    } catch (error) {
      console.error('登出API调用失败:', error);
    } finally {
      // 无论API调用是否成功，都清除本地认证信息
      this.clearAuth();
    }
  },

  // 获取认证头信息
  getAuthHeaders() {
    const sessionId = this.getSessionId();
    if (!sessionId) return {};

    return {
      'Session-ID': sessionId
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
  notify(isAuthenticated, user) {
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

// 登录函数
export async function login(username, password) {
  // 如果username不包含@，则添加@freeagent.live后缀
  const email = username.includes('@') ? username : `${username}@freeagent.live`;
  
  const response = await fetch('/api/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email, password })
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(errorText || '登录失败');
  }

  const data = await response.json();
  
  // 保存认证信息
  auth.setAuth(data.session_id, data.email || email);
  
  // 通知状态变化
  authListener.notify(true, auth.getCurrentUser());
  
  return data;
}

// 登出函数
export async function logout() {
  await auth.logout();
  
  // 通知状态变化
  authListener.notify(false, null);
}

// 检查认证状态的Hook（用于React组件）
// 注意：这个函数需要在React组件中使用时，确保已经导入React
export function useAuth() {
  // 需要在使用时导入React
  // const [isAuthenticated, setIsAuthenticated] = React.useState(auth.isAuthenticated());
  // const [user, setUser] = React.useState(auth.getCurrentUser());

  // React.useEffect(() => {
  //   // 监听认证状态变化
  //   const unsubscribe = authListener.addListener((authenticated, userData) => {
  //     setIsAuthenticated(authenticated);
  //     setUser(userData);
  //   });

  //   return unsubscribe;
  // }, []);

  return {
    isAuthenticated: auth.isAuthenticated(),
    user: auth.getCurrentUser(),
    login,
    logout
  };
}