package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/internal/service"
)

type SubmissionHandler struct {
	submissionService *service.SubmissionService
}

func NewSubmissionHandler(submissionService *service.SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{
		submissionService: submissionService,
	}
}

// CreateSubmission 새 제출 생성 (파일 업로드)
func (h *SubmissionHandler) CreateSubmission(c *gin.Context) {
	userId, _ := c.Get("userId")
	agentId := c.PostForm("agentId")

	if agentId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "agentId is required",
		})
		return
	}

	// 파일 가져오기
	file, err := c.FormFile("file")
	if err != nil {
		// 파일이 없으면 URL 방식 시도
		var req models.CreateSubmissionRequest
		if err := c.ShouldBindJSON(&req); err == nil && req.CodeURL != "" {
			submission, err := h.submissionService.CreateFromURL(req.AgentID, userId.(string), req.CodeURL)
			if err != nil {
				h.handleError(c, err)
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"message":    "Submission created successfully",
				"submission": submission,
			})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "file or codeUrl is required",
		})
		return
	}

	// 파일 업로드 방식
	submission, err := h.submissionService.Create(agentId, userId.(string), file)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Submission created and uploaded successfully",
		"submission": submission,
	})
}

// GetSubmission 특정 제출 조회
func (h *SubmissionHandler) GetSubmission(c *gin.Context) {
	id := c.Param("id")

	submission, err := h.submissionService.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrSubmissionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Submission not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get submission",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"submission": submission,
	})
}

// ListSubmissionsByAgent 특정 에이전트의 제출 목록 조회
func (h *SubmissionHandler) ListSubmissionsByAgent(c *gin.Context) {
	agentId := c.Param("agentId")

	submissions, err := h.submissionService.GetByAgentID(agentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get submissions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"submissions": submissions,
		"total":       len(submissions),
	})
}

// SetActiveSubmission 제출을 활성화
func (h *SubmissionHandler) SetActiveSubmission(c *gin.Context) {
	id := c.Param("id")
	userId, _ := c.Get("userId")

	err := h.submissionService.SetActive(id, userId.(string))
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Submission activated successfully",
	})
}

// handleError 에러 처리 헬퍼
func (h *SubmissionHandler) handleError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrAgentNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
	} else if errors.Is(err, service.ErrUnauthorized) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
	} else if errors.Is(err, service.ErrInvalidFile) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file"})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}
}
