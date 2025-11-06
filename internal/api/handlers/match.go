package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/internal/service"
)

type MatchHandler struct {
	matchService *service.MatchService
}

func NewMatchHandler(matchService *service.MatchService) *MatchHandler {
	return &MatchHandler{
		matchService: matchService,
	}
}

// CreateMatchRequest 매치 생성 요청
type CreateMatchRequest struct {
	Agent1ID string `json:"agent1Id" binding:"required"`
	Agent2ID string `json:"agent2Id" binding:"required"`
}

// CreateMatch 매치 생성 및 실행 (수동)
func (h *MatchHandler) CreateMatch(c *gin.Context) {
	var req CreateMatchRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 매치 생성 및 실행
	match, err := h.matchService.CreateAndExecute(req.Agent1ID, req.Agent2ID)
	if err != nil {
		if errors.Is(err, service.ErrSameAgent) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Cannot match agent against itself",
			})
			return
		}

		if errors.Is(err, service.ErrDifferentEnvironment) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Agents must be in the same environment",
			})
			return
		}

		if errors.Is(err, service.ErrAgentNotReady) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create match",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Match created and executed",
		"match":   match,
	})
}

// GetMatch 특정 매치 조회
func (h *MatchHandler) GetMatch(c *gin.Context) {
	id := c.Param("id")

	match, err := h.matchService.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrMatchNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Match not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get match",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"match": match,
	})
}

// ListMatchesByAgent 특정 에이전트의 매치 목록 조회
func (h *MatchHandler) ListMatchesByAgent(c *gin.Context) {
	agentId := c.Param("agentId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	matches, err := h.matchService.GetByAgentID(agentId, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get matches",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"matches":  matches,
		"total":    len(matches),
		"page":     page,
		"pageSize": pageSize,
	})
}

// ListMatches 모든 매치 목록 조회 (TODO: 구현)
func ListMatches(c *gin.Context) {
	// TODO: 전체 매치 목록
	c.JSON(http.StatusOK, gin.H{
		"matches": []interface{}{},
		"message": "List all matches - TODO",
	})
}
