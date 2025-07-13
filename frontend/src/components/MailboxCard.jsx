import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../utils/api.js';
import { cacheManager } from '../utils/cache.js';
import EmailItem from './EmailItem.jsx';

const MailboxCard = ({ mailbox }) => {
    const [emails, setEmails] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [expandedEmailId, setExpandedEmailId] = useState(null);

    const loadEmails = useCallback(async () => {
        try {
            setLoading(true);
            setError(null);
            const emailData = await api.getEmails(mailbox);
            setEmails(emailData.reverse()); // 最新的在前面
        } catch (err) {
            setError('加载邮件失败');
            console.error('加载邮件失败:', err);
        } finally {
            setLoading(false);
        }
    }, [mailbox]);

    useEffect(() => {
        loadEmails();
    }, [loadEmails]);

    const handleEmailToggle = (emailId) => {
        setExpandedEmailId(prev => prev === emailId ? null : emailId);
    };

    const handleDeleteEmail = async (emailId) => {
        try {
            await api.deleteEmail(mailbox, emailId);
            // 删除成功，从本地状态中移除邮件
            setEmails(prev => prev.filter(email => email.ID !== emailId));
            // 如果删除的是展开的邮件，关闭展开状态
            if (expandedEmailId === emailId) {
                setExpandedEmailId(null);
            }
        } catch (err) {
            console.error('删除邮件失败:', err);
            alert('删除失败: ' + err.message);
        }
    };

    return (
        <div className="mailbox-card">
            <div className="mailbox-header">
                <div className="mailbox-name">{mailbox}</div>
                <div className="email-count">{emails.length}</div>
            </div>
            <div className="email-list">
                {loading && <div className="loading">正在加载邮件...</div>}
                {error && <div className="error">{error}</div>}
                {!loading && !error && emails.length === 0 && (
                    <div className="loading">暂无邮件</div>
                )}
                {!loading && !error && emails.map((email, index) => (
                    <EmailItem
                        key={email.ID || index}
                        email={email}
                        expanded={expandedEmailId === (email.ID || index)}
                        onToggle={() => handleEmailToggle(email.ID || index)}
                        onDelete={handleDeleteEmail}
                    />
                ))}
            </div>
        </div>
    );
};

export default MailboxCard;