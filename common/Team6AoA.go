package common

import (
	"github.com/google/uuid"
)

type CheatingRecord struct {
	Expected      int
	Actual        int
	CheatedAmount int
}

type Team6AoA struct {
	weight float64 // Weight for current turn contributions
	decay  float64 // Decay rate for cumulative contributions

	cumulativeContributions map[uuid.UUID]float64           // Cumulative contributions with decay
	currentContributions    map[uuid.UUID]float64           // Current turn contributions
	auditHistory            map[uuid.UUID][]*CheatingRecord // Audit history per agent

}

func (t *Team6AoA) GetAuditCost(commonPool int) int

func (t *Team6AoA) GetExpectedContribution(agentId uuid.UUID, agentScore int) int {
	return int(float64(agentScore) * 0.3)
}
func (t *Team6AoA) SetContributionAuditResult(agentId uuid.UUID, agentScore int, actualContribution int, agentStatedContribution int) {
	// Store current contribution
	t.currentContributions[agentId] = float64(actualContribution)

	// Update cumulative contributions with decay
	oldCumulative := t.cumulativeContributions[agentId]
	newCumulative := (oldCumulative * t.decay) + float64(actualContribution)
	t.cumulativeContributions[agentId] = newCumulative

	expectedContribution := int(float64(agentScore) * 0.3)
	// Check for cheating
	if actualContribution < expectedContribution {
		cheatedAmount := expectedContribution - actualContribution
		record := &CheatingRecord{
			Expected:      expectedContribution,
			Actual:        actualContribution,
			CheatedAmount: cheatedAmount,
		}
		t.auditHistory[agentId] = append(t.auditHistory[agentId], record)
	} else {
		// Append nil to indicate no cheating in this contribution turn
		t.auditHistory[agentId] = append(t.auditHistory[agentId], nil)
	}

}

func (t *Team6AoA) GetVoteResult(votes []Vote) uuid.UUID {

	votingPower := t.CalculateVotingPower()

	voteTotals := make(map[uuid.UUID]float64)

	for _, vote := range votes {
		if vote.IsVote == 1 {
			voteTotals[vote.VotedForID] += votingPower[vote.VoterID]
		}
	}

	// Find the agent with the highest vote count above threshold (50%)
	const auditThreshold = 0.5
	var maxVotedAgent uuid.UUID
	maxVotes := 0.0

	for agentID, voteTotal := range voteTotals {
		if voteTotal > maxVotes && voteTotal > auditThreshold {
			maxVotedAgent = agentID
			maxVotes = voteTotal
		}
	}

	return maxVotedAgent // Returns uuid.Nil if no agent meets the threshold

}

// Will just check if the agent cheated in their last contribution
// TODO: Implement probability based detection:
// - Baseline detection: 50% + (1-50% based on how much the agent cheated)
// - So that the probability of detection is based on how much the agent cheated
func (t *Team6AoA) GetContributionAuditResult(agentId uuid.UUID) bool {
	// Check if agent has any audit history
	if history, exists := t.auditHistory[agentId]; exists && len(history) > 0 {
		// Get the most recent audit record
		lastRecord := history[len(history)-1]
		// If lastRecord is not nil, it means the agent cheated in their last contribution
		return lastRecord != nil
	}
	return false // No history or empty history means no detected cheating
}

func (t *Team6AoA) RunPostContributionAoaLogic(team *Team, agentMap map[uuid.UUID]IExtendedAgent)

func (t *Team6AoA) GetWithdrawalOrder(agentIDs []uuid.UUID) []uuid.UUID
func (t *Team6AoA) GetExpectedWithdrawal(agentId uuid.UUID, agentScore int, commonPool int) int
func (t *Team6AoA) SetWithdrawalAuditResult(agentId uuid.UUID, agentScore int, agentActualWithdrawal int, agentStatedWithdrawal int, commonPool int)
func (t *Team6AoA) GetWithdrawalAuditResult(agentId uuid.UUID) bool

func (t *Team6AoA) CalculateVotingPower() map[uuid.UUID]float64 {
	weightedContributions := make(map[uuid.UUID]float64)
	totalWeighted := 0.0

	// Calculate weighted contributions for each agent using the formula:
	// weighted_contribution = (w × current_contribution) + ((1-w) × cumulative_contribution)
	for agentID, current := range t.currentContributions {
		cumulative := t.cumulativeContributions[agentID]
		// Apply weighting of current vs. cumulative
		weighted := (t.weight * current) + ((1 - t.weight) * cumulative)
		weightedContributions[agentID] = weighted
		totalWeighted += weighted
	}

	// Convert weighted contributions into voting power proportions
	// voting_power = weighted_contribution / total_weighted_contributions
	votingPower := make(map[uuid.UUID]float64)
	for agentID, weighted := range weightedContributions {
		if totalWeighted == 0 {
			votingPower[agentID] = 0
		} else {
			// This ensures all voting powers sum to 1.0
			votingPower[agentID] = weighted / totalWeighted
		}
	}

	return votingPower
}

func createTeam6AoA() IArticlesOfAssociation {
	return &Team6AoA{}
}
