import React, { useState } from 'react';
import { sendEmail } from '../utils/api';

const SendEmail = ({ userEmail = '' }) => {
  const [emailData, setEmailData] = useState({
    from: userEmail || '',
    to: '',
    subject: '',
    body: ''
  });
  const [sending, setSending] = useState(false);
  const [message, setMessage] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!emailData.from || !emailData.to || !emailData.subject || !emailData.body) {
      setMessage('请填写所有必填字段');
      return;
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
      <form onSubmit={handleSubmit} className="send-email-form">
        <div className="form-group">
          <label htmlFor="from">发件人 *</label>
          <input
            type="email"
            id="from"
            value={emailData.from}
            onChange={handleChange('from')}
            placeholder="your-name@freeagent.live"
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
            placeholder="recipient@example.com"
            required
          />
        </div>

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

        <div className="form-group">
          <label htmlFor="body">邮件内容 *</label>
          <textarea
            id="body"
            value={emailData.body}
            onChange={handleChange('body')}
            placeholder="请输入邮件内容..."
            rows="8"
            required
          />
        </div>

        <button 
          type="submit" 
          disabled={sending}
          className="send-btn"
        >
          {sending ? '发送中...' : '📤 发送邮件'}
        </button>

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