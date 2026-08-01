import React, { useState, useEffect, useMemo } from 'react';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import './Layout.css';

interface LayoutProps {
  children: React.ReactNode;
}

interface User {
  id: number;
  nome: string;
  email: string;
  role: 'admin' | 'gerente' | 'atendente';
  loja_id?: number;
}

interface MenuItem {
  path: string;
  icon: string;
  label: string;
  roles?: ('admin' | 'gerente' | 'atendente')[];
}

const Layout: React.FC<LayoutProps> = ({ children }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const [user, setUser] = useState<User | null>(null);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const userStr = localStorage.getItem('user');
    if (userStr) {
      try {
        setUser(JSON.parse(userStr));
      } catch {
        setUser(null);
      }
    }
    setIsLoading(false);
  }, []);

  const handleLogout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    navigate('/login');
  };

  // Menu items com roles
  const allMenuItems: MenuItem[] = [
    { path: '/dashboard', icon: '📊', label: 'Dashboard' },
    { path: '/atendimento', icon: '💬', label: 'Atendimento' },
    { 
      path: '/produtos', 
      icon: '📦', 
      label: 'Produtos',
      roles: ['admin', 'gerente'] 
    },
    { 
      path: '/orcamentos', 
      icon: '💰', 
      label: 'Orçamentos',
      roles: ['admin', 'gerente'] 
    },
    { 
      path: '/reclamacoes', 
      icon: '📋', 
      label: 'Reclamações',
      roles: ['admin', 'gerente'] 
    },
    { 
      path: '/usuarios', 
      icon: '👥', 
      label: 'Usuários',
      roles: ['admin'] 
    },
    { 
      path: '/metricas', 
      icon: '📈', 
      label: 'Métricas',
      roles: ['admin', 'gerente'] 
    },
    { 
      path: '/configuracoes', 
      icon: '⚙️', 
      label: 'Configurações',
      roles: ['admin'] 
    },
  ];

  // Filtrar itens baseado no role do usuário
  const menuItems = useMemo(() => {
    if (!user) return allMenuItems.filter(item => !item.roles);
    
    return allMenuItems.filter(item => {
      if (!item.roles) return true;
      return item.roles.includes(user.role);
    });
  }, [user]);

  const getPageTitle = () => {
    const currentItem = menuItems.find(item => isActive(item.path));
    return currentItem?.label || 'Dashboard';
  };

  const isActive = (path: string) => {
    if (path === '/dashboard' && location.pathname === '/') return true;
    return location.pathname === path || location.pathname.startsWith(path + '/');
  };

  // Verificar se o usuário tem permissão para acessar a rota atual
  const hasPermission = useMemo(() => {
    const currentPath = location.pathname;
    const menuItem = allMenuItems.find(item => 
      currentPath === item.path || currentPath.startsWith(item.path + '/')
    );
    
    if (!menuItem || !menuItem.roles) return true;
    if (!user) return false;
    
    return menuItem.roles.includes(user.role);
  }, [location.pathname, user]);

  // Redirecionar se não tiver permissão
  useEffect(() => {
    if (!isLoading && !hasPermission && user) {
      navigate('/dashboard');
    }
  }, [isLoading, hasPermission, user, navigate]);

  if (isLoading) {
    return (
      <div className="layout-loading">
        <div className="loading-spinner">⏳</div>
        <p>Carregando...</p>
      </div>
    );
  }

  return (
    <div className="layout-container">
      {/* Sidebar */}
      <aside className={`layout-sidebar ${isMobileMenuOpen ? 'mobile-open' : ''}`}>
        <div className="sidebar-brand" onClick={() => navigate('/dashboard')}>
          <span className="brand-icon">💊</span>
          <span className="brand-text">SaaS Atendimento</span>
        </div>

        <nav className="sidebar-nav">
          {menuItems.map((item) => (
            <Link
              key={item.path}
              to={item.path}
              className={`sidebar-link ${isActive(item.path) ? 'active' : ''}`}
              onClick={() => setIsMobileMenuOpen(false)}
              title={item.label}
            >
              <span className="link-icon">{item.icon}</span>
              <span className="link-label">{item.label}</span>
            </Link>
          ))}
        </nav>

        <div className="sidebar-footer">
          <div className="user-info">
            <div className="user-avatar" style={{ 
              background: user?.role === 'admin' 
                ? 'linear-gradient(135deg, #8b5cf6, #6d28d9)'
                : user?.role === 'gerente'
                ? 'linear-gradient(135deg, #3b82f6, #1d4ed8)'
                : 'linear-gradient(135deg, #22c55e, #16a34a)'
            }}>
              {user?.nome?.charAt(0)?.toUpperCase() || '?'}
            </div>
            <div className="user-details">
              <p className="user-name">{user?.nome || 'Usuário'}</p>
              <p className="user-role">
                {user?.role === 'admin' ? '👑 Administrador' :
                 user?.role === 'gerente' ? '📋 Gerente' :
                 '🎯 Atendente'}
              </p>
            </div>
          </div>
          <button 
            onClick={handleLogout} 
            className="btn-logout-sidebar" 
            title="Sair"
            aria-label="Sair"
          >
            🚪
          </button>
        </div>
      </aside>

      {/* Overlay para mobile */}
      {isMobileMenuOpen && (
        <div 
          className="sidebar-overlay" 
          onClick={() => setIsMobileMenuOpen(false)}
        />
      )}

      {/* Conteúdo principal */}
      <main className="layout-main">
        {/* Header */}
        <header className="layout-header">
          <div className="header-left">
            <button
              className="btn-mobile-menu"
              onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
              aria-label="Menu"
            >
              ☰
            </button>
            <div className="header-title">
              {getPageTitle()}
            </div>
          </div>
          <div className="header-right">
            <div className="header-status">
              <span className="status-dot online"></span>
              <span className="status-text">Online</span>
            </div>
            <span className="header-user">
              {user?.nome || 'Usuário'}
            </span>
            <button onClick={handleLogout} className="btn-logout-header">
              Sair
            </button>
          </div>
        </header>

        {/* Conteúdo */}
        <div className="layout-content">
          {children}
        </div>
      </main>
    </div>
  );
};

export default Layout;