import { useState, useEffect, useCallback, useRef } from 'react';
import SocketService from '../services/socket';

export const useWebSocket = () => {
  const [isConnected, setIsConnected] = useState(false);
  const [reconnectAttempts, setReconnectAttempts] = useState(0);
  const isMounted = useRef(true);

  useEffect(() => {
    isMounted.current = true;

    const connect = async () => {
      try {
        const token = localStorage.getItem('token') || '';
        await SocketService.connect(token);
        if (isMounted.current) {
          setIsConnected(true);
        }
      } catch (error) {
        console.error('Erro ao conectar WebSocket:', error);
        if (isMounted.current) {
          setIsConnected(false);
        }
      }
    };

    connect();

    const handleConnect = () => {
      if (isMounted.current) {
        setIsConnected(true);
        setReconnectAttempts(0);
      }
    };

    const handleDisconnect = () => {
      if (isMounted.current) {
        setIsConnected(false);
      }
    };

    const handleReconnectAttempt = (data: { attemptNumber: number }) => {
      if (isMounted.current) {
        setReconnectAttempts(data.attemptNumber);
      }
    };

    SocketService.on('connect', handleConnect);
    SocketService.on('disconnect', handleDisconnect);
    SocketService.on('reconnect_attempt', handleReconnectAttempt);

    return () => {
      isMounted.current = false;
      SocketService.off('connect', handleConnect);
      SocketService.off('disconnect', handleDisconnect);
      SocketService.off('reconnect_attempt', handleReconnectAttempt);
      SocketService.disconnect();
      // Remover chamada para clearAllListeners
    };
  }, []);

  const emit = useCallback((event: string, data: any) => {
    return SocketService.emit(event, data);
  }, []);

  const on = useCallback((event: string, callback: Function) => {
    SocketService.on(event, callback);
  }, []);

  const off = useCallback((event: string, callback?: Function) => {
    SocketService.off(event, callback);
  }, []);

  const once = useCallback((event: string, callback: Function) => {
    SocketService.once(event, callback);
  }, []);

  const sendMessage = useCallback((clienteId: number, mensagem: string, remetente: string = 'atendente') => {
    return SocketService.sendMessage(clienteId, mensagem, remetente);
  }, []);

  const puxarCliente = useCallback(() => {
    return SocketService.puxarCliente();
  }, []);

  const finalizarAtendimento = useCallback((clienteId: number) => {
    return SocketService.finalizarAtendimento(clienteId);
  }, []);

  const sendTyping = useCallback((clienteId: number, isTyping: boolean) => {
    return SocketService.emit('typing', {
      cliente_id: clienteId,
      is_typing: isTyping,
      time: new Date().toISOString()
    });
  }, []);

  return {
    isConnected,
    reconnectAttempts,
    emit,
    on,
    off,
    once,
    sendMessage,
    puxarCliente,
    finalizarAtendimento,
    sendTyping,
  };
};

export default useWebSocket;