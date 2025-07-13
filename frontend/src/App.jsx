import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { api } from './utils/api.js';
import { cacheManager } from './utils/cache.js';
import { auth, authListener, logout } from './utils/auth.js';
import MailboxCard from './components/MailboxCard.jsx';
import Stats from './components/Stats.jsx';
import SendEmail from './components/SendEmail.jsx';
import CreateMailbox from './components/CreateMailbox.jsx';
import Login from './components/Login.jsx';

const App = () => {
    const [mailboxes, setMailboxes] = useState([]);
    const [loading, setLoading] = useState(true);
    const [refreshKey, setRefreshKey] = useState(0);
    const [activeTab, setActiveTab] = useState('mailboxes'); // 'mailboxes', 'send', 'create'
    const [isAuthenticated, setIsAuthenticated] = useState(auth.isAuthenticated());
    const [currentUser, setCurrentUser] = useState(auth.getCurrentUser());

    const loadMailboxes = useCallback(async () => {
        try {
            setLoading(true);
            const mailboxData = await api.getMailboxes();
            setMailboxes(mailboxData);
        } catch (err) {
            console.error('加载邮箱失败:', err);
        } finally {
            setLoading(false);
        }
    }, []);

    // 监听认证状态变化
    useEffect(() => {
        const unsubscribe = authListener.addListener((authenticated, user) => {
            setIsAuthenticated(authenticated);
            setCurrentUser(user);
            
            if (authenticated) {
                // 登录成功后加载数据
                loadMailboxes();
            } else {
                // 登出后清除数据
                setMailboxes([]);
                cacheManager.clear();
            }
        });

        return unsubscribe;
    }, [loadMailboxes]);

    useEffect(() => {
        if (isAuthenticated) {
            loadMailboxes();
        }
    }, [loadMailboxes, refreshKey, isAuthenticated]);

    const handleRefresh = () => {
        cacheManager.clear();
        setRefreshKey(prev => prev + 1);
    };

    const handleMailboxCreated = () => {
        // 刷新邮箱列表
        handleRefresh();
        // 切换回邮箱列表视图
        setActiveTab('mailboxes');
    };

    const handleLoginSuccess = (loginData) => {
        // 认证状态会通过 authListener 自动更新
        // 这里可以添加登录成功后的其他逻辑
        console.log('登录成功:', loginData);
    };

    const handleLogout = async () => {
        try {
            await logout();
            // 认证状态会通过 authListener 自动更新
        } catch (error) {
            console.error('登出失败:', error);
        }
    };

    // 计算邮件总数的占位符
    const totalEmails = useMemo(() => {
        // 这里可以通过API获取或累计计算
        return mailboxes.length * 2; // 临时估算
    }, [mailboxes]);

    // 如果未认证，显示登录页面
    if (!isAuthenticated) {
        return <Login onLoginSuccess={handleLoginSuccess} />;
    }

    return (
        <div className="admin-layout">
            {/* 侧边栏 */}
            <div className="sidebar">
                <div className="sidebar-header">
                    <div className="logo">
                        <span className="logo-icon">✉️</span>
                        <span className="logo-text">FreeAgent Mail</span>
                    </div>
                </div>

                <nav className="sidebar-nav">
                    <button 
                        className={`nav-item ${activeTab === 'mailboxes' ? 'active' : ''}`}
                        onClick={() => setActiveTab('mailboxes')}
                    >
                        <span className="nav-icon">📮</span>
                        <span className="nav-text">邮箱管理</span>
                    </button>
                    <button 
                        className={`nav-item ${activeTab === 'send' ? 'active' : ''}`}
                        onClick={() => setActiveTab('send')}
                    >
                        <span className="nav-icon">📤</span>
                        <span className="nav-text">发送邮件</span>
                    </button>
                    <button 
                        className={`nav-item ${activeTab === 'create' ? 'active' : ''}`}
                        onClick={() => setActiveTab('create')}
                    >
                        <span className="nav-icon">➕</span>
                        <span className="nav-text">创建邮箱</span>
                    </button>
                </nav>

                <div className="sidebar-footer">
                    <div className="user-profile">
                        <div className="user-avatar">👤</div>
                        <div className="user-details">
                            <div className="user-name">管理员</div>
                            <div className="user-email">{currentUser?.email}</div>
                        </div>
                    </div>
                    <button onClick={handleLogout} className="logout-btn" title="登出">
                        <span className="logout-icon">🚪</span>
                    </button>
                </div>
            </div>

            {/* 主内容区域 */}
            <div className="main-content">
                <div className="content-header">
                    <div className="page-title">
                        {activeTab === 'mailboxes' && '📮 邮箱管理'}
                        {activeTab === 'send' && '📤 发送邮件'}
                        {activeTab === 'create' && '➕ 创建邮箱'}
                    </div>
                    <button onClick={handleRefresh} className="refresh-btn" title="刷新数据">
                        <span className="refresh-icon">🔄</span>
                        <span className="refresh-text">刷新</span>
                    </button>
                </div>

                <div className="content-body">
                    {activeTab === 'mailboxes' && (
                        <>
                            <Stats mailboxes={mailboxes} totalEmails={totalEmails} />
                            {loading ? (
                                <div className="loading-container">
                                    <div className="loading-spinner"></div>
                                    <div className="loading-text">正在加载邮箱数据...</div>
                                </div>
                            ) : (
                                <div className="mailbox-container">
                                    <div className="section-header">
                                        <h3>邮箱列表</h3>
                                        <span className="mailbox-count">{mailboxes.length} 个邮箱</span>
                                    </div>
                                    <div className="mailbox-grid">
                                        {mailboxes.map((mailbox, index) => (
                                            <MailboxCard 
                                                key={`${mailbox}-${refreshKey}`} 
                                                mailbox={mailbox} 
                                            />
                                        ))}
                                    </div>
                                </div>
                            )}
                        </>
                    )}

                    {activeTab === 'send' && (
                        <SendEmail userEmail={currentUser?.email || (mailboxes.length > 0 ? mailboxes[0] : '')} />
                    )}

                    {activeTab === 'create' && (
                        <CreateMailbox onMailboxCreated={handleMailboxCreated} />
                    )}
                </div>
            </div>
        </div>
    );
};

export default App;