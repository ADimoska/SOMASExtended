package common

// import "github.com/google/uuid"
import (
	"container/list"
	"math"
	"math/rand"

	"github.com/google/uuid"
)

// Warning -> Implicit to the AoA, not formalized until a successful audit
// Offence -> Formalized warning, 3 offences result in a kick
// Need to formalize the first offence punishment -> Server needs to enforce this.

// ---------------------------------------- Audit Queue Functionality ----------------------------------------

type IAuditQueue interface {
	AddToQueue(auditResult bool)
	GetLength() int
	SetLength(length int)
	GetWarnings() int
	GetLastAuditWarning() bool
	SetLastAuditWarning(value bool)
	Reset()
}

type AuditQueue struct {
	length int
	rounds list.List
}

func (aq *AuditQueue) AddToQueue(auditResult bool) {
	if aq.length == aq.rounds.Len() {
		aq.rounds.Remove(aq.rounds.Front())
	}
	aq.rounds.PushBack(auditResult)
}

func (aq *AuditQueue) GetLength() int {
	return aq.length
}

func (aq *AuditQueue) SetLength(length int) {
	aq.length = length
}

func (aq *AuditQueue) GetWarnings() int {
	warnings := 0
	for e := aq.rounds.Front(); e != nil; e = e.Next() {
		// Avoid run-time panic (which would happen if this is enforced to be an int)
		if e.Value.(bool) {
			warnings++
		}
	}
	return warnings
}

func (aq *AuditQueue) GetLastAuditWarning() bool {
	back := aq.rounds.Back()
	if back != nil {
		return back.Value.(bool)
	}
	return false
}

func (aq *AuditQueue) SetLastAuditWarning(value bool) {
	back := aq.rounds.Back()
	if back != nil {
		back.Value = value
	}
}

func (aq *AuditQueue) Reset() {
	aq.rounds.Init()
}

func NewAuditQueue(length int) *AuditQueue {
	return &AuditQueue{
		length: length,
		rounds: list.List{},
	}
}

// ---------------------------------------- Articles of Association Functionality ----------------------------------------

type Team2AoA struct {
	AuditMap map[uuid.UUID]*AuditQueue
	// Used by the server in order to track which agents need to be kicked/fined/rolling privileges revoked
	OffenceMap map[uuid.UUID]int
	Leader     uuid.UUID
}

func (t *Team2AoA) ResetAuditMap() {
	t.AuditMap = make(map[uuid.UUID]*AuditQueue)
}

func (t *Team2AoA) GetExpectedContribution(agentId uuid.UUID, agentScore int) int {
	return agentScore
}

// Probably not very relevant, the punishment is levied based on offences committed and is enforced by the server
func (t *Team2AoA) GetAuditResult(agentId uuid.UUID) bool {
	// Only deduct from the common pool for a successful audit
	warnings := t.AuditMap[agentId].GetWarnings()
	offences := t.OffenceMap[agentId]
	offences += warnings

	if offences > 3 {
		offences = 3
	}

	t.OffenceMap[agentId] = offences

	// Reset the audit queue after an audit to prevent double counting of offences
	t.AuditMap[agentId].Reset()

	return offences > 0
}

func (t *Team2AoA) GetContributionAuditResult(agentId uuid.UUID) bool {
	return t.GetAuditResult(agentId)
}

func (t *Team2AoA) SetContributionAuditResult(agentId uuid.UUID, agentScore int, agentActualContribution int, agentStatedContribution int) {
	t.AuditMap[agentId].AddToQueue(agentActualContribution != agentScore)
}

func (t *Team2AoA) GetWithdrawalAuditResult(agentId uuid.UUID) bool {
	return t.GetAuditResult(agentId)
}

func (t *Team2AoA) GetExpectedWithdrawal(agentId uuid.UUID, agentScore int, commonPool int) int {
	if agentId == t.Leader {
		return int(float64(commonPool) * 0.25)
	}
	return int(float64(commonPool) * 0.20)
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
	auditResult := math.Abs(expectedWithdrawal-actualWithdrawal) > epsilon || t.AuditMap[agentId].GetLastAuditWarning()

	t.AuditMap[agentId].SetLastAuditWarning(auditResult)
}

func (t *Team2AoA) GetAuditCost(commonPool int) int {
	if commonPool < 5 {
		return 2
	}
	return 2 + (commonPool / 5)
}

func (t *Team2AoA) GetVoteResult(votes []Vote) uuid.UUID {
	voteMap := make(map[uuid.UUID]int)
	for _, vote := range votes {
		if vote.IsVote == 1 {
			if vote.VoterID == t.Leader {
				voteMap[vote.VotedForID] += 2
			} else {
				voteMap[vote.VotedForID]++
			}
		}
		// TODO: 4 is the fixed threshold of votes, this may change depending on team size
		if voteMap[vote.VotedForID] > 4 {
			return vote.VotedForID
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

// To be ran every round, so that the server has enough information to enforce the AoA
func (t *Team2AoA) RunAoAStuff() {
	for agentId, offences := range t.OffenceMap {
		currentAuditDuration := t.AuditMap[agentId].GetLength()
		if offences == 1 {
			if currentAuditDuration < 6 {
				t.AuditMap[agentId].SetLength(6) // After one warning, increase the duration of the audit memory to 6 audits
			}
		} else if offences == 2 {
			if currentAuditDuration < 8 {
				t.AuditMap[agentId].SetLength(8) // After two warnings, increase the duration of the audit memory to 8 audits
			}
		}
	}
}

func (t *Team2AoA) SetLeader(leader uuid.UUID) {
	t.Leader = leader
}

func (t *Team2AoA) GetLeader() uuid.UUID {
	return t.Leader
}

// After the AoA stuff has been run, the server can use this to determine what punishment to impose
func (t *Team2AoA) GetOffenders(numOffences int) []uuid.UUID {
	offenders := make([]uuid.UUID, 0)
	for agentId, offences := range t.OffenceMap {
		if offences == numOffences {
			offenders = append(offenders, agentId)
		}
	}
	return offenders
}

func CreateTeam2AoA(team *Team, leader uuid.UUID) IArticlesOfAssociation {
	auditMap := make(map[uuid.UUID]*AuditQueue)
	offenceMap := make(map[uuid.UUID]int)

	if leader == uuid.Nil {
		shuffledAgents := make([]uuid.UUID, len(team.Agents))
		copy(shuffledAgents, team.Agents)
		rand.Shuffle(len(shuffledAgents), func(i, j int) {
			shuffledAgents[i], shuffledAgents[j] = shuffledAgents[j], shuffledAgents[i]
		})
		leader = shuffledAgents[0]
	}

	for _, memberId := range team.Agents {
		auditMap[memberId] = NewAuditQueue(4)
		offenceMap[memberId] = 0
	}

	return &Team2AoA{
		AuditMap:   auditMap,
		OffenceMap: offenceMap,
		Leader:     leader,
	}
}
