package aoa

import (
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type Team5AOA struct {
	ContributionAuditMap map[uuid.UUID]bool
	WithdrawalAuditMap   map[uuid.UUID]bool
}

// ResetAuditMap resets the audit maps for both contribution and withdrawal
func (f *Team5AOA) ResetAuditMap() {
	f.ContributionAuditMap = make(map[uuid.UUID]bool)
	f.WithdrawalAuditMap = make(map[uuid.UUID]bool)
}

// GetExpectedContribution returns the expected contribution from an agent based on its score
func (f *Team5AOA) GetExpectedContribution(agentId uuid.UUID, agentScore int) int {
	// For simplicity, we assume the expected contribution is equal to the agent's score
	return agentScore
}

// SetContributionAuditResult sets the audit result for an agent's contribution
func (f *Team5AOA) SetContributionAuditResult(agentId uuid.UUID, agentScore int, agentActualContribution int, agentStatedContribution int) {
	// If the agent's actual contribution does not match the stated contribution, mark it as failed
	if agentActualContribution != agentStatedContribution {
		f.ContributionAuditMap[agentId] = true
	} else {
		f.ContributionAuditMap[agentId] = false
	}
}

// GetContributionAuditResult returns the audit result for an agent's contribution
func (f *Team5AOA) GetContributionAuditResult(agentId uuid.UUID) bool {
	// true means agent failed the audit (cheated)
	return f.ContributionAuditMap[agentId]
}

// GetExpectedWithdrawal returns the expected withdrawal for an agent based on its score and common pool
func (f *Team5AOA) GetExpectedWithdrawal(agentId uuid.UUID, agentScore int, commonPool int) int {
	// For simplicity, return a fixed amount of 2
	if commonPool >= 2 {
		return 2
	}
	return commonPool // If common pool has less than 2, return whatever is available
}

// SetWithdrawalAuditResult sets the audit result for an agent's withdrawal
func (f *Team5AOA) SetWithdrawalAuditResult(agentId uuid.UUID, agentScore int, agentActualWithdrawal int, agentStatedWithdrawal int, commonPool int) {
	// If the agent's actual withdrawal does not match the stated withdrawal, mark it as failed
	if agentActualWithdrawal != agentStatedWithdrawal {
		f.WithdrawalAuditMap[agentId] = true
	} else {
		f.WithdrawalAuditMap[agentId] = false
	}
}

// GetWithdrawalAuditResult returns the audit result for an agent's withdrawal
func (f *Team5AOA) GetWithdrawalAuditResult(agentId uuid.UUID) bool {
	// true means agent failed the audit (cheated)
	return f.WithdrawalAuditMap[agentId]
}

// GetAuditCost returns the cost of performing an audit
func (f *Team5AOA) GetAuditCost(commonPool int) int {
	// For simplicity, return a fixed audit cost of 5
	if commonPool >= 5 {
		return 5
	}
	return commonPool // If common pool has less than 5, return whatever is available
}

// GetVoteResult determines the agent to be audited based on votes
// MUST return UUID nil if audit should not be executed
func (f *Team5AOA) GetVoteResult(votes []Vote) uuid.UUID {
	voteCount := make(map[uuid.UUID]int)
	// Count the votes for each agent
	for _, vote := range votes {
		if vote.IsVote != 0 {
			voteCount[vote.VotedForID]++
		}
	}

	// Find the agent with the majority of votes
	var maxVotes int
	var selectedAgent uuid.UUID
	for agentID, count := range voteCount {
		if count > maxVotes {
			maxVotes = count
			selectedAgent = agentID
		}
	}

	// If no agent has more than 50% of the votes, return nil
	totalVotes := len(votes)
	if maxVotes > totalVotes/2 {
		return selectedAgent
	}
	return uuid.Nil
}

// GetWithdrawalOrder returns a shuffled order of agents for withdrawal
func (t *Team5AOA) GetWithdrawalOrder(agentIDs []uuid.UUID) []uuid.UUID {
	// Seed the random number generator to ensure different shuffles each time
	rand.Seed(time.Now().UnixNano())

	// Create a copy of the agentIDs to avoid modifying the original list
	shuffledAgents := make([]uuid.UUID, len(agentIDs))
	copy(shuffledAgents, agentIDs)

	// Shuffle the agent list
	rand.Shuffle(len(shuffledAgents), func(i, j int) {
		shuffledAgents[i], shuffledAgents[j] = shuffledAgents[j], shuffledAgents[i]
	})

	return shuffledAgents
}

// RunAoAStuff runs any AoA specific tasks (placeholder for future extension)
func (t *Team5AOA) RunAoAStuff() {
	// Placeholder for future AoA specific operations
}

// CreateFixedAoA creates a new instance of Team5AOA
func CreateTeam5AoA() IArticlesOfAssociation {
	return &Team5AOA{
		ContributionAuditMap: make(map[uuid.UUID]bool),
		WithdrawalAuditMap:   make(map[uuid.UUID]bool),
	}
}
