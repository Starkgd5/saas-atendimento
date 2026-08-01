import AuthService from './auth.service';

class ApiService {
  private static instance: ApiService;
  private baseUrl = process.env.REACT_APP_API_URL || '/api/v1';
  private defaultTimeout = 30000;

  private constructor() {}

  static getInstance(): ApiService {
    if (!ApiService.instance) {
      ApiService.instance = new ApiService();
    }
    return ApiService.instance;
  }

  private async request<T = any>(
    endpoint: string,
    options: {
      method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
      headers?: Record<string, string>;
      body?: any;
      requireAuth?: boolean;
      timeout?: number;
    } = {}
  ): Promise<T> {
    const {
      method = 'GET',
      headers = {},
      body,
      requireAuth = true,
      timeout = this.defaultTimeout,
    } = options;

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeout);

    try {
      const requestHeaders: Record<string, string> = {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
        ...headers,
      };

      if (requireAuth) {
        const token = AuthService.getToken();
        if (!token) {
          throw new Error('Não autenticado');
        }
        requestHeaders['Authorization'] = `Bearer ${token}`;
      }

      const response = await fetch(`${this.baseUrl}${endpoint}`, {
        method,
        headers: requestHeaders,
        body: body ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (response.status === 401) {
        AuthService.logout();
        throw new Error('Sessão expirada');
      }

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || data.error || `Erro ${response.status}`);
      }

      return data;
    } catch (err: any) {
      clearTimeout(timeoutId);
      if (err.name === 'AbortError') {
        throw new Error('Tempo limite excedido');
      }
      throw err;
    }
  }

  // Métodos HTTP
  get<T = any>(endpoint: string, options?: Omit<Parameters<typeof this.request>[1], 'method' | 'body'>) {
    return this.request<T>(endpoint, { ...options, method: 'GET' });
  }

  post<T = any>(endpoint: string, body?: any, options?: Omit<Parameters<typeof this.request>[1], 'method' | 'body'>) {
    return this.request<T>(endpoint, { ...options, method: 'POST', body });
  }

  put<T = any>(endpoint: string, body?: any, options?: Omit<Parameters<typeof this.request>[1], 'method' | 'body'>) {
    return this.request<T>(endpoint, { ...options, method: 'PUT', body });
  }

  patch<T = any>(endpoint: string, body?: any, options?: Omit<Parameters<typeof this.request>[1], 'method' | 'body'>) {
    return this.request<T>(endpoint, { ...options, method: 'PATCH', body });
  }

  delete<T = any>(endpoint: string, options?: Omit<Parameters<typeof this.request>[1], 'method' | 'body'>) {
    return this.request<T>(endpoint, { ...options, method: 'DELETE' });
  }
}

export default ApiService.getInstance();