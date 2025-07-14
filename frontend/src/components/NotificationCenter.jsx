import React, { useState, useEffect, useCallback, useRef } from 'react';

const NotificationCenter = ({
  onNotificationClick,
  onNotificationDismiss,
  onMarkAllRead,
  maxNotifications = 50,
  className = '',
  style = {}
}) => {
  const [notifications, setNotifications] = useState([]);
  const [isOpen, setIsOpen] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const [filter, setFilter] = useState('all'); // 'all', 'unread', 'emails', 'system'
  const [isLoading, setIsLoading] = useState(false);
  
  const notificationRef = useRef(null);
  const soundRef = useRef(null);
  
  // 通知类型配置
  const notificationTypes = {
    email: {
      icon: '📧',
      color: '#2196F3',
      title: '新邮件'
    },
    system: {
      icon: '⚙️',
      color: '#FF9800',
      title: '系统通知'
    },
    security: {
      icon: '🔒',
      color: '#f44336',
      title: '安全警告'
    },
    success: {
      icon: '✅',
      color: '#4CAF50',
      title: '操作成功'
    },
    warning: {
      icon: '⚠️',
      color: '#FF9800',
      title: '警告'
    },
    error: {
      icon: '❌',
      color: '#f44336',
      title: '错误'
    }
  };
  
  // 添加通知
  const addNotification = useCallback((notification) => {
    const newNotification = {
      id: Date.now() + Math.random(),
      timestamp: new Date().toISOString(),
      isRead: false,
      priority: 'normal',
      ...notification
    };
    
    setNotifications(prev => {
      const updated = [newNotification, ...prev].slice(0, maxNotifications);
      return updated;
    });
    
    // 播放通知声音
    if (notification.playSound !== false && soundRef.current) {
      soundRef.current.play().catch(() => {
        // 忽略音频播放失败
      });
    }
    
    // 显示浏览器通知
    if (notification.showBrowserNotification && 'Notification' in window && Notification.permission === 'granted') {
      new Notification(notification.title || '新通知', {
        body: notification.message,
        icon: '/assets/icon-192.png',
        tag: notification.id
      });
    }
    
    // 自动关闭
    if (notification.autoClose !== false) {
      const timeout = notification.timeout || 5000;
      setTimeout(() => {
        dismissNotification(newNotification.id);
      }, timeout);
    }
  }, [maxNotifications]);
  
  // 标记为已读
  const markAsRead = useCallback((notificationId) => {
    setNotifications(prev =>
      prev.map(notification =>
        notification.id === notificationId
          ? { ...notification, isRead: true }
          : notification
      )
    );
  }, []);
  
  // 删除通知
  const dismissNotification = useCallback((notificationId) => {
    setNotifications(prev => prev.filter(n => n.id !== notificationId));
    onNotificationDismiss?.(notificationId);
  }, [onNotificationDismiss]);
  
  // 标记所有为已读
  const markAllAsRead = useCallback(() => {
    setNotifications(prev =>
      prev.map(notification => ({ ...notification, isRead: true }))
    );
    onMarkAllRead?.();
  }, [onMarkAllRead]);
  
  // 清除所有通知
  const clearAllNotifications = useCallback(() => {
    setNotifications([]);
  }, []);
  
  // 清除已读通知
  const clearReadNotifications = useCallback(() => {
    setNotifications(prev => prev.filter(n => !n.isRead));
  }, []);
  
  // 处理通知点击
  const handleNotificationClick = useCallback((notification) => {
    markAsRead(notification.id);
    onNotificationClick?.(notification);
    
    // 如果有URL，跳转到指定页面
    if (notification.url) {
      window.location.href = notification.url;
    }
  }, [markAsRead, onNotificationClick]);
  
  // 筛选通知
  const filteredNotifications = notifications.filter(notification => {
    if (filter === 'unread') return !notification.isRead;
    if (filter === 'emails') return notification.type === 'email';
    if (filter === 'system') return ['system', 'security', 'warning', 'error'].includes(notification.type);
    return true;
  });
  
  // 计算未读数量
  useEffect(() => {
    const count = notifications.filter(n => !n.isRead).length;
    setUnreadCount(count);
  }, [notifications]);
  
  // 点击外部关闭
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (notificationRef.current && !notificationRef.current.contains(event.target)) {
        setIsOpen(false);
      }
    };
    
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen]);
  
  // 格式化时间
  const formatTime = (timestamp) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffInSeconds = Math.floor((now - date) / 1000);
    
    if (diffInSeconds < 60) return '刚刚';
    if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)}分钟前`;
    if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)}小时前`;
    if (diffInSeconds < 604800) return `${Math.floor(diffInSeconds / 86400)}天前`;
    
    return date.toLocaleDateString();
  };
  
  // 请求通知权限
  const requestNotificationPermission = useCallback(async () => {
    if ('Notification' in window && Notification.permission === 'default') {
      await Notification.requestPermission();
    }
  }, []);
  
  // 组件挂载时请求通知权限
  useEffect(() => {
    requestNotificationPermission();
  }, [requestNotificationPermission]);
  
  // 暴露给父组件的方法
  React.useImperativeHandle(notificationRef, () => ({
    addNotification,
    markAsRead,
    dismissNotification,
    markAllAsRead,
    clearAllNotifications,
    clearReadNotifications
  }));
  
  return (
    <div className={`notification-center ${className}`} style={style} ref={notificationRef}>
      {/* 通知按钮 */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        style={{
          position: 'relative',
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          padding: '8px',
          borderRadius: '6px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: '20px',
          color: '#666',
          transition: 'all 0.2s ease'
        }}
        onMouseEnter={(e) => {
          e.target.style.backgroundColor = '#f0f0f0';
        }}
        onMouseLeave={(e) => {
          e.target.style.backgroundColor = 'transparent';
        }}
      >
        🔔
        {unreadCount > 0 && (
          <span
            style={{
              position: 'absolute',
              top: '2px',
              right: '2px',
              backgroundColor: '#f44336',
              color: 'white',
              borderRadius: '10px',
              fontSize: '10px',
              fontWeight: 'bold',
              minWidth: '16px',
              height: '16px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              lineHeight: 1
            }}
          >
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </button>
      
      {/* 通知面板 */}
      {isOpen && (
        <div
          style={{
            position: 'absolute',
            top: '100%',
            right: 0,
            width: '360px',
            maxHeight: '500px',
            backgroundColor: '#fff',
            border: '1px solid #e0e0e0',
            borderRadius: '8px',
            boxShadow: '0 4px 20px rgba(0, 0, 0, 0.15)',
            zIndex: 1000,
            overflow: 'hidden'
          }}
        >
          {/* 头部 */}
          <div
            style={{
              padding: '16px',
              borderBottom: '1px solid #e0e0e0',
              backgroundColor: '#f8f9fa'
            }}
          >
            <div style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginBottom: '12px'
            }}>
              <h3 style={{ margin: 0, fontSize: '16px', color: '#333' }}>
                通知中心
                {unreadCount > 0 && (
                  <span style={{ color: '#f44336', marginLeft: '8px' }}>
                    ({unreadCount} 未读)
                  </span>
                )}
              </h3>
              <button
                onClick={() => setIsOpen(false)}
                style={{
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  fontSize: '18px',
                  color: '#999'
                }}
              >
                ×
              </button>
            </div>
            
            {/* 筛选按钮 */}
            <div style={{ display: 'flex', gap: '4px' }}>
              {[
                { key: 'all', label: '全部' },
                { key: 'unread', label: '未读' },
                { key: 'emails', label: '邮件' },
                { key: 'system', label: '系统' }
              ].map(filterOption => (
                <button
                  key={filterOption.key}
                  onClick={() => setFilter(filterOption.key)}
                  style={{
                    padding: '6px 12px',
                    border: '1px solid #ddd',
                    borderRadius: '12px',
                    backgroundColor: filter === filterOption.key ? '#e3f2fd' : '#fff',
                    color: filter === filterOption.key ? '#1976d2' : '#666',
                    cursor: 'pointer',
                    fontSize: '12px',
                    fontWeight: filter === filterOption.key ? '500' : 'normal'
                  }}
                >
                  {filterOption.label}
                </button>
              ))}
            </div>
          </div>
          
          {/* 操作按钮 */}
          {notifications.length > 0 && (
            <div style={{
              padding: '8px 16px',
              borderBottom: '1px solid #e0e0e0',
              backgroundColor: '#f8f9fa',
              display: 'flex',
              gap: '8px'
            }}>
              {unreadCount > 0 && (
                <button
                  onClick={markAllAsRead}
                  style={{
                    padding: '4px 8px',
                    border: '1px solid #4CAF50',
                    borderRadius: '4px',
                    backgroundColor: '#fff',
                    color: '#4CAF50',
                    cursor: 'pointer',
                    fontSize: '12px'
                  }}
                >
                  全部已读
                </button>
              )}
              <button
                onClick={clearReadNotifications}
                style={{
                  padding: '4px 8px',
                  border: '1px solid #FF9800',
                  borderRadius: '4px',
                  backgroundColor: '#fff',
                  color: '#FF9800',
                  cursor: 'pointer',
                  fontSize: '12px'
                }}
              >
                清除已读
              </button>
              <button
                onClick={clearAllNotifications}
                style={{
                  padding: '4px 8px',
                  border: '1px solid #f44336',
                  borderRadius: '4px',
                  backgroundColor: '#fff',
                  color: '#f44336',
                  cursor: 'pointer',
                  fontSize: '12px'
                }}
              >
                清除全部
              </button>
            </div>
          )}
          
          {/* 通知列表 */}
          <div
            style={{
              maxHeight: '350px',
              overflowY: 'auto'
            }}
          >
            {isLoading ? (
              <div style={{
                padding: '20px',
                textAlign: 'center',
                color: '#666'
              }}>
                正在加载...
              </div>
            ) : filteredNotifications.length === 0 ? (
              <div style={{
                padding: '40px 20px',
                textAlign: 'center',
                color: '#999'
              }}>
                <div style={{ fontSize: '48px', marginBottom: '8px' }}>🔔</div>
                <div>暂无{filter === 'unread' ? '未读' : ''}通知</div>
              </div>
            ) : (
              filteredNotifications.map(notification => {
                const typeConfig = notificationTypes[notification.type] || notificationTypes.system;
                
                return (
                  <div
                    key={notification.id}
                    onClick={() => handleNotificationClick(notification)}
                    style={{
                      padding: '12px 16px',
                      borderBottom: '1px solid #f0f0f0',
                      cursor: 'pointer',
                      backgroundColor: notification.isRead ? '#fff' : '#f0f8ff',
                      transition: 'background-color 0.2s ease',
                      position: 'relative'
                    }}
                    onMouseEnter={(e) => {
                      e.target.style.backgroundColor = notification.isRead ? '#f8f9fa' : '#e3f2fd';
                    }}
                    onMouseLeave={(e) => {
                      e.target.style.backgroundColor = notification.isRead ? '#fff' : '#f0f8ff';
                    }}
                  >
                    {/* 未读指示器 */}
                    {!notification.isRead && (
                      <div
                        style={{
                          position: 'absolute',
                          left: '4px',
                          top: '50%',
                          transform: 'translateY(-50%)',
                          width: '6px',
                          height: '6px',
                          borderRadius: '50%',
                          backgroundColor: typeConfig.color
                        }}
                      />
                    )}
                    
                    <div style={{ display: 'flex', alignItems: 'flex-start', gap: '8px' }}>
                      {/* 图标 */}
                      <span style={{
                        fontSize: '16px',
                        marginTop: '2px'
                      }}>
                        {typeConfig.icon}
                      </span>
                      
                      {/* 内容 */}
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{
                          fontSize: '14px',
                          fontWeight: notification.isRead ? 'normal' : '500',
                          color: '#333',
                          marginBottom: '4px',
                          lineHeight: '1.3'
                        }}>
                          {notification.title || typeConfig.title}
                        </div>
                        <div style={{
                          fontSize: '13px',
                          color: '#666',
                          lineHeight: '1.4',
                          marginBottom: '6px'
                        }}>
                          {notification.message}
                        </div>
                        <div style={{
                          fontSize: '11px',
                          color: '#999'
                        }}>
                          {formatTime(notification.timestamp)}
                        </div>
                      </div>
                      
                      {/* 关闭按钮 */}
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          dismissNotification(notification.id);
                        }}
                        style={{
                          background: 'none',
                          border: 'none',
                          cursor: 'pointer',
                          color: '#ccc',
                          fontSize: '16px',
                          padding: '0',
                          lineHeight: 1
                        }}
                      >
                        ×
                      </button>
                    </div>
                    
                    {/* 优先级指示器 */}
                    {notification.priority === 'high' && (
                      <div
                        style={{
                          position: 'absolute',
                          right: '8px',
                          top: '8px',
                          width: '8px',
                          height: '8px',
                          borderRadius: '50%',
                          backgroundColor: '#f44336'
                        }}
                      />
                    )}
                  </div>
                );
              })
            )}
          </div>
        </div>
      )}
      
      {/* 通知声音 */}
      <audio
        ref={soundRef}
        preload="auto"
        style={{ display: 'none' }}
      >
        <source src="/assets/notification.mp3" type="audio/mpeg" />
        <source src="/assets/notification.ogg" type="audio/ogg" />
      </audio>
    </div>
  );
};

export default NotificationCenter;