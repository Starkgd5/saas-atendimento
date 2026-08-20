class SocketService {
  private static instance: SocketService;
  private ws: WebSocket | null = null;
  private isConnected = false;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectTimeout: NodeJS.Timeout | null = null;
  private eventHandlers: Map<string, Set<Function>> = new Map();
  private pingInterval: NodeJS.Timeout | null = null;

  private constructor() {}

  static getInstance(): SocketService {
    if (!SocketService.instance) {
      SocketService.instance = new SocketService();
    }
    return SocketService.instance;
  }

  connect(token: string = ''): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }

      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = process.env.REACT_APP_WS_URL || `${protocol}//localhost:8080`;
      const tokenParam = token || localStorage.getItem('token') || '';
      const url = `${host}/ws?token=${encodeURIComponent(tokenParam)}`;

      console.log(`🔌 Conectando WebSocket: ${url}`);

      try {
        this.ws = new WebSocket(url);

        this.ws.onopen = () => {
          console.log('✅ WebSocket conectado');
          this.isConnected = true;
          this.reconnectAttempts = 0;
          this.emitEvent('connect', {});
          
          if (this.pingInterval) {
            clearInterval(this.pingInterval);
          }
          this.pingInterval = setInterval(() => {
            if (this.ws?.readyState === WebSocket.OPEN) {
              this.ws.send(JSON.stringify({ type: 'ping' }));
            }
          }, 30000);
          
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data);
            console.log('📩 Mensagem recebida:', data);
            
            if (data.type === 'pong') {
              console.log('🏓 Pong recebido');
              return;
            }
            
            this.emitEvent(data.type || 'message', data);
          } catch (error) {
            console.error('❌ Erro ao processar mensagem:', error);
          }
        };

        this.ws.onerror = (error) => {
          console.error('❌ Erro no WebSocket:', error);
          this.emitEvent('error', { error });
        };

        this.ws.onclose = (event) => {
          console.log(`🔌 WebSocket desconectado: ${event.code} - ${event.reason}`);
          this.isConnected = false;
          this.emitEvent('disconnect', { code: event.code, reason: event.reason });

          if (this.pingInterval) {
            clearInterval(this.pingInterval);
            this.pingInterval = null;
          }

          if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
            console.log(`🔄 Reconectando em ${delay}ms... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
            
            if (this.reconnectTimeout) {
              clearTimeout(this.reconnectTimeout);
            }
            this.reconnectTimeout = setTimeout(() => {
              this.connect().catch(() => {});
            }, delay);
          } else {
            console.error('❌ Falha na reconexão após múltiplas tentativas');
            this.emitEvent('reconnect_failed', {});
          }
        };

      } catch (error) {
        console.error('❌ Erro ao criar WebSocket:', error);
        reject(error);
      }
    });
  }

  disconnect(): void {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
    if (this.ws) {
      this.ws.close(1000, 'Desconectado manualmente');
      this.ws = null;
    }
    this.isConnected = false;
    this.reconnectAttempts = 0;
    console.log('🔌 WebSocket desconectado manualmente');
  }

  emit(event: string, data: any): boolean {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn(`⚠️ Tentativa de emitir evento "${event}" sem conexão`);
      return false;
    }

    try {
      const message = JSON.stringify({ type: event, payload: data });
      this.ws.send(message);
      return true;
    } catch (error) {
      console.error(`❌ Erro ao emitir evento "${event}":`, error);
      return false;
    }
  }

  on(event: string, callback: Function): void {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, new Set());
    }
    this.eventHandlers.get(event)?.add(callback);
  }

  off(event: string, callback?: Function): void {
    if (!callback) {
      this.eventHandlers.delete(event);
      return;
    }
    const handlers = this.eventHandlers.get(event);
    if (handlers) {
      handlers.delete(callback);
      if (handlers.size === 0) {
        this.eventHandlers.delete(event);
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

  private emitEvent(event: string, data: any): void {
    const handlers = this.eventHandlers.get(event);
    if (handlers) {
      handlers.forEach((callback) => {
        try {
          callback(data);
        } catch (error) {
          console.error(`❌ Erro no handler do evento "${event}":`, error);
        }
      });
    }
  }

  isConnectedToServer(): boolean {
    return this.isConnected && this.ws?.readyState === WebSocket.OPEN;
  }

  getSocketId(): string | null {
    return this.ws?.url || null;
  }

  getReconnectAttempts(): number {
    return this.reconnectAttempts;
  }

  getConnectionStatus(): { isConnected: boolean; socketId: string | null; reconnectAttempts: number } {
    return {
      isConnected: this.isConnected,
      socketId: this.getSocketId(),
      reconnectAttempts: this.reconnectAttempts,
    };
  }

  // Métodos de conveniência
  sendMessage(clienteId: number, mensagem: string, remetente: string = 'atendente'): boolean {
    return this.emit('nova_mensagem', {
      cliente_id: clienteId,
      mensagem,
      remetente,
      time: new Date().toISOString()
    });
  }

  puxarCliente(): boolean {
    return this.emit('puxar_cliente', {});
  }

  finalizarAtendimento(clienteId: number): boolean {
    return this.emit('atendimento_finalizado', {
      cliente_id: clienteId,
      time: new Date().toISOString()
    });
  }

  // Remover método que não existe
  // clearAllListeners() foi removido
}

export default SocketService.getInstance();