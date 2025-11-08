package websocket

import (
	"sync"

	"go.uber.org/zap"
)

// Hub WebSocket 연결 관리 및 브로드캐스트
type Hub struct {
	// 사용자별 연결 저장 (userID -> *Client)
	clients map[string]*Client
	mu      sync.RWMutex

	// 브로드캐스트 채널
	broadcast chan *Message

	// 등록/해제 채널
	register   chan *Client
	unregister chan *Client

	logger *zap.Logger
}

// Message WebSocket 메시지
type Message struct {
	UserID  string      `json:"-"`         // 수신자 (빈 문자열이면 전체 브로드캐스트)
	Type    string      `json:"type"`      // 메시지 타입
	Payload interface{} `json:"payload"`   // 메시지 내용
}

// BuildStatusMessage 빌드 상태 변경 메시지
type BuildStatusMessage struct {
	SubmissionID string `json:"submissionId"`
	Status       string `json:"status"`
	Message      string `json:"message,omitempty"`
	ImageURL     string `json:"imageUrl,omitempty"`
}

// NewHub Hub 생성
func NewHub() *Hub {
	logger, _ := zap.NewProduction()
	return &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

// Run Hub 실행
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// registerClient 클라이언트 등록
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 기존 연결이 있으면 닫기
	if oldClient, exists := h.clients[client.userID]; exists {
		close(oldClient.send)
		h.logger.Info("Replaced existing WebSocket connection",
			zap.String("userId", client.userID))
	}

	h.clients[client.userID] = client
	h.logger.Info("WebSocket client registered",
		zap.String("userId", client.userID),
		zap.Int("totalClients", len(h.clients)))
}

// unregisterClient 클라이언트 해제
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[client.userID]; exists {
		delete(h.clients, client.userID)
		close(client.send)
		h.logger.Info("WebSocket client unregistered",
			zap.String("userId", client.userID),
			zap.Int("totalClients", len(h.clients)))
	}
}

// broadcastMessage 메시지 브로드캐스트
func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if message.UserID == "" {
		// 전체 브로드캐스트
		for _, client := range h.clients {
			select {
			case client.send <- message:
			default:
				// 채널이 가득 찬 경우 연결 해제
				h.logger.Warn("Client send channel full, unregistering",
					zap.String("userId", client.userID))
				go func(c *Client) {
					h.unregister <- c
				}(client)
			}
		}
	} else {
		// 특정 사용자에게만 전송
		if client, exists := h.clients[message.UserID]; exists {
			select {
			case client.send <- message:
			default:
				h.logger.Warn("Client send channel full",
					zap.String("userId", message.UserID))
			}
		}
	}
}

// SendToUser 특정 사용자에게 메시지 전송
func (h *Hub) SendToUser(userID string, msgType string, payload interface{}) {
	h.broadcast <- &Message{
		UserID:  userID,
		Type:    msgType,
		Payload: payload,
	}
}

// Broadcast 모든 사용자에게 메시지 전송
func (h *Hub) Broadcast(msgType string, payload interface{}) {
	h.broadcast <- &Message{
		UserID:  "",
		Type:    msgType,
		Payload: payload,
	}
}

// SendBuildStatus 빌드 상태 변경 알림
func (h *Hub) SendBuildStatus(userID, submissionID, status, message, imageURL string) {
	h.SendToUser(userID, "build_status", BuildStatusMessage{
		SubmissionID: submissionID,
		Status:       status,
		Message:      message,
		ImageURL:     imageURL,
	})
}
