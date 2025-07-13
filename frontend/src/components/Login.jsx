import React, { useState } from 'react';
import { login } from '../utils/auth.js';

const Login = ({ onLoginSuccess }) => {
  const [credentials, setCredentials] = useState({
    username: '',
    password: ''
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    
    if (!credentials.username || !credentials.password) {
      setError('请填写用户名和密码');
      return;
    }

    setLoading(true);
    setError('');

    try {
      // 使用auth.js的login函数，它会处理用户名到邮箱的转换
      const data = await login(credentials.username, credentials.password);
      
      // 通知父组件登录成功
      onLoginSuccess(data);
    } catch (error) {
      setError(error.message || '登录失败，请检查用户名和密码');
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (field) => (e) => {
    setCredentials(prev => ({
      ...prev,
      [field]: e.target.value
    }));
    // 清除错误信息
    if (error) setError('');
  };

  return (
    <div className="login-container">
      <div className="login-card">
        <div className="login-header">
          <h2>🔐 登录 FreeAgent 邮箱</h2>
          <p>请输入您的管理员凭据</p>
        </div>

        <form onSubmit={handleSubmit} className="login-form">
          <div className="form-group">
            <label htmlFor="username">用户名</label>
            <input
              type="text"
              id="username"
              value={credentials.username}
              onChange={handleChange('username')}
              placeholder="输入用户名"
              required
              disabled={loading}
            />
          </div>

          <div className="form-group">
            <label htmlFor="password">密码</label>
            <input
              type="password"
              id="password"
              value={credentials.password}
              onChange={handleChange('password')}
              placeholder="输入密码"
              required
              disabled={loading}
            />
          </div>

          {error && (
            <div className="error-message">
              ❌ {error}
            </div>
          )}

          <button 
            type="submit" 
            disabled={loading}
            className="login-btn"
          >
            {loading ? '登录中...' : '🚀 登录'}
          </button>
        </form>

        <div className="login-footer">
          <div className="default-credentials">
            <h4>💡 默认管理员账户</h4>
            <p><strong>用户名:</strong> admin</p>
            <p><strong>邮箱:</strong> admin@freeagent.live</p>
            <p><strong>密码:</strong> admin123</p>
            <p><small>用户名会自动转换为完整邮箱地址</small></p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Login;