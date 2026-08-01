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
  redirectTo = '/dashboard'
}) => {
  const location = useLocation();
  const token = localStorage.getItem('token');
  const userStr = localStorage.getItem('user');

  // Verificar se está autenticado
  if (!token) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // Tentar fazer parse do usuário
  let user: User | null = null;
  try {
    user = userStr ? JSON.parse(userStr) : null;
  } catch (error) {
    console.error('Erro ao parsear usuário:', error);
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // Se não tiver usuário, redirecionar para login
  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // Verificar role específica (requiredRole)
  if (requiredRole && user.role !== requiredRole) {
    return <Navigate to={redirectTo} replace />;
  }

  // Verificar roles permitidas (allowedRoles)
  if (allowedRoles && allowedRoles.length > 0) {
    if (!allowedRoles.includes(user.role)) {
      return <Navigate to={redirectTo} replace />;
    }
  }

  // Verificar se o usuário está ativo (role = admin não precisa de loja)
  if (user.role !== 'admin' && !user.loja_id) {
    console.warn('Usuário sem loja associada');
    // Pode redirecionar para uma página de erro ou continuar
  }

  return <>{children}</>;
};

// HOC para proteger rotas com roles
export const withAuth = (
  Component: React.ComponentType,
  options?: Omit<PrivateRouteProps, 'children'>
) => {
  return function AuthenticatedComponent(props: any) {
    return (
      <PrivateRoute {...options}>
        <Component {...props} />
      </PrivateRoute>
    );
  };
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