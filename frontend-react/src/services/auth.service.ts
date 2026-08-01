import ApiService from './api.service';

interface User {
  id: number;
  nome: string;
  email: string;
  role: 'admin' | 'gerente' | 'atendente';
  loja_id?: number;
}

interface LoginResponse {
  token: string;
  user: User;
}

class AuthService {
  private static instance: AuthService;
  private _user: User | null = null;
  private _token: string | null = null;

  private constructor() {
    this._token = localStorage.getItem('token');
    const userStr = localStorage.getItem('user');
    if (userStr) {
      try {
        this._user = JSON.parse(userStr);
      } catch {
        this._user = null;
      }
    }
  }

  static getInstance(): AuthService {
    if (!AuthService.instance) {
      AuthService.instance = new AuthService();
    }
    return AuthService.instance;
  }

  async login(email: string, password: string): Promise<LoginResponse> {
    const response = await ApiService.post<LoginResponse>('/auth/login', { email, password }, { requireAuth: false });
    
    this._token = response.token;
    this._user = response.user;
    
    localStorage.setItem('token', response.token);
    localStorage.setItem('user', JSON.stringify(response.user));
    
    return response;
  }

  logout(): void {
    this._token = null;
    this._user = null;
    localStorage.removeItem('token');
    localStorage.removeItem('user');
  }

  getToken(): string | null {
    return this._token || localStorage.getItem('token');
  }

  getUser(): User | null {
    return this._user;
  }

  isAuthenticated(): boolean {
    return !!this.getToken();
  }

  hasRole(role: User['role']): boolean {
    return this._user?.role === role;
  }

  hasAnyRole(roles: User['role'][]): boolean {
    return this._user ? roles.includes(this._user.role) : false;
  }

  async refreshToken(): Promise<string> {
    const response = await ApiService.post<{ token: string }>('/auth/refresh', {}, { requireAuth: true });
    this._token = response.token;
    localStorage.setItem('token', response.token);
    return response.token;
  }
}

export default AuthService.getInstance();