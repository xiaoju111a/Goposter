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
                
                {/* 邮件标签 */}
                <div className="email-tags">
                    {email.IsAutoReply && (
                        <span className="email-tag auto-reply">自动回复</span>
                    )}
                    {email.Attachments && email.Attachments.length > 0 && (
                        <span className="email-tag attachments">
                            📎 {email.Attachments.length}个附件
                        </span>
                    )}
                    {email.Charset && email.Charset !== 'utf-8' && (
                        <span className="email-tag charset">{email.Charset}</span>
                    )}
                </div>
                
                <div className="email-body-expanded">
                    {/* 邮件正文 */}
                    <div className="email-content">
                        {email.HTMLBody ? (
                            <div className="email-html-content">
                                <div className="content-type-toggle">
                                    <button 
                                        className="toggle-btn active"
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            const htmlContent = e.target.closest('.email-body-expanded').querySelector('.email-html-content');
                                            const textContent = e.target.closest('.email-body-expanded').querySelector('.email-text-content');
                                            htmlContent.style.display = 'block';
                                            textContent.style.display = 'none';
                                            e.target.classList.add('active');
                                            e.target.nextElementSibling.classList.remove('active');
                                        }}
                                    >
                                        HTML
                                    </button>
                                    <button 
                                        className="toggle-btn"
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            const htmlContent = e.target.closest('.email-body-expanded').querySelector('.email-html-content');
                                            const textContent = e.target.closest('.email-body-expanded').querySelector('.email-text-content');
                                            htmlContent.style.display = 'none';
                                            textContent.style.display = 'block';
                                            e.target.classList.add('active');
                                            e.target.previousElementSibling.classList.remove('active');
                                        }}
                                    >
                                        纯文本
                                    </button>
                                </div>
                                <div className="html-content" dangerouslySetInnerHTML={{
                                    __html: email.HTMLBody ? email.HTMLBody.replace(/cid:([^"'\s>]+)/g, (match, cid) => {
                                        const mailbox = window.location.pathname.split('/').pop();
                                        return `/api/attachments/inline/${mailbox}/${email.ID}/${cid}`;
                                    }) : ''
                                }} />
                            </div>
                        ) : null}
                        <div className="email-text-content" style={{display: email.HTMLBody ? 'none' : 'block'}}>
                            {email.Body || '(无内容)'}
                        </div>
                    </div>
                    
                    {/* 附件列表 */}
                    {email.Attachments && email.Attachments.length > 0 && (
                        <div className="email-attachments">
                            <h4>📎 附件 ({email.Attachments.length})</h4>
                            <div className="attachments-list">
                                {email.Attachments.map((attachment, index) => (
                                    <div key={index} className="attachment-item">
                                        <div className="attachment-info">
                                            <span className="attachment-name">{attachment.Filename || '未知文件'}</span>
                                            <span className="attachment-size">
                                                {(attachment.Size / 1024).toFixed(1)} KB
                                            </span>
                                            <span className="attachment-type">{attachment.ContentType}</span>
                                        </div>
                                        <div className="attachment-actions">
                                            {attachment.Disposition === 'inline' && (
                                                <span className="inline-badge">内联</span>
                                            )}
                                            {attachment.ContentType && attachment.ContentType.startsWith('image/') && (
                                                <button 
                                                    className="preview-btn"
                                                    onClick={(e) => {
                                                        e.stopPropagation();
                                                        const imageUrl = attachment.CID 
                                                            ? `/api/attachments/inline/${window.location.pathname.split('/').pop()}/${email.ID}/${attachment.CID}`
                                                            : `/api/attachments/${window.location.pathname.split('/').pop()}/${email.ID}/${index}`;
                                                        window.open(imageUrl, '_blank');
                                                    }}
                                                    title="预览图片"
                                                >
                                                    🖼️
                                                </button>
                                            )}
                                            <button 
                                                className="download-btn"
                                                onClick={(e) => {
                                                    e.stopPropagation();
                                                    const downloadUrl = `/api/attachments/${window.location.pathname.split('/').pop()}/${email.ID}/${index}`;
                                                    const link = document.createElement('a');
                                                    link.href = downloadUrl;
                                                    link.download = attachment.Filename || 'attachment';
                                                    link.click();
                                                }}
                                                title="下载附件"
                                            >
                                                📥
                                            </button>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                    
                    {/* 嵌入式内容 */}
                    {email.Embedded && (
                        <div className="email-embedded">
                            {email.Embedded.links && email.Embedded.links.length > 0 && (
                                <div className="embedded-links">
                                    <h4>🔗 链接</h4>
                                    <div className="links-list">
                                        {email.Embedded.links.slice(0, 5).map((link, index) => (
                                            <div key={index} className="link-item">
                                                <a href={link} target="_blank" rel="noopener noreferrer">
                                                    {link.length > 50 ? link.substring(0, 50) + '...' : link}
                                                </a>
                                            </div>
                                        ))}
                                        {email.Embedded.links.length > 5 && (
                                            <div className="more-links">
                                                还有 {email.Embedded.links.length - 5} 个链接...
                                            </div>
                                        )}
                                    </div>
                                </div>
                            )}
                            
                            {email.Embedded.images && email.Embedded.images.length > 0 && (
                                <div className="embedded-images">
                                    <h4>🖼️ 图片</h4>
                                    <div className="images-list">
                                        {email.Embedded.images.slice(0, 3).map((image, index) => (
                                            <div key={index} className="image-item">
                                                <img 
                                                    src={image} 
                                                    alt={`图片 ${index + 1}`}
                                                    className="embedded-image"
                                                    onError={(e) => {
                                                        e.target.style.display = 'none';
                                                        e.target.nextElementSibling.style.display = 'block';
                                                    }}
                                                />
                                                <div className="image-placeholder" style={{display: 'none'}}>
                                                    无法加载图片: {image}
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}
                        </div>
                    )}
                    
                    {/* 签名 */}
                    {email.Signature && (
                        <div className="email-signature">
                            <h4>✍️ 签名</h4>
                            <div className="signature-content">
                                {email.Signature}
                            </div>
                        </div>
                    )}
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