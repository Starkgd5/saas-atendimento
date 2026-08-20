import React from 'react';
import { Navigate, useLocation } from 'react-router-dom';

interface PrivateRouteProps {
  children: React.ReactNode;
  requiredRole?: 'admin' | 'gerente' | 'atendente';
  allowedRoles?: ('admin' | 'gerente' | 'atendente')[];
  redirectTo?: string;
}

interface User {
  id: number;
  nome: string;
  email: string;
  role: 'admin' | 'gerente' | 'atendente';
  loja_id?: number;
}

const PrivateRoute: React.FC<PrivateRouteProps> = ({
  children,
  requiredRole,
  allowedRoles,
  redirectTo = '/login'
}) => {
  const location = useLocation();
  const token = localStorage.getItem('token');
  const userStr = localStorage.getItem('user');

  // Verificar autenticação
  if (!token) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  let user: User | null = null;
  try {
    user = userStr ? JSON.parse(userStr) : null;
  } catch (error) {
    console.error('Erro ao parsear usuário:', error);
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // Verificar role específica
  if (requiredRole && user.role !== requiredRole) {
    return <Navigate to="/dashboard/farmacia" replace />;
  }

  // Verificar roles permitidas
  if (allowedRoles && allowedRoles.length > 0) {
    if (!allowedRoles.includes(user.role)) {
      return <Navigate to="/dashboard/farmacia" replace />;
    }
  }

  // Se o usuário for atendente e tentar acessar rotas admin/gerente
  if (user.role === 'atendente') {
    const adminRoutes = ['/usuarios', '/configuracoes', '/relatorios', '/financeiro'];
    const currentPath = location.pathname;
    if (adminRoutes.some(route => currentPath.startsWith(route))) {
      return <Navigate to="/dashboard/farmacia" replace />;
    }
  }

  return <>{children}</>;
};

// Hook para verificar autenticação
export const useAuth = () => {
  const token = localStorage.getItem('token');
  const userStr = localStorage.getItem('user');

  const isAuthenticated = !!token;

  let user: User | null = null;
  try {
    user = userStr ? JSON.parse(userStr) : null;
  } catch {
    user = null;
  }

  const hasRole = (role: User['role']) => {
    return user?.role === role;
  };

  const hasAnyRole = (roles: User['role'][]) => {
    return user ? roles.includes(user.role) : false;
  };

  const logout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    window.location.href = '/login';
  };

  return {
    isAuthenticated,
    user,
    hasRole,
    hasAnyRole,
    logout,
    token,
  };
};

export default PrivateRoute;