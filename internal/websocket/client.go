package websocket

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: 프로덕션에서는 특정 origin만 허용
		return true
	},
}

// Client WebSocket 클라이언트
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan *Message
	userID string
	logger *zap.Logger
}

// NewClient 클라이언트 생성
func NewClient(hub *Hub, conn *websocket.Conn, userID string) *Client {
	logger, _ := zap.NewProduction()
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan *Message, 256),
		userID: userID,
		logger: logger,
	}
}

// readPump 클라이언트로부터 메시지 읽기 (핑/퐁 유지)
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("WebSocket read error",
					zap.String("userId", c.userID),
					zap.Error(err))
			}
			break
		}
		// 클라이언트로부터 메시지는 무시 (단방향 통신)
	}
}

// writePump Hub로부터 메시지를 받아 클라이언트에게 전송
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
				// Hub가 채널을 닫음
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// JSON으로 인코딩
			data, err := json.Marshal(message)
			if err != nil {
				c.logger.Error("Failed to marshal message",
					zap.String("userId", c.userID),
					zap.Error(err))
				continue
			}

			// 메시지 전송
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				c.logger.Error("Failed to write message",
					zap.String("userId", c.userID),
					zap.Error(err))
				return
			}

		case <-ticker.C:
			// Ping 전송
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWs WebSocket 연결 업그레이드 및 클라이언트 시작
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, userID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger, _ := zap.NewProduction()
		logger.Error("Failed to upgrade WebSocket connection", zap.Error(err))
		return
	}

	client := NewClient(hub, conn, userID)
	client.hub.register <- client

	// 고루틴 시작
	go client.writePump()
	go client.readPump()
}
