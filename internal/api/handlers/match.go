package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/internal/service"
	"github.com/rl-arena/rl-arena-backend/pkg/storage"
)

type MatchHandler struct {
	matchService *service.MatchService
	storage      *storage.Storage
}

func NewMatchHandler(matchService *service.MatchService, storage *storage.Storage) *MatchHandler {
	return &MatchHandler{
		matchService: matchService,
		storage:      storage,
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

// GetMatchReplay 매치 리플레이 파일 다운로드
// Supports format query parameter: json (default) or html (Kaggle-style visualization)
func (h *MatchHandler) GetMatchReplay(c *gin.Context) {
	id := c.Param("id")
	format := c.DefaultQuery("format", "json") // json or html

	// ID 유효성 검사
	if id == "" || id == "match_undefined" || id == "undefined" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No match available yet. This agent hasn't played any matches.",
			"message": "진행한 매치가 없습니다.",
		})
		return
	}

	// 매치 조회
	match, err := h.matchService.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrMatchNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Match not found",
				"message": "매치를 찾을 수 없습니다.",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get match",
			"message": "매치 정보를 가져오는데 실패했습니다.",
		})
		return
	}

	// Determine which replay URL to use based on format
	var replayURL *string
	var contentType string
	var fileExtension string

	switch format {
	case "html":
		replayURL = match.ReplayHTMLURL
		contentType = "text/html"
		fileExtension = "html"
	case "json":
		fallthrough
	default:
		replayURL = match.ReplayURL
		contentType = "application/json"
		fileExtension = "json"
	}

	// 리플레이 URL 확인
	if replayURL == nil || *replayURL == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Replay not available for this match",
		})
		return
	}

	replayPath := *replayURL
	
	// 파일 존재 여부 확인
	if !h.storage.FileExists(replayPath) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Replay file not found",
		})
		return
	}

	// 파일 전체 경로
	fullPath := h.storage.GetFilePath(replayPath)
	
	// 파일 정보 가져오기
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to access replay file",
		})
		return
	}

	// For JSON format, return the content as JSON response (not download)
	if format == "json" {
		// Read JSON file
		data, err := os.ReadFile(fullPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to read replay file",
			})
			return
		}

		// Parse and transform JSON for frontend compatibility
		var executorReplay map[string]interface{}
		if err := json.Unmarshal(data, &executorReplay); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to parse replay file",
			})
			return
		}

		// Transform executor format to frontend-compatible format
		transformedReplay := transformExecutorReplayForFrontend(executorReplay)

		c.JSON(http.StatusOK, transformedReplay)
		return
	}

	// For HTML format, serve as downloadable file
	filename := fmt.Sprintf("replay_%s.%s", match.ID, fileExtension)
	
	// Content-Disposition 헤더 설정 (다운로드)
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
	
	// 파일 전송
	c.File(fullPath)
}

// GetMatchReplayURL 매치 리플레이 URL 조회 (파일 다운로드 없이 URL만)
// Supports format parameter: json (default) or html (Kaggle-style visualization)
func (h *MatchHandler) GetMatchReplayURL(c *gin.Context) {
	id := c.Param("id")
	format := c.DefaultQuery("format", "json") // json or html

	// ID 유효성 검사
	if id == "" || id == "match_undefined" || id == "undefined" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No match available yet. This agent hasn't played any matches.",
			"message": "진행한 매치가 없습니다.",
		})
		return
	}

	// 매치 조회
	match, err := h.matchService.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrMatchNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Match not found",
				"message": "매치를 찾을 수 없습니다.",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get match",
			"message": "매치 정보를 가져오는데 실패했습니다.",
		})
		return
	}

	// Determine which replay URL to use based on format
	var replayURL *string
	switch format {
	case "html":
		replayURL = match.ReplayHTMLURL
	case "json":
		fallthrough
	default:
		replayURL = match.ReplayURL
	}

	// 리플레이 URL 확인
	if replayURL == nil || *replayURL == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Replay not available for this match",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"replayUrl": h.storage.GetFileURL(*replayURL),
		"matchId":   match.ID,
		"format":    format,
	})
}

// GetMatchesWithReplays 리플레이가 있는 매치 목록 조회 (Watch 기능용)
func (h *MatchHandler) GetMatchesWithReplays(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	agentID := c.Query("agentId") // Optional: 특정 Agent 필터링

	var matches []*models.Match
	var err error

	if agentID != "" {
		// 특정 Agent의 매치만 조회
		matches, err = h.matchService.GetByAgentID(agentID, page, pageSize)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get matches",
			})
			return
		}
	} else {
		// agentId 파라미터 필수
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "agentId query parameter is required",
		})
		return
	}

	// 리플레이가 있는 매치만 필터링
	matchesWithReplays := []*models.Match{}
	for _, match := range matches {
		if match.ReplayURL != nil && *match.ReplayURL != "" {
			// ReplayURL을 실제 접근 가능한 URL로 변환
			replayURL := h.storage.GetFileURL(*match.ReplayURL)
			match.ReplayURL = &replayURL
			matchesWithReplays = append(matchesWithReplays, match)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"matches":  matchesWithReplays,
		"total":    len(matchesWithReplays),
		"page":     page,
		"pageSize": pageSize,
	})
}

// transformExecutorReplayForFrontend transforms executor replay format to frontend-compatible format
func transformExecutorReplayForFrontend(executorReplay map[string]interface{}) map[string]interface{} {
	frames, ok := executorReplay["frames"].([]interface{})
	if !ok {
		return executorReplay
	}

	transformedFrames := make([]interface{}, len(frames))
	for i, frameData := range frames {
		frame, ok := frameData.(map[string]interface{})
		if !ok {
			transformedFrames[i] = frameData
			continue
		}

		// Create info object if it doesn't exist
		info, ok := frame["info"].(map[string]interface{})
		if !ok {
			info = make(map[string]interface{})
			frame["info"] = info
		}

		// Parse ball_pos: "0.5 0.5" -> [0.5, 0.5]
		if ballPosStr, ok := info["ball_pos"].(string); ok {
			var ballX, ballY float64
			if n, _ := fmt.Sscanf(ballPosStr, "%f %f", &ballX, &ballY); n == 2 {
				info["ball_pos"] = []float64{ballX, ballY}
			}
		}

		// Parse paddle_positions: "0.5 0.5" -> [0.5, 0.5]
		if paddlePosStr, ok := info["paddle_positions"].(string); ok {
			var paddle1Y, paddle2Y float64
			if n, _ := fmt.Sscanf(paddlePosStr, "%f %f", &paddle1Y, &paddle2Y); n == 2 {
				info["paddle_positions"] = []float64{paddle1Y, paddle2Y}
			}
		}

		// Parse scores: "4 9" -> [4, 9] or keep as array
		// Frontend expects "score" key, backend has "scores"
		if scoresStr, ok := info["scores"].(string); ok {
			var score1, score2 int
			if n, _ := fmt.Sscanf(scoresStr, "%d %d", &score1, &score2); n == 2 {
				info["score"] = []int{score1, score2}
				delete(info, "scores") // Remove old key
			}
		} else if scoresArr, ok := info["scores"].([]interface{}); ok {
			// Already an array, just rename the key
			info["score"] = scoresArr
			delete(info, "scores")
		}

		transformedFrames[i] = frame
	}

	result := make(map[string]interface{})
	for k, v := range executorReplay {
		result[k] = v
	}
	result["frames"] = transformedFrames

	return result
}
