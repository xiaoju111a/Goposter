import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { api } from './utils/api.js';
import { cacheManager } from './utils/cache.js';
import { auth, authListener, logout } from './utils/auth.js';
import MailboxCard from './components/MailboxCard.jsx';
import EmailList from './components/EmailList.jsx';
import SendEmail from './components/SendEmail.jsx';
import CreateMailbox from './components/CreateMailbox.jsx';
import Login from './components/Login.jsx';
// 高级功能组件
import VirtualList from './components/VirtualList.jsx';
import FilterBar from './components/FilterBar.jsx';
import NotificationCenter from './components/NotificationCenter.jsx';
// import EmailEditor from './components/EmailEditor.jsx';
// import EmailTemplates from './components/EmailTemplates.jsx';
import BatchOperations from './components/BatchOperations.jsx';
import MailboxBatchOperations from './components/MailboxBatchOperations.jsx';
import SwipeActions from './components/SwipeActions.jsx';
import PullToRefresh from './components/PullToRefresh.jsx';
import AttachmentViewer from './components/AttachmentViewer.jsx';
import SecuritySettings from './components/SecuritySettings.jsx';

const App = () => {
    const [mailboxes, setMailboxes] = useState([]);
    const [loading, setLoading] = useState(true);
    const [refreshKey, setRefreshKey] = useState(0);
    const [activeTab, setActiveTab] = useState('mailboxes'); // 'mailboxes', 'send', 'create', 'stats', 'security'
    const [isAuthenticated, setIsAuthenticated] = useState(auth.isAuthenticated());
    const [currentUser, setCurrentUser] = useState(auth.getCurrentUser());
    const [searchQuery, setSearchQuery] = useState('');
    const [selectedMailboxes, setSelectedMailboxes] = useState([]);
    const [currentMailbox, setCurrentMailbox] = useState(null); // 当前查看的邮箱
    // 高级功能状态
    const [notifications, setNotifications] = useState([]);
    const [showNotifications, setShowNotifications] = useState(false);
    const [filterConfig, setFilterConfig] = useState({});
    const [viewMode, setViewMode] = useState('grid'); // 'grid', 'list'
    const [isRefreshing, setIsRefreshing] = useState(false);

    const loadMailboxes = useCallback(async () => {
        try {
            setLoading(true);
            console.log('开始加载邮箱数据...');
            const mailboxData = await api.getMailboxes();
            console.log('获取到的邮箱数据:', mailboxData);
            // 过滤掉 undefined 或空值
            const validMailboxes = mailboxData.filter(mailbox => 
                mailbox && typeof mailbox === 'string' && mailbox.trim() !== ''
            );
            console.log('过滤后的邮箱数据:', validMailboxes);
            setMailboxes(validMailboxes);
        } catch (err) {
            console.error('加载邮箱失败:', err);
            // 设置错误状态
            setMailboxes([]);
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

    const handleRefresh = async () => {
        setIsRefreshing(true);
        cacheManager.clear();
        setRefreshKey(prev => prev + 1);
        await loadMailboxes();
        setIsRefreshing(false);
        // 添加通知
        addNotification({
            id: Date.now(),
            type: 'success',
            title: '刷新成功',
            message: '邮箱数据已更新',
            timestamp: new Date()
        });
    };

    // 通知管理
    const addNotification = useCallback((notification) => {
        setNotifications(prev => [notification, ...prev.slice(0, 49)]); // 最多50条
    }, []);

    const removeNotification = useCallback((id) => {
        setNotifications(prev => prev.filter(n => n.id !== id));
    }, []);

    const clearAllNotifications = useCallback(() => {
        setNotifications([]);
    }, []);

    // 筛选邮箱
    const filteredMailboxes = useMemo(() => {
        let filtered = mailboxes;
        
        // 基本搜索
        if (searchQuery.trim()) {
            filtered = filtered.filter(mailbox => 
                mailbox.toLowerCase().includes(searchQuery.toLowerCase())
            );
        }
        
        // 高级筛选
        if (filterConfig.domain) {
            filtered = filtered.filter(mailbox => 
                mailbox.includes(filterConfig.domain)
            );
        }
        
        return filtered;
    }, [mailboxes, searchQuery, filterConfig]);

    const handleMailboxCreated = () => {
        // 刷新邮箱列表
        handleRefresh();
        // 切换回邮箱列表视图
        setActiveTab('mailboxes');
    };


    const handleMailboxClick = (mailbox) => {
        setCurrentMailbox(mailbox);
    };

    const handleBackToMailboxes = () => {
        setCurrentMailbox(null);
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


    // 如果未认证，显示登录页面
    if (!isAuthenticated) {
        console.log('用户未认证，显示登录页面');
        return <Login onLoginSuccess={handleLoginSuccess} />;
    }
    
    console.log('用户已认证，显示主界面');
    
    // 调试信息（仅在开发环境）
    if (process.env.NODE_ENV === 'development') {
        console.log('App渲染状态:', {
            isAuthenticated,
            currentUser,
            mailboxes: mailboxes,
            mailboxesLength: mailboxes.length,
            loading,
            activeTab
        });
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
                    <button 
                        className={`nav-item ${activeTab === 'security' ? 'active' : ''}`}
                        onClick={() => setActiveTab('security')}
                    >
                        <span className="nav-icon">🔒</span>
                        <span className="nav-text">安全设置</span>
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
                        {activeTab === 'security' && '🔒 安全设置'}
                    </div>
                    <div className="header-actions">
                        <button 
                            onClick={() => setShowNotifications(!showNotifications)}
                            className="notification-btn" 
                            title="通知中心"
                        >
                            <span className="notification-icon">🔔</span>
                            {notifications.length > 0 && (
                                <span className="notification-badge">{notifications.length}</span>
                            )}
                        </button>
                        <button 
                            onClick={handleRefresh} 
                            className={`refresh-btn ${isRefreshing ? 'refreshing' : ''}`} 
                            title="刷新数据"
                            disabled={isRefreshing}
                        >
                            <span className="refresh-icon">🔄</span>
                            <span className="refresh-text">{isRefreshing ? '刷新中' : '刷新'}</span>
                        </button>
                    </div>
                </div>

                {/* 通知中心 */}
                {showNotifications && (
                    <NotificationCenter
                        notifications={notifications}
                        onClose={() => setShowNotifications(false)}
                        onRemove={removeNotification}
                        onClearAll={clearAllNotifications}
                    />
                )}

                <div className="content-body">
                    {activeTab === 'mailboxes' && (
                        <>
                            {currentMailbox ? (
                                <EmailList 
                                    mailbox={currentMailbox}
                                    onBack={handleBackToMailboxes}
                                />
                            ) : (
                                <>
                                    {/* 高级筛选栏 */}
                                    <FilterBar
                                        searchQuery={searchQuery}
                                        onSearchChange={setSearchQuery}
                                        filterConfig={filterConfig}
                                        onFilterChange={setFilterConfig}
                                        totalCount={mailboxes.length}
                                        filteredCount={filteredMailboxes.length}
                                    />
                                    
                                    {/* 批量操作 */}
                                    {selectedMailboxes.length > 0 && (
                                        <MailboxBatchOperations
                                            selectedMailboxes={selectedMailboxes}
                                            onClearSelection={() => setSelectedMailboxes([])}
                                            onDelete={async (items) => {
                                                // 检查是否包含默认管理员邮箱
                                                if (items.includes('admin@freeagent.live')) {
                                                    alert('默认管理员邮箱不能删除');
                                                    return;
                                                }
                                                
                                                try {
                                                    // 批量删除邮箱
                                                    for (const mailbox of items) {
                                                        await api.deleteMailbox(mailbox);
                                                    }
                                                    
                                                    // 删除成功，更新状态
                                                    setMailboxes(prev => prev.filter(mailbox => !items.includes(mailbox)));
                                                    setSelectedMailboxes([]);
                                                    
                                                    // 如果当前正在查看被删除的邮箱，返回邮箱列表
                                                    if (currentMailbox && items.includes(currentMailbox)) {
                                                        setCurrentMailbox(null);
                                                    }
                                                    
                                                    addNotification({
                                                        id: Date.now(),
                                                        type: 'success',
                                                        title: '批量删除完成',
                                                        message: `已删除 ${items.length} 个邮箱`,
                                                        timestamp: new Date()
                                                    });
                                                } catch (err) {
                                                    console.error('批量删除失败:', err);
                                                    alert('批量删除失败: ' + err.message);
                                                }
                                            }}
                                        />
                                    )}
                                    
                                    {/* 视图模式切换 */}
                                    <div className="view-controls">
                                        <button 
                                            className={`view-btn ${viewMode === 'grid' ? 'active' : ''}`}
                                            onClick={() => setViewMode('grid')}
                                            title="网格视图"
                                        >
                                            <span>⊞</span>
                                        </button>
                                        <button 
                                            className={`view-btn ${viewMode === 'list' ? 'active' : ''}`}
                                            onClick={() => setViewMode('list')}
                                            title="列表视图"
                                        >
                                            <span>☰</span>
                                        </button>
                                    </div>
                                    
                                    {loading ? (
                                        <div className="loading-container">
                                            <div className="loading-spinner"></div>
                                            <div className="loading-text">正在加载邮箱数据...</div>
                                        </div>
                                    ) : (
                                        <div className="mailbox-container">
                                            <div className="section-header">
                                                <h3>邮箱列表</h3>
                                                <span className="mailbox-count">
                                                    {searchQuery ? 
                                                        `找到 ${filteredMailboxes.length} 个邮箱` :
                                                        `${mailboxes.length} 个邮箱`
                                                    }
                                                </span>
                                            </div>
                                            
                                            {/* 简化的邮箱列表渲染 */}
                                            <div className={`mailbox-list ${viewMode}`}>
                                                {console.log('渲染状态:', { 
                                                    filteredMailboxes, 
                                                    length: filteredMailboxes.length,
                                                    loading,
                                                    mailboxes,
                                                    searchQuery 
                                                })}
                                                {filteredMailboxes.length === 0 ? (
                                                    <div className="empty-state">
                                                        <div className="empty-icon">📭</div>
                                                        <div className="empty-text">暂无邮箱</div>
                                                        <div className="empty-description">
                                                            {searchQuery ? '没有找到匹配的邮箱' : '请先创建邮箱'}
                                                        </div>
                                                    </div>
                                                ) : (
                                                    filteredMailboxes.map((mailbox, index) => {
                                                        console.log('渲染邮箱:', mailbox, typeof mailbox);
                                                        // 确保邮箱名称有效
                                                        if (!mailbox || typeof mailbox !== 'string') {
                                                            console.log('跳过无效邮箱:', mailbox);
                                                            return null;
                                                        }
                                                        
                                                        return (
                                                            <div key={`${mailbox}-${refreshKey}`} className="mailbox-item">
                                                                <MailboxCard 
                                                                    mailbox={mailbox}
                                                                    selected={selectedMailboxes.includes(mailbox)}
                                                                    viewMode={viewMode}
                                                                    onSelect={mailbox === 'admin@freeagent.live' ? undefined : (selected) => {
                                                                        if (selected) {
                                                                            setSelectedMailboxes(prev => [...prev, mailbox]);
                                                                        } else {
                                                                            setSelectedMailboxes(prev => prev.filter(m => m !== mailbox));
                                                                        }
                                                                    }}
                                                                    onMailboxClick={handleMailboxClick}
                                                                />
                                                            </div>
                                                        );
                                                    })
                                                )}
                                            </div>
                                        </div>
                                    )}
                                </>
                            )}
                        </>
                    )}

                    {activeTab === 'send' && (
                        <div className="send-email-wrapper">
                            <SendEmail userEmail={currentUser?.email || (mailboxes.length > 0 ? mailboxes[0] : 'admin@freeagent.live')} />
                        </div>
                    )}
                    

                    {activeTab === 'create' && (
                        <CreateMailbox onMailboxCreated={handleMailboxCreated} />
                    )}


                    {activeTab === 'security' && (
                        <SecuritySettings />
                    )}
                </div>
            </div>
        </div>
    );
};

export default App;