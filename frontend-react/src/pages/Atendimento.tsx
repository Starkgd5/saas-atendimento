import React, { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import Button from '../components/ui/Button';
import { useWebSocket } from '../hooks/useWebSocket';
import { useAuth } from '../hooks/useAuth';
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

const Atendimento: React.FC = () => {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [clients, setClients] = useState<Client[]>([
    { id: 1, name: 'João Silva', phone: '(11) 99999-9999', status: 'attending' },
    { id: 2, name: 'Maria Santos', phone: '(11) 88888-8888', status: 'waiting' },
    { id: 3, name: 'Pedro Oliveira', phone: '(11) 77777-7777', status: 'waiting' },
  ]);
  const [filaStatus, setFilaStatus] = useState({ emAtendimento: 1, emEspera: 2, limite: 3 });
  const [selectedClient, setSelectedClient] = useState<Client | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();
  const { user } = useAuth();
  const { isConnected, sendMessage, puxarCliente, on, off } = useWebSocket();
  const { success, error: showError } = useToast();

  useEffect(() => {
    // Escutar eventos
    on('nova_mensagem', (data: any) => {
      setMessages(prev => [...prev, {
        id: Date.now(),
        text: data.mensagem || data.payload?.mensagem,
        sender: 'user',
        timestamp: new Date(),
        status: 'delivered'
      }]);
    });

    on('fila_atualizada', (data: any) => {
      setFilaStatus(prev => ({
        ...prev,
        emAtendimento: data.em_atendimento || prev.emAtendimento,
        emEspera: data.em_espera || prev.emEspera
      }));
    });

    return () => {
      off('nova_mensagem');
      off('fila_atualizada');
    };
  }, [on, off]);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const handleSendMessage = useCallback(() => {
    if (!input.trim() || !selectedClient) return;

    const newMessage: Message = {
      id: Date.now(),
      text: input,
      sender: 'attendant',
      timestamp: new Date(),
      status: 'sent'
    };

    setMessages(prev => [...prev, newMessage]);
    
    // Enviar via WebSocket
    sendMessage(selectedClient.id, input);

    setInput('');

    // Simular entrega
    setTimeout(() => {
      setMessages(prev => prev.map(msg => 
        msg.id === newMessage.id ? { ...msg, status: 'delivered' } : msg
      ));
    }, 500);
  }, [input, selectedClient, sendMessage]);

  const handlePuxarCliente = useCallback(() => {
    puxarCliente();
    
    const waitingClients = clients.filter(c => c.status === 'waiting');
    if (waitingClients.length > 0 && filaStatus.emAtendimento < filaStatus.limite) {
      const client = waitingClients[0];
      setClients(prev => prev.map(c => 
        c.id === client.id ? { ...c, status: 'attending' } : c
      ));
      setFilaStatus(prev => ({
        ...prev,
        emAtendimento: prev.emAtendimento + 1,
        emEspera: prev.emEspera - 1
      }));
      setSelectedClient(client);
      setMessages([]);
      success(`Cliente ${client.name} puxado para atendimento`);
    }
  }, [clients, filaStatus, puxarCliente, success]);

  const handleSelectClient = (client: Client) => {
    setSelectedClient(client);
    setMessages([]);
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSendMessage();
    }
  };

  const timeAgo = FormatterService.timeAgo;

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
        </div>

        <div className="client-list">
          {clients.map(client => (
            <div
              key={client.id}
              onClick={() => handleSelectClient(client)}
              className={`client-item ${selectedClient?.id === client.id ? 'active' : ''}`}
            >
              <div className="client-info">
                <p className="client-name">{client.name}</p>
                <p className="client-phone">{client.phone}</p>
              </div>
              <span className={`client-status ${client.status}`}>
                {client.status === 'attending' ? '🟢 Atendendo' : '🟡 Aguardando'}
              </span>
            </div>
          ))}
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