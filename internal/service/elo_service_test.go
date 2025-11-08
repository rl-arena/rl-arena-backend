package service

import (
	"testing"
)

func TestELOService_GetKFactor(t *testing.T) {
	eloService := NewELOService()

	tests := []struct {
		name        string
		matchCount  int
		expectedK   float64
		description string
	}{
		{
			name:        "New player - 0 matches",
			matchCount:  0,
			expectedK:   40.0,
			description: "Provisional rating for brand new player",
		},
		{
			name:        "New player - 5 matches",
			matchCount:  5,
			expectedK:   40.0,
			description: "Provisional rating for new player",
		},
		{
			name:        "New player - 9 matches",
			matchCount:  9,
			expectedK:   40.0,
			description: "Last match with provisional K-factor",
		},
		{
			name:        "Intermediate player - 10 matches",
			matchCount:  10,
			expectedK:   32.0,
			description: "First match with intermediate K-factor",
		},
		{
			name:        "Intermediate player - 15 matches",
			matchCount:  15,
			expectedK:   32.0,
			description: "Mid-range intermediate player",
		},
		{
			name:        "Intermediate player - 19 matches",
			matchCount:  19,
			expectedK:   32.0,
			description: "Last match with intermediate K-factor",
		},
		{
			name:        "Established player - 20 matches",
			matchCount:  20,
			expectedK:   24.0,
			description: "First match with established K-factor",
		},
		{
			name:        "Established player - 100 matches",
			matchCount:  100,
			expectedK:   24.0,
			description: "Veteran player with stable rating",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualK := eloService.GetKFactor(tt.matchCount)
			if actualK != tt.expectedK {
				t.Errorf("GetKFactor(%d) = %v, want %v (%s)",
					tt.matchCount, actualK, tt.expectedK, tt.description)
			}
		})
	}
}

func TestELOService_CalculateNewRatingsWithMatchCounts(t *testing.T) {
	eloService := NewELOService()

	tests := []struct {
		name            string
		agent1ELO       int
		agent2ELO       int
		agent1Matches   int
		agent2Matches   int
		result          float64
		expectedChange1 int // Approximate expected change for agent1
		description     string
	}{
		{
			name:            "New player beats established player",
			agent1ELO:       1200,
			agent2ELO:       1200,
			agent1Matches:   5,  // K=40 (provisional)
			agent2Matches:   50, // K=24 (established)
			result:          1.0,
			expectedChange1: 20, // Higher change due to higher K-factor
			description:     "New player should gain more ELO when winning",
		},
		{
			name:            "Established player beats new player",
			agent1ELO:       1200,
			agent2ELO:       1200,
			agent1Matches:   50, // K=24 (established)
			agent2Matches:   5,  // K=40 (provisional)
			result:          1.0,
			expectedChange1: 12, // Lower change due to lower K-factor
			description:     "Established player should gain less ELO",
		},
		{
			name:            "Two new players draw",
			agent1ELO:       1200,
			agent2ELO:       1200,
			agent1Matches:   3,  // K=40
			agent2Matches:   7,  // K=40
			result:          0.5, // draw
			expectedChange1: 0,   // Equal ratings, draw = no change
			description:     "Equal players drawing should have no rating change",
		},
		{
			name:            "New player loses to established player",
			agent1ELO:       1200,
			agent2ELO:       1200,
			agent1Matches:   5,  // K=40 (provisional)
			agent2Matches:   50, // K=24 (established)
			result:          0.0,
			expectedChange1: -20, // Larger loss due to higher K-factor
			description:     "New player should lose more ELO when losing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newAgent1ELO, newAgent2ELO, agent1Change, agent2Change :=
				eloService.CalculateNewRatingsWithMatchCounts(
					tt.agent1ELO,
					tt.agent2ELO,
					tt.agent1Matches,
					tt.agent2Matches,
					tt.result,
				)

			// Verify rating changes are applied
			if newAgent1ELO != tt.agent1ELO+agent1Change {
				t.Errorf("Rating calculation mismatch: %d != %d + %d",
					newAgent1ELO, tt.agent1ELO, agent1Change)
			}

			// Verify zero-sum property (total rating change should be ~0 for equal opponents)
			if tt.agent1ELO == tt.agent2ELO && tt.result == 0.5 {
				if agent1Change != 0 || agent2Change != 0 {
					t.Errorf("Equal players drawing should have zero change, got agent1=%d, agent2=%d",
						agent1Change, agent2Change)
				}
			}

			// Verify K-factor effect (approximate check)
			k1 := eloService.GetKFactor(tt.agent1Matches)
			k2 := eloService.GetKFactor(tt.agent2Matches)

			// For equal opponents, rating change should be proportional to K-factor
			if tt.agent1ELO == tt.agent2ELO && tt.result != 0.5 {
				expectedRatio := k1 / k2
				actualRatio := float64(agent1Change) / float64(-agent2Change)

				// Allow 10% tolerance due to rounding
				if actualRatio < expectedRatio*0.9 || actualRatio > expectedRatio*1.1 {
					t.Errorf("K-factor ratio mismatch: expected ~%.2f, got %.2f (agent1Change=%d, agent2Change=%d, k1=%.1f, k2=%.1f)",
						expectedRatio, actualRatio, agent1Change, agent2Change, k1, k2)
				}
			}

			t.Logf("%s: Agent1 ELO %d→%d (%+d, K=%.0f), Agent2 ELO %d→%d (%+d, K=%.0f)",
				tt.description,
				tt.agent1ELO, newAgent1ELO, agent1Change, k1,
				tt.agent2ELO, newAgent2ELO, agent2Change, k2,
			)
		})
	}
}

func TestELOService_BackwardCompatibility(t *testing.T) {
	eloService := NewELOService()

	// Test that old method still works
	agent1ELO := 1200
	agent2ELO := 1300
	result := 1.0 // agent1 wins

	newAgent1ELO, newAgent2ELO, agent1Change, agent2Change := eloService.CalculateNewRatings(
		agent1ELO,
		agent2ELO,
		result,
	)

	if newAgent1ELO == agent1ELO || newAgent2ELO == agent2ELO {
		t.Error("Old CalculateNewRatings method should still calculate rating changes")
	}

	if agent1Change <= 0 || agent2Change >= 0 {
		t.Errorf("Winner should gain ELO and loser should lose ELO, got agent1Change=%d, agent2Change=%d",
			agent1Change, agent2Change)
	}

	t.Logf("Backward compatibility verified: Agent1 %d→%d (%+d), Agent2 %d→%d (%+d)",
		agent1ELO, newAgent1ELO, agent1Change,
		agent2ELO, newAgent2ELO, agent2Change)
}
