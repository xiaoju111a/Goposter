import React, { useState, useEffect } from 'react';

const ForwardingSettings = () => {
    const [settings, setSettings] = useState({
        forward_enabled: false,
        forward_to: '',
        keep_original: true
    });
    const [loading, setLoading] = useState(false);
    const [currentUser, setCurrentUser] = useState('');

    useEffect(() => {
        fetchSettings();
    }, []);

    const fetchSettings = async () => {
        try {
            const response = await fetch('/api/forwarding/settings', {
                headers: {
                    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
                }
            });
            if (response.ok) {
                const data = await response.json();
                setSettings(data);
                setCurrentUser(data.mailbox || '');
            }
        } catch (error) {
            console.error('获取转发设置失败:', error);
        }
    };

    const updateSettings = async () => {
        setLoading(true);
        try {
            const response = await fetch('/api/forwarding/update', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
                },
                body: JSON.stringify({
                    forward_to: settings.forward_to,
                    forward_enabled: settings.forward_enabled,
                    keep_original: settings.keep_original
                })
            });
            if (response.ok) {
                alert('转发设置更新成功');
            } else {
                alert('转发设置更新失败');
            }
        } catch (error) {
            console.error('更新转发设置失败:', error);
            alert('更新失败');
        } finally {
            setLoading(false);
        }
    };

    const validateEmail = (email) => {
        const re = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        return re.test(email);
    };

    const isFormValid = () => {
        if (!settings.forward_enabled) return true;
        return settings.forward_to && validateEmail(settings.forward_to);
    };

    return (
        <div className="forwarding-settings">
            <h3>邮件转发设置</h3>
            <p>当前邮箱: <strong>{currentUser}</strong></p>
            
            <div className="setting-item">
                <label>
                    <input
                        type="checkbox"
                        checked={settings.forward_enabled}
                        onChange={(e) => setSettings({
                            ...settings,
                            forward_enabled: e.target.checked
                        })}
                    />
                    启用邮件转发
                </label>
            </div>

            {settings.forward_enabled && (
                <>
                    <div className="setting-item">
                        <label>转发到邮箱:</label>
                        <input
                            type="email"
                            value={settings.forward_to}
                            onChange={(e) => setSettings({
                                ...settings,
                                forward_to: e.target.value
                            })}
                            placeholder="输入转发邮箱地址"
                            className={settings.forward_to && !validateEmail(settings.forward_to) ? 'invalid' : ''}
                        />
                        {settings.forward_to && !validateEmail(settings.forward_to) && (
                            <span className="error">请输入有效的邮箱地址</span>
                        )}
                    </div>

                    <div className="setting-item">
                        <label>
                            <input
                                type="checkbox"
                                checked={settings.keep_original}
                                onChange={(e) => setSettings({
                                    ...settings,
                                    keep_original: e.target.checked
                                })}
                            />
                            保留原邮件副本
                        </label>
                        <small>
                            {settings.keep_original ? 
                                '邮件将同时保存在本邮箱和转发到目标邮箱' : 
                                '邮件只会转发到目标邮箱，不会保存在本邮箱'
                            }
                        </small>
                    </div>
                </>
            )}

            <div className="setting-actions">
                <button 
                    onClick={updateSettings}
                    disabled={!isFormValid() || loading}
                    className="save-btn"
                >
                    {loading ? '保存中...' : '保存设置'}
                </button>
            </div>

            {settings.forward_enabled && (
                <div className="warning">
                    <strong>注意:</strong> 
                    <ul>
                        <li>转发功能将自动转发所有接收到的邮件</li>
                        <li>请确保转发邮箱地址正确且有效</li>
                        <li>如果关闭"保留原邮件副本"，邮件将不会存储在本系统中</li>
                    </ul>
                </div>
            )}
        </div>
    );
};

export default ForwardingSettings;