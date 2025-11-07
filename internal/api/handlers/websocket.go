package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/internal/websocket"
)

// WebSocketHandler WebSocket 연결 처리
type WebSocketHandler struct {
	hub *websocket.Hub
}

// NewWebSocketHandler WebSocketHandler 생성
func NewWebSocketHandler(hub *websocket.Hub) *WebSocketHandler {
	return &WebSocketHandler{
		hub: hub,
	}
}

// HandleWebSocket WebSocket 연결 엔드포인트
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// 인증 미들웨어에서 설정한 userID 가져오기
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// WebSocket 연결 업그레이드
	websocket.ServeWs(h.hub, c.Writer, c.Request, userID.(string))
}
