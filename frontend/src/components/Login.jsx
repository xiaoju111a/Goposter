import React, { useState, useEffect } from 'react';
import { login } from '../utils/auth.js';
import configManager from '../utils/config.js';

const Login = ({ onLoginSuccess }) => {
  const [credentials, setCredentials] = useState({
    username: '',
    password: '',
    twoFactorCode: ''
  });
  const [loading, setLoading] = useState(false);
  const [config, setConfig] = useState({
    domain: 'freeagent.live',
    admin_email: 'admin@freeagent.live'
  });
  const [error, setError] = useState('');
  const [step, setStep] = useState(1); // 1: 基础登录, 2: 2FA验证
  const [requires2FA, setRequires2FA] = useState(false);
  const [loginAttempts, setLoginAttempts] = useState(0);
  const [lockoutTime, setLockoutTime] = useState(null);
  const [passwordStrength, setPasswordStrength] = useState({ score: 0, feedback: [] });
  const [showPassword, setShowPassword] = useState(false);

  // 密码强度检查
  const checkPasswordStrength = (password) => {
    const feedback = [];
    let score = 0;
    
    if (password.length >= 8) score++; else feedback.push('至少8个字符');
    if (/[a-z]/.test(password)) score++; else feedback.push('包含小写字母');
    if (/[A-Z]/.test(password)) score++; else feedback.push('包含大写字母');
    if (/\d/.test(password)) score++; else feedback.push('包含数字');
    if (/[^\w\s]/.test(password)) score++; else feedback.push('包含特殊字符');
    
    return { score, feedback };
  };

  useEffect(() => {
    if (credentials.password) {
      setPasswordStrength(checkPasswordStrength(credentials.password));
    }
  }, [credentials.password]);

  // 加载配置
  useEffect(() => {
    const loadConfig = async () => {
      try {
        const configData = await configManager.getConfig();
        setConfig(configData);
      } catch (error) {
        console.error('Failed to load config:', error);
      }
    };
    loadConfig();
  }, []);

  // 检查锁定状态
  useEffect(() => {
    const checkLockout = () => {
      const lockoutData = localStorage.getItem('loginLockout');
      if (lockoutData) {
        const { attempts, lockedUntil } = JSON.parse(lockoutData);
        const now = new Date().getTime();
        
        if (lockedUntil && now < lockedUntil) {
          setLockoutTime(new Date(lockedUntil));
          setLoginAttempts(attempts);
        } else if (lockedUntil && now >= lockedUntil) {
          // 锁定时间已过，清除锁定状态
          localStorage.removeItem('loginLockout');
          setLockoutTime(null);
          setLoginAttempts(0);
        } else {
          setLoginAttempts(attempts || 0);
        }
      }
    };
    
    checkLockout();
    const interval = setInterval(checkLockout, 1000);
    return () => clearInterval(interval);
  }, []);

  const handleSubmit = async (e) => {
    e.preventDefault();
    
    // 检查是否被锁定
    if (lockoutTime && new Date().getTime() < lockoutTime.getTime()) {
      setError(`账户已锁定，请在 ${lockoutTime.toLocaleTimeString()} 后重试`);
      return;
    }
    
    if (step === 1) {
      // 第一步：基础验证
      if (!credentials.username || !credentials.password) {
        setError('请填写用户名和密码');
        return;
      }
      
      // 密码强度检查（管理员账户允许弱密码）
      if (passwordStrength.score < 3 && credentials.username !== config.admin_email && credentials.username !== 'admin') {
        setError('密码强度不足：' + passwordStrength.feedback.join('、'));
        return;
      }
    } else if (step === 2) {
      // 第二步：2FA验证
      if (!credentials.twoFactorCode || credentials.twoFactorCode.length !== 6) {
        setError('请输入6位验证码');
        return;
      }
    }

    setLoading(true);
    setError('');

    try {
      const loginData = {
        username: credentials.username,
        password: credentials.password
      };
      
      if (step === 2) {
        loginData.twoFactorCode = credentials.twoFactorCode;
      }
      
      const data = await login(loginData.username, loginData.password, loginData.twoFactorCode);
      
      // 清除登录失败记录
      localStorage.removeItem('loginLockout');
      setLoginAttempts(0);
      setLockoutTime(null);
      
      // 通知父组件登录成功
      onLoginSuccess(data);
    } catch (error) {
      // 处理不同类型的错误
      if (error.message.includes('2fa_required') || error.message.includes('two-factor code required')) {
        setRequires2FA(true);
        setStep(2);
        setError('请输入双因素认证码');
      } else if (error.message.includes('Invalid 2FA code') || error.message.includes('invalid two-factor code')) {
        setError('验证码错误，请重新输入');
      } else {
        // 记录登录失败
        const newAttempts = loginAttempts + 1;
        setLoginAttempts(newAttempts);
        
        if (newAttempts >= 5) {
          // 锁定账户30分钟
          const lockedUntil = new Date().getTime() + 30 * 60 * 1000;
          localStorage.setItem('loginLockout', JSON.stringify({
            attempts: newAttempts,
            lockedUntil
          }));
          setLockoutTime(new Date(lockedUntil));
          setError('登录失败次数过多，账户已锁定30分钟');
        } else {
          localStorage.setItem('loginLockout', JSON.stringify({ attempts: newAttempts }));
          setError(`登录失败 (${newAttempts}/5次)：${error.message || '请检查用户名和密码'}`);
        }
      }
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

  const handleBackToStep1 = () => {
    setStep(1);
    setRequires2FA(false);
    setCredentials(prev => ({ ...prev, twoFactorCode: '' }));
    setError('');
  };

  const getPasswordStrengthColor = () => {
    if (passwordStrength.score >= 4) return '#10b981'; // 绿色
    if (passwordStrength.score >= 3) return '#f59e0b'; // 黄色
    return '#ef4444'; // 红色
  };

  const getPasswordStrengthText = () => {
    if (passwordStrength.score >= 4) return '强';
    if (passwordStrength.score >= 3) return '中等';
    if (passwordStrength.score >= 1) return '弱';
    return '很弱';
  };

  const formatTimeRemaining = () => {
    if (!lockoutTime) return '';
    const now = new Date().getTime();
    const remaining = lockoutTime.getTime() - now;
    const minutes = Math.floor(remaining / 60000);
    const seconds = Math.floor((remaining % 60000) / 1000);
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  };

  return (
    <div className="login-container">
      <div className="login-card">
        <div className="login-header">
          <h2>🔐 {step === 1 ? '登录 FreeAgent 邮箱' : '双因素认证'}</h2>
          <p>{step === 1 ? '请输入您的管理员凭据' : '请输入6位验证码'}</p>
          {step === 2 && (
            <button 
              type="button" 
              className="back-btn"
              onClick={handleBackToStep1}
            >
              ← 返回
            </button>
          )}
        </div>

        <form onSubmit={handleSubmit} className="login-form">
          {step === 1 && (
            <>
              {/* 第一步：基础登录 */}
              <div className="form-group">
                <label htmlFor="username">用户名</label>
                <input
                  type="text"
                  id="username"
                  value={credentials.username}
                  onChange={handleChange('username')}
                  placeholder="输入用户名或邮箱"
                  required
                  disabled={loading || lockoutTime}
                />
              </div>

              <div className="form-group">
                <label htmlFor="password">密码</label>
                <div className="password-input-container">
                  <input
                    type={showPassword ? "text" : "password"}
                    id="password"
                    value={credentials.password}
                    onChange={handleChange('password')}
                    placeholder="输入密码"
                    required
                    disabled={loading || lockoutTime}
                  />
                  <button
                    type="button"
                    className="password-toggle"
                    onClick={() => setShowPassword(!showPassword)}
                    disabled={loading}
                  >
                    {showPassword ? '👁️' : '🙈'}
                  </button>
                </div>
                
                {/* 密码强度指示器 */}
                {credentials.password && (
                  <div className="password-strength">
                    <div className="strength-bar">
                      <div 
                        className="strength-fill"
                        style={{
                          width: `${(passwordStrength.score / 5) * 100}%`,
                          backgroundColor: getPasswordStrengthColor()
                        }}
                      ></div>
                    </div>
                    <div className="strength-text" style={{ color: getPasswordStrengthColor() }}>
                      密码强度: {getPasswordStrengthText()}
                    </div>
                    {passwordStrength.feedback.length > 0 && (
                      <div className="strength-feedback">
                        <small>建议: {passwordStrength.feedback.join('、')}</small>
                      </div>
                    )}
                  </div>
                )}
              </div>
              
              {/* 登录失败信息 */}
              {loginAttempts > 0 && !lockoutTime && (
                <div className="warning-message">
                  ⚠️ 登录失败 {loginAttempts}/5 次，{5 - loginAttempts} 次后将锁定账户
                </div>
              )}
              
              {/* 锁定状态 */}
              {lockoutTime && (
                <div className="lockout-message">
                  🔒 账户已锁定，剩余时间: {formatTimeRemaining()}
                </div>
              )}
            </>
          )}
          
          {step === 2 && (
            <>
              {/* 第二步：2FA验证 */}
              <div className="user-info">
                <p>正在为 <strong>{credentials.username}</strong> 进行双因素认证</p>
              </div>
              
              <div className="form-group">
                <label htmlFor="twoFactorCode">验证码</label>
                <input
                  type="text"
                  id="twoFactorCode"
                  value={credentials.twoFactorCode}
                  onChange={handleChange('twoFactorCode')}
                  placeholder="输入6位验证码"
                  maxLength="6"
                  pattern="[0-9]{6}"
                  required
                  disabled={loading}
                  className="totp-input"
                  autoComplete="one-time-code"
                />
              </div>
              
              <div className="totp-help">
                <p><small>📱 请打开 Google Authenticator 或其他 TOTP 应用获取验证码</small></p>
              </div>
            </>
          )}

          {error && (
            <div className="error-message">
              ❌ {error}
            </div>
          )}

          <button 
            type="submit" 
            disabled={loading || (lockoutTime && new Date().getTime() < lockoutTime.getTime())}
            className={`login-btn ${step === 2 ? 'verify-btn' : ''}`}
          >
            {loading ? (
              step === 1 ? '登录中...' : '验证中...'
            ) : (
              step === 1 ? '🚀 登录' : '✓ 验证并登录'
            )}
          </button>
        </form>

        {step === 1 && (
          <div className="login-footer">
            <div className="default-credentials">
              <h4>💡 默认管理员账户</h4>
              <p><strong>用户名:</strong> admin</p>
              <p><strong>邮箱:</strong> {config.admin_email}</p>
              <p><strong>密码:</strong> admin123</p>
              <p><small>用户名会自动转换为完整邮箱地址</small></p>
            </div>
            
            <div className="security-features">
              <h4>🔒 安全特性</h4>
              <ul>
                <li>✓ 双因素认证 (2FA) 支持</li>
                <li>✓ JWT 令牌系统</li>
                <li>✓ 密码强度检测</li>
                <li>✓ 登录失败防护</li>
                <li>✓ AES-256-GCM 加密</li>
              </ul>
            </div>
          </div>
        )}
        
        {step === 2 && (
          <div className="login-footer">
            <div className="security-notice">
              <p><small>🔒 您的账户已启用双因素认证，请输入验证码完成登录</small></p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default Login;