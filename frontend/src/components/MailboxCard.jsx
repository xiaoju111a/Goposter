import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../utils/api.js';
import { cacheManager } from '../utils/cache.js';
import EmailItem from './EmailItem.jsx';

const MailboxCard = ({ mailbox, selected = false, onSelect, viewMode = 'grid', onMailboxClick }) => {
    const [emails, setEmails] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

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

    const handleMailboxClick = () => {
        if (onMailboxClick) {
            onMailboxClick(mailbox);
        }
    };


    return (
        <div className={`mailbox-card ${selected ? 'selected' : ''} ${viewMode}`}>
            <div className="mailbox-header">
                {onSelect && (
                    <input
                        type="checkbox"
                        checked={selected}
                        onChange={(e) => onSelect(e.target.checked)}
                        className="mailbox-checkbox"
                        onClick={(e) => e.stopPropagation()}
                    />
                )}
                <div className="mailbox-name" onClick={handleMailboxClick}>
                    {mailbox}
                </div>
                <div className="email-count-badge">{loading ? '...' : emails.length}</div>
            </div>
            <div className="mailbox-preview">
                {loading && <div className="loading">正在加载...</div>}
                {error && <div className="error">{error}</div>}
                {!loading && !error && emails.length === 0 && (
                    <div className="empty-preview">暂无邮件</div>
                )}
                {!loading && !error && emails.length > 0 && (
                    <div className="email-preview">
                        最近邮件: {emails[0]?.Subject || '(无主题)'}
                    </div>
                )}
            </div>
        </div>
    );
};

export default MailboxCard;