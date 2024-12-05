package common

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distuv"
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
	agentsToMonitor         map[uuid.UUID]int64             // Monitoring tracking

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

// Mointoring: 3 stages
// 0 -> not being monitored
// 1 -> monitoring stage 1
// 2 -> stage 2
// 3 -> stage 3
func (t *Team6AoA) RunContributionMonitoring() {
	// for all agent in monitoring system

	// for now, we're saying monitoring is free
	// auditCost = t.GetAuditCost()
	// monitCosts =

	for monitAgent, monitStage := range t.agentsToMonitor {

		source := rand.NewSource(uint64(time.Now().UnixNano()))

		// Check if agent has any audit history
		if monitHistory, monitExists := t.auditHistory[monitAgent]; monitExists && len(monitHistory) > 0 {

			lastMonitRecord := monitHistory[len(monitHistory)-1]

			if lastMonitRecord == nil {
				// if agent being monitored was good last turn, move them down a stage
				t.agentsToMonitor[monitAgent] -= 1
			} else {
				if monitStage == 1 {
					// stage 1, half of actual
					lambda := float64(lastMonitRecord.Actual / 2)
					poisson := distuv.Poisson{
						Lambda: lambda,           // The rate parameter
						Src:    rand.New(source), // Random source
					}
					monitCheck := poisson.Rand()

					if monitCheck <= float64(lastMonitRecord.Expected) {
						// agent gets away with it
						t.agentsToMonitor[monitAgent] -= 1
					} else {
						// agent gets caught
						t.agentsToMonitor[monitAgent] += 1
					}

				} else if monitStage == 2 {
					// stage 2, 3/4 of actual
					lambda := float64(3 * lastMonitRecord.Actual / 4)
					poisson := distuv.Poisson{
						Lambda: lambda,           // The rate parameter
						Src:    rand.New(source), // Random source
					}
					monitCheck := poisson.Rand()

					if monitCheck <= float64(lastMonitRecord.Expected) {
						// agent gets away with it
						t.agentsToMonitor[monitAgent] -= 1
					} else {
						// agent gets caught
						t.agentsToMonitor[monitAgent] += 1
					}

				} else if monitStage == 3 {
					// stage 3, full actual
					lambda := float64(lastMonitRecord.Actual)
					poisson := distuv.Poisson{
						Lambda: lambda,           // The rate parameter
						Src:    rand.New(source), // Random source
					}
					monitCheck := poisson.Rand()

					if monitCheck <= float64(lastMonitRecord.Expected) {
						// agent gets away with it
						t.agentsToMonitor[monitAgent] -= 1
					} else {
						// agent gets caught
						t.agentsToMonitor[monitAgent] += 1
					}
				}
			}

			if monitStage < 0 {
				// monit stage can't be negative
				t.agentsToMonitor[monitAgent] = 0
			} else if monitStage > 3 {
				// agent has passed stage 3 of monitoring, must get kicked out
			}
		}

		if monitStage == 0 {
			// if the monit stage of this agent in system drops from 1 -> 0, free it from monitoring
			delete(t.agentsToMonitor, monitAgent)
		}

	}
}

// Will just check if the agent cheated in their last contribution
// TODO: Implement probability based detection:
// - Baseline detection: 50% + (1-50% based on how much the agent cheated)
// - So that the probability of detection is based on how much the agent cheated
func (t *Team6AoA) GetContributionAuditResult(agentId uuid.UUID) bool {

	// first, run monitoring for all agents being monitored!
	t.RunContributionMonitoring()

	// Check if agent has any audit history
	if history, exists := t.auditHistory[agentId]; exists && len(history) > 0 {
		// Get the most recent audit record
		lastRecord := history[len(history)-1]
		// If lastRecord is not nil, it means the agent cheated in their last contribution

		cheatCheck := lastRecord != nil
		_, inMonitMap := t.agentsToMonitor[agentId]

		if inMonitMap {
			// if this agent is already in monitoring map, do nothing
			// if it cheated, that'll already have been sorted by the RunMonitoring Function
		} else if cheatCheck == true {
			// this agent cheated, but isn't in monitoring map
			// we must add it in at stage 1!
			t.agentsToMonitor[agentId] = 1
		}

		return cheatCheck
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

// Punishment fn! (finally)
func (t *Team6AoA) GetPunishment(agentScore int, agentId uuid.UUID) int {
	return (agentScore * 25) / 100
}

// not needed, dw abt it, here to fix error complaints
func (f *Team6AoA) ResourceAllocation(agentScores map[uuid.UUID]int, remainingResources int) map[uuid.UUID]int {
	return make(map[uuid.UUID]int)
}

// not needed, dw abt it, here to fix error complaints
func (t *Team6AoA) RunPreIterationAoaLogic(team *Team, agentMap map[uuid.UUID]IExtendedAgent) {}

// not needed, dw abt it, here to fix error complaints
func (t *Team6AoA) Team4_HandlePunishmentVote(map[uuid.UUID]map[int]int) int {
	return 0
}

// not needed, dw abt it, here to fix error complaints
func (t *Team6AoA) Team4_RunProposedWithdrawalVote(map[uuid.UUID]int, map[uuid.UUID]map[uuid.UUID]int) {
}

// not needed, dw abt it, here to fix error complaints
func (t *Team6AoA) Team4_SetRankUp(rankUpVoteMap map[uuid.UUID]map[uuid.UUID]int) {
}

func createTeam6AoA() IArticlesOfAssociation {
	return &Team6AoA{
		weight: float64, // Weight for current turn contributions
		decay:  float64, // Decay rate for cumulative contributions

		cumulativeContributions: make(map[uuid.UUID]float64),
		currentContributions:    make(map[uuid.UUID]float64),           // Current turn contributions
		auditHistory:            make(map[uuid.UUID][]*CheatingRecord), // Audit history per agent
		agentsToMonitor:         make(map[uuid.UUID]int64),
	}
}
