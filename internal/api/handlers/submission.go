package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/internal/models"
)

// CreateSubmission 새 제출 생성
func CreateSubmission(c *gin.Context) {
	var req models.CreateSubmissionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	userId, _ := c.Get("userId")

	// TODO:
	// 1. Agent 소유자 확인
	// 2. 코드 다운로드 및 검증
	// 3. 빌드 큐에 추가
	// submission, err := submissionService.Create(req, userId.(string))

	c.JSON(http.StatusCreated, gin.H{
		"message": "Submission created and queued for building",
		"agentId": req.AgentID,
		"userId":  userId, // ← 사용!
	})
}

// GetSubmission 특정 제출 조회
func GetSubmission(c *gin.Context) {
	id := c.Param("id")

	// TODO: 데이터베이스에서 조회

	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "Submission details - TODO: implement",
	})
}

// ListSubmissionsByAgent 특정 에이전트의 제출 목록 조회
func ListSubmissionsByAgent(c *gin.Context) {
	agentId := c.Param("agentId")

	submissions := []models.Submission{}

	c.JSON(http.StatusOK, gin.H{
		"agentId":     agentId,
		"submissions": submissions,
		"total":       len(submissions),
	})
}
