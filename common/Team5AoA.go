package common

import (
	// environmentServer "SOMAS_Extended/server"
	"container/list"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type Team5AOA struct {
	ContributionAuditMap map[uuid.UUID]*list.List
	WithdrawalAuditMap   map[uuid.UUID]bool
	ContributionRoundMap map[uuid.UUID]int // Tracks the number of successful contribution rounds for each agent
	Allocation           map[uuid.UUID]int // Stores the resource allocation for each agent
}

// ResetAuditMap resets the audit maps for both contribution and withdrawal
func (f *Team5AOA) ResetAuditMap() {
	f.ContributionAuditMap = make(map[uuid.UUID]*list.List)
	f.WithdrawalAuditMap = make(map[uuid.UUID]bool)
	f.ContributionRoundMap = make(map[uuid.UUID]int)
	f.Allocation = make(map[uuid.UUID]int)
}

// GetExpectedContribution returns the expected contribution from an agent based on its score
func (f *Team5AOA) GetExpectedContribution(agentId uuid.UUID, agentScore int) int {
	// According to the AoA document, each member contributes 75% of their current resources
	return int(float64(agentScore) * 0.75)
}

// SetContributionAuditResult sets the audit result for an agent's contribution
func (f *Team5AOA) SetContributionAuditResult(agentId uuid.UUID, agentScore int, agentActualContribution int, agentStatedContribution int) {
	if f.ContributionAuditMap[agentId] == nil {
		f.ContributionAuditMap[agentId] = list.New()
	}
	// If the agent's actual contribution does not match the stated contribution, mark it as failed and add to the list
	f.ContributionAuditMap[agentId].PushBack(agentStatedContribution > agentActualContribution)

	// Track successful contributions for bonus
	if agentStatedContribution == agentActualContribution {
		f.ContributionRoundMap[agentId]++
	} else {
		f.ContributionRoundMap[agentId] = 0 // Reset the count if the contribution is incorrect
	}
}

// GetContributionAuditResult returns the audit result for an agent's contribution
func (f *Team5AOA) GetContributionAuditResult(agentId uuid.UUID) bool {
	// true means agent failed the audit (cheated)
	if f.ContributionAuditMap[agentId] == nil {
		return false
	}
	for e := f.ContributionAuditMap[agentId].Front(); e != nil; e = e.Next() {
		if e.Value.(bool) {
			return true
		}
	}
	return false
}

// GetExpectedWithdrawal returns the expected withdrawal for an agent based on ResourceAllocation
func (f *Team5AOA) GetExpectedWithdrawal(agentId uuid.UUID, agentScore int, commonPool int) int {
	if val, ok := f.Allocation[agentId]; ok {
		return val
	}
	return 0
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
	// According to the AoA document, auditing consumes 5% of the resources from the common pool
	cost := int(float64(commonPool) * 0.05)
	if cost < 1 {
		cost = 1 // Ensure a minimum cost of 1
	}
	return cost
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

// GetBonusContribution returns the bonus for contributing correctly for three consecutive rounds
func (f *Team5AOA) GetBonusContribution(agentId uuid.UUID, commonPool int) int {
	if f.ContributionRoundMap[agentId] >= 3 {
		// According to the AoA document, agents receive a bonus for contributing correctly for three consecutive rounds
		return int(float64(commonPool) * 0.05) // Bonus amount is set to 5% of the common pool
	}
	return 0
}

// ApplyPunishment applies punishment to an agent if they failed an audit
func (f *Team5AOA) ApplyPunishment(agentId uuid.UUID) bool {
	if f.GetContributionAuditResult(agentId) || f.GetWithdrawalAuditResult(agentId) {
		return true // Forfeit 100% of withdrawn resources (represented as -100% deduction)
	}
	return false
}

// KickOutAgent checks if an agent should be kicked out based on audit failures
func (f *Team5AOA) KickOutAgent(agentId uuid.UUID) bool {
	// If an agent fails three audits in a row, they are kicked out
	failCount := 0
	if f.ContributionAuditMap[agentId] != nil {
		for e := f.ContributionAuditMap[agentId].Front(); e != nil; e = e.Next() {
			if e.Value.(bool) {
				failCount++
			}
			if failCount >= 3 {
				return true
			}
		}
	}
	return false
}

func (t *Team5AOA) RunPreIterationAoaLogic(team *Team, agentMap map[uuid.UUID]IExtendedAgent)     {}
func (t *Team5AOA) RunPostContributionAoaLogic(team *Team, agentMap map[uuid.UUID]IExtendedAgent) {}

func (f *Team5AOA) ResourceAllocation(agentScores map[uuid.UUID]int, remainingResources int) map[uuid.UUID]int {
	// Step 1: Calculate the need threshold (T)
	var scores []int
	for _, score := range agentScores {
		scores = append(scores, score)
	}
	medianScore := calculateMedian(scores)
	meanScore := calculateMean(scores)
	alpha := 0.7 // α is set between 0.5 to 0.8 as per the requirement
	threshold := max(medianScore, int(float64(meanScore)*alpha))

	// Step 2: Allocate resources based on need level until needs are met or resources are depleted
	agentIDs := make([]uuid.UUID, 0, len(agentScores))
	for agentID := range agentScores {
		agentIDs = append(agentIDs, agentID)
	}

	// Sort agent IDs based on scores in ascending order (lower scores get higher priority)
	sortedAgents := make([]uuid.UUID, len(agentIDs))
	copy(sortedAgents, agentIDs)
	for i := 0; i < len(sortedAgents)-1; i++ {
		for j := i + 1; j < len(sortedAgents); j++ {
			if agentScores[sortedAgents[i]] > agentScores[sortedAgents[j]] {
				sortedAgents[i], sortedAgents[j] = sortedAgents[j], sortedAgents[i]
			}
		}
	}

	allocation := make(map[uuid.UUID]int)
	nPriority := 0
	for _, agentID := range sortedAgents {
		if remainingResources <= 0 {
			break
		}
		need := threshold - agentScores[agentID]
		if need > 0 {
			allocation[agentID] = min(need, remainingResources/(nPriority+1))
			remainingResources -= allocation[agentID]
			nPriority++
		}
	}

	// Step 3: Residual allocation - distribute remaining resources equally among all agents
	if remainingResources > 0 {
		residual := remainingResources / len(agentScores)
		for agentID := range agentScores {
			allocation[agentID] += residual
		}
	}

	// Update the Allocation map
	f.Allocation = allocation

	return allocation
}

// Utility functions
func calculateMedian(numbers []int) int {
	size := len(numbers)
	if size == 0 {
		return 0
	}
	sorted := make([]int, size)
	copy(sorted, numbers)
	for i := 0; i < size-1; i++ {
		for j := i + 1; j < size; j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	if size%2 == 0 {
		return (sorted[size/2-1] + sorted[size/2]) / 2
	}
	return sorted[size/2]
}

func calculateMean(numbers []int) int {
	sum := 0
	for _, number := range numbers {
		sum += number
	}
	if len(numbers) == 0 {
		return 0
	}
	return sum / len(numbers)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CreateFixedAoA creates a new instance of Team5AOA
func CreateTeam5AoA() IArticlesOfAssociation {
	return &Team5AOA{
		ContributionAuditMap: make(map[uuid.UUID]*list.List),
		WithdrawalAuditMap:   make(map[uuid.UUID]bool),
		ContributionRoundMap: make(map[uuid.UUID]int),
		Allocation:           make(map[uuid.UUID]int),
	}
}

// Do nothing
func (t *Team5AOA) Team4_SetRankUp(map[uuid.UUID]map[uuid.UUID]int) {
}

func (t *Team5AOA) Team4_RunProposedWithdrawalVote(map[uuid.UUID]int, map[uuid.UUID]map[uuid.UUID]int) {
}
func (t *Team5AOA) Team4_HandlePunishmentVote(map[uuid.UUID]map[int]int) int {
	return 0
}

func (t *Team5AOA) GetPunishment(agentScore int, agentId uuid.UUID) int {
	return (agentScore * 25) / 100
}
