import React, { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import Button from '../components/ui/Button';
import { useWebSocket } from '../hooks/useWebSocket';
import { useAuth } from '../hooks/useAuth';
import { useApi } from '../hooks/useApi';
import { useToast } from '../hooks/useToast';
import FormatterService from '../services/formatter.service';
require('./Atendimento.css');

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

// Interface FilaStatus com os nomes corretos (underline)
interface FilaStatus {
  em_atendimento: number;  // ← underline
  em_espera: number;       // ← underline
  limite: number;
}

// Interface para o estado interno do componente (camelCase)
interface FilaStatusDisplay {
  emAtendimento: number;
  emEspera: number;
  limite: number;
}

const Atendimento: React.FC = () => {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [clients, setClients] = useState<Client[]>([]);
  const [filaStatus, setFilaStatus] = useState<FilaStatusDisplay>({ 
    emAtendimento: 0, 
    emEspera: 0, 
    limite: 3 
  });
  const [selectedClient, setSelectedClient] = useState<Client | null>(null);
  const [loading, setLoading] = useState(true);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();
  const { user } = useAuth();
  const { isConnected, sendMessage, puxarCliente, on, off } = useWebSocket();
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
      
      // Buscar status da fila (resposta com underline)
      const fila: FilaStatus = await get('/fila/status');
      
      // Converter para o formato do componente (camelCase)
      setFilaStatus({
        emAtendimento: fila.em_atendimento || 0,
        emEspera: fila.em_espera || 0,
        limite: fila.limite || 3
      });

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
      const historicoClientes = data.map((item: any) => ({
        id: item.id,
        name: item.cliente || 'Cliente',
        phone: item.telefone || '(00) 00000-0000',
        status: 'done' as const,
        lastMessage: item.ultima_mensagem || '',
        lastMessageTime: item.finalizado_em ? new Date(item.finalizado_em) : undefined
      }));
      
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
    fetchData();
    fetchHistorico();

    const interval = setInterval(() => {
      fetchData();
    }, 30000);

    return () => clearInterval(interval);
  }, [fetchData, fetchHistorico]);

  // Escutar eventos WebSocket
  useEffect(() => {
    on('nova_mensagem', (data: any) => {
      const clienteId = data.cliente_id || data.clienteId;
      
      if (selectedClient && clienteId === selectedClient.id) {
        setMessages(prev => [...prev, {
          id: Date.now(),
          text: data.mensagem || data.text,
          sender: 'user',
          timestamp: new Date(),
          status: 'delivered'
        }]);
      }

      setClients(prev => prev.map(c => 
        c.id === clienteId ? { 
          ...c, 
          lastMessage: data.mensagem || data.text,
          lastMessageTime: new Date()
        } : c
      ));
    });

    on('fila_atualizada', (data: any) => {
      // Converter dados da fila (underline) para o formato do componente (camelCase)
      setFilaStatus(prev => ({
        ...prev,
        emAtendimento: data.em_atendimento || data.emAtendimento || prev.emAtendimento,
        emEspera: data.em_espera || data.emEspera || prev.emEspera
      }));
      fetchData();
    });

    on('cliente_entrou', (data: any) => {
      success(`Cliente ${data.nome || 'Novo cliente'} entrou na fila`);
      fetchData();
    });

    on('cliente_saiu', (data: any) => {
      setClients(prev => prev.filter(c => c.id !== data.cliente_id));
    });

    on('atendimento_finalizado', (data: any) => {
      const clienteId = data.cliente_id || data.clienteId;
      success('Atendimento finalizado com sucesso!');
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
      await post(`/atendimentos/${selectedClient.id}/enviar`, {
        mensagem: input,
        remetente: 'atendente'
      });

      sendMessage(selectedClient.id, input);

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

  // Usar emAtendimento (camelCase) no JSX
  const { emAtendimento, emEspera, limite } = filaStatus;

  return (
    <div className="flex h-screen bg-gray-100 font-sans">
      {/* Sidebar */}
      <div className="w-80 bg-white border-r border-gray-200 flex flex-col shadow-lg">
        <div className="p-4 border-b border-gray-200 bg-gradient-to-r from-blue-50 to-white">
          <div className="header-top flex justify-between items-center">
            <h2 className="text-lg font-bold text-gray-800">💬 Atendimentos</h2>
            <span className={`text-xs px-2 py-1 rounded-full ${isConnected ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>
              {isConnected ? '🟢 Online' : '🔴 Offline'}
            </span>
          </div>
          <div className="user-info text-xs text-gray-500 mt-1">
            👤 {user?.nome} ({user?.role})
          </div>
          <div className="mt-3 flex justify-between text-sm">
            <span className="bg-green-100 text-green-700 px-3 py-1 rounded-full">
              👤 {emAtendimento} atendendo
            </span>
            <span className="bg-yellow-100 text-yellow-700 px-3 py-1 rounded-full">
              ⏳ {emEspera} na fila
            </span>
            <span className="bg-gray-100 text-gray-600 px-3 py-1 rounded-full">
              📊 {limite} limite
            </span>
          </div>
          <div className="flex gap-2 mt-3">
            <Button
              onClick={handlePuxarCliente}
              disabled={emAtendimento >= limite || !isConnected}
              variant="primary"
              fullWidth
              icon="🎯"
            >
              {!isConnected ? '🔌 Conectando...' : 
               emAtendimento >= limite ? '⛔ Limite atingido' : 'Puxar Próximo'}
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

        <div className="flex-1 overflow-y-auto">
          {clients.length === 0 ? (
            <div className="text-center text-gray-400 py-8">
              <p>Nenhum cliente na fila</p>
            </div>
          ) : (
            clients.map(client => (
              <div
                key={client.id}
                onClick={() => handleSelectClient(client)}
                className={`p-4 border-b border-gray-100 cursor-pointer transition ${
                  selectedClient?.id === client.id 
                    ? 'bg-blue-50 border-r-4 border-blue-500' 
                    : 'hover:bg-gray-50'
                }`}
              >
                <div className="flex justify-between items-start">
                  <div className="flex-1 min-w-0">
                    <p className="font-medium text-gray-800 truncate">{client.name}</p>
                    <p className="text-sm text-gray-500 truncate">{client.phone}</p>
                    {client.lastMessage && (
                      <p className="text-xs text-gray-400 truncate mt-1">{client.lastMessage}</p>
                    )}
                  </div>
                  <span className={`text-xs px-2.5 py-1 rounded-full whitespace-nowrap ml-2 ${
                    client.status === 'attending' 
                      ? 'bg-green-100 text-green-700' 
                      : client.status === 'waiting'
                      ? 'bg-yellow-100 text-yellow-700'
                      : 'bg-gray-100 text-gray-600'
                  }`}>
                    {client.status === 'attending' ? '🟢 Atendendo' : 
                     client.status === 'waiting' ? '🟡 Aguardando' : '✅ Finalizado'}
                  </span>
                </div>
              </div>
            ))
          )}
        </div>

        <div className="p-3 border-t border-gray-200 bg-gray-50 flex justify-between items-center">
          <span className="text-xs text-gray-500">Total: {clients.length} clientes</span>
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
      <div className="flex-1 flex flex-col bg-gray-50">
        {selectedClient ? (
          <>
            <div className="bg-white border-b border-gray-200 p-4 flex items-center shadow-sm">
              <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-blue-600 rounded-full flex items-center justify-center text-white font-bold text-lg shadow">
                {selectedClient.name.charAt(0)}
              </div>
              <div className="ml-3">
                <p className="font-medium text-gray-800">{selectedClient.name}</p>
                <p className="text-sm text-gray-500">{selectedClient.phone}</p>
              </div>
              <div className="ml-auto flex items-center gap-3">
                <span className={`text-sm ${isConnected ? 'text-green-500' : 'text-red-500'}`}>
                  {isConnected ? '🟢 Online' : '🔴 Offline'}
                </span>
              </div>
            </div>

            <div className="flex-1 overflow-y-auto p-4 space-y-2.5 bg-gray-50">
              {messages.length === 0 ? (
                <div className="flex items-center justify-center h-full text-gray-400">
                  <p>Nenhuma mensagem ainda</p>
                </div>
              ) : (
                messages.map((msg, index) => (
                  <div
                    key={index}
                    className={`flex ${msg.sender === 'attendant' || msg.sender === 'bot' ? 'justify-end' : 'justify-start'}`}
                  >
                    <div
                      className={`max-w-[75%] rounded-2xl px-4 py-2.5 shadow-sm ${
                        msg.sender === 'attendant' || msg.sender === 'bot'
                          ? 'bg-blue-500 text-white rounded-br-sm'
                          : 'bg-white text-gray-800 rounded-bl-sm'
                      }`}
                    >
                      <p className="text-sm leading-relaxed">{msg.text}</p>
                      <p className={`text-xs mt-1 ${
                        msg.sender === 'attendant' || msg.sender === 'bot'
                          ? 'text-blue-100'
                          : 'text-gray-400'
                      }`}>
                        {FormatterService.formatTime(msg.timestamp)}
                        {msg.sender === 'attendant' && (
                          <span className="ml-1">
                            {msg.status === 'sent' && ' ✓'}
                            {msg.status === 'delivered' && ' ✓✓'}
                            {msg.status === 'read' && ' ✓✓'}
                          </span>
                        )}
                      </p>
                    </div>
                  </div>
                ))
              )}
              <div ref={messagesEndRef} />
            </div>

            <div className="bg-white border-t border-gray-200 p-4 shadow-lg">
              <div className="flex gap-2.5">
                <input
                  type="text"
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyPress={handleKeyPress}
                  placeholder={isConnected ? "Digite sua mensagem..." : "🔌 Aguardando conexão..."}
                  disabled={!isConnected}
                  className="flex-1 px-4 py-2.5 border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition disabled:bg-gray-100"
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