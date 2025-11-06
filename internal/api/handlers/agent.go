package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/internal/service"
)

type AgentHandler struct {
	agentService *service.AgentService
}

func NewAgentHandler(agentService *service.AgentService) *AgentHandler {
	return &AgentHandler{
		agentService: agentService,
	}
}

// ListAgents 모든 에이전트 목록 조회
func (h *AgentHandler) ListAgents(c *gin.Context) {
	// 페이지네이션 파라미터
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	agents, total, err := h.agentService.List(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get agents",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agents":   agents,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetAgent 특정 에이전트 조회
func (h *AgentHandler) GetAgent(c *gin.Context) {
	id := c.Param("id")

	agent, err := h.agentService.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrAgentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Agent not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get agent",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agent": agent,
	})
}

// CreateAgent 새 에이전트 생성
func (h *AgentHandler) CreateAgent(c *gin.Context) {
	var req models.CreateAgentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// context에서 userId 가져오기
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// 에이전트 생성
	agent, err := h.agentService.Create(
		userId.(string),
		req.Name,
		req.Description,
		req.EnvironmentID,
	)
	if err != nil {
		// 에러 로깅 추가
		fmt.Printf("Agent creation error: %v\n", err)

		if errors.Is(err, service.ErrInvalidEnvironment) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid environment",
			})
			return
		}

		if errors.Is(err, service.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid input",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create agent",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Agent created successfully",
		"agent":   agent,
	})
}

// UpdateAgent 에이전트 수정
func (h *AgentHandler) UpdateAgent(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateAgentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	userId, _ := c.Get("userId")

	// 에이전트 업데이트
	err := h.agentService.Update(id, userId.(string), req.Name, req.Description)
	if err != nil {
		if errors.Is(err, service.ErrAgentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Agent not found",
			})
			return
		}

		if errors.Is(err, service.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You don't have permission to update this agent",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update agent",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Agent updated successfully",
	})
}

// DeleteAgent 에이전트 삭제
func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	id := c.Param("id")
	userId, _ := c.Get("userId")

	// 에이전트 삭제
	err := h.agentService.Delete(id, userId.(string))
	if err != nil {
		if errors.Is(err, service.ErrAgentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Agent not found",
			})
			return
		}

		if errors.Is(err, service.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You don't have permission to delete this agent",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete agent",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Agent deleted successfully",
	})
}

// GetMyAgents 내 에이전트 목록 조회
func (h *AgentHandler) GetMyAgents(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	agents, err := h.agentService.GetByUserID(userId.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get agents",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agents": agents,
		"total":  len(agents),
	})
}
