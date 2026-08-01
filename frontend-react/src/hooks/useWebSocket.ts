import { useState, useEffect, useCallback, useRef } from 'react';
import socketService from '../services/socket';

export const useWebSocket = () => {
  const [isConnected, setIsConnected] = useState(false);
  const [socketId, setSocketId] = useState<string | null>(null);
  const [reconnectAttempts, setReconnectAttempts] = useState(0);
  const eventHandlersRef = useRef<Map<string, Set<Function>>>(new Map());

  useEffect(() => {
    const connect = async () => {
      try {
        const socket = await socketService.connect({
          token: localStorage.getItem('token') || '',
        });
        setIsConnected(true);
        setSocketId(socket.id || null);
      } catch (error) {
        console.error('Erro ao conectar WebSocket:', error);
        setIsConnected(false);
      }
    };

    connect();

    socketService.on('connect', () => {
      setIsConnected(true);
      setSocketId(socketService.getSocketId());
    });

    socketService.on('disconnect', () => {
      setIsConnected(false);
      setSocketId(null);
    });

    socketService.on('reconnect_attempt', (data: { attemptNumber: number }) => {
      setReconnectAttempts(data.attemptNumber);
    });

    return () => {
      socketService.disconnect();
      socketService.clearAllListeners();
    };
  }, []);

  const emit = useCallback((event: string, data: any) => {
    return socketService.emit(event, data);
  }, []);

  const on = useCallback((event: string, callback: Function) => {
    if (!eventHandlersRef.current.has(event)) {
      eventHandlersRef.current.set(event, new Set());
    }
    eventHandlersRef.current.get(event)?.add(callback);
    socketService.on(event, callback);
  }, []);

  const off = useCallback((event: string, callback?: Function) => {
    if (callback) {
      const handlers = eventHandlersRef.current.get(event);
      if (handlers) {
        handlers.delete(callback);
        if (handlers.size === 0) {
          eventHandlersRef.current.delete(event);
        }
      }
    } else {
      eventHandlersRef.current.delete(event);
    }
    socketService.off(event, callback);
  }, []);

  const once = useCallback((event: string, callback: Function) => {
    const onceCallback = (data: any) => {
      callback(data);
      off(event, onceCallback);
    };
    on(event, onceCallback);
  }, [on, off]);

  const sendMessage = useCallback((clienteId: number, mensagem: string, remetente: string = 'atendente') => {
    return socketService.sendMessage(clienteId, mensagem, remetente);
  }, []);

  const puxarCliente = useCallback(() => {
    return socketService.puxarCliente();
  }, []);

  const finalizarAtendimento = useCallback((clienteId: number) => {
    return socketService.finalizarAtendimento(clienteId);
  }, []);

  const sendTyping = useCallback((clienteId: number, isTyping: boolean) => {
    return socketService.sendTyping(clienteId, isTyping);
  }, []);

  const reconnect = useCallback(() => {
    socketService.reconnect();
  }, []);

  return {
    isConnected,
    socketId,
    reconnectAttempts,
    emit,
    on,
    off,
    once,
    sendMessage,
    puxarCliente,
    finalizarAtendimento,
    sendTyping,
    reconnect,
  };
};

export default useWebSocket;