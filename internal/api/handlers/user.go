package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/internal/models"
)

// GetCurrentUser 현재 사용자 정보 조회
func GetCurrentUser(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// TODO: 데이터베이스에서 조회
	// user, err := userService.GetByID(userId.(string))

	c.JSON(http.StatusOK, gin.H{
		"userId":  userId, // ← 사용!
		"message": "User details - TODO: implement",
	})
}

// UpdateCurrentUser 현재 사용자 정보 수정
func UpdateCurrentUser(c *gin.Context) {
	var req models.UpdateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	userId, _ := c.Get("userId")

	// TODO: 데이터베이스 업데이트
	// user, err := userService.Update(userId.(string), req)

	c.JSON(http.StatusOK, gin.H{
		"message": "User updated successfully",
		"userId":  userId, // ← 사용!
	})
}
