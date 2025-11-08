package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/rl-arena/rl-arena-backend/pkg/logger"
	pb "github.com/rl-arena/rl-arena-backend/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	address    string
	conn       *grpc.ClientConn
	grpcClient pb.ExecutorClient
}

// NewClient Executor gRPC 클라이언트 생성
func NewClient(address string) (*Client, error) {
	// gRPC connection options
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(10 * time.Second),
	}

	// gRPC 연결
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to executor: %w", err)
	}

	client := pb.NewExecutorClient(conn)

	logger.Info("Connected to Executor gRPC service", "address", address)

	return &Client{
		address:    address,
		conn:       conn,
		grpcClient: client,
	}, nil
}

// Close gRPC 연결 종료
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ExecuteMatchRequest Executor에 보낼 요청 (레거시 호환용)
type ExecuteMatchRequest struct {
	MatchID       string      `json:"matchId"`
	EnvironmentID string      `json:"environmentId"`
	Agent1        AgentInfo   `json:"agent1"`
	Agent2        AgentInfo   `json:"agent2"`
	Config        interface{} `json:"config,omitempty"`
}

type AgentInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CodeURL      string `json:"codeUrl"`
	DockerImage  string `json:"dockerImage,omitempty"`
	Version      string `json:"version,omitempty"`
}

// ExecuteMatchResponse Executor로부터 받는 응답 (레거시 호환용)
type ExecuteMatchResponse struct {
	MatchID       string  `json:"matchId"`
	Status        string  `json:"status"` // "success", "error", "timeout"
	WinnerID      *string `json:"winnerId,omitempty"`
	Agent1Score   float64 `json:"agent1Score"`
	Agent2Score   float64 `json:"agent2Score"`
	ReplayURL     string  `json:"replayUrl,omitempty"`
	ReplayHTMLURL string  `json:"replayHtmlUrl,omitempty"`
	Duration      int64   `json:"duration"` // milliseconds
	ErrorMsg      string  `json:"error,omitempty"`
}

// ExecuteMatch 게임 실행 요청 (gRPC)
func (c *Client) ExecuteMatch(req ExecuteMatchRequest) (*ExecuteMatchResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	logger.Info("Sending match execution request to Executor via gRPC",
		"address", c.address,
		"matchId", req.MatchID,
		"environment", req.EnvironmentID,
	)

	// ExecuteMatchRequest를 proto MatchRequest로 변환
	agents := []*pb.AgentData{
		{
			AgentId:     req.Agent1.ID,
			DockerImage: req.Agent1.DockerImage,
			Version:     req.Agent1.Version,
			CodeUrl:     req.Agent1.CodeURL, // deprecated, backward compatibility
			Metadata: map[string]string{
				"name": req.Agent1.Name,
			},
		},
		{
			AgentId:     req.Agent2.ID,
			DockerImage: req.Agent2.DockerImage,
			Version:     req.Agent2.Version,
			CodeUrl:     req.Agent2.CodeURL,
			Metadata: map[string]string{
				"name": req.Agent2.Name,
			},
		},
	}

	grpcReq := &pb.MatchRequest{
		MatchId:       req.MatchID,
		Environment:   req.EnvironmentID,
		Agents:        agents,
		TimeoutSec:    300, // 5 minutes
		RecordReplay:  true,
	}

	// gRPC 호출
	resp, err := c.grpcClient.RunMatch(ctx, grpcReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute match: %w", err)
	}

	// proto MatchResponse를 ExecuteMatchResponse로 변환
	result := &ExecuteMatchResponse{
		MatchID:       resp.MatchId,
		Status:        convertStatus(resp.Status),
		ReplayURL:     resp.ReplayUrl,
		ReplayHTMLURL: resp.ReplayHtmlUrl,
		Duration:      int64(resp.ExecutionTimeSec * 1000), // seconds to milliseconds
		ErrorMsg:      resp.ErrorMessage,
	}

	// Winner ID 설정
	if resp.WinnerAgentId != "" {
		result.WinnerID = &resp.WinnerAgentId
	}

	// Agent scores 계산
	if len(resp.AgentResults) >= 2 {
		result.Agent1Score = resp.AgentResults[0].Score
		result.Agent2Score = resp.AgentResults[1].Score
	}

	logger.Info("Match execution completed via gRPC",
		"matchId", result.MatchID,
		"status", result.Status,
		"duration", result.Duration,
	)

	return result, nil
}

// convertStatus proto status enum을 문자열로 변환
func convertStatus(status pb.MatchStatus) string {
	switch status {
	case pb.MatchStatus_STATUS_SUCCESS:
		return "success"
	case pb.MatchStatus_STATUS_TIMEOUT:
		return "timeout"
	case pb.MatchStatus_STATUS_ERROR:
		return "error"
	case pb.MatchStatus_STATUS_CANCELLED:
		return "cancelled"
	default:
		return "unknown"
	}
}

// HealthCheck Executor 상태 확인 (gRPC)
func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.HealthCheckRequest{}
	resp, err := c.grpcClient.HealthCheck(ctx, req)
	if err != nil {
		return fmt.Errorf("executor health check failed: %w", err)
	}

	if !resp.Healthy {
		return fmt.Errorf("executor is not healthy")
	}

	logger.Info("Executor health check passed",
		"version", resp.Version,
		"activeMatches", resp.ActiveMatches,
	)

	return nil
}
