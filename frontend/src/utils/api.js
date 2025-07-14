import { cacheManager } from './cache.js';
import { auth } from './auth.js';

// Base64解码工具
const decodeBase64IfNeeded = (text) => {
    if (!text || typeof text !== 'string') return text;
    
    const base64Regex = /^[A-Za-z0-9+/]+=*$/;
    if (text.length >= 8 && text.length % 4 === 0 && base64Regex.test(text)) {
        try {
            const decoded = atob(text);
            if (decoded && decoded.length > 0) {
                return decoded;
            }
        } catch (e) {
            // 解码失败，返回原文
        }
    }
    return text;
};

// 创建带认证的fetch函数
const authenticatedFetch = async (url, options = {}) => {
    const authHeaders = auth.getAuthHeaders();
    
    const fetchOptions = {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options.headers,
            ...authHeaders
        }
    };

    const response = await fetch(url, fetchOptions);
    
    // 如果返回401，表示认证失败，清除认证状态
    if (response.status === 401) {
        auth.clearAuth();
        window.location.reload(); // 强制刷新页面回到登录状态
        throw new Error('认证已过期，请重新登录');
    }
    
    return response;
};

// API接口
export const api = {
    async getMailboxes() {
        const cached = cacheManager.get('mailboxes');
        if (cached) return cached;

        // 直接调用API，不使用认证
        const response = await fetch('/api/mailboxes', {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json'
            }
        });
        
        if (!response.ok) {
            console.error('API响应失败:', response.status, response.statusText);
            throw new Error(`获取邮箱列表失败: ${response.status}`);
        }
        
        let data;
        try {
            const responseText = await response.text();
            console.log('API响应原始数据:', responseText);
            data = JSON.parse(responseText);
        } catch (jsonError) {
            console.error('JSON解析错误:', jsonError);
            throw new Error('服务器返回了无效的JSON数据');
        }
        
        // 确保返回的是数组
        if (!Array.isArray(data)) {
            console.error('邮箱数据不是数组:', data);
            return [];
        }
        
        console.log('解析后的邮箱数据:', data);
        cacheManager.set('mailboxes', data);
        return data;
    },

    async getEmails(mailbox) {
        if (!mailbox || typeof mailbox !== 'string') {
            throw new Error('邮箱名称不能为空');
        }
        
        const cached = cacheManager.get(`emails-${mailbox}`);
        if (cached) return cached;

        // 直接调用API，不使用认证
        const response = await fetch(`/api/emails/${encodeURIComponent(mailbox)}`, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json'
            }
        });
        
        if (!response.ok) {
            throw new Error('获取邮件失败');
        }
        
        const data = await response.json();
        
        // 确保数据是数组
        if (!Array.isArray(data)) {
            console.warn('API返回的数据不是数组:', data);
            return [];
        }
        
        // 自动解码Base64内容
        const decodedData = data.map(email => ({
            ...email,
            Body: decodeBase64IfNeeded(email.Body),
            Subject: decodeBase64IfNeeded(email.Subject)
        }));

        cacheManager.set(`emails-${mailbox}`, decodedData);
        return decodedData;
    },

    async deleteEmail(mailbox, emailId) {
        const response = await authenticatedFetch(`/api/emails/delete/${encodeURIComponent(mailbox)}/${emailId}`, {
            method: 'DELETE',
        });

        if (response.ok) {
            // 删除成功，只清除该邮箱的邮件缓存
            cacheManager.clear(`emails-${mailbox}`);
            cacheManager.clear('mailboxes'); // 也清除邮箱列表缓存以更新统计
            return true;
        } else {
            const errorText = await response.text();
            throw new Error(errorText);
        }
    },

    async sendEmail(emailData) {
        const response = await authenticatedFetch('/api/send', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(emailData)
        });

        if (response.ok) {
            return await response.json();
        } else {
            const errorText = await response.text();
            throw new Error(errorText || '发送邮件失败');
        }
    },

    async createMailbox(mailboxData) {
        const response = await authenticatedFetch('/api/mailboxes/create', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(mailboxData)
        });

        if (response.ok) {
            // 创建成功，清除邮箱列表缓存
            cacheManager.clear('mailboxes');
            return await response.json();
        } else {
            const errorText = await response.text();
            throw new Error(errorText || '创建邮箱失败');
        }
    }
};

// 导出便捷方法
export const sendEmail = api.sendEmail.bind(api);
export const createMailbox = api.createMailbox.bind(api);