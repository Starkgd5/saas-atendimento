package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ============================================
// CONSTANTES
// ============================================

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512KB
)

// ============================================
// TIPOS DE MENSAGEM
// ============================================

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	From    string          `json:"from,omitempty"`
	To      string          `json:"to,omitempty"`
	Room    string          `json:"room,omitempty"`
	Time    time.Time       `json:"time,omitempty"`
}

// ============================================
// CLIENTE
// ============================================

type Client struct {
	conn      *websocket.Conn
	userID    int
	lojaID    int
	role      string
	send      chan []byte
	manager   *Manager
	mu        sync.Mutex
	lastPing  time.Time
	connected bool
}

// ============================================
// MANAGER
// ============================================

type Manager struct {
	clients       map[*Client]bool
	clientsByUser map[int]*Client
	rooms         map[string]map[*Client]bool
	broadcast     chan []byte
	register      chan *Client
	unregister    chan *Client
	mu            sync.RWMutex
	upgrader      websocket.Upgrader
	pingInterval  time.Duration
}

// ============================================
// CONSTRUTOR
// ============================================

func NewManager() *Manager {
	return &Manager{
		clients:       make(map[*Client]bool),
		clientsByUser: make(map[int]*Client),
		rooms:         make(map[string]map[*Client]bool),
		broadcast:     make(chan []byte, 100),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		pingInterval: pingPeriod,
	}
}

// ============================================
// RUN - LOOP PRINCIPAL
// ============================================

func (m *Manager) Run() {
	ticker := time.NewTicker(m.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case client := <-m.register:
			m.registerClient(client)

		case client := <-m.unregister:
			m.unregisterClient(client)

		case message := <-m.broadcast:
			m.broadcastMessage(message)

		case <-ticker.C:
			m.pingClients()
		}
	}
}

// ============================================
// OPERAÇÕES DE CLIENTE
// ============================================

func (m *Manager) registerClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[client] = true
	m.clientsByUser[client.userID] = client

	roomName := m.getRoomName(client.lojaID)
	if m.rooms[roomName] == nil {
		m.rooms[roomName] = make(map[*Client]bool)
	}
	m.rooms[roomName][client] = true

	log.Printf("✅ Cliente %d (loja %d) conectado", client.userID, client.lojaID)
}

func (m *Manager) unregisterClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.clients[client]; ok {
		delete(m.clients, client)
		delete(m.clientsByUser, client.userID)

		roomName := m.getRoomName(client.lojaID)
		if room, ok := m.rooms[roomName]; ok {
			delete(room, client)
			if len(room) == 0 {
				delete(m.rooms, roomName)
			}
		}

		close(client.send)
		log.Printf("🔌 Cliente %d desconectado", client.userID)
	}
}

func (m *Manager) broadcastMessage(message []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for client := range m.clients {
		select {
		case client.send <- message:
		default:
			go m.unregisterClient(client)
		}
	}
}

func (m *Manager) pingClients() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for client := range m.clients {
		if err := client.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
			log.Printf("Erro ao enviar ping para cliente %d: %v", client.userID, err)
			go m.unregisterClient(client)
		}
	}
}

// ============================================
// MÉTODOS PÚBLICOS
// ============================================

func (m *Manager) EnviarParaClientes(lojaID int, message []byte) {
	roomName := m.getRoomName(lojaID)
	m.EnviarParaSala(roomName, message)
}

func (m *Manager) EnviarParaSala(room string, message []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if clients, ok := m.rooms[room]; ok {
		for client := range clients {
			select {
			case client.send <- message:
			default:
				go m.unregisterClient(client)
			}
		}
	}
}

func (m *Manager) EnviarParaUsuario(userID int, message []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if client, ok := m.clientsByUser[userID]; ok {
		select {
		case client.send <- message:
		default:
			go m.unregisterClient(client)
		}
	}
}

func (m *Manager) EnviarParaTodos(message []byte) {
	m.broadcast <- message
}

func (m *Manager) GetClientesConectados() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

func (m *Manager) GetClientesPorLoja(lojaID int) int {
	roomName := m.getRoomName(lojaID)
	m.mu.RLock()
	defer m.mu.RUnlock()

	if clients, ok := m.rooms[roomName]; ok {
		return len(clients)
	}
	return 0
}

func (m *Manager) GetClientesConectadosDetalhes() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var clients []map[string]interface{}
	for client := range m.clients {
		clients = append(clients, map[string]interface{}{
			"user_id": client.userID,
			"loja_id": client.lojaID,
			"role":    client.role,
		})
	}

	return map[string]interface{}{
		"total":    len(clients),
		"clientes": clients,
	}
}

// ============================================
// MÉTODOS AUXILIARES
// ============================================

func (m *Manager) getRoomName(lojaID int) string {
	return "loja_" + string(rune(lojaID))
}

// ============================================
// SERVE WS
// ============================================

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request, userID, lojaID int) {
	// Atualizar o upgrader para permitir todas as origens
	m.upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("❌ Erro ao upgrade WebSocket:", err)
		return
	}

	role := r.URL.Query().Get("role")
	if role == "" {
		role = "atendente"
	}

	client := &Client{
		conn:      conn,
		userID:    userID,
		lojaID:    lojaID,
		role:      role,
		send:      make(chan []byte, 256),
		manager:   m,
		lastPing:  time.Now(),
		connected: true,
	}

	m.register <- client

	go client.writePump()
	go client.readPump()
}

// ============================================
// CLIENT PUMPS
// ============================================

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.manager.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.lastPing = time.Now()
		return nil
	})

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Erro ao ler mensagem do cliente %d: %v", c.userID, err)
			}
			break
		}

		c.manager.handleMessage(msg, c)
	}
}

// ============================================
// HANDLE MESSAGE
// ============================================

func (m *Manager) handleMessage(msg Message, client *Client) {
	switch msg.Type {
	case "nova_mensagem":
		// Verificar se o payload é válido
		if len(msg.Payload) > 0 {
			m.EnviarParaClientes(client.lojaID, msg.Payload)
		}

	case "puxar_cliente":
		response, _ := json.Marshal(map[string]interface{}{
			"type":    "fila_atualizada",
			"payload": map[string]interface{}{"action": "puxar_cliente"},
		})
		m.EnviarParaClientes(client.lojaID, response)

	case "finalizar_atendimento":
		response, _ := json.Marshal(map[string]interface{}{
			"type":    "atendimento_finalizado",
			"payload": map[string]interface{}{"cliente_id": msg.Payload},
		})
		m.EnviarParaClientes(client.lojaID, response)

	case "ping":
		response, _ := json.Marshal(map[string]interface{}{
			"type": "pong",
			"time": time.Now(),
		})
		client.send <- response

	case "join_room":
		if msg.Room != "" {
			m.mu.Lock()
			if m.rooms[msg.Room] == nil {
				m.rooms[msg.Room] = make(map[*Client]bool)
			}
			m.rooms[msg.Room][client] = true
			m.mu.Unlock()
		}

	case "leave_room":
		if msg.Room != "" {
			m.mu.Lock()
			if room, ok := m.rooms[msg.Room]; ok {
				delete(room, client)
				if len(room) == 0 {
					delete(m.rooms, msg.Room)
				}
			}
			m.mu.Unlock()
		}

	default:
		log.Printf("Mensagem desconhecida do cliente %d: %s", client.userID, msg.Type)
	}
}

// ============================================
// MÉTODOS DE CONFIGURAÇÃO
// ============================================

func (m *Manager) SetPingInterval(interval time.Duration) {
	m.pingInterval = interval
}

func (m *Manager) SetCheckOrigin(checkOrigin func(r *http.Request) bool) {
	m.upgrader.CheckOrigin = checkOrigin
}