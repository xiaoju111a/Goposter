import React, { useState, useEffect } from 'react';
import { createMailbox } from '../utils/api';
import configManager from '../utils/config.js';

const CreateMailbox = ({ onMailboxCreated }) => {
  const [mailboxData, setMailboxData] = useState({
    username: '',
    password: '',
    description: ''
  });
  const [creating, setCreating] = useState(false);
  const [config, setConfig] = useState({
    domain: 'freeagent.live'
  });
  const [message, setMessage] = useState('');

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

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!mailboxData.username || !mailboxData.password) {
      setMessage('请填写用户名和密码');
      return;
    }

    // 验证用户名格式
    if (!/^[a-zA-Z0-9._-]+$/.test(mailboxData.username)) {
      setMessage('用户名只能包含字母、数字、点、下划线和连字符');
      return;
    }

    setCreating(true);
    setMessage('');

    try {
      const result = await createMailbox(mailboxData);
      setMessage(`✅ 邮箱创建成功！邮箱地址: ${mailboxData.username}@${config.domain}`);
      
      // 清空表单
      setMailboxData({
        username: '',
        password: '',
        description: ''
      });

      // 通知父组件刷新邮箱列表
      if (onMailboxCreated) {
        onMailboxCreated();
      }
    } catch (error) {
      setMessage(`❌ 创建失败: ${error.message}`);
    } finally {
      setCreating(false);
    }
  };

  const handleChange = (field) => (e) => {
    setMailboxData(prev => ({
      ...prev,
      [field]: e.target.value
    }));
  };

  const generatePassword = () => {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let password = '';
    for (let i = 0; i < 12; i++) {
      password += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    setMailboxData(prev => ({ ...prev, password }));
  };

  return (
    <div className="create-mailbox-container">
      <h3>✉️ 创建新邮箱</h3>
      <form onSubmit={handleSubmit} className="create-mailbox-form">
        <div className="form-group">
          <label htmlFor="username">用户名 *</label>
          <div className="username-input-group">
            <input
              type="text"
              id="username"
              value={mailboxData.username}
              onChange={handleChange('username')}
              placeholder="输入用户名"
              pattern="[a-zA-Z0-9._-]+"
              title="只能包含字母、数字、点、下划线和连字符"
              required
            />
            <span className="domain-suffix">@{config.domain}</span>
          </div>
          <small>邮箱地址将为: {mailboxData.username ? `${mailboxData.username}@${config.domain}` : `username@${config.domain}`}</small>
        </div>

        <div className="form-group">
          <label htmlFor="password">密码 *</label>
          <div className="password-input-group">
            <input
              type="text"
              id="password"
              value={mailboxData.password}
              onChange={handleChange('password')}
              placeholder="设置密码"
              required
            />
            <button 
              type="button" 
              onClick={generatePassword}
              className="generate-btn"
              title="生成随机密码"
            >
              🎲
            </button>
          </div>
        </div>

        <div className="form-group">
          <label htmlFor="description">描述（可选）</label>
          <input
            type="text"
            id="description"
            value={mailboxData.description}
            onChange={handleChange('description')}
            placeholder="邮箱用途描述，如：客服邮箱、个人邮箱等"
          />
        </div>

        <button 
          type="submit" 
          disabled={creating}
          className="create-btn"
        >
          {creating ? '创建中...' : '📬 创建邮箱'}
        </button>

        {message && (
          <div className={`message ${message.includes('成功') ? 'success' : 'error'}`}>
            {message}
          </div>
        )}
      </form>

      <div className="tips">
        <h4>💡 使用提示</h4>
        <ul>
          <li>用户名可以包含字母、数字、点(.)、下划线(_)和连字符(-)</li>
          <li>所有邮箱都使用 @{config.domain} 域名</li>
          <li>密码用于IMAP客户端登录邮箱</li>
          <li>创建后即可接收和发送邮件</li>
        </ul>
      </div>
    </div>
  );
};

export default CreateMailbox;