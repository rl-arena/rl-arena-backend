package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetLeaderboard 전체 리더보드 조회
func GetLeaderboard(c *gin.Context) {
	// TODO: ELO 순으로 정렬된 에이전트 목록
	// agents, err := agentService.GetLeaderboard(limit)

	c.JSON(http.StatusOK, gin.H{
		"leaderboard": []gin.H{},
	})
}

// GetLeaderboardByEnvironment 특정 환경의 리더보드 조회
func GetLeaderboardByEnvironment(c *gin.Context) {
	envId := c.Param("envId")

	// TODO: 특정 환경의 리더보드

	c.JSON(http.StatusOK, gin.H{
		"environmentId": envId,
		"leaderboard":   []gin.H{},
	})
}
