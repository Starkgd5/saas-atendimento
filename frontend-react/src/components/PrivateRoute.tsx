import React from 'react';
import { Navigate } from 'react-router-dom';

interface PrivateRouteProps {
  children: React.ReactNode;
  requiredRole?: 'admin' | 'gerente' | 'atendente';
}

const PrivateRoute: React.FC<PrivateRouteProps> = ({ children, requiredRole }) => {
  const token = localStorage.getItem('token');
  const userStr = localStorage.getItem('user');
  
  // Verificar se está autenticado
  if (!token) {
    return <Navigate to="/login" replace />;
  }

  // Verificar role se necessário
  if (requiredRole) {
    try {
      const user = userStr ? JSON.parse(userStr) : null;
      if (!user || user.role !== requiredRole) {
        // Se não tem permissão, redirecionar para dashboard
        return <Navigate to="/dashboard" replace />;
      }
    } catch (error) {
      return <Navigate to="/login" replace />;
    }
  }

  return <>{children}</>;
};

export default PrivateRoute;