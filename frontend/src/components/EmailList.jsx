import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../utils/api.js';
import EmailItem from './EmailItem.jsx';

const EmailList = ({ mailbox, onBack }) => {
    const [emails, setEmails] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [selectedEmail, setSelectedEmail] = useState(null);
    const [searchQuery, setSearchQuery] = useState('');

    const loadEmails = useCallback(async () => {
        if (!mailbox) {
            setLoading(false);
            setError('邮箱名称不能为空');
            return;
        }
        
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

    const handleEmailClick = (email) => {
        setSelectedEmail(email);
    };

    const handleBackToList = () => {
        setSelectedEmail(null);
    };

    const handleDeleteEmail = async (emailId) => {
        try {
            await api.deleteEmail(mailbox, emailId);
            // 删除成功，从本地状态中移除邮件
            setEmails(prev => prev.filter(email => email.ID !== emailId));
            // 如果删除的是当前查看的邮件，返回列表
            if (selectedEmail && selectedEmail.ID === emailId) {
                setSelectedEmail(null);
            }
        } catch (err) {
            console.error('删除邮件失败:', err);
            alert('删除失败: ' + err.message);
        }
    };

    const filteredEmails = emails.filter(email => {
        if (!searchQuery) return true;
        const query = searchQuery.toLowerCase();
        return (
            (email.Subject || '').toLowerCase().includes(query) ||
            (email.From || '').toLowerCase().includes(query) ||
            (email.Body || '').toLowerCase().includes(query)
        );
    });

    const formatDate = (dateStr) => {
        try {
            const date = new Date(dateStr);
            const now = new Date();
            const diff = now - date;
            const days = Math.floor(diff / (1000 * 60 * 60 * 24));
            
            if (days === 0) {
                return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
            } else if (days < 7) {
                return `${days}天前`;
            } else {
                return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' });
            }
        } catch {
            return '未知时间';
        }
    };

    const getSenderName = (from) => {
        if (!from) return '未知发件人';
        const match = from.match(/^(.+?)\s*<(.+)>$/);
        if (match) {
            return match[1].trim() || match[2];
        }
        return from;
    };

    const getAvatarInitial = (from) => {
        const name = getSenderName(from);
        return name.charAt(0).toUpperCase();
    };

    const generateAvatarColor = (from) => {
        const colors = [
            '#4285f4', '#34a853', '#fbbc04', '#ea4335', 
            '#9c27b0', '#ff9800', '#795548', '#607d8b',
            '#e91e63', '#009688', '#ff5722', '#3f51b5'
        ];
        let hash = 0;
        for (let i = 0; i < from.length; i++) {
            hash = from.charCodeAt(i) + ((hash << 5) - hash);
        }
        return colors[Math.abs(hash) % colors.length];
    };

    // 如果选中了某封邮件，显示邮件详情
    if (selectedEmail) {
        return (
            <div className="email-detail-view">
                <div className="email-detail-header">
                    <button className="back-btn" onClick={handleBackToList}>
                        ← 返回邮件列表
                    </button>
                    <div className="mailbox-info">
                        <span className="mailbox-name">{mailbox}</span>
                    </div>
                </div>
                
                <div className="email-detail-content">
                    <EmailItem 
                        email={selectedEmail}
                        expanded={true}
                        onToggle={() => {}}
                        onDelete={handleDeleteEmail}
                    />
                </div>
            </div>
        );
    }

    return (
        <div className="email-list-view">
            <div className="email-list-header">
                <button className="back-btn" onClick={onBack}>
                    ← 返回邮箱列表
                </button>
                <div className="mailbox-info">
                    <span className="mailbox-name">{mailbox}</span>
                    <span className="email-count-badge">{emails.length} 封邮件</span>
                </div>
            </div>

            <div className="email-search-bar">
                <input
                    type="text"
                    placeholder="搜索邮件..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="email-search-input"
                />
            </div>

            <div className="email-list-container">
                {loading && (
                    <div className="loading-container">
                        <div className="loading-spinner"></div>
                        <div className="loading-text">正在加载邮件...</div>
                    </div>
                )}
                
                {error && (
                    <div className="error-container">
                        <div className="error-message">{error}</div>
                        <button onClick={loadEmails} className="retry-btn">重试</button>
                    </div>
                )}
                
                {!loading && !error && filteredEmails.length === 0 && (
                    <div className="empty-state">
                        <div className="empty-icon">📭</div>
                        <div className="empty-text">
                            {searchQuery ? '没有找到匹配的邮件' : '暂无邮件'}
                        </div>
                    </div>
                )}
                
                {!loading && !error && filteredEmails.length > 0 && (
                    <div className="email-list">
                        {filteredEmails.map((email, index) => (
                            <div 
                                key={email.ID || index} 
                                className="email-list-item"
                                onClick={() => handleEmailClick(email)}
                            >
                                <div className="email-list-row">
                                    <div 
                                        className="sender-avatar"
                                        style={{ backgroundColor: generateAvatarColor(email.From) }}
                                    >
                                        {getAvatarInitial(email.From)}
                                    </div>
                                    <div className="email-list-content">
                                        <div className="email-list-header-row">
                                            <span className="sender-name">{getSenderName(email.From)}</span>
                                            <span className="email-date">{formatDate(email.Date)}</span>
                                        </div>
                                        <div className="email-subject-row">
                                            {email.Subject || '(无主题)'}
                                        </div>
                                        <div className="email-preview-row">
                                            {email.Body && email.Body.length > 100 
                                                ? email.Body.substring(0, 100) + '...' 
                                                : email.Body || '(无内容)'}
                                        </div>
                                    </div>
                                    <div className="email-list-actions">
                                        <button 
                                            className="delete-email-btn"
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                handleDeleteEmail(email.ID);
                                            }}
                                            title="删除邮件"
                                        >
                                            🗑️
                                        </button>
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
};

export default EmailList;