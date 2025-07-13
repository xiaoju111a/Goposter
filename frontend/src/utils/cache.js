class CacheManager {
    constructor() {
        this.cache = new Map();
        this.ttl = 30000; // 30秒缓存
    }

    get(key) {
        const item = this.cache.get(key);
        if (item && Date.now() - item.timestamp < this.ttl) {
            return item.data;
        }
        return null;
    }

    set(key, data) {
        this.cache.set(key, {
            data,
            timestamp: Date.now()
        });
    }

    clear() {
        this.cache.clear();
    }
}

export const cacheManager = new CacheManager();