package common

import (
	"container/list"
	"fmt"
	"log"
	"math/rand"
	"time"

	gameRecorder "github.com/ADimoska/SOMASExtended/gameRecorder"
	"github.com/google/uuid"
)

// Strategy defines the punishment severity: Lenient, Moderates, or Resolutes.
type Strategy int

const (
	Lenient   Strategy = 0
	Moderates Strategy = 1
	Resolutes Strategy = 2
)

// Punishment represents the outcome of an audit, detailing the strategy used
// and the amount of score reduction.
type Punishment struct {
	Strategy       Strategy // Strategy used (Lenient, Moderates, or Resolutes)
	ScoreReduction int      // Amount to deduct from the agent's score
}

// AuditQueue manages a fixed-length queue of audit results (true for lied, false for honest).
type Team3AuditQueue struct {
	length int
	rounds list.List // Stores the audit results in a linked list
}

// NewAuditQueue creates and initializes a new AuditQueue with a specified maximum length.
func NewTeam3AuditQueue(length int) *Team3AuditQueue {
	return &Team3AuditQueue{
		length: length,
		rounds: list.List{},
	}
}

// AddToQueue adds a new audit result to the queue. If the queue is full, the oldest result is removed.
func (aq *Team3AuditQueue) AddToQueue(auditResult bool) {
	if aq.length == aq.rounds.Len() {
		aq.rounds.Remove(aq.rounds.Front()) // Remove the oldest result if the queue is full
	}
	aq.rounds.PushBack(auditResult) // Add the new result to the back of the queue
}

// GetWarnings counts the number of "true" entries in the queue, representing instances of lying.
func (aq *Team3AuditQueue) GetWarnings() int {
	warnings := 0
	for e := aq.rounds.Front(); e != nil; e = e.Next() {
		if e.Value.(bool) { // Check if the value is true (indicating a lie)
			warnings++
		}
	}
	return warnings
}

// Team2AoA represents the Articles of Association system for managing agents,
// audits, punishments, and withdrawal orders.
type Team3AoA struct {
	AuditMap         map[uuid.UUID]*Team3AuditQueue // Tracks audit results for each agent
	OffenceMap       map[uuid.UUID]int              // Tracks cumulative score reductions for agents
	LyingHistory     map[uuid.UUID]*Team3AuditQueue // Tracks the history of lying for agents
	PunishmentPeriod int                            // Number of rounds to remember lies (varies by strategy)
}

// CreateTeam3AoA initializes a new instance of Team3AoA with default settings.
func CreateTeam3AoA() *Team3AoA {
	return &Team3AoA{
		AuditMap:         make(map[uuid.UUID]*Team3AuditQueue),
		OffenceMap:       make(map[uuid.UUID]int),
		LyingHistory:     make(map[uuid.UUID]*Team3AuditQueue),
		PunishmentPeriod: 3, // Default to Moderates (remembers lies for 3 rounds)
	}
}

// ResetAuditMap clears all audit data, effectively resetting the state of the audits.
func (t *Team3AoA) ResetAuditMap() {
	t.AuditMap = make(map[uuid.UUID]*Team3AuditQueue)
}

// GetExpectedContribution calculates how much of their score an agent is expected to contribute
// based on the size of the team.
func (t *Team3AoA) GetExpectedContribution(agentId uuid.UUID, agentScore int) int {
	teamSize := len(t.AuditMap)
	var sharePercentage float64

	if teamSize < 5 {
		sharePercentage = 1.0 // Small team: contribute 100%
	} else if teamSize < 10 {
		sharePercentage = 0.75 // Medium team: contribute 75%
	} else {
		sharePercentage = 0.5 // Large team: contribute 50%
	}

	return int(float64(agentScore)*sharePercentage + 0.5) // Round up the result
}

// GetExpectedWithdrawal determines the amount an agent is allowed to withdraw based on their score.
func (t *Team3AoA) GetExpectedWithdrawal(agentId uuid.UUID, agentScore int, commonPool int) int {
	switch {
	case agentScore < 6:
		return 6 // Low score: withdraw 6
	case agentScore <= 12:
		return 2 // Medium score: withdraw 2
	default:
		return 0 // High score: withdraw nothing
	}
}

// GetAuditCost always returns 0, as audits are free in this implementation.
func (t *Team3AoA) GetAuditCost(commonPool int) int {
	return 1
}

// DetermineStrategy calculates the majority vote using Instant Runoff Voting (IRV).
func (t *Team3AoA) DetermineStrategy(votes []Vote) Strategy {
	voteCounts := make(map[Strategy]int)  // Maps strategy to its vote count
	eliminated := make(map[Strategy]bool) // Tracks eliminated strategies

	for {
		// Reset vote counts
		for strategy := range voteCounts {
			voteCounts[strategy] = 0
		}

		// Count first-choice votes
		for _, vote := range votes {
			if vote.IsVote == 1 {
				strategy := Strategy(vote.VotedForID.ID() % 3) // Convert UUID to one of the three strategies
				if !eliminated[strategy] {
					voteCounts[strategy]++
				}
			}
		}

		// Check for majority
		totalVotes := 0
		for _, count := range voteCounts {
			totalVotes += count
		}

		for strategy, count := range voteCounts {
			if count > totalVotes/2 {
				// Return the strategy with the majority
				if strategy == Resolutes {
					t.PunishmentPeriod = 7 // Resolutes: Remember lies for 7 rounds
				} else if strategy == Moderates {
					t.PunishmentPeriod = 3 // Moderates: Remember lies for 3 rounds
				} else {
					t.PunishmentPeriod = 0 // Lenient: Remember nothing
				}
				return strategy
			}
		}

		// Find the strategy with the fewest votes
		minVotes := totalVotes
		toEliminate := Strategy(-1)
		for strategy, count := range voteCounts {
			if !eliminated[strategy] && count < minVotes {
				minVotes = count
				toEliminate = strategy
			}
		}

		// Eliminate the strategy with the fewest votes
		eliminated[toEliminate] = true

		// If all strategies are eliminated (tie), default to Lenient
		if len(eliminated) == len(voteCounts) {
			t.PunishmentPeriod = 0
			return Lenient
		}
	}
}

// ApplyPunishment applies the calculated punishment to the agent, reducing their score.
func (t *Team3AoA) ApplyPunishment(agentId uuid.UUID, strategy Strategy, liedBy int) {
	punishment := t.CalculatePunishment(agentId, strategy, liedBy)

	if punishment.ScoreReduction > 0 {
		// Reduce the agent's score
		t.OffenceMap[agentId] -= punishment.ScoreReduction
		fmt.Printf("Agent %s's score reduced by %d.\n", agentId, punishment.ScoreReduction)
	} else {
		// Lenient or no reduction applied
		fmt.Printf("Agent %s's score remains unchanged under Lenient strategy.\n", agentId)
	}
}

// Audit checks whether an agent lied in a round and applies the appropriate punishment.
func (t *Team3AoA) Audit(agentId uuid.UUID, actual int, stated int, votes []Vote) {
	strategy := t.DetermineStrategy(votes) // Determine the punishment strategy
	liedBy := stated - actual              // Calculate the amount the agent lied by
	if liedBy > 0 {                        // Apply punishment only if lying occurred
		t.ApplyPunishment(agentId, strategy, liedBy)
	}
}

// GetVoteResult calculates the majority vote and returns the winning agent's UUID.
func (t *Team3AoA) GetVoteResult(votes []Vote) uuid.UUID {
	voteMap := make(map[uuid.UUID]int)
	for _, vote := range votes {
		voteMap[vote.VotedForID]++
		if voteMap[vote.VotedForID] > 4 {
			return vote.VotedForID
		}
	}
	return uuid.Nil
}

// GetWithdrawalOrder sorts agents into non-liars and liars.
// Non-liars are shuffled randomly and appear first; liars appear at the bottom.
func (t *Team3AoA) GetWithdrawalOrder(agentIDs []uuid.UUID) []uuid.UUID {
	liars := []uuid.UUID{}
	nonLiars := []uuid.UUID{}

	// Separate agents into liars and non-liars
	for _, agentID := range agentIDs {
		if t.AuditMap[agentID] != nil && t.AuditMap[agentID].rounds.Len() > 0 {
			lastResult := t.AuditMap[agentID].rounds.Back().Value.(bool)
			if lastResult {
				liars = append(liars, agentID) // Agent lied
			} else {
				nonLiars = append(nonLiars, agentID) // Agent did not lie
			}
		} else {
			// No audit data; assume the agent did not lie
			nonLiars = append(nonLiars, agentID)
		}
	}

	// Shuffle non-liars randomly
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(nonLiars), func(i, j int) {
		nonLiars[i], nonLiars[j] = nonLiars[j], nonLiars[i]
	})

	// Combine non-liars and liars
	return append(nonLiars, liars...)
}

// CalculatePunishment computes the punishment for lying, including score reduction.
func (t *Team3AoA) CalculatePunishment(agentId uuid.UUID, strategy Strategy, liedBy int) Punishment {
	// Get or initialize the lying history for the agent
	lyingQueue := t.LyingHistory[agentId]
	if lyingQueue == nil {
		lyingQueue = NewTeam3AuditQueue(t.PunishmentPeriod)
		t.LyingHistory[agentId] = lyingQueue
	}

	// Lenient strategy: No punishment applied
	if strategy == Lenient {
		lyingQueue.AddToQueue(true)
		return Punishment{
			Strategy:       Lenient,
			ScoreReduction: 0, // No reduction
		}
	}

	// Count the total number of lies (including this round)
	lyingCount := lyingQueue.GetWarnings() + 1

	// Determine the multiplier based on the strategy
	multiplier := 1.0
	if strategy == Moderates {
		multiplier = 0.4 // 40% deduction for Moderates
	}

	// Calculate the score reduction
	scoreReduction := int(float64(lyingCount) * float64(liedBy) * multiplier)

	// Add the current lie to the lying history
	lyingQueue.AddToQueue(true)

	return Punishment{
		Strategy:       strategy,
		ScoreReduction: scoreReduction,
	}
}

func (t *Team3AoA) SetWithdrawalAuditResult(agentId uuid.UUID, agentScore int, agentActualWithdrawal int, agentStatedWithdrawal int, commonPool int) {
	expected := t.GetExpectedWithdrawal(agentId, agentScore, commonPool)
	auditResult := (agentActualWithdrawal > expected) || (agentActualWithdrawal != agentStatedWithdrawal)
	if t.AuditMap[agentId] == nil {
		t.AuditMap[agentId] = NewTeam3AuditQueue(t.PunishmentPeriod)
	}
	t.AuditMap[agentId].AddToQueue(auditResult)
}

func (t *Team3AoA) GetContributionAuditResult(agentId uuid.UUID) bool {
	if queue, exists := t.AuditMap[agentId]; exists && queue.rounds.Len() > 0 {
		return queue.rounds.Back().Value.(bool)
	}
	return false
}

func (t *Team3AoA) GetWithdrawalAuditResult(agentId uuid.UUID) bool {
	if queue, exists := t.AuditMap[agentId]; exists && queue.rounds.Len() > 0 {
		return queue.rounds.Back().Value.(bool)
	}
	return false
}

func (t *Team3AoA) SetContributionAuditResult(agentId uuid.UUID, agentScore int, agentActualContribution int, agentStatedContribution int) {
	auditResult := agentActualContribution != agentStatedContribution
	if t.AuditMap[agentId] == nil {
		t.AuditMap[agentId] = NewTeam3AuditQueue(t.PunishmentPeriod)
	}
	t.AuditMap[agentId].AddToQueue(auditResult)
}

// GetPunishment uses the voted strategy to determine punishment
func (t *Team3AoA) GetPunishment(agentScore int, agentId uuid.UUID) int {
	// Get the current strategy from votes
	votes := make([]Vote, 0) // You'll need to maintain votes somewhere in the struct
	strategy := t.DetermineStrategy(votes)

	// Calculate how much they lied by using their last audit result
	var liedBy int
	if queue, exists := t.AuditMap[agentId]; exists && queue.rounds.Len() > 0 {
		// If they lied in their last audit, calculate punishment
		if queue.rounds.Back().Value.(bool) {
			// You'll need to track the actual vs stated amounts to calculate liedBy
			// For now, using a base value
			liedBy = 5 // Replace with actual calculation of how much they lied by
		}
	}

	// Use existing CalculatePunishment function to determine punishment
	punishment := t.CalculatePunishment(agentId, strategy, liedBy)
	return punishment.ScoreReduction
}

// RunPreIterationAoaLogic collects votes from agents and determines the strategy for the iteration
func (t *Team3AoA) RunPreIterationAoaLogic(team *Team, agentMap map[uuid.UUID]IExtendedAgent, dataRecorder *gameRecorder.ServerDataRecorder) {
	votes := make([]Vote, 0)

	// Collect votes and log each agent's vote
	for _, agentID := range team.Agents {
		if agent, exists := agentMap[agentID]; exists {
			rankedVotes := agent.Team3_GetStrategyVote()

			// Log agent's votes
			log.Printf("Agent %v primary vote: %v, secondary vote: %v",
				agentID,
				getStrategyName(rankedVotes[0]),
				getStrategyName(rankedVotes[1]))

			// Add both votes with different weights
			for rank, strategy := range rankedVotes {
				strategyUUID := uuid.NewSHA1(uuid.Nil, []byte{byte(strategy)})
				vote := Vote{
					IsVote:     2 - rank, // First choice (rank 0) gets weight 2, second choice (rank 1) gets weight 1
					VoterID:    agentID,
					VotedForID: strategyUUID,
				}
				votes = append(votes, vote)
			}
		}
	}

	// Determine and log final strategy
	strategy := t.DetermineStrategy(votes)
	log.Printf("Final strategy chosen: %v with %d total votes", getStrategyName(strategy), len(votes))

	// Set punishment period based on strategy
	switch strategy {
	case Resolutes:
		t.PunishmentPeriod = 7
		log.Printf("Resolute strategy: Remembering lies for 7 rounds")
	case Moderates:
		t.PunishmentPeriod = 3
		log.Printf("Moderate strategy: Remembering lies for 3 rounds")
	case Lenient:
		t.PunishmentPeriod = 0
		log.Printf("Lenient strategy: Not remembering any lies")
	}
}

// Helper function to convert Strategy to string
func getStrategyName(s Strategy) string {
	switch s {
	case Lenient:
		return "Lenient"
	case Moderates:
		return "Moderates"
	case Resolutes:
		return "Resolutes"
	default:
		return "Unknown"
	}
}

func (t *Team3AoA) ResourceAllocation(agentScores map[uuid.UUID]int, remainingResources int) map[uuid.UUID]int {
	return make(map[uuid.UUID]int)
}

func (t *Team3AoA) RunPostContributionAoaLogic(team *Team, agentMap map[uuid.UUID]IExtendedAgent) {
	// Empty implementation as Team3 doesn't need post-contribution logic
}

func (t *Team3AoA) Team4_HandlePunishmentVote(map[uuid.UUID]map[int]int) int {
	// Empty implementation as this is Team4-specific
	return 0
}

func (t *Team3AoA) Team4_RunProposedWithdrawalVote(map[uuid.UUID]int, map[uuid.UUID]map[uuid.UUID]int) {
	// Empty implementation as this is Team4-specific
}

func (t *Team3AoA) Team4_SetRankUp(rankUpVoteMap map[uuid.UUID]map[uuid.UUID]int) {
	// Empty implementation as this is Team4-specific
}
