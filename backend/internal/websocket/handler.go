package websocket

import (
	"net/http"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Handler обрабатывает WebSocket-подключение
func Handler(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hub.logger.Info("WebSocket connection attempt", zap.String("remote_addr", r.RemoteAddr))

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			hub.logger.Warn("WebSocket upgrade failed", zap.Error(err))
			return
		}
		hub.logger.Info("WebSocket upgraded, registering client")

		hub.register <- conn
		defer func() {
			hub.unregister <- conn
		}()

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				hub.logger.Debug("WebSocket read error, closing", zap.Error(err))
				break
			}
		}
	}
}
