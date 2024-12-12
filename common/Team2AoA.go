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
type AuditRecord struct {
	auditMap map[uuid.UUID][]int
	duration int
	cost     int
	// reliability float64
}

func NewAuditRecord(duration int) *AuditRecord {
	cost := calculateCost(duration)

	return &AuditRecord{
		auditMap: make(map[uuid.UUID][]int),
		duration: duration,
		cost:     cost,
	}
}

// Getters
func (a *AuditRecord) GetAuditMap() map[uuid.UUID][]int {
	return a.auditMap
}

func (a *AuditRecord) GetAuditDuration() int {
	return a.duration
}

func (a *AuditRecord) GetAuditCost() int {
	return a.cost
}

// Setters
func (a *AuditRecord) SetAuditDuration(duration int) {
	a.duration, a.cost = duration, calculateCost(duration)
}

// Implement a more sophisticated cost calculation if needed, could compound with reliability if implemented
func calculateCost(duration int) int {
	return duration
}

// Get the number of infractions in the last n rounds, given by the quality of the audit
func (a *AuditRecord) GetAllInfractions(agentId uuid.UUID) int {
	infractions := 0
	records := a.auditMap[agentId]

	history := min(a.duration, len(records))

	for _, infraction := range records[len(records)-history:] {
		infractions += infraction
	}

	return infractions
}

/**
* Clear all infractions for a given agent
* This may/may not be called in case the audit system is converted into a probability-based hybrid.
* In such a case, the infractions may need to be kept in case there is an unsuccessful audit.
 */
func (a *AuditRecord) ClearAllInfractions(agentId uuid.UUID) {
	a.auditMap[agentId] = []int{}
}

// After an agent's contribution, add a new record to the audit map - infraction could be 1 or 0 instead of bool
func (a *AuditRecord) AddRecord(agentId uuid.UUID, infraction int) {
	if _, ok := a.auditMap[agentId]; !ok {
		a.auditMap[agentId] = []int{}
	}

	a.auditMap[agentId] = append(a.auditMap[agentId], infraction)
}

// In case this is needed by individual AoAs
func (a *AuditRecord) GetLastRecord(agentId uuid.UUID) int {
	if _, ok := a.auditMap[agentId]; !ok {
		return 0
	}

	records := a.auditMap[agentId]

	if len(records) == 0 {
		return 0
	}

	return records[len(records)-1]
}

// After the agent's withdrawal, which is after the contribution, update the last record instead of adding a new one
func (a *AuditRecord) IncrementLastRecord(agentId uuid.UUID) {
	if _, ok := a.auditMap[agentId]; !ok {
		a.auditMap[agentId] = []int{}
	}

	records := a.auditMap[agentId]
	if len(records) == 0 {
		return
	}

	records[len(records)-1]++
}

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
	// TODO: If probabilistic auditing is implemented, this should be removed
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

// TODO: Implement a borda vote here instead?
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
