package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/internal/models"
)

// ListMatches 모든 매치 목록 조회
func ListMatches(c *gin.Context) {
	// 쿼리 파라미터로 페이지네이션
	// page := c.DefaultQuery("page", "1")
	// limit := c.DefaultQuery("limit", "20")

	matches := []models.Match{}

	c.JSON(http.StatusOK, gin.H{
		"matches": matches,
		"total":   len(matches),
	})
}

// GetMatch 특정 매치 조회
func GetMatch(c *gin.Context) {
	id := c.Param("id")

	// TODO: 데이터베이스에서 조회
	// match, err := matchService.GetByID(id)

	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "Match details - TODO: implement",
	})
}

// ListMatchesByAgent 특정 에이전트의 매치 목록 조회
func ListMatchesByAgent(c *gin.Context) {
	agentId := c.Param("agentId")

	// TODO: 데이터베이스에서 조회
	matches := []models.Match{}

	c.JSON(http.StatusOK, gin.H{
		"agentId": agentId,
		"matches": matches,
		"total":   len(matches),
	})
}
