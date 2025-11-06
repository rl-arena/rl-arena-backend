package service

import "math"

// ELOService ELO 레이팅 계산 서비스
type ELOService struct {
	kFactor float64
}

// NewELOService ELO 서비스 생성
func NewELOService() *ELOService {
	return &ELOService{
		kFactor: 32, // K-factor: 레이팅 변동 폭
	}
}

// CalculateNewRatings 매치 결과에 따른 새로운 ELO 레이팅 계산
// result: 1.0 (agent1 승), 0.5 (무승부), 0.0 (agent2 승)
func (s *ELOService) CalculateNewRatings(agent1ELO, agent2ELO int, result float64) (newAgent1ELO, newAgent2ELO, agent1Change, agent2Change int) {
	// 기대 승률 계산
	expectedAgent1 := s.expectedScore(float64(agent1ELO), float64(agent2ELO))
	expectedAgent2 := 1.0 - expectedAgent1

	// 새 레이팅 계산
	newAgent1ELOFloat := float64(agent1ELO) + s.kFactor*(result-expectedAgent1)
	newAgent2ELOFloat := float64(agent2ELO) + s.kFactor*((1.0-result)-expectedAgent2)

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
