import React, { useState, useEffect } from 'react';

const AdminPanel = () => {
    const [users, setUsers] = useState([]);
    const [mailboxes, setMailboxes] = useState([]);
    const [activeTab, setActiveTab] = useState('users');
    const [newUser, setNewUser] = useState({ email: '', password: '', isAdmin: false });
    const [newMailbox, setNewMailbox] = useState({ email: '', password: '', description: '', owner: '' });

    useEffect(() => {
        fetchUsers();
        fetchMailboxes();
    }, []);

    const fetchUsers = async () => {
        try {
            const response = await fetch('/api/admin/users', {
                headers: {
                    'Authorization': `Bearer ${localStorage.getItem('token')}`
                }
            });
            if (response.ok) {
                const data = await response.json();
                setUsers(data.users);
            }
        } catch (error) {
            console.error('获取用户列表失败:', error);
        }
    };

    const fetchMailboxes = async () => {
        try {
            const response = await fetch('/api/admin/mailboxes', {
                headers: {
                    'Authorization': `Bearer ${localStorage.getItem('token')}`
                }
            });
            if (response.ok) {
                const data = await response.json();
                setMailboxes(data.mailboxes);
            }
        } catch (error) {
            console.error('获取邮箱列表失败:', error);
        }
    };

    const createUser = async () => {
        try {
            const response = await fetch('/api/admin/users/create', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${localStorage.getItem('token')}`
                },
                body: JSON.stringify(newUser)
            });
            if (response.ok) {
                fetchUsers();
                setNewUser({ email: '', password: '', isAdmin: false });
                alert('用户创建成功');
            } else {
                alert('用户创建失败');
            }
        } catch (error) {
            console.error('创建用户失败:', error);
        }
    };

    const createMailbox = async () => {
        try {
            const response = await fetch('/api/admin/mailboxes/create', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${localStorage.getItem('token')}`
                },
                body: JSON.stringify(newMailbox)
            });
            if (response.ok) {
                fetchMailboxes();
                setNewMailbox({ email: '', password: '', description: '', owner: '' });
                alert('邮箱创建成功');
            } else {
                alert('邮箱创建失败');
            }
        } catch (error) {
            console.error('创建邮箱失败:', error);
        }
    };

    return (
        <div className="admin-panel">
            <h2>管理员控制面板</h2>
            
            <div className="tab-nav">
                <button 
                    className={activeTab === 'users' ? 'active' : ''}
                    onClick={() => setActiveTab('users')}
                >
                    用户管理
                </button>
                <button 
                    className={activeTab === 'mailboxes' ? 'active' : ''}
                    onClick={() => setActiveTab('mailboxes')}
                >
                    邮箱管理
                </button>
            </div>

            {activeTab === 'users' && (
                <div className="users-section">
                    <h3>用户管理</h3>
                    
                    <div className="create-user-form">
                        <h4>创建新用户</h4>
                        <input
                            type="email"
                            placeholder="邮箱"
                            value={newUser.email}
                            onChange={(e) => setNewUser({...newUser, email: e.target.value})}
                        />
                        <input
                            type="password"
                            placeholder="密码"
                            value={newUser.password}
                            onChange={(e) => setNewUser({...newUser, password: e.target.value})}
                        />
                        <label>
                            <input
                                type="checkbox"
                                checked={newUser.isAdmin}
                                onChange={(e) => setNewUser({...newUser, isAdmin: e.target.checked})}
                            />
                            管理员权限
                        </label>
                        <button onClick={createUser}>创建用户</button>
                    </div>

                    <div className="users-list">
                        <h4>用户列表</h4>
                        <table>
                            <thead>
                                <tr>
                                    <th>邮箱</th>
                                    <th>管理员</th>
                                    <th>创建时间</th>
                                    <th>最后登录</th>
                                    <th>2FA</th>
                                </tr>
                            </thead>
                            <tbody>
                                {users.map(user => (
                                    <tr key={user.email}>
                                        <td>{user.email}</td>
                                        <td>{user.is_admin ? '是' : '否'}</td>
                                        <td>{new Date(user.created_at).toLocaleString()}</td>
                                        <td>{user.last_login ? new Date(user.last_login).toLocaleString() : '从未'}</td>
                                        <td>{user.two_factor_enabled ? '已启用' : '未启用'}</td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                </div>
            )}

            {activeTab === 'mailboxes' && (
                <div className="mailboxes-section">
                    <h3>邮箱管理</h3>
                    
                    <div className="create-mailbox-form">
                        <h4>创建新邮箱</h4>
                        <input
                            type="email"
                            placeholder="邮箱地址"
                            value={newMailbox.email}
                            onChange={(e) => setNewMailbox({...newMailbox, email: e.target.value})}
                        />
                        <input
                            type="password"
                            placeholder="密码"
                            value={newMailbox.password}
                            onChange={(e) => setNewMailbox({...newMailbox, password: e.target.value})}
                        />
                        <input
                            type="text"
                            placeholder="描述"
                            value={newMailbox.description}
                            onChange={(e) => setNewMailbox({...newMailbox, description: e.target.value})}
                        />
                        <input
                            type="email"
                            placeholder="所有者邮箱"
                            value={newMailbox.owner}
                            onChange={(e) => setNewMailbox({...newMailbox, owner: e.target.value})}
                        />
                        <button onClick={createMailbox}>创建邮箱</button>
                    </div>

                    <div className="mailboxes-list">
                        <h4>邮箱列表</h4>
                        <table>
                            <thead>
                                <tr>
                                    <th>邮箱</th>
                                    <th>用户名</th>
                                    <th>描述</th>
                                    <th>所有者</th>
                                    <th>转发</th>
                                    <th>状态</th>
                                </tr>
                            </thead>
                            <tbody>
                                {mailboxes.map(mailbox => (
                                    <tr key={mailbox.email}>
                                        <td>{mailbox.email}</td>
                                        <td>{mailbox.username}</td>
                                        <td>{mailbox.description}</td>
                                        <td>{mailbox.owner}</td>
                                        <td>
                                            {mailbox.forward_enabled ? 
                                                `→ ${mailbox.forward_to}` : 
                                                '未启用'
                                            }
                                        </td>
                                        <td>{mailbox.is_active ? '激活' : '停用'}</td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                </div>
            )}
        </div>
    );
};

export default AdminPanel;