package common

import (
	"container/list"
	"fmt"

	"github.com/google/uuid"
)

// Strategy defines the punishment severity: Lenient, Moderates, or Resolutes.
type Strategy int

const (
	Lenient   Strategy = iota // 0: No punishment
	Moderates                 // 1: Less severe punishment
	Resolutes                 // 2: Harsher punishment
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
