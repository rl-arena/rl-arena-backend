package service

import "math"

// ELOService ELO 레이팅 계산 서비스
type ELOService struct {
	defaultKFactor float64 // Default K-factor for established players
}

// NewELOService ELO 서비스 생성
func NewELOService() *ELOService {
	return &ELOService{
		defaultKFactor: 32, // K-factor: 레이팅 변동 폭 (기본값)
	}
}

// GetKFactor returns the appropriate K-factor based on the number of matches played.
// Implements provisional rating system similar to Kaggle competitions:
// - New players (< 10 matches): K=40 for faster convergence
// - Intermediate players (10-20 matches): K=32 for moderate adjustment
// - Established players (> 20 matches): K=24 for rating stability
func (s *ELOService) GetKFactor(matchCount int) float64 {
	if matchCount < 10 {
		return 40.0 // Provisional rating - faster convergence
	} else if matchCount < 20 {
		return 32.0 // Intermediate - moderate adjustment
	}
	return 24.0 // Established rating - stable
}

// CalculateNewRatings 매치 결과에 따른 새로운 ELO 레이팅 계산
// result: 1.0 (agent1 승), 0.5 (무승부), 0.0 (agent2 승)
// Backward compatibility: uses default K-factor for both players
func (s *ELOService) CalculateNewRatings(agent1ELO, agent2ELO int, result float64) (newAgent1ELO, newAgent2ELO, agent1Change, agent2Change int) {
	// 기대 승률 계산
	expectedAgent1 := s.expectedScore(float64(agent1ELO), float64(agent2ELO))
	expectedAgent2 := 1.0 - expectedAgent1

	// 새 레이팅 계산 (기본 K-factor 사용)
	newAgent1ELOFloat := float64(agent1ELO) + s.defaultKFactor*(result-expectedAgent1)
	newAgent2ELOFloat := float64(agent2ELO) + s.defaultKFactor*((1.0-result)-expectedAgent2)

	newAgent1ELO = int(math.Round(newAgent1ELOFloat))
	newAgent2ELO = int(math.Round(newAgent2ELOFloat))

	agent1Change = newAgent1ELO - agent1ELO
	agent2Change = newAgent2ELO - agent2ELO

	return
}

// CalculateNewRatingsWithMatchCounts calculates new ELO ratings using dynamic K-factors.
// This method should be used when match count data is available for provisional rating support.
// result: 1.0 (agent1 wins), 0.5 (draw), 0.0 (agent2 wins)
func (s *ELOService) CalculateNewRatingsWithMatchCounts(
	agent1ELO, agent2ELO int,
	agent1Matches, agent2Matches int,
	result float64,
) (newAgent1ELO, newAgent2ELO, agent1Change, agent2Change int) {
	// 기대 승률 계산
	expectedAgent1 := s.expectedScore(float64(agent1ELO), float64(agent2ELO))
	expectedAgent2 := 1.0 - expectedAgent1

	// Use dynamic K-factors based on match count (provisional rating system)
	kFactor1 := s.GetKFactor(agent1Matches)
	kFactor2 := s.GetKFactor(agent2Matches)

	// 새 레이팅 계산 (동적 K-factor 사용)
	newAgent1ELOFloat := float64(agent1ELO) + kFactor1*(result-expectedAgent1)
	newAgent2ELOFloat := float64(agent2ELO) + kFactor2*((1.0-result)-expectedAgent2)

	newAgent1ELO = int(math.Round(newAgent1ELOFloat))
	newAgent2ELO = int(math.Round(newAgent2ELOFloat))

	agent1Change = newAgent1ELO - agent1ELO
	agent2Change = newAgent2ELO - agent2ELO

	return
}

// expectedScore ELO에 기반한 기대 승률 계산
func (s *ELOService) expectedScore(ratingA, ratingB float64) float64 {
	return 1.0 / (1.0 + math.Pow(10, (ratingB-ratingA)/400.0))
}
