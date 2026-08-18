import React, { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import Button from '../components/ui/Button';
import { useWebSocket } from '../hooks/useWebSocket';
import { useAuth } from '../hooks/useAuth';
import { useApi } from '../hooks/useApi';
import { useToast } from '../hooks/useToast';
import FormatterService from '../services/formatter.service';
import './Atendimento.css';

interface Message {
  id: number;
  text: string;
  sender: 'user' | 'bot' | 'attendant';
  timestamp: Date;
  status?: 'sent' | 'delivered' | 'read';
}

interface Client {
  id: number;
  name: string;
  phone: string;
  status: 'waiting' | 'attending' | 'done';
  lastMessage?: string;
  lastMessageTime?: Date;
}

interface FilaStatus {
  em_atendimento: number;
  em_espera: number;
  limite: number;
}

// Removido AtendimentoData não utilizado

const Atendimento: React.FC = () => {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [clients, setClients] = useState<Client[]>([]);
  const [filaStatus, setFilaStatus] = useState<FilaStatus>({ emAtendimento: 0, emEspera: 0, limite: 3 });
  const [selectedClient, setSelectedClient] = useState<Client | null>(null);
  const [loading, setLoading] = useState(true);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();
  const { user } = useAuth();
  const { isConnected, sendMessage, on, off } = useWebSocket();
  const { get, post } = useApi();
  const { success, error: showError } = useToast();

  // Buscar mensagens de um cliente
  const fetchMessages = useCallback(async (clienteId: number) => {
    try {
      const data = await get(`/mensagens?cliente_id=${clienteId}`);
      
      const formattedMessages: Message[] = data.map((msg: any) => ({
        id: msg.id,
        text: msg.conteudo || msg.text,
        sender: msg.remetente === 'cliente' ? 'user' :
                msg.remetente === 'atendente' ? 'attendant' : 'bot',
        timestamp: new Date(msg.enviado_em || msg.created_at),
        status: msg.lida ? 'read' : 'delivered'
      }));

      setMessages(formattedMessages);
    } catch (error: any) {
      showError(error.message || 'Erro ao buscar mensagens');
    }
  }, [get, showError]);

  // Buscar dados da fila e clientes
  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      
      // Buscar status da fila
      const fila = await get('/fila/status');
      setFilaStatus(fila);

      // Buscar clientes na fila
      const clientesFila = await get('/fila/clientes');
      
      // Transformar dados da API para o formato do componente
      const formattedClients: Client[] = clientesFila.map((item: any) => ({
        id: item.id,
        name: item.nome || item.cliente || 'Cliente',
        phone: item.telefone || item.phone || '(00) 00000-0000',
        status: item.status === 'em_atendimento' ? 'attending' : 
                item.status === 'aguardando' ? 'waiting' : 'done',
        lastMessage: item.ultima_mensagem || '',
        lastMessageTime: item.ultimo_atendimento ? new Date(item.ultimo_atendimento) : undefined
      }));

      setClients(formattedClients);

      // Se houver clientes e nenhum selecionado, selecionar o primeiro em atendimento
      if (formattedClients.length > 0 && !selectedClient) {
        const attending = formattedClients.find(c => c.status === 'attending');
        if (attending) {
          setSelectedClient(attending);
          await fetchMessages(attending.id);
        } else {
          setSelectedClient(formattedClients[0]);
        }
      }

    } catch (error: any) {
      showError(error.message || 'Erro ao carregar dados do atendimento');
    } finally {
      setLoading(false);
    }
  }, [get, selectedClient, fetchMessages, showError]);

  // Buscar histórico de atendimentos
  const fetchHistorico = useCallback(async () => {
    try {
      const data = await get('/atendimentos');
      // Atualizar lista de clientes com histórico
      const historicoClientes = data.map((item: any) => ({
        id: item.id,
        name: item.cliente || 'Cliente',
        phone: item.telefone || '(00) 00000-0000',
        status: 'done',
        lastMessage: item.ultima_mensagem || '',
        lastMessageTime: item.finalizado_em ? new Date(item.finalizado_em) : undefined
      }));
      
      // Combinar com clientes atuais
      setClients(prev => {
        const currentIds = new Set(prev.map(c => c.id));
        const novos = historicoClientes.filter((c: any) => !currentIds.has(c.id));
        return [...prev, ...novos];
      });
    } catch (error: any) {
      console.error('Erro ao buscar histórico:', error);
    }
  }, [get]);

  // Função para puxar cliente
  const handlePuxarCliente = useCallback(async () => {
    try {
      const result = await post('/fila/proximo', {});
      
      if (result.cliente_id) {
        success('Cliente puxado para atendimento!');
        await fetchData();
        
        // Selecionar o cliente puxado
        const novoCliente = clients.find(c => c.id === result.cliente_id);
        if (novoCliente) {
          setSelectedClient(novoCliente);
          await fetchMessages(novoCliente.id);
        }
      } else {
        showError('Fila vazia');
      }
    } catch (error: any) {
      showError(error.message || 'Erro ao puxar cliente');
    }
  }, [post, fetchData, clients, fetchMessages, success, showError]);

  useEffect(() => {
    // Carregar dados iniciais
    fetchData();
    fetchHistorico();

    // Configurar intervalo para atualizar a fila
    const interval = setInterval(() => {
      fetchData();
    }, 30000); // Atualizar a cada 30 segundos

    return () => clearInterval(interval);
  }, [fetchData, fetchHistorico]);

  // Escutar eventos WebSocket
  useEffect(() => {
    // Nova mensagem
    on('nova_mensagem', (data: any) => {
      const clienteId = data.cliente_id || data.clienteId;
      
      // Adicionar mensagem ao chat se for do cliente selecionado
      if (selectedClient && clienteId === selectedClient.id) {
        setMessages(prev => [...prev, {
          id: Date.now(),
          text: data.mensagem || data.text,
          sender: 'user',
          timestamp: new Date(),
          status: 'delivered'
        }]);
      }

      // Atualizar última mensagem do cliente na lista
      setClients(prev => prev.map(c => 
        c.id === clienteId ? { 
          ...c, 
          lastMessage: data.mensagem || data.text,
          lastMessageTime: new Date()
        } : c
      ));
    });

    // Fila atualizada
    on('fila_atualizada', (data: any) => {
      setFilaStatus(prev => ({
        ...prev,
        emAtendimento: data.em_atendimento || data.emAtendimento || prev.emAtendimento,
        emEspera: data.em_espera || data.emEspera || prev.emEspera
      }));
      
      // Recarregar lista de clientes
      fetchData();
    });

    // Cliente entrou na fila
    on('cliente_entrou', (data: any) => {
      success(`Cliente ${data.nome || 'Novo cliente'} entrou na fila`);
      fetchData();
    });

    // Cliente saiu da fila
    on('cliente_saiu', (data: any) => {
      setClients(prev => prev.filter(c => c.id !== data.cliente_id));
    });

    // Atendimento finalizado
    on('atendimento_finalizado', (data: any) => {
      const clienteId = data.cliente_id || data.clienteId;
      success('Atendimento finalizado com sucesso!');
      
      // Remover da lista de atendimento
      setClients(prev => prev.filter(c => c.id !== clienteId));
      
      if (selectedClient?.id === clienteId) {
        setSelectedClient(null);
        setMessages([]);
      }
      
      fetchData();
    });

    return () => {
      off('nova_mensagem');
      off('fila_atualizada');
      off('cliente_entrou');
      off('cliente_saiu');
      off('atendimento_finalizado');
    };
  }, [on, off, selectedClient, fetchData, success, showError]);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const handleSendMessage = useCallback(async () => {
    if (!input.trim() || !selectedClient) return;

    const newMessage: Message = {
      id: Date.now(),
      text: input,
      sender: 'attendant',
      timestamp: new Date(),
      status: 'sent'
    };

    setMessages(prev => [...prev, newMessage]);
    
    try {
      // Enviar via API
      await post(`/atendimentos/${selectedClient.id}/enviar`, {
        mensagem: input,
        remetente: 'atendente'
      });

      // Enviar via WebSocket
      sendMessage(selectedClient.id, input);

      // Atualizar status para entregue
      setTimeout(() => {
        setMessages(prev => prev.map(msg => 
          msg.id === newMessage.id ? { ...msg, status: 'delivered' } : msg
        ));
      }, 500);

    } catch (error: any) {
      showError(error.message || 'Erro ao enviar mensagem');
      setMessages(prev => prev.map(msg => 
        msg.id === newMessage.id ? { ...msg, status: 'sent' } : msg
      ));
    }

    setInput('');
  }, [input, selectedClient, sendMessage, post, showError]);

  const handleSelectClient = useCallback(async (client: Client) => {
    setSelectedClient(client);
    await fetchMessages(client.id);
  }, [fetchMessages]);

  const handleFinalizarAtendimento = useCallback(async () => {
    if (!selectedClient) return;

    if (!window.confirm(`Tem certeza que deseja finalizar o atendimento de ${selectedClient.name}?`)) {
      return;
    }

    try {
      await post(`/atendimentos/${selectedClient.id}/finalizar`, {});
      
      success('Atendimento finalizado com sucesso!');
      
      // Remover da lista
      setClients(prev => prev.filter(c => c.id !== selectedClient.id));
      setSelectedClient(null);
      setMessages([]);
      
      await fetchData();
    } catch (error: any) {
      showError(error.message || 'Erro ao finalizar atendimento');
    }
  }, [selectedClient, post, fetchData, success, showError]);

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSendMessage();
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-center">
          <div className="text-4xl animate-spin mb-4">⏳</div>
          <p className="text-gray-600">Carregando atendimentos...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="atendimento-container">
      {/* Sidebar */}
      <div className="atendimento-sidebar">
        <div className="atendimento-sidebar-header">
          <div className="header-top">
            <h2>💬 Atendimentos</h2>
            <span className={`status-badge ${isConnected ? 'online' : 'offline'}`}>
              {isConnected ? '🟢 Online' : '🔴 Offline'}
            </span>
          </div>
          <div className="user-info">
            👤 {user?.nome} ({user?.role})
          </div>
          <div className="fila-stats">
            <span className="atendendo">👤 {filaStatus.emAtendimento} atendendo</span>
            <span className="espera">⏳ {filaStatus.emEspera} na fila</span>
            <span className="limite">📊 {filaStatus.limite} limite</span>
          </div>
          <div className="flex gap-2">
            <Button
              onClick={handlePuxarCliente}
              disabled={filaStatus.emAtendimento >= filaStatus.limite || !isConnected}
              variant="primary"
              fullWidth
              className="btn-puxar"
              icon="🎯"
            >
              {!isConnected ? '🔌 Conectando...' : 
               filaStatus.emAtendimento >= filaStatus.limite ? '⛔ Limite atingido' : 'Puxar Próximo'}
            </Button>
            {selectedClient && (
              <Button
                onClick={handleFinalizarAtendimento}
                variant="success"
                icon="✅"
              >
                Finalizar
              </Button>
            )}
          </div>
        </div>

        <div className="client-list">
          {clients.length === 0 ? (
            <div className="text-center text-gray-400 py-8">
              <p>Nenhum cliente na fila</p>
            </div>
          ) : (
            clients.map(client => (
              <div
                key={client.id}
                onClick={() => handleSelectClient(client)}
                className={`client-item ${selectedClient?.id === client.id ? 'active' : ''}`}
              >
                <div className="client-info">
                  <p className="client-name">{client.name}</p>
                  <p className="client-phone">{client.phone}</p>
                  {client.lastMessage && (
                    <p className="client-last-message text-xs text-gray-400 truncate">
                      {client.lastMessage}
                    </p>
                  )}
                </div>
                <span className={`client-status ${client.status}`}>
                  {client.status === 'attending' ? '🟢 Atendendo' : 
                   client.status === 'waiting' ? '🟡 Aguardando' : '✅ Finalizado'}
                </span>
              </div>
            ))
          )}
        </div>

        <div className="sidebar-footer">
          <span>Total: {clients.length} clientes</span>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate('/dashboard')}
            icon="📊"
          >
            Dashboard
          </Button>
        </div>
      </div>

      {/* Chat */}
      <div className="atendimento-chat">
        {selectedClient ? (
          <>
            <div className="chat-header">
              <div className="chat-avatar">
                {selectedClient.name.charAt(0)}
              </div>
              <div className="chat-client-info">
                <p className="chat-client-name">{selectedClient.name}</p>
                <p className="chat-client-phone">{selectedClient.phone}</p>
              </div>
              <div className="chat-status online">
                {isConnected ? '🟢 Online' : '🔴 Offline'}
              </div>
            </div>

            <div className="chat-messages">
              {messages.length === 0 ? (
                <div className="chat-empty">
                  <div className="chat-empty-icon">💬</div>
                  <p className="chat-empty-title">Nenhuma mensagem ainda</p>
                  <p className="chat-empty-sub">Inicie a conversa com o cliente</p>
                </div>
              ) : (
                messages.map((msg, index) => (
                  <div
                    key={index}
                    className={`message-wrapper ${msg.sender === 'attendant' || msg.sender === 'bot' ? 'own' : 'other'}`}
                  >
                    <div className={`message-bubble ${msg.sender === 'attendant' || msg.sender === 'bot' ? 'own' : 'other'}`}>
                      <p className="text">{msg.text}</p>
                      <span className="time">
                        {FormatterService.formatTime(msg.timestamp)}
                        {msg.sender === 'attendant' && (
                          <span className="check">
                            {msg.status === 'sent' && ' ✓'}
                            {msg.status === 'delivered' && ' ✓✓'}
                            {msg.status === 'read' && ' ✓✓'}
                          </span>
                        )}
                      </span>
                    </div>
                  </div>
                ))
              )}
              <div ref={messagesEndRef} />
            </div>

            <div className="chat-input-area">
              <div className="input-wrapper">
                <input
                  type="text"
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyPress={handleKeyPress}
                  placeholder={isConnected ? "Digite sua mensagem..." : "🔌 Aguardando conexão..."}
                  disabled={!isConnected}
                  className="chat-input"
                />
                <Button
                  onClick={handleSendMessage}
                  disabled={!input.trim() || !isConnected}
                  variant="primary"
                  icon="📤"
                >
                  Enviar
                </Button>
              </div>
            </div>
          </>
        ) : (
          <div className="flex-1 flex flex-col items-center justify-center text-gray-400">
            <div className="text-6xl mb-4">💬</div>
            <p className="text-lg font-medium">Selecione um cliente</p>
            <p className="text-sm">Para começar o atendimento</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default Atendimento;