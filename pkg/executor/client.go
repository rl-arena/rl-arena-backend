package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rl-arena/rl-arena-backend/pkg/logger"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient Executor 클라이언트 생성
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // 게임 실행은 오래 걸릴 수 있음
		},
	}
}

// ExecuteMatchRequest Executor에 보낼 요청
type ExecuteMatchRequest struct {
	MatchID       string      `json:"matchId"`
	EnvironmentID string      `json:"environmentId"`
	Agent1        AgentInfo   `json:"agent1"`
	Agent2        AgentInfo   `json:"agent2"`
	Config        interface{} `json:"config,omitempty"`
}

type AgentInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	CodeURL string `json:"codeUrl"`
}

// ExecuteMatchResponse Executor로부터 받는 응답
type ExecuteMatchResponse struct {
	MatchID     string  `json:"matchId"`
	Status      string  `json:"status"` // "success", "error"
	WinnerID    *string `json:"winnerId,omitempty"`
	Agent1Score float64 `json:"agent1Score"`
	Agent2Score float64 `json:"agent2Score"`
	ReplayURL   string  `json:"replayUrl,omitempty"`
	Duration    int64   `json:"duration"` // milliseconds
	ErrorMsg    string  `json:"error,omitempty"`
}

// ExecuteMatch 게임 실행 요청
func (c *Client) ExecuteMatch(req ExecuteMatchRequest) (*ExecuteMatchResponse, error) {
	url := fmt.Sprintf("%s/execute", c.baseURL)

	logger.Info("Sending match execution request to Executor",
		"url", url,
		"matchId", req.MatchID,
		"environment", req.EnvironmentID,
	)

	// JSON 변환
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// HTTP 요청
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// 요청 전송
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 응답 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 상태 코드 확인
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("executor returned error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	// JSON 파싱
	var result ExecuteMatchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	logger.Info("Match execution completed",
		"matchId", result.MatchID,
		"status", result.Status,
		"duration", result.Duration,
	)

	return &result, nil
}

// HealthCheck Executor 상태 확인
func (c *Client) HealthCheck() error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("executor health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("executor is not healthy: status=%d", resp.StatusCode)
	}

	return nil
}
