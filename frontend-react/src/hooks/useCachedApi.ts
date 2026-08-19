import { useState, useCallback, useEffect, useRef } from 'react';
import ApiService from '../services/api.service';
import CacheService from '../services/cache.service';
import { useToast } from './useToast';

interface UseCachedApiOptions {
  cacheKey?: string;
  ttl?: number; // Tempo de vida em milissegundos
  staleTime?: number; // Tempo para considerar stale
  enabled?: boolean;
  onSuccess?: (data: any) => void;
  onError?: (error: any) => void;
}

interface UseCachedApiResult<T> {
  data: T | null;
  loading: boolean;
  error: Error | null;
  isStale: boolean;
  refetch: () => Promise<void>;
  invalidate: () => void;
  clearCache: () => void;
}

export function useCachedApi<T = any>(
  endpoint: string,
  params?: Record<string, any>,
  options: UseCachedApiOptions = {}
): UseCachedApiResult<T> {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const [isStale, setIsStale] = useState(false);
  const { error: showError } = useToast();
  const isMounted = useRef(true);

  const cacheKey = options.cacheKey || CacheService.generateKey(endpoint, params);
  const ttl = options.ttl || 5 * 60 * 1000; // 5 minutos
  const enabled = options.enabled !== false;

  const fetchData = useCallback(async () => {
    if (!enabled) return;

    setLoading(true);
    setError(null);

    try {
      // Verificar cache primeiro
      const cachedData = CacheService.get<T>(cacheKey);
      if (cachedData !== null) {
        setData(cachedData);
        setIsStale(CacheService.isStale(cacheKey));
        setLoading(false);
        
        // Se estiver stale, fazer refetch em background
        if (CacheService.isStale(cacheKey)) {
          // Buscar em background
          ApiService.get<T>(endpoint, { params })
            .then((freshData) => {
              if (isMounted.current) {
                setData(freshData);
                setIsStale(false);
                CacheService.set(cacheKey, freshData, { ttl });
                options.onSuccess?.(freshData);
              }
            })
            .catch((err) => {
              console.warn('Erro ao atualizar dados em background:', err);
            });
        }
        return;
      }

      // Buscar da API
      const response = await ApiService.get<T>(endpoint, { params });
      
      if (isMounted.current) {
        setData(response);
        setIsStale(false);
        CacheService.set(cacheKey, response, { ttl });
        options.onSuccess?.(response);
      }
    } catch (err: any) {
      if (isMounted.current) {
        setError(err);
        showError(err.message || 'Erro ao carregar dados');
        options.onError?.(err);
      }
    } finally {
      if (isMounted.current) {
        setLoading(false);
      }
    }
  }, [endpoint, params, cacheKey, ttl, enabled, showError, options]);

  // Forçar refetch
  const refetch = useCallback(async () => {
    // Invalidar cache antes de buscar
    CacheService.delete(cacheKey);
    await fetchData();
  }, [cacheKey, fetchData]);

  // Invalidar cache específico
  const invalidate = useCallback(() => {
    CacheService.delete(cacheKey);
    setIsStale(true);
  }, [cacheKey]);

  // Limpar todo o cache
  const clearCache = useCallback(() => {
    CacheService.clear();
  }, []);

  // Carregar dados iniciais
  useEffect(() => {
    isMounted.current = true;
    fetchData();

    return () => {
      isMounted.current = false;
    };
  }, [fetchData]);

  return {
    data,
    loading,
    error,
    isStale,
    refetch,
    invalidate,
    clearCache,
  };
}