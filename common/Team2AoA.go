package common

import (
	"log"
	"math"
	"math/rand"

	"github.com/google/uuid"
)

// Warning -> Implicit to the AoA, not formalized until a successful audit
// Offence -> Formalized warning, 3 offences result in a kick

// ---------------------------------------- Articles of Association Functionality ----------------------------------------

type Team2AoA struct {
	auditRecord *AuditRecord
	// Used by the server in order to track which agents need to be kicked/fined/rolling privileges revoked
	OffenceMap   map[uuid.UUID]int
	RollsLeftMap map[uuid.UUID]int
	Leader       uuid.UUID
	Team         *Team
}

func (t *Team2AoA) GetExpectedContribution(agentId uuid.UUID, agentScore int) int {
	return agentScore
}

// Probably not very relevant, the punishment is levied based on offences committed and is enforced by the server
func (t *Team2AoA) GetAuditResult(agentId uuid.UUID) bool {
	// Only deduct from the common pool for a successful audit
	warnings := t.auditRecord.GetAllInfractions(agentId)
	offences := t.OffenceMap[agentId]
	offences += warnings

	if offences == 1 {
		t.RollsLeftMap[agentId] = 3
	} else if offences == 2 {
		t.RollsLeftMap[agentId] = 2
	} else if offences >= 3 {
		offences = 3
	}

	t.OffenceMap[agentId] = offences

	// Reset the audit queue after an audit to prevent double counting of offences
	t.auditRecord.ClearAllInfractions(agentId)

	return offences > 0
}

func (t *Team2AoA) GetContributionAuditResult(agentId uuid.UUID) bool {
	return t.GetAuditResult(agentId)
}

func (t *Team2AoA) SetContributionAuditResult(agentId uuid.UUID, agentScore int, agentActualContribution int, agentStatedContribution int) {
	var infraction int
	if agentActualContribution != agentStatedContribution {
		infraction = 1
	} else {
		infraction = 0
	}

	t.auditRecord.AddRecord(agentId, infraction)
}

func (t *Team2AoA) GetWithdrawalAuditResult(agentId uuid.UUID) bool {
	return t.GetAuditResult(agentId)
}

func (t *Team2AoA) GetExpectedWithdrawal(agentId uuid.UUID, agentScore int, commonPool int) int {
	// Get the precomputed withdrawal map
	expectedWithdrawals := t.mapExpectedWithdrawal()
	if amount, exists := expectedWithdrawals[agentId]; exists {
		return amount
	}
	return 0
}

func (t *Team2AoA) mapExpectedWithdrawal() map[uuid.UUID]int {
	team := t.Team
	commonPool := team.GetCommonPool()
	count := len(team.Agents)

	reserved := float64(commonPool) * 0.15 // 15% reserved from the common pool
	availablePool := float64(commonPool) - reserved

	// Calculate the multipliers
	leaderMultiplier := 2.5
	totalMultiplier := leaderMultiplier + (float64(count - 1))
	multForLeader := (availablePool * leaderMultiplier) / totalMultiplier
	multForCitizen := (availablePool) / totalMultiplier

	expectedWithdrawals := make(map[uuid.UUID]int)
	for _, agentId := range team.Agents {
		if agentId == t.Leader {
			expectedWithdrawals[agentId] = int(multForLeader)
		} else {
			expectedWithdrawals[agentId] = int(multForCitizen)
		}
	}
	return expectedWithdrawals
}

func (t *Team2AoA) SetWithdrawalAuditResult(agentId uuid.UUID, agentScore int, agentActualWithdrawal int, agentStatedWithdrawal int, commonPool int) {
	multiplier := 0.10
	if agentId == t.Leader {
		multiplier = 0.25
	}
	const epsilon = 1e-9 // Define a small threshold
	expectedWithdrawal := float64(agentScore) * multiplier
	actualWithdrawal := float64(agentActualWithdrawal)

	// Compare using epsilon to handle floating-point inaccuracies
	infraction := math.Abs(expectedWithdrawal-actualWithdrawal) > epsilon

	if infraction && t.auditRecord.GetLastRecord(agentId) == 0 {
		t.auditRecord.IncrementLastRecord(agentId)
	}
}

func (t *Team2AoA) GetAuditCost(commonPool int) int {
	return t.auditRecord.GetAuditCost()
}

func (t *Team2AoA) GetVoteResult(votes []Vote) uuid.UUID {
	if len(votes) == 0 {
		return uuid.Nil
	}

	voteMap := make(map[uuid.UUID]int)
	duration := 0
	count := len(t.Team.Agents)

	for _, vote := range votes {
		durationVote, agentVotedFor := vote.AuditDuration, vote.VotedForID
		votes := 1
		if vote.VotedForID == t.Leader {
			durationVote = durationVote * 2
			votes = 2
		}
		if vote.IsVote == 1 {
			voteMap[agentVotedFor] += votes
		}
		duration += durationVote
	}

	duration /= len(votes)
	if duration > 0 {
		t.auditRecord.SetAuditDuration(duration)
	}

	// Make it easier to audit a leader, this ensures the leader can't outvote the rest and stay in power
	leaderVotes := voteMap[t.Leader]
	if leaderVotes >= max(1, (count/2)-1) {
		return t.Leader
	}

	// If the leader is not the one votes up, then check if someone else has been voted up
	for votedFor, votes := range voteMap {
		if votes >= ((count / 2) + 1) {
			return votedFor
		}
	}

	return uuid.Nil
}

func (t *Team2AoA) GetWithdrawalOrder(agentIDs []uuid.UUID) []uuid.UUID {
	// Create a copy of agentIDs to avoid modifying the original slice
	shuffledAgents := make([]uuid.UUID, len(agentIDs))
	copy(shuffledAgents, agentIDs)

	// Shuffle the agent list
	rand.Shuffle(len(shuffledAgents), func(i, j int) {
		shuffledAgents[i], shuffledAgents[j] = shuffledAgents[j], shuffledAgents[i]
	})

	withdrawalOrder := make([]uuid.UUID, 0, len(agentIDs))

	// Add the leader ID to the first position
	withdrawalOrder = append(withdrawalOrder, t.Leader)

	// Append all other IDs, excluding the leader
	for _, agentID := range shuffledAgents {
		if agentID != t.Leader {
			withdrawalOrder = append(withdrawalOrder, agentID)
		}
	}

	return withdrawalOrder
}

func (t *Team2AoA) RunPreIterationAoaLogic(team *Team, agentMap map[uuid.UUID]IExtendedAgent)     {}
func (t *Team2AoA) RunPostContributionAoaLogic(team *Team, agentMap map[uuid.UUID]IExtendedAgent) {}

func (f *Team2AoA) ResourceAllocation(agentScores map[uuid.UUID]int, remainingResources int) map[uuid.UUID]int {
	return make(map[uuid.UUID]int)
}

func (t *Team2AoA) SetLeader(leader uuid.UUID) {
	t.Leader = leader
}

func (t *Team2AoA) GetLeader() uuid.UUID {
	return t.Leader
}

func (t *Team2AoA) GetOffences(agentId uuid.UUID) int {
	return t.OffenceMap[agentId]
}

func (t *Team2AoA) GetRollsLeft(agentId uuid.UUID) int {
	return t.RollsLeftMap[agentId]
}

func (t *Team2AoA) RollOnce(agentId uuid.UUID) {
	t.RollsLeftMap[agentId] = max(0, t.RollsLeftMap[agentId]-1)
}

func (t *Team2AoA) GetPunishment(agentScore int, agentId uuid.UUID) int {
	multiplier := 50
	if t.OffenceMap[agentId] == 2 {
		multiplier = 100
	}
	return (agentScore * multiplier) / 100
}

func CreateTeam2AoA(team *Team, leader uuid.UUID, auditDuration int) IArticlesOfAssociation {
	log.Println("Creating Team2AoA")
	offenceMap := make(map[uuid.UUID]int)
	rollsLeftMap := make(map[uuid.UUID]int)

	if leader == uuid.Nil {
		shuffledAgents := make([]uuid.UUID, len(team.Agents))
		copy(shuffledAgents, team.Agents)
		rand.Shuffle(len(shuffledAgents), func(i, j int) {
			shuffledAgents[i], shuffledAgents[j] = shuffledAgents[j], shuffledAgents[i]
		})
		leader = shuffledAgents[0]
	}

	return &Team2AoA{
		auditRecord:  NewAuditRecord(auditDuration),
		OffenceMap:   offenceMap,
		RollsLeftMap: rollsLeftMap,
		Leader:       leader,
		Team:         team,
	}
}

// Do nothing
func (t *Team2AoA) Team4_SetRankUp(rankUpVoteMap map[uuid.UUID]map[uuid.UUID]int) {
}
func (t *Team2AoA) Team4_RunProposedWithdrawalVote(map[uuid.UUID]int, map[uuid.UUID]map[uuid.UUID]int) {
}
func (t *Team2AoA) Team4_HandlePunishmentVote(map[uuid.UUID]map[int]int) int {
	return 0
}
