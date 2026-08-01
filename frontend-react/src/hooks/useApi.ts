import { useState, useCallback } from 'react';
import ApiService from '../services/api.service';
import AuthService from '../services/auth.service';

export const useApi = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const request = useCallback(async <T = any>(
    endpoint: string,
    options: {
      method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
      headers?: Record<string, string>;
      body?: any;
      requireAuth?: boolean;
    } = {}
  ): Promise<T> => {
    setLoading(true);
    setError(null);

    try {
      const result = await (ApiService as unknown as {
        request<T = any>(
          endpoint: string,
          options: {
            method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
            headers?: Record<string, string>;
            body?: any;
            requireAuth?: boolean;
          }
        ): Promise<T>;
      }).request<T>(endpoint, options);
      return result;
    } catch (err: any) {
      setError(err.message || 'Erro na requisição');
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  const get = useCallback(<T = any>(endpoint: string, options?: Omit<Parameters<typeof request>[1], 'method' | 'body'>) => {
    return request<T>(endpoint, { ...options, method: 'GET' });
  }, [request]);

  const post = useCallback(<T = any>(endpoint: string, body?: any, options?: Omit<Parameters<typeof request>[1], 'method' | 'body'>) => {
    return request<T>(endpoint, { ...options, method: 'POST', body });
  }, [request]);

  const put = useCallback(<T = any>(endpoint: string, body?: any, options?: Omit<Parameters<typeof request>[1], 'method' | 'body'>) => {
    return request<T>(endpoint, { ...options, method: 'PUT', body });
  }, [request]);

  const patch = useCallback(<T = any>(endpoint: string, body?: any, options?: Omit<Parameters<typeof request>[1], 'method' | 'body'>) => {
    return request<T>(endpoint, { ...options, method: 'PATCH', body });
  }, [request]);

  const del = useCallback(<T = any>(endpoint: string, options?: Omit<Parameters<typeof request>[1], 'method' | 'body'>) => {
    return request<T>(endpoint, { ...options, method: 'DELETE' });
  }, [request]);

  return {
    loading,
    error,
    request,
    get,
    post,
    put,
    patch,
    delete: del,
  };
};

export default useApi;