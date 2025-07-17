import React, { useState, useEffect } from 'react';
import { sendEmail } from '../utils/api';
import configManager from '../utils/config.js';

const SendEmail = ({ userEmail = '' }) => {
  const [config, setConfig] = useState({
    admin_email: 'admin@ygocard.org',
    domain: 'ygocard.org'
  });
  const [emailData, setEmailData] = useState({
    from: userEmail || config.admin_email,
    to: '',
    subject: '',
    body: ''
  });
  const [relayStatus, setRelayStatus] = useState(null);
  const [relayLoading, setRelayLoading] = useState(true);
  
  // 加载配置和SMTP中继状态
  useEffect(() => {
    const loadConfig = async () => {
      try {
        const configData = await configManager.getConfig();
        setConfig(configData);
        // 如果没有userEmail，使用管理员邮箱
        if (!userEmail) {
          setEmailData(prev => ({
            ...prev,
            from: configData.admin_email
          }));
        }
      } catch (error) {
        console.error('Failed to load config:', error);
      }
    };
    
    const loadRelayStatus = async () => {
      try {
        const token = localStorage.getItem('access_token');
        if (!token) {
          setRelayLoading(false);
          return;
        }
        
        const response = await fetch('/api/relay/status', {
          headers: {
            'Authorization': `Bearer ${token}`
          }
        });
        
        if (response.ok) {
          const data = await response.json();
          setRelayStatus(data);
        } else {
          console.error('Failed to load relay status:', response.status);
        }
      } catch (error) {
        console.error('Failed to load relay status:', error);
      } finally {
        setRelayLoading(false);
      }
    };
    
    loadConfig();
    loadRelayStatus();
  }, [userEmail]);
  
  // 当userEmail变化时更新from字段
  React.useEffect(() => {
    if (userEmail) {
      setEmailData(prev => ({
        ...prev,
        from: userEmail
      }));
    }
  }, [userEmail]);
  const [sending, setSending] = useState(false);
  const [message, setMessage] = useState('');

  // 检查邮箱是否为外部邮箱
  const isExternalEmail = (email) => {
    if (!email || !config.domain) return false;
    return !email.endsWith(`@${config.domain}`);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!emailData.from || !emailData.to || !emailData.subject || !emailData.body) {
      setMessage('请填写所有必填字段');
      return;
    }

    // 检查是否为外部邮箱且没有配置SMTP中继
    if (isExternalEmail(emailData.to)) {
      if (!relayStatus || !relayStatus.enabled) {
        setMessage('⚠️ 发送外部邮件需要配置SMTP中继服务。请联系管理员配置SMTP中继后再发送外部邮件。');
        return;
      }
      
      if (!relayStatus.connection_ok) {
        setMessage('⚠️ SMTP中继服务连接异常，无法发送外部邮件。请联系管理员检查SMTP中继配置。');
        return;
      }
    }

    setSending(true);
    setMessage('');

    try {
      await sendEmail(emailData);
      setMessage('✅ 邮件发送成功！');
      // 清空表单（除了发件人）
      setEmailData({
        from: emailData.from,
        to: '',
        subject: '',
        body: ''
      });
    } catch (error) {
      setMessage(`❌ 发送失败: ${error.message}`);
    } finally {
      setSending(false);
    }
  };

  const handleChange = (field) => (e) => {
    setEmailData(prev => ({
      ...prev,
      [field]: e.target.value
    }));
  };

  return (
    <div className="send-email-container">
      <h3>📧 发送邮件</h3>
      
      {/* SMTP中继状态指示器 */}
      {!relayLoading && (
        <div className="smtp-relay-status">
          <div className="status-indicator">
            <span className="status-label">SMTP中继状态:</span>
            {relayStatus && relayStatus.enabled ? (
              <span className={`status-badge ${relayStatus.connection_ok ? 'success' : 'error'}`}>
                {relayStatus.connection_ok ? '✅ 正常' : '❌ 异常'}
              </span>
            ) : (
              <span className="status-badge disabled">⚠️ 未配置</span>
            )}
          </div>
          <div className="status-description">
            {relayStatus && relayStatus.enabled && relayStatus.connection_ok ? (
              <small>✅ 可以发送外部邮件</small>
            ) : (
              <small>⚠️ 仅支持内部邮件（@{config.domain}）</small>
            )}
          </div>
        </div>
      )}
      
      <form onSubmit={handleSubmit} className="send-email-form">
        <div className="form-row">
          <div className="form-group">
            <label htmlFor="from">发件人 *</label>
            <input
              type="email"
              id="from"
              value={emailData.from}
              onChange={handleChange('from')}
              placeholder={`your-name@${config.domain || 'ygocard.org'}`}
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="to">收件人 *</label>
            <input
              type="email"
              id="to"
              value={emailData.to}
              onChange={handleChange('to')}
              placeholder={`recipient@${config.domain || 'example.com'}`}
              required
            />
            {/* 动态提示信息 */}
            {emailData.to && (
              <div className="recipient-hint">
                {isExternalEmail(emailData.to) ? (
                  <small style={{ color: '#f39c12' }}>
                    ⚠️ 外部邮件 - 需要SMTP中继服务
                  </small>
                ) : (
                  <small style={{ color: '#27ae60' }}>
                    ✅ 内部邮件 - 可直接发送
                  </small>
                )}
              </div>
            )}
          </div>
        </div>

        <div className="form-row single">
          <div className="form-group">
            <label htmlFor="subject">主题 *</label>
            <input
              type="text"
              id="subject"
              value={emailData.subject}
              onChange={handleChange('subject')}
              placeholder="邮件主题"
              required
            />
          </div>
        </div>

        <div className="form-row content">
          <div className="form-group">
            <label htmlFor="body">邮件内容 *</label>
            <textarea
              id="body"
              value={emailData.body}
              onChange={handleChange('body')}
              placeholder="请输入邮件内容..."
              rows="12"
              required
              className="email-content-textarea"
            />
          </div>
        </div>

        <div className="form-row single">
          <button 
            type="submit" 
            disabled={sending}
            className="send-btn"
          >
            {sending ? '发送中...' : '📤 发送邮件'}
          </button>
        </div>

        {message && (
          <div className={`message ${message.includes('成功') ? 'success' : 'error'}`}>
            {message}
          </div>
        )}
      </form>
    </div>
  );
};

export default SendEmail;