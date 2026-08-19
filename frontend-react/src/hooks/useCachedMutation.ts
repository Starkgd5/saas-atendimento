import { useState, useCallback } from 'react';
import ApiService from '../services/api.service';
import CacheService from '../services/cache.service';
import { useToast } from './useToast';

interface UseCachedMutationOptions {
  invalidateKeys?: string[]; // Chaves do cache para invalidar
  invalidatePatterns?: string[]; // Padrões para invalidar
  onSuccess?: (data: any) => void;
  onError?: (error: any) => void;
}

export function useCachedMutation<T = any, V = any>(
  endpoint: string,
  method: 'POST' | 'PUT' | 'PATCH' | 'DELETE' = 'POST',
  options: UseCachedMutationOptions = {}
) {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const { success, error: showError } = useToast();

  const mutate = useCallback(
    async (body?: V, params?: Record<string, any>) => {
      setLoading(true);
      setError(null);

      try {
        let response: T;

        switch (method) {
          case 'POST':
            response = await ApiService.post<T>(endpoint, body, { params });
            break;
          case 'PUT':
            response = await ApiService.put<T>(endpoint, body, { params });
            break;
          case 'PATCH':
            response = await ApiService.patch<T>(endpoint, body, { params });
            break;
          case 'DELETE':
            response = await ApiService.delete<T>(endpoint, { params });
            break;
        }

        setData(response);
        options.onSuccess?.(response);

        // Invalidar cache
        if (options.invalidateKeys) {
          options.invalidateKeys.forEach((key) => {
            CacheService.delete(key);
          });
        }

        if (options.invalidatePatterns) {
          options.invalidatePatterns.forEach((pattern) => {
            CacheService.clearByPattern(pattern);
          });
        }

        // Se não especificou, invalidar por padrão
        if (!options.invalidateKeys && !options.invalidatePatterns) {
          // Invalidar cache baseado no endpoint
          const baseEndpoint = endpoint.split('/')[0];
          CacheService.clearByPattern(baseEndpoint);
        }

        return response;
      } catch (err: any) {
        setError(err);
        showError(err.message || 'Erro ao realizar operação');
        options.onError?.(err);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [endpoint, method, options, showError]
  );

  return {
    mutate,
    loading,
    data,
    error,
  };
}