interface CacheItem<T = any> {
  data: T;
  timestamp: number;
  expiresAt: number;
}

interface CacheOptions {
  ttl?: number; // Tempo de vida em milissegundos
  staleTime?: number; // Tempo para considerar stale (refetch)
  key?: string;
}

class CacheService {
  private static instance: CacheService;
  private cache: Map<string, CacheItem> = new Map();
  private defaultTTL = 5 * 60 * 1000; // 5 minutos
  private defaultStaleTime = 30 * 1000; // 30 segundos

  private constructor() {}

  static getInstance(): CacheService {
    if (!CacheService.instance) {
      CacheService.instance = new CacheService();
    }
    return CacheService.instance;
  }

  // ============================================
  // OPERAÇÕES BÁSICAS
  // ============================================

  set<T>(key: string, data: T, options: CacheOptions = {}): void {
    const ttl = options.ttl || this.defaultTTL;
    const now = Date.now();

    this.cache.set(key, {
      data,
      timestamp: now,
      expiresAt: now + ttl,
    });
  }

  get<T>(key: string): T | null {
    const item = this.cache.get(key);
    if (!item) return null;

    // Verificar se expirou
    if (Date.now() > item.expiresAt) {
      this.cache.delete(key);
      return null;
    }

    return item.data as T;
  }

  has(key: string): boolean {
    const item = this.cache.get(key);
    if (!item) return false;

    if (Date.now() > item.expiresAt) {
      this.cache.delete(key);
      return false;
    }

    return true;
  }

  isStale(key: string): boolean {
    const item = this.cache.get(key);
    if (!item) return true;

    const staleTime = this.defaultStaleTime;
    return Date.now() - item.timestamp > staleTime;
  }

  delete(key: string): void {
    this.cache.delete(key);
  }

  clear(): void {
    this.cache.clear();
  }

  clearByPattern(pattern: string): void {
    const regex = new RegExp(pattern);
    for (const key of Array.from(this.cache.keys())) {
      if (regex.test(key)) {
        this.cache.delete(key);
      }
    }
  }

  // ============================================
  // GERAÇÃO DE CHAVES
  // ============================================

  generateKey(endpoint: string, params?: Record<string, any>): string {
    if (!params || Object.keys(params).length === 0) {
      return endpoint;
    }

    const sortedParams = Object.keys(params)
      .sort()
      .reduce((acc, key) => {
        if (params[key] !== undefined && params[key] !== null && params[key] !== '') {
          acc[key] = params[key];
        }
        return acc;
      }, {} as Record<string, any>);

    const queryString = new URLSearchParams(sortedParams).toString();
    return `${endpoint}?${queryString}`;
  }

  // ============================================
  // ESTATÍSTICAS
  // ============================================

  getStats(): { total: number; keys: string[] } {
    const keys = Array.from(this.cache.keys());
    return {
      total: keys.length,
      keys,
    };
  }

  getSize(): number {
    return this.cache.size;
  }

  // ============================================
  // PERSISTÊNCIA (opcional)
  // ============================================

  saveToLocalStorage(prefix: string = 'cache_'): void {
    const data: Record<string, any> = {};
    for (const [key, value] of Array.from(this.cache.entries())) {
      data[`${prefix}${key}`] = value;
    }
    try {
      localStorage.setItem('app_cache', JSON.stringify(data));
    } catch (error) {
      console.warn('Erro ao salvar cache no localStorage:', error);
    }
  }

  loadFromLocalStorage(prefix: string = 'cache_'): void {
    try {
      const data = localStorage.getItem('app_cache');
      if (!data) return;

      const parsed = JSON.parse(data);
      const now = Date.now();

      for (const [key, value] of Object.entries(parsed)) {
        const item = value as CacheItem;
        // Verificar se ainda é válido
        if (item.expiresAt > now) {
          const originalKey = key.startsWith(prefix) ? key.substring(prefix.length) : key;
          this.cache.set(originalKey, item);
        }
      }
    } catch (error) {
      console.warn('Erro ao carregar cache do localStorage:', error);
    }
  }
}

export default CacheService.getInstance();