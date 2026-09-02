package websocket

import (
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Message — структура события, отправляемого клиентам
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Hub — менеджер WebSocket-соединений
type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan Message
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
	logger     *zap.Logger
}

func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan Message),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		logger:     logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.logger.Info("WebSocket client registered", zap.Int("total", len(h.clients)))
			h.mu.Unlock()

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
				h.logger.Info("WebSocket client unregistered", zap.Int("total", len(h.clients)))
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.Lock()
			h.logger.Debug("Broadcasting message",
				zap.String("type", msg.Type),
				zap.Int("clients", len(h.clients)),
			)
			for conn := range h.clients {
				err := conn.WriteJSON(msg)
				if err != nil {
					h.logger.Warn("WriteJSON error", zap.Error(err))
					conn.Close()
					delete(h.clients, conn)
				}
			}
			h.mu.Unlock()
		}
	}
}

// Broadcast отправляет сообщение всем клиентам
func (h *Hub) Broadcast(msg Message) {
	h.broadcast <- msg
}
