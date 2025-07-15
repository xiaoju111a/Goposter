// FreeAgent Mail Service Worker
const CACHE_NAME = 'freeagent-mail-v1.0.0';
const API_CACHE_NAME = 'freeagent-mail-api-v1.0.0';

// 需要缓存的静态资源
const STATIC_ASSETS = [
  '/',
  '/index.html',
  '/manifest.json',
  '/assets/icon-192.png',
  '/assets/icon-512.png'
];

// API缓存策略
const API_ENDPOINTS = [
  '/api/mailboxes',
  '/api/auth/profile',
  '/api/security/stats'
];

// 安装事件 - 缓存静态资源
self.addEventListener('install', (event) => {
  console.log('[SW] 安装中...');
  
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => {
        console.log('[SW] 缓存静态资源');
        return cache.addAll(STATIC_ASSETS.map(url => 
          new Request(url, { cache: 'reload' })
        ));
      })
      .then(() => {
        console.log('[SW] 安装完成');
        return self.skipWaiting();
      })
      .catch((error) => {
        console.error('[SW] 安装失败:', error);
      })
  );
});

// 激活事件 - 清理旧缓存
self.addEventListener('activate', (event) => {
  console.log('[SW] 激活中...');
  
  event.waitUntil(
    caches.keys()
      .then((cacheNames) => {
        return Promise.all(
          cacheNames.map((cacheName) => {
            if (cacheName !== CACHE_NAME && cacheName !== API_CACHE_NAME) {
              console.log('[SW] 删除旧缓存:', cacheName);
              return caches.delete(cacheName);
            }
          })
        );
      })
      .then(() => {
        console.log('[SW] 激活完成');
        return self.clients.claim();
      })
  );
});

// 网络请求拦截
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);
  
  // 跳过非同源请求
  if (url.origin !== location.origin) {
    return;
  }
  
  // API请求处理
  if (url.pathname.startsWith('/api/')) {
    event.respondWith(handleAPIRequest(request));
    return;
  }
  
  // 静态资源处理
  event.respondWith(handleStaticRequest(request));
});

// API请求处理策略：网络优先，缓存备用
async function handleAPIRequest(request) {
  const url = new URL(request.url);
  const cacheKey = `${request.method}_${url.pathname}${url.search}`;
  
  try {
    // 尝试网络请求
    const networkResponse = await fetch(request.clone());
    
    if (networkResponse.ok) {
      // 缓存成功的GET请求
      if (request.method === 'GET') {
        const cache = await caches.open(API_CACHE_NAME);
        cache.put(cacheKey, networkResponse.clone());
      }
      return networkResponse;
    }
    
    throw new Error(`网络请求失败: ${networkResponse.status}`);
    
  } catch (error) {
    console.log('[SW] 网络请求失败，尝试缓存:', error);
    
    // 网络失败时返回缓存
    if (request.method === 'GET') {
      const cache = await caches.open(API_CACHE_NAME);
      const cachedResponse = await cache.match(cacheKey);
      
      if (cachedResponse) {
        console.log('[SW] 返回缓存响应:', url.pathname);
        return cachedResponse;
      }
    }
    
    // 返回离线页面或错误响应
    return createOfflineResponse(request);
  }
}

// 静态资源处理策略：缓存优先，网络备用
async function handleStaticRequest(request) {
  try {
    // 先尝试缓存
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }
    
    // 缓存未命中，尝试网络
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      // 缓存新资源
      const cache = await caches.open(CACHE_NAME);
      cache.put(request, networkResponse.clone());
    }
    
    return networkResponse;
    
  } catch (error) {
    console.log('[SW] 静态资源加载失败:', error);
    
    // 对于HTML请求，返回离线页面
    if (request.destination === 'document') {
      return caches.match('/offline.html') || createOfflineResponse(request);
    }
    
    return createOfflineResponse(request);
  }
}

// 创建离线响应
function createOfflineResponse(request) {
  if (request.destination === 'document') {
    return new Response(`
      <!DOCTYPE html>
      <html>
      <head>
        <title>FreeAgent Mail - 离线模式</title>
        <meta charset="utf-8">
        <meta name="viewport" content="width=device-width, initial-scale=1">
        <style>
          body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            text-align: center; 
            padding: 50px; 
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            margin: 0;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            flex-direction: column;
          }
          .offline-container {
            background: rgba(255, 255, 255, 0.1);
            padding: 40px;
            border-radius: 20px;
            backdrop-filter: blur(10px);
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
          }
          .offline-icon { font-size: 4rem; margin-bottom: 20px; }
          .offline-title { font-size: 2rem; margin-bottom: 20px; }
          .offline-message { font-size: 1.2rem; margin-bottom: 30px; opacity: 0.9; }
          .retry-btn {
            background: rgba(255, 255, 255, 0.2);
            border: 2px solid rgba(255, 255, 255, 0.3);
            color: white;
            padding: 12px 24px;
            border-radius: 25px;
            cursor: pointer;
            font-size: 1rem;
            transition: all 0.3s ease;
          }
          .retry-btn:hover {
            background: rgba(255, 255, 255, 0.3);
            transform: translateY(-2px);
          }
        </style>
      </head>
      <body>
        <div class="offline-container">
          <div class="offline-icon">📧</div>
          <h1 class="offline-title">FreeAgent Mail</h1>
          <p class="offline-message">当前处于离线模式<br>请检查网络连接后重试</p>
          <button class="retry-btn" onclick="window.location.reload()">重新连接</button>
        </div>
        
        <script>
          // 监听网络状态恢复
          window.addEventListener('online', () => {
            window.location.reload();
          });
        </script>
      </body>
      </html>
    `, {
      headers: { 'Content-Type': 'text/html' }
    });
  }
  
  return new Response('离线模式 - 资源不可用', {
    status: 503,
    statusText: 'Service Unavailable'
  });
}

// 消息处理
self.addEventListener('message', (event) => {
  const { type, payload } = event.data;
  
  switch (type) {
    case 'SKIP_WAITING':
      self.skipWaiting();
      break;
      
    case 'CLEAR_CACHE':
      clearAllCaches().then(() => {
        event.ports[0]?.postMessage({ success: true });
      });
      break;
      
    case 'CACHE_EMAILS':
      cacheEmails(payload).then(() => {
        event.ports[0]?.postMessage({ success: true });
      });
      break;
      
    default:
      console.log('[SW] 未知消息类型:', type);
  }
});

// 清理所有缓存
async function clearAllCaches() {
  const cacheNames = await caches.keys();
  await Promise.all(
    cacheNames.map(cacheName => caches.delete(cacheName))
  );
  console.log('[SW] 所有缓存已清理');
}

// 缓存邮件数据
async function cacheEmails(emails) {
  try {
    const cache = await caches.open(API_CACHE_NAME);
    
    for (const mailbox of emails) {
      const cacheKey = `GET_/api/emails/${mailbox}`;
      const response = new Response(JSON.stringify(emails[mailbox] || []), {
        headers: { 'Content-Type': 'application/json' }
      });
      await cache.put(cacheKey, response);
    }
    
    console.log('[SW] 邮件数据缓存完成');
  } catch (error) {
    console.error('[SW] 邮件缓存失败:', error);
  }
}

// 定期清理过期缓存
self.addEventListener('periodicsync', (event) => {
  if (event.tag === 'cleanup-cache') {
    event.waitUntil(cleanupExpiredCache());
  }
});

async function cleanupExpiredCache() {
  const cache = await caches.open(API_CACHE_NAME);
  const requests = await cache.keys();
  const maxAge = 24 * 60 * 60 * 1000; // 24小时
  
  for (const request of requests) {
    const response = await cache.match(request);
    if (response) {
      const date = new Date(response.headers.get('date'));
      if (Date.now() - date.getTime() > maxAge) {
        await cache.delete(request);
      }
    }
  }
}

// 推送通知处理
self.addEventListener('push', (event) => {
  if (!event.data) return;
  
  try {
    const data = event.data.json();
    const options = {
      body: data.body || '您有新邮件',
      icon: '/assets/icon-192.png',
      badge: '/assets/icon-72.png',
      tag: 'email-notification',
      requireInteraction: true,
      actions: [
        {
          action: 'view',
          title: '查看邮件',
          icon: '/assets/view-icon.png'
        },
        {
          action: 'close',
          title: '关闭',
          icon: '/assets/close-icon.png'
        }
      ],
      data: {
        url: data.url || '/',
        mailbox: data.mailbox
      }
    };
    
    event.waitUntil(
      self.registration.showNotification(data.title || 'FreeAgent Mail', options)
    );
  } catch (error) {
    console.error('[SW] 推送通知处理失败:', error);
  }
});

// 通知点击处理
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  
  const { action, data } = event;
  
  if (action === 'view' || !action) {
    event.waitUntil(
      clients.matchAll({ type: 'window', includeUncontrolled: true })
        .then((clients) => {
          // 查找已打开的窗口
          for (const client of clients) {
            if (client.url.includes(location.origin)) {
              client.focus();
              client.postMessage({
                type: 'NAVIGATE_TO_MAILBOX',
                mailbox: data?.mailbox
              });
              return;
            }
          }
          
          // 打开新窗口
          return clients.openWindow(data?.url || '/');
        })
    );
  }
});

console.log('[SW] Service Worker 已加载');