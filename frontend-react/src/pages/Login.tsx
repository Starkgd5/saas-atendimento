import React, { useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import Button from '../components/ui/Button';
import Input from '../components/ui/Input';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';

const Login: React.FC = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const { success, error: showError } = useToast();
  const navigate = useNavigate();
  const location = useLocation();

  // Removido 'from' não utilizado
  // const from = (location.state as any)?.from?.pathname || '/dashboard';

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    try {
      const response = await login(email, password);
      success('Login realizado com sucesso!');
      
      if (response.user.role === 'admin' || response.user.role === 'gerente') {
        navigate('/dashboard');
      } else {
        navigate('/atendimento');
      }
    } catch (err: any) {
      showError(err.message || 'Erro ao fazer login');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 via-indigo-50 to-purple-50 p-4">
      <div className="bg-white p-8 rounded-2xl shadow-2xl w-full max-w-md border border-gray-100">
        {/* Logo */}
        <div className="text-center mb-8">
          <div className="text-5xl mb-3">💊</div>
          <h1 className="text-2xl font-bold text-gray-800">
            {process.env.REACT_APP_NAME || 'SaaS Atendimento'}
          </h1>
          <p className="text-gray-500 mt-1 text-sm">
            Sistema de atendimento com IA
          </p>
          <p className="text-xs text-gray-400 mt-2">
            v{process.env.REACT_APP_VERSION || '2.0.0'}
          </p>
        </div>

        {/* Formulário */}
        <form onSubmit={handleLogin}>
          <div className="mb-4">
            <Input
              type="email"
              label="E-mail"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="admin@saas.com"
              required
              autoFocus
              icon="📧"
            />
          </div>

          <div className="mb-6">
            <Input
              type="password"
              label="Senha"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              required
              icon="🔒"
            />
          </div>

          <Button
            type="submit"
            loading={loading}
            fullWidth
            size="lg"
            icon="🔑"
          >
            Entrar
          </Button>
        </form>

        {/* Credenciais de demonstração */}
        <div className="mt-6 pt-6 border-t border-gray-100">
          <p className="text-center text-xs text-gray-400 mb-3">Credenciais de demonstração</p>
          <div className="grid grid-cols-2 gap-2 text-xs">
            <div className="p-2 bg-gray-50 rounded-lg">
              <span className="font-medium text-gray-700">Admin</span>
              <p className="text-gray-400">admin@saas.com</p>
              <p className="text-gray-400">admin123</p>
            </div>
            <div className="p-2 bg-gray-50 rounded-lg">
              <span className="font-medium text-gray-700">Gerente</span>
              <p className="text-gray-400">gerente@saas.com</p>
              <p className="text-gray-400">gerente123</p>
            </div>
            <div className="p-2 bg-gray-50 rounded-lg col-span-2">
              <span className="font-medium text-gray-700">Atendente</span>
              <p className="text-gray-400">atendente@saas.com</p>
              <p className="text-gray-400">atendente123</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Login;