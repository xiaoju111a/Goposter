import React from 'react';

const Stats = ({ mailboxes, totalEmails }) => (
    <div className="stats-container">
        <div className="stat-card">
            <div className="stat-value">{mailboxes.length}</div>
            <div className="stat-label">邮箱总数</div>
        </div>
        <div className="stat-card">
            <div className="stat-value">{totalEmails}</div>
            <div className="stat-label">邮件总数</div>
        </div>
        <div className="stat-card">
            <div className="stat-value">25/143</div>
            <div className="stat-label">SMTP/IMAP端口</div>
        </div>
        <div className="stat-card">
            <div className="stat-value">在线</div>
            <div className="stat-label">服务状态</div>
        </div>
    </div>
);

export default Stats;