import { useState, useEffect, useCallback } from 'react';
import AuthService from '../services/auth.service';

interface User {
  id: number;
  nome: string;
  email: string;
  role: 'admin' | 'gerente' | 'atendente';
  loja_id?: number;
}

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
}

export const useAuth = () => {
  const [state, setState] = useState<AuthState>({
    user: null,
    token: null,
    isAuthenticated: false,
    isLoading: true,
  });

  useEffect(() => {
    const token = AuthService.getToken();
    const user = AuthService.getUser();
    
    setState({
      user,
      token,
      isAuthenticated: !!token && !!user,
      isLoading: false,
    });
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const response = await AuthService.login(email, password);
    setState({
      user: response.user,
      token: response.token,
      isAuthenticated: true,
      isLoading: false,
    });
    return response;
  }, []);

  const logout = useCallback(() => {
    AuthService.logout();
    setState({
      user: null,
      token: null,
      isAuthenticated: false,
      isLoading: false,
    });
  }, []);

  const hasRole = useCallback((role: User['role']) => {
    return AuthService.hasRole(role);
  }, []);

  const hasAnyRole = useCallback((roles: User['role'][]) => {
    return AuthService.hasAnyRole(roles);
  }, []);

  const refreshToken = useCallback(async () => {
    const token = await AuthService.refreshToken();
    setState(prev => ({ ...prev, token }));
    return token;
  }, []);

  return {
    ...state,
    login,
    logout,
    hasRole,
    hasAnyRole,
    refreshToken,
  };
};

export default useAuth;