import React, { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import './Atendimento.css';

interface Message {
  id: number;
  text: string;
  sender: 'user' | 'bot' | 'attendant';
  timestamp: Date;
}

interface Client {
  id: number;
  name: string;
  phone: string;
  status: 'waiting' | 'attending' | 'done';
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
  const [ws, setWs] = useState<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  // Pegar usuário
  const user = JSON.parse(localStorage.getItem('user') || '{"nome":"Atendente","role":"atendente"}');

  useEffect(() => {
    // Conectar WebSocket
    const token = localStorage.getItem('token') || '';
    const wsUrl = `ws://localhost:8080/ws?token=${token}`;
    const websocket = new WebSocket(wsUrl);

    websocket.onopen = () => {
      console.log('✅ WebSocket conectado');
      setIsConnected(true);
    };

    websocket.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        console.log('📩 Mensagem recebida:', data);

        if (data.type === 'nova_mensagem') {
          setMessages(prev => [...prev, {
            id: Date.now(),
            text: data.payload.mensagem || data.payload.text,
            sender: 'user',
            timestamp: new Date()
          }]);
        } else if (data.type === 'fila_atualizada') {
          setFilaStatus(prev => ({
            ...prev,
            emAtendimento: data.payload.em_atendimento || prev.emAtendimento,
            emEspera: data.payload.em_espera || prev.emEspera
          }));
        }
      } catch (error) {
        console.error('Erro ao processar mensagem WebSocket:', error);
      }
    };

    websocket.onerror = (error) => {
      console.error('❌ Erro no WebSocket:', error);
    };

    websocket.onclose = () => {
      console.log('🔌 WebSocket desconectado');
      setIsConnected(false);
    };

    setWs(websocket);

    // Mensagens iniciais
    setMessages([
      { id: 1, text: 'Olá! Gostaria de um orçamento para Dipirona.', sender: 'user', timestamp: new Date() },
      { id: 2, text: 'Olá! Vou verificar o estoque para você.', sender: 'attendant', timestamp: new Date() },
    ]);

    return () => {
      if (websocket.readyState === WebSocket.OPEN) {
        websocket.close();
      }
    };
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const sendMessage = () => {
    if (!input.trim() || !selectedClient || !ws || ws.readyState !== WebSocket.OPEN) return;

    ws.send(JSON.stringify({
      type: 'nova_mensagem',
      payload: {
        cliente_id: selectedClient.id,
        mensagem: input,
        remetente: 'atendente'
      }
    }));

    setMessages(prev => [...prev, {
      id: Date.now(),
      text: input,
      sender: 'attendant',
      timestamp: new Date()
    }]);
    setInput('');

    setTimeout(() => {
      setMessages(prev => [...prev, {
        id: Date.now() + 1,
        text: '✅ Mensagem enviada com sucesso!',
        sender: 'bot',
        timestamp: new Date()
      }]);
    }, 1000);
  };

  const handlePuxarCliente = () => {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;

    ws.send(JSON.stringify({
      type: 'puxar_cliente',
      payload: {}
    }));

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
    }
  };

  const handleLogout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    navigate('/login');
  };

  return (
    <div className="flex h-screen bg-gray-100 font-sans">
      {/* Sidebar */}
      <div className="w-80 bg-white border-r border-gray-200 flex flex-col shadow-lg">
        {/* Header da Sidebar */}
        <div className="p-4 border-b border-gray-200 bg-gradient-to-r from-blue-50 to-white">
          <div className="flex justify-between items-center">
            <h2 className="text-lg font-bold text-gray-800">💬 Atendimentos</h2>
            <span className={`text-xs px-2 py-1 rounded-full ${isConnected ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>
              {isConnected ? '🟢 Online' : '🔴 Offline'}
            </span>
          </div>

          {/* Info do usuário */}
          <div className="mt-2 text-xs text-gray-500">
            👤 {user.nome} ({user.role})
          </div>

          <div className="mt-3 flex justify-between text-sm">
            <span className="bg-green-100 text-green-700 px-3 py-1 rounded-full">
              👤 {filaStatus.emAtendimento} atendendo
            </span>
            <span className="bg-yellow-100 text-yellow-700 px-3 py-1 rounded-full">
              ⏳ {filaStatus.emEspera} na fila
            </span>
            <span className="bg-gray-100 text-gray-600 px-3 py-1 rounded-full">
              📊 {filaStatus.limite} limite
            </span>
          </div>

          <button
            onClick={handlePuxarCliente}
            disabled={filaStatus.emAtendimento >= filaStatus.limite || !isConnected}
            className={`mt-3 w-full px-4 py-2.5 rounded-lg font-medium transition ${
              filaStatus.emAtendimento >= filaStatus.limite || !isConnected
                ? 'bg-gray-300 text-gray-500 cursor-not-allowed'
                : 'bg-blue-500 text-white hover:bg-blue-600 active:scale-95'
            }`}
          >
            {!isConnected ? '🔌 Conectando...' : 
             filaStatus.emAtendimento >= filaStatus.limite ? '⛔ Limite atingido' : '🎯 Puxar Próximo'}
          </button>
        </div>

        {/* Lista de clientes */}
        <div className="flex-1 overflow-y-auto">
          {clients.map(client => (
            <div
              key={client.id}
              onClick={() => setSelectedClient(client)}
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
                </div>
                <span className={`text-xs px-2.5 py-1 rounded-full whitespace-nowrap ml-2 ${
                  client.status === 'attending' 
                    ? 'bg-green-100 text-green-700' 
                    : 'bg-yellow-100 text-yellow-700'
                }`}>
                  {client.status === 'attending' ? '🟢 Atendendo' : '🟡 Aguardando'}
                </span>
              </div>
            </div>
          ))}
        </div>

        {/* Footer da Sidebar */}
        <div className="p-3 border-t border-gray-200 bg-gray-50 flex justify-between items-center">
          <span className="text-xs text-gray-500">
            Total: {clients.length} clientes
          </span>
          <button
            onClick={handleLogout}
            className="text-xs text-red-500 hover:text-red-700"
          >
            🚪 Sair
          </button>
        </div>
      </div>

      {/* Chat */}
      <div className="flex-1 flex flex-col bg-gray-50">
        {selectedClient ? (
          <>
            {/* Header do Chat */}
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
                <button className="text-gray-400 hover:text-gray-600">
                  ⋮
                </button>
              </div>
            </div>

            {/* Mensagens */}
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
                        {msg.timestamp.toLocaleTimeString('pt-BR', { 
                          hour: '2-digit', 
                          minute: '2-digit' 
                        })}
                        {msg.sender === 'attendant' && ' ✅'}
                      </p>
                    </div>
                  </div>
                ))
              )}
              <div ref={messagesEndRef} />
            </div>

            {/* Input */}
            <div className="bg-white border-t border-gray-200 p-4 shadow-lg">
              <div className="flex gap-2.5">
                <input
                  type="text"
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyPress={(e) => e.key === 'Enter' && sendMessage()}
                  placeholder={isConnected ? "Digite sua mensagem..." : "🔌 Aguardando conexão..."}
                  disabled={!isConnected}
                  className="flex-1 px-4 py-2.5 border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition disabled:bg-gray-100"
                />
                <button
                  onClick={sendMessage}
                  disabled={!input.trim() || !isConnected}
                  className={`px-6 py-2.5 rounded-xl font-medium transition ${
                    input.trim() && isConnected
                      ? 'bg-blue-500 text-white hover:bg-blue-600 active:scale-95'
                      : 'bg-gray-200 text-gray-400 cursor-not-allowed'
                  }`}
                >
                  📤 Enviar
                </button>
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