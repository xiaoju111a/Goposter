import React, { useState } from 'react';

const EmailItem = ({ email, expanded, onToggle, onDelete }) => {
    const [isHovered, setIsHovered] = useState(false);
    
    const bodyPreview = email.Body && email.Body.length > 120 
        ? email.Body.substring(0, 120) + '...' 
        : email.Body;

    const handleDelete = (e) => {
        e.stopPropagation();
        if (window.confirm(`确定要删除这封邮件吗？\n\n主题: ${email.Subject || '无主题'}\n发件人: ${email.From}`)) {
            onDelete(email.ID);
        }
    };

    const formatDate = (dateStr) => {
        try {
            const date = new Date(dateStr || email.timestamp);
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
        // 提取发件人姓名（如果有的话）
        const match = from.match(/^(.+?)\s*<(.+)>$/);
        if (match) {
            return match[1].trim() || match[2];
        }
        return from;
    };

    const getSenderEmail = (from) => {
        if (!from) return '';
        const match = from.match(/^(.+?)\s*<(.+)>$/);
        if (match) {
            return match[2];
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

    if (expanded) {
        return (
            <div className="email-item expanded" onClick={onToggle}>
                <div className="email-expanded-header">
                    <div className="email-sender-info">
                        <div 
                            className="sender-avatar expanded-avatar"
                            style={{ backgroundColor: generateAvatarColor(email.From) }}
                        >
                            {getAvatarInitial(email.From)}
                        </div>
                        <div className="sender-details">
                            <div className="sender-name">{getSenderName(email.From)}</div>
                            <div className="sender-email">&lt;{getSenderEmail(email.From)}&gt;</div>
                            <div className="email-meta">
                                <span>收件人: {email.To}</span>
                                <span className="email-date-full">
                                    {new Date(email.Date || email.timestamp).toLocaleString('zh-CN')}
                                </span>
                            </div>
                        </div>
                    </div>
                    <div className="email-actions">
                        <button 
                            className="action-btn delete-btn" 
                            onClick={handleDelete}
                            title="删除"
                        >
                            🗑️
                        </button>
                    </div>
                </div>
                <div className="email-subject-expanded">{email.Subject || '(无主题)'}</div>
                <div className="email-body-expanded">
                    <div className="email-content">
                        {email.Body || '(无内容)'}
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div 
            className={`email-item ${isHovered ? 'hovered' : ''}`}
            onClick={onToggle}
            onMouseEnter={() => setIsHovered(true)}
            onMouseLeave={() => setIsHovered(false)}
        >
            <div className="email-content-row">
                <div className="email-left">
                    <div 
                        className="sender-avatar"
                        style={{ backgroundColor: generateAvatarColor(email.From) }}
                    >
                        {getAvatarInitial(email.From)}
                    </div>
                    <div className="email-main-content">
                        <div className="email-first-line">
                            <span className="sender-name">{getSenderName(email.From)}</span>
                            <span className="email-subject">{email.Subject || '(无主题)'}</span>
                        </div>
                        <div className="email-preview-line">
                            {bodyPreview || '(无内容)'}
                        </div>
                    </div>
                </div>
                <div className="email-right">
                    <div className="email-date">{formatDate(email.Date)}</div>
                    {isHovered && (
                        <div className="email-hover-actions">
                            <button 
                                className="hover-action-btn" 
                                onClick={(e) => e.stopPropagation()}
                                title="标记为已读"
                            >
                                ✉️
                            </button>
                            <button 
                                className="hover-action-btn" 
                                onClick={handleDelete}
                                title="删除"
                            >
                                🗑️
                            </button>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

export default EmailItem;