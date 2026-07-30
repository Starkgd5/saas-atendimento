import React, { useState, useEffect } from 'react';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import './Layout.css';

interface LayoutProps {
  children: React.ReactNode;
}

interface User {
  id: number;
  nome: string;
  email: string;
  role: string;
}

const Layout: React.FC<LayoutProps> = ({ children }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const [user, setUser] = useState<User | null>(null);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

  useEffect(() => {
    const userStr = localStorage.getItem('user');
    if (userStr) {
      try {
        setUser(JSON.parse(userStr));
      } catch {
        setUser(null);
      }
    }
  }, []);

  const handleLogout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    navigate('/login');
  };

  const menuItems = [
    { path: '/dashboard', icon: '📊', label: 'Dashboard' },
    { path: '/atendimento', icon: '💬', label: 'Atendimento' },
  ];

  // Itens apenas para admin/gerente
  if (user?.role === 'admin' || user?.role === 'gerente') {
    menuItems.push({ path: '/usuarios', icon: '👥', label: 'Usuários' });
    menuItems.push({ path: '/metricas', icon: '📈', label: 'Métricas' });
    menuItems.push({ path: '/reclamacoes', icon: '📋', label: 'Reclamações' });
  }

  const isActive = (path: string) => {
    return location.pathname === path || location.pathname.startsWith(path + '/');
  };

  return (
    <div className="layout-container">
      {/* Sidebar */}
      <aside className={`layout-sidebar ${isMobileMenuOpen ? 'mobile-open' : ''}`}>
        <div className="sidebar-brand">
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
            >
              <span className="link-icon">{item.icon}</span>
              <span className="link-label">{item.label}</span>
            </Link>
          ))}
        </nav>

        <div className="sidebar-footer">
          <div className="user-info">
            <div className="user-avatar">
              {user?.nome?.charAt(0) || '?'}
            </div>
            <div className="user-details">
              <p className="user-name">{user?.nome || 'Usuário'}</p>
              <p className="user-role">{user?.role || 'atendente'}</p>
            </div>
          </div>
          <button onClick={handleLogout} className="btn-logout-sidebar" title="Sair">
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
        {/* Header mobile */}
        <header className="layout-header">
          <button
            className="btn-mobile-menu"
            onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
          >
            ☰
          </button>
          <div className="header-title">
            {menuItems.find(item => isActive(item.path))?.label || 'Dashboard'}
          </div>
          <div className="header-right">
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