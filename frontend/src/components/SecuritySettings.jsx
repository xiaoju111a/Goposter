import React, { useState, useEffect } from 'react';
import { twoFactorAuth, securityLogs } from '../utils/auth.js';
import configManager from '../utils/config.js';

const SecuritySettings = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [config, setConfig] = useState({
    admin_email: 'admin@freeagent.live'
  });
  const [success, setSuccess] = useState('');
  const [qrCode, setQrCode] = useState('');
  const [secret, setSecret] = useState('');
  const [verificationCode, setVerificationCode] = useState('');
  const [is2FAEnabled, setIs2FAEnabled] = useState(false);
  const [securityStats, setSecurityStats] = useState(null);
  const [auditLogs, setAuditLogs] = useState([]);

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

  useEffect(() => {
    loadSecurityData();
    load2FAStatus();
  }, []);

  const loadSecurityData = async () => {
    try {
      // 加载安全统计
      const stats = await securityLogs.getStats();
      setSecurityStats(stats);
      
      // 加载审计日志
      const logs = await securityLogs.getAuditLogs(null, 20);
      setAuditLogs(logs);
    } catch (error) {
      console.error('Failed to load security data:', error);
    }
  };

  const load2FAStatus = async () => {
    try {
      const status = await twoFactorAuth.getStatus();
      setIs2FAEnabled(status.enabled || false);
    } catch (error) {
      console.error('Failed to load 2FA status:', error);
      // 如果API调用失败，默认为未启用
      setIs2FAEnabled(false);
    }
  };

  const handleEnable2FA = async () => {
    setLoading(true);
    setError('');
    setSuccess('');

    try {
      const result = await twoFactorAuth.enable();
      setQrCode(result.qr_code);
      setSecret(result.secret);
      setSuccess('2FA已启用！请扫描二维码设置您的认证器应用。');
    } catch (error) {
      setError(error.message);
    } finally {
      setLoading(false);
    }
  };

  const handleDisable2FA = async () => {
    if (!confirm('确定要禁用双因素认证吗？这将降低您的账户安全性。')) {
      return;
    }

    setLoading(true);
    setError('');
    setSuccess('');

    try {
      await twoFactorAuth.disable();
      setIs2FAEnabled(false);
      setQrCode('');
      setSecret('');
      setSuccess('2FA已禁用。');
    } catch (error) {
      setError(error.message);
    } finally {
      setLoading(false);
    }
  };

  const handleVerify2FA = async () => {
    if (!verificationCode || verificationCode.length !== 6) {
      setError('请输入6位验证码');
      return;
    }

    setLoading(true);
    setError('');

    try {
      const userEmail = localStorage.getItem('userEmail') || config.admin_email;
      const result = await twoFactorAuth.verify(userEmail, verificationCode);
      if (result.valid) {
        setIs2FAEnabled(true);
        setQrCode('');
        setSecret('');
        setSuccess('验证成功！双因素认证已激活。');
        setVerificationCode('');
        // 重新加载2FA状态以确保同步
        await load2FAStatus();
      } else {
        setError('验证码无效，请重试。');
      }
    } catch (error) {
      setError(error.message);
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (timestamp) => {
    return new Date(timestamp).toLocaleString();
  };

  const getMaskedEmail = (email) => {
    if (!email || !email.includes('@')) return email;
    const [local, domain] = email.split('@');
    const maskedLocal = local.length > 2 
      ? local[0] + '*'.repeat(local.length - 2) + local[local.length - 1]
      : local;
    return `${maskedLocal}@${domain}`;
  };

  const getActionIcon = (action) => {
    const icons = {
      'LOGIN': '🔑',
      'LOGOUT': '🚪',
      'ENABLE_2FA': '🔐',
      'DISABLE_2FA': '🔓',
      'STORE_EMAIL': '📧',
      'DELETE_EMAIL': '🗑️'
    };
    return icons[action] || '📋';
  };

  return (
    <div className="security-settings">
      <div className="settings-header">
        <h2>🔒 安全设置</h2>
        <p>管理您的账户安全功能和查看安全日志</p>
      </div>

      {error && (
        <div className="error-message">
          ❌ {error}
        </div>
      )}

      {success && (
        <div className="success-message">
          ✅ {success}
        </div>
      )}

      {/* 双因素认证设置 */}
      <div className="setting-section">
        <h3>🔐 双因素认证 (2FA)</h3>
        <p>增加额外的安全层，保护您的账户免受未授权访问</p>
        
        {!is2FAEnabled && !qrCode && (
          <div className="setting-control">
            <button 
              onClick={handleEnable2FA}
              disabled={loading}
              className="enable-btn"
            >
              {loading ? '启用中...' : '启用 2FA'}
            </button>
          </div>
        )}

        {qrCode && !is2FAEnabled && (
          <div className="qr-setup">
            <div className="qr-code-container">
              <h4>设置认证器应用</h4>
              <div className="qr-code">
                <img src={qrCode} alt="2FA QR Code" />
              </div>
              <div className="secret-key">
                <p><strong>手动输入密钥:</strong></p>
                <code>{secret}</code>
              </div>
            </div>
            
            <div className="verification-step">
              <h4>验证设置</h4>
              <p>请输入认证器应用中显示的6位代码：</p>
              <div className="verification-input">
                <input
                  type="text"
                  value={verificationCode}
                  onChange={(e) => setVerificationCode(e.target.value)}
                  placeholder="输入6位验证码"
                  maxLength="6"
                  pattern="[0-9]{6}"
                  className="totp-input"
                />
                <button 
                  onClick={handleVerify2FA}
                  disabled={loading || verificationCode.length !== 6}
                  className="verify-btn"
                >
                  {loading ? '验证中...' : '验证并启用'}
                </button>
              </div>
            </div>
          </div>
        )}

        {is2FAEnabled && (
          <div className="setting-control enabled">
            <div className="status-indicator">
              <span className="status-icon">✅</span>
              <span>双因素认证已启用</span>
            </div>
            <button 
              onClick={handleDisable2FA}
              disabled={loading}
              className="disable-btn"
            >
              {loading ? '禁用中...' : '禁用 2FA'}
            </button>
          </div>
        )}
      </div>

      {/* 安全统计 */}
      {securityStats && (
        <div className="setting-section">
          <h3>📊 安全统计</h3>
          <div className="security-stats">
            <div className="stat-item">
              <span className="stat-label">加密状态:</span>
              <span className={`stat-value ${securityStats.encryption_enabled ? 'enabled' : 'disabled'}`}>
                {securityStats.encryption_enabled ? '✅ 已启用' : '❌ 未启用'}
              </span>
            </div>
            <div className="stat-item">
              <span className="stat-label">审计日志:</span>
              <span className={`stat-value ${securityStats.audit_logging ? 'enabled' : 'disabled'}`}>
                {securityStats.audit_logging ? '✅ 已启用' : '❌ 未启用'}
              </span>
            </div>
            <div className="stat-item">
              <span className="stat-label">Redis缓存:</span>
              <span className={`stat-value ${securityStats.redis_caching ? 'enabled' : 'disabled'}`}>
                {securityStats.redis_caching ? '✅ 已启用' : '❌ 未启用'}
              </span>
            </div>
            {securityStats.encrypted_emails !== undefined && (
              <div className="stat-item">
                <span className="stat-label">加密邮件数:</span>
                <span className="stat-value">{securityStats.encrypted_emails}</span>
              </div>
            )}
            {securityStats.audit_logs !== undefined && (
              <div className="stat-item">
                <span className="stat-label">审计记录数:</span>
                <span className="stat-value">{securityStats.audit_logs}</span>
              </div>
            )}
          </div>
        </div>
      )}

      {/* 安全日志 */}
      <div className="setting-section">
        <h3>📋 最近安全活动</h3>
        <div className="audit-logs">
          {auditLogs.length === 0 ? (
            <p className="no-logs">暂无安全日志记录</p>
          ) : (
            <div className="logs-list">
              {auditLogs.map((log, index) => (
                <div key={index} className="log-item">
                  <div className="log-icon">
                    {getActionIcon(log.action)}
                  </div>
                  <div className="log-content">
                    <div className="log-action">
                      <span className="action-name">{log.action}</span>
                      <span className={`action-status ${log.success ? 'success' : 'failed'}`}>
                        {log.success ? '✅' : '❌'}
                      </span>
                    </div>
                    <div className="log-details">
                      <span className="log-user">{getMaskedEmail(log.user_email)}</span>
                      <span className="log-time">{formatDate(log.created_at)}</span>
                    </div>
                    {log.details && (
                      <div className="log-description">{log.details}</div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* 密码安全提示 */}
      <div className="setting-section">
        <h3>🛡️ 密码安全建议</h3>
        <div className="security-tips">
          <ul>
            <li>✓ 使用至少8个字符的复杂密码</li>
            <li>✓ 包含大小写字母、数字和特殊字符</li>
            <li>✓ 定期更换密码</li>
            <li>✓ 不要在多个网站使用相同密码</li>
            <li>✓ 启用双因素认证增强安全性</li>
          </ul>
        </div>
      </div>
    </div>
  );
};

export default SecuritySettings;