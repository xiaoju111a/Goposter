import React from 'react';

const EmailItem = ({ email, expanded, onToggle, onDelete }) => {
    const bodyPreview = email.Body && email.Body.length > 80 
        ? email.Body.substring(0, 80) + '...' 
        : email.Body;

    const handleDelete = (e) => {
        e.stopPropagation(); // 防止触发邮件展开
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

    return (
        <div className={`email-item ${expanded ? 'expanded' : ''}`} onClick={onToggle}>
            <div className="email-header">
                <div className="email-from">{email.From}</div>
                <div className="email-date">{formatDate(email.Date)}</div>
            </div>
            <div className="email-subject">{email.Subject || '无主题'}</div>
            {!expanded && (
                <div className="email-preview">{bodyPreview}</div>
            )}
            {expanded && (
                <div className="email-expanded">
                    <div className="email-full-header">
                        <div className="email-full-meta">
                            <div><strong>发件人:</strong> {email.From}</div>
                            <div><strong>收件人:</strong> {email.To}</div>
                            <div><strong>时间:</strong> {new Date(email.Date || email.timestamp).toLocaleString()}</div>
                        </div>
                        <button className="delete-btn" onClick={handleDelete}>
                            🗑️
                        </button>
                    </div>
                    <div className="email-body">{email.Body}</div>
                </div>
            )}
        </div>
    );
};

export default EmailItem;