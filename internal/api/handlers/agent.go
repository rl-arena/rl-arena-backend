package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/internal/models"
)

// ListAgents 모든 에이전트 목록 조회
func ListAgents(c *gin.Context) {
	// TODO: 데이터베이스에서 조회
	agents := []models.Agent{}

	c.JSON(http.StatusOK, gin.H{
		"agents": agents,
		"total":  len(agents),
	})
}

// GetAgent 특정 에이전트 조회
func GetAgent(c *gin.Context) {
	id := c.Param("id")

	// TODO: 데이터베이스에서 조회
	// agent, err := agentService.GetByID(id)

	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "Agent details - TODO: implement",
	})
}

// CreateAgent 새 에이전트 생성
func CreateAgent(c *gin.Context) {
	var req models.CreateAgentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// context에서 userId 가져오기 (Auth 미들웨어에서 설정)
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// TODO: 데이터베이스에 저장
	// agent, err := agentService.Create(userId.(string), req)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Agent created successfully",
		"name":    req.Name,
		"userId":  userId, // ← 이제 userId 사용!
	})
}

// UpdateAgent 에이전트 수정
func UpdateAgent(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateAgentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	userId, _ := c.Get("userId")

	// TODO: 소유자 확인 및 업데이트
	// agent, err := agentService.Update(id, userId.(string), req)

	c.JSON(http.StatusOK, gin.H{
		"message": "Agent updated successfully",
		"id":      id,
		"userId":  userId, // ← 사용!
	})
}

// DeleteAgent 에이전트 삭제
func DeleteAgent(c *gin.Context) {
	id := c.Param("id")
	userId, _ := c.Get("userId")

	// TODO: 소유자 확인 및 삭제
	// err := agentService.Delete(id, userId.(string))

	c.JSON(http.StatusOK, gin.H{
		"message": "Agent deleted successfully",
		"id":      id,
		"userId":  userId, // ← 사용!
	})
}
