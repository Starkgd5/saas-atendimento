import { io, Socket } from 'socket.io-client';

// Tipos de eventos do WebSocket
export const SOCKET_EVENTS = {
  CONNECT: 'connect',
  DISCONNECT: 'disconnect',
  CONNECT_ERROR: 'connect_error',
  RECONNECT: 'reconnect',
  RECONNECT_ATTEMPT: 'reconnect_attempt',
  RECONNECT_ERROR: 'reconnect_error',
  RECONNECT_FAILED: 'reconnect_failed',
  
  // Eventos customizados
  NOVA_MENSAGEM: 'nova_mensagem',
  FILA_ATUALIZADA: 'fila_atualizada',
  CLIENTE_ENTROU: 'cliente_entrou',
  CLIENTE_SAIU: 'cliente_saiu',
  ATENDIMENTO_FINALIZADO: 'atendimento_finalizado',
  TYPING: 'typing',
  PONG: 'pong',
} as const;

export type SocketEvent = typeof SOCKET_EVENTS[keyof typeof SOCKET_EVENTS];

interface SocketOptions {
  url?: string;
  token?: string;
  autoConnect?: boolean;
  reconnection?: boolean;
  reconnectionAttempts?: number;
  reconnectionDelay?: number;
  reconnectionDelayMax?: number;
  timeout?: number;
}

interface MessagePayload {
  type: string;
  payload: any;
  from?: string;
  to?: string;
  room?: string;
  time?: Date;
}

class SocketService {
  private static instance: SocketService;
  private socket: Socket | null = null;
  private isConnected = false;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private eventHandlers: Map<string, Set<Function>> = new Map();
  private connectionPromise: Promise<Socket> | null = null;
  private connectionResolve: (() => void) | null = null;

  private constructor() {}

  static getInstance(): SocketService {
    if (!SocketService.instance) {
      SocketService.instance = new SocketService();
    }
    return SocketService.instance;
  }

  // ============================================
  // CONEXÃO
  // ============================================

  connect(options: SocketOptions = {}): Promise<Socket> {
    if (this.socket?.connected) {
      return Promise.resolve(this.socket);
    }

    // Se já estiver conectando, retorna a promise existente
    if (this.connectionPromise) {
      return this.connectionPromise;
    }

    this.connectionPromise = new Promise((resolve, reject) => {
      const {
        url = process.env.REACT_APP_WS_URL || 'ws://localhost:8080',
        token = localStorage.getItem('token') || '',
        autoConnect = true,
        reconnection = true,
        reconnectionAttempts = 5,
        reconnectionDelay = 1000,
        reconnectionDelayMax = 5000,
        timeout = 10000,
      } = options;

      this.maxReconnectAttempts = reconnectionAttempts;

      this.socket = io(url, {
        transports: ['websocket'],
        query: { token },
        autoConnect,
        reconnection,
        reconnectionAttempts,
        reconnectionDelay,
        reconnectionDelayMax,
        timeout,
        forceNew: true,
        path: '/socket.io/',
      });

      // Eventos padrão
      this.socket.on(SOCKET_EVENTS.CONNECT, () => {
        this.isConnected = true;
        this.reconnectAttempts = 0;
        console.log('✅ WebSocket conectado');
        this.emit(SOCKET_EVENTS.PONG, { time: new Date() });
        this.connectionResolve?.();
        this.connectionResolve = null;
        resolve(this.socket!);
      });

      this.socket.on(SOCKET_EVENTS.DISCONNECT, (reason) => {
        this.isConnected = false;
        console.log(`🔌 WebSocket desconectado: ${reason}`);
        this.emitEvent(SOCKET_EVENTS.DISCONNECT, { reason });
      });

      this.socket.on(SOCKET_EVENTS.CONNECT_ERROR, (error) => {
        console.error('❌ Erro no WebSocket:', error);
        this.emitEvent(SOCKET_EVENTS.CONNECT_ERROR, { error });
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
          reject(error);
          this.connectionResolve = null;
        }
      });

      this.socket.on(SOCKET_EVENTS.RECONNECT, (attemptNumber) => {
        console.log(`🔄 WebSocket reconectado após ${attemptNumber} tentativas`);
        this.emitEvent(SOCKET_EVENTS.RECONNECT, { attemptNumber });
      });

      this.socket.on(SOCKET_EVENTS.RECONNECT_ATTEMPT, (attemptNumber) => {
        this.reconnectAttempts = attemptNumber;
        console.log(`🔄 Tentativa de reconexão ${attemptNumber}/${this.maxReconnectAttempts}`);
        this.emitEvent(SOCKET_EVENTS.RECONNECT_ATTEMPT, { attemptNumber });
      });

      this.socket.on(SOCKET_EVENTS.RECONNECT_ERROR, (error) => {
        console.error('❌ Erro na reconexão:', error);
        this.emitEvent(SOCKET_EVENTS.RECONNECT_ERROR, { error });
      });

      this.socket.on(SOCKET_EVENTS.RECONNECT_FAILED, () => {
        console.error('❌ Falha na reconexão após múltiplas tentativas');
        this.emitEvent(SOCKET_EVENTS.RECONNECT_FAILED, {});
        reject(new Error('Falha na reconexão'));
        this.connectionResolve = null;
      });

      // Eventos customizados
      this.socket.on(SOCKET_EVENTS.NOVA_MENSAGEM, (data) => {
        this.emitEvent(SOCKET_EVENTS.NOVA_MENSAGEM, data);
      });

      this.socket.on(SOCKET_EVENTS.FILA_ATUALIZADA, (data) => {
        this.emitEvent(SOCKET_EVENTS.FILA_ATUALIZADA, data);
      });

      this.socket.on(SOCKET_EVENTS.CLIENTE_ENTROU, (data) => {
        this.emitEvent(SOCKET_EVENTS.CLIENTE_ENTROU, data);
      });

      this.socket.on(SOCKET_EVENTS.CLIENTE_SAIU, (data) => {
        this.emitEvent(SOCKET_EVENTS.CLIENTE_SAIU, data);
      });

      this.socket.on(SOCKET_EVENTS.ATENDIMENTO_FINALIZADO, (data) => {
        this.emitEvent(SOCKET_EVENTS.ATENDIMENTO_FINALIZADO, data);
      });

      this.socket.on(SOCKET_EVENTS.TYPING, (data) => {
        this.emitEvent(SOCKET_EVENTS.TYPING, data);
      });

      // Qualquer outro evento
      this.socket.onAny((event, ...args) => {
        this.emitEvent(event, ...args);
      });

      this.connectionResolve = () => {
        resolve(this.socket!);
      };

      if (autoConnect && !this.socket.connected) {
        this.socket.connect();
      }
    });

    return this.connectionPromise;
  }

  // ============================================
  // DESCONEXÃO
  // ============================================

  disconnect(): void {
    if (this.socket) {
      this.socket.disconnect();
      this.socket.removeAllListeners();
      this.socket = null;
    }
    this.isConnected = false;
    this.connectionPromise = null;
    this.connectionResolve = null;
    console.log('🔌 WebSocket desconectado manualmente');
  }

  // ============================================
  // ENVIO DE MENSAGENS
  // ============================================

  emit(event: string, data: any): boolean {
    if (!this.socket || !this.isConnected) {
      console.warn(`⚠️ Tentativa de emitir evento "${event}" sem conexão`);
      return false;
    }

    try {
      this.socket.emit(event, data);
      return true;
    } catch (error) {
      console.error(`❌ Erro ao emitir evento "${event}":`, error);
      return false;
    }
  }

  // Enviar mensagem no chat
  sendMessage(clienteId: number, mensagem: string, remetente: string = 'atendente'): boolean {
    return this.emit(SOCKET_EVENTS.NOVA_MENSAGEM, {
      cliente_id: clienteId,
      mensagem,
      remetente,
      time: new Date(),
    });
  }

  // Puxar próximo cliente da fila
  puxarCliente(): boolean {
    return this.emit('puxar_cliente', {});
  }

  // Finalizar atendimento
  finalizarAtendimento(clienteId: number): boolean {
    return this.emit(SOCKET_EVENTS.ATENDIMENTO_FINALIZADO, {
      cliente_id: clienteId,
      time: new Date(),
    });
  }

  // Indicar digitação
  sendTyping(clienteId: number, isTyping: boolean): boolean {
    return this.emit(SOCKET_EVENTS.TYPING, {
      cliente_id: clienteId,
      is_typing: isTyping,
      time: new Date(),
    });
  }

  // ============================================
  // EVENTOS
  // ============================================

  on(event: string, callback: Function): void {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, new Set());
    }
    this.eventHandlers.get(event)?.add(callback);

    // Se já estiver conectado e tiver listener no socket
    if (this.socket) {
      this.socket.on(event, (data) => {
        this.emitEvent(event, data);
      });
    }
  }

  off(event: string, callback?: Function): void {
    if (!callback) {
      this.eventHandlers.delete(event);
      if (this.socket) {
        this.socket.off(event);
      }
      return;
    }

    const handlers = this.eventHandlers.get(event);
    if (handlers) {
      handlers.delete(callback);
      if (handlers.size === 0) {
        this.eventHandlers.delete(event);
        if (this.socket) {
          this.socket.off(event);
        }
      }
    }
  }

  once(event: string, callback: Function): void {
    const onceCallback = (data: any) => {
      callback(data);
      this.off(event, onceCallback);
    };
    this.on(event, onceCallback);
  }

  private emitEvent(event: string, ...args: any[]): void {
    const handlers = this.eventHandlers.get(event);
    if (handlers) {
      handlers.forEach((callback) => {
        try {
          callback(...args);
        } catch (error) {
          console.error(`❌ Erro no handler do evento "${event}":`, error);
        }
      });
    }
  }

  // ============================================
  // STATUS
  // ============================================

  isConnectedToServer(): boolean {
    return (this.isConnected && this.socket?.connected) || false;
  }

  getSocketId(): string | null {
    return this.socket?.id || null;
  }

  getReconnectAttempts(): number {
    return this.reconnectAttempts;
  }

  // ============================================
  // RECONEXÃO MANUAL
  // ============================================

  reconnect(): void {
    if (this.socket && !this.isConnected) {
      console.log('🔄 Tentando reconectar manualmente...');
      this.socket.connect();
    }
  }

  // ============================================
  // LIMPEZA
  // ============================================

  clearAllListeners(): void {
    this.eventHandlers.clear();
    if (this.socket) {
      this.socket.removeAllListeners();
    }
  }

  // ============================================
  // UTILITÁRIOS
  // ============================================

  getConnectionStatus(): {
    isConnected: boolean;
    socketId: string | null;
    reconnectAttempts: number;
  } {
    return {
      isConnected: this.isConnected,
      socketId: this.getSocketId(),
      reconnectAttempts: this.reconnectAttempts,
    };
  }
}

// Exportar instância única
export default SocketService.getInstance();

// Exportar tipos
export type { SocketOptions, MessagePayload };