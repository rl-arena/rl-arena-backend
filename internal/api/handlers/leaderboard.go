package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/internal/service"
)

type LeaderboardHandler struct {
	agentService *service.AgentService
}

func NewLeaderboardHandler(agentService *service.AgentService) *LeaderboardHandler {
	return &LeaderboardHandler{
		agentService: agentService,
	}
}

// GetLeaderboard 전체 리더보드 조회
func (h *LeaderboardHandler) GetLeaderboard(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	agents, err := h.agentService.GetLeaderboard("", limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get leaderboard",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"leaderboard": agents,
		"total":       len(agents),
	})
}

// GetLeaderboardByEnvironment 특정 환경의 리더보드 조회
func (h *LeaderboardHandler) GetLeaderboardByEnvironment(c *gin.Context) {
	envId := c.Param("envId")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	
	// type 파라미터: public, private, all (기본값: public)
	leaderboardType := c.DefaultQuery("type", "public")

	agents, err := h.agentService.GetLeaderboardWithType(envId, leaderboardType, limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"environmentId": envId,
		"type":          leaderboardType,
		"leaderboard":   agents,
		"total":         len(agents),
	})
}
