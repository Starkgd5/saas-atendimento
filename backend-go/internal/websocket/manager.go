package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type    string          `json:"type"`    // "nova_mensagem", "fila_atualizada", "cliente_entrou"
	Payload json.RawMessage `json:"payload"`
}

type Client struct {
	conn     *websocket.Conn
	userID   int
	lojaID   int
	send     chan []byte
	manager  *Manager
}

type Manager struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func NewManager() *Manager {
	return &Manager{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 100),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (m *Manager) Run() {
	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			m.clients[client] = true
			m.mu.Unlock()
			log.Printf("Cliente %d conectado", client.userID)

		case client := <-m.unregister:
			m.mu.Lock()
			if _, ok := m.clients[client]; ok {
				delete(m.clients, client)
				close(client.send)
			}
			m.mu.Unlock()
			log.Printf("Cliente %d desconectado", client.userID)

		case message := <-m.broadcast:
			m.mu.RLock()
			for client := range m.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(m.clients, client)
				}
			}
			m.mu.RUnlock()
		}
	}
}

// EnviarParaClientes envia mensagem para clientes de uma loja específica
func (m *Manager) EnviarParaClientes(lojaID int, message []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for client := range m.clients {
		if client.lojaID == lojaID {
			select {
			case client.send <- message:
			default:
				// Cliente pode estar desconectado
			}
		}
	}
}

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request, userID, lojaID int) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Erro ao upgrade WebSocket:", err)
		return
	}

	client := &Client{
		conn:    conn,
		userID:  userID,
		lojaID:  lojaID,
		send:    make(chan []byte, 256),
		manager: m,
	}

	m.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
		c.manager.unregister <- c
	}()

	for message := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			return
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.conn.Close()
		c.manager.unregister <- c
	}()

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		// Processar mensagem recebida
		c.manager.handleMessage(msg, c)
	}
}

func (m *Manager) handleMessage(msg Message, client *Client) {
	switch msg.Type {
	case "nova_mensagem":
		// Reenviar para todos os clientes da mesma loja
		m.EnviarParaClientes(client.lojaID, []byte(msg.Payload))
	case "puxar_cliente":
		// Solicitação para puxar próximo da fila
		m.broadcast <- []byte(`{"type":"fila_atualizada","payload":"{}"}`)
	}
}