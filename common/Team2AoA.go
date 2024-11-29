package common

// import "github.com/google/uuid"
import (
	"container/list"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type AuditQueue struct {
	length int
	rounds list.List
}

func NewAuditQueue(length int) *AuditQueue {
	return &AuditQueue{
		length: length,
		rounds: list.List{},
	}
}

func (aq *AuditQueue) AddToQueue(auditResult bool) {
	if aq.length == aq.rounds.Len() {
		aq.rounds.Remove(aq.rounds.Front())
	}
	aq.rounds.PushBack(auditResult)
}

func (aq *AuditQueue) GetWarnings() int {
	warnings := 0
	for e := aq.rounds.Front(); e != nil; e = e.Next() {
		warnings += e.Value.(int)
	}
	return warnings
}

type Team2AoA struct {
	AuditMap   map[uuid.UUID]*AuditQueue
	OffenceMap map[uuid.UUID]int
	Leader     uuid.UUID
}

func (t *Team2AoA) ResetAuditMap() {
	t.AuditMap = make(map[uuid.UUID]*AuditQueue)
}

func (t *Team2AoA) GetExpectedContribution(agentId uuid.UUID, agentScore int) int {
	return agentScore
}

// TODO: Team2 to implement the actual functionality
func (t *Team2AoA) GetContributionAuditResult(agentId uuid.UUID) bool {
	return false
}

func (t *Team2AoA) SetContributionAuditResult(agentId uuid.UUID, agentScore int, agentActualContribution int, agentStatedContribution int) {
	// ignore agentStatedContribution
	// check if agent actually contributed it's entire score
	if t.AuditMap[agentId] == nil {
		t.AuditMap[agentId] = NewAuditQueue(5)
	}
	t.AuditMap[agentId].AddToQueue(agentActualContribution != agentScore)
}

func (t *Team2AoA) GetWithdrawalAuditResult(agentId uuid.UUID) bool {
	return false
}

func (t *Team2AoA) GetExpectedWithdrawal(agentId uuid.UUID, agentScore int, commonPool int) int {
	if agentId == t.Leader {
		return int(float64(agentScore) * 0.25)
	}
	return int(float64(agentScore) * 0.10)
}

func (t *Team2AoA) SetWithdrawalAuditResult(agentId uuid.UUID, agentScore int, agentActualWithdrawal int, agentStatedWithdrawal int, commonPool int) {
	if t.AuditMap[agentId] == nil {
		t.AuditMap[agentId] = NewAuditQueue(5)
	}
	if agentId == t.Leader {
		t.AuditMap[agentId].AddToQueue(float64(agentScore)*0.25 != float64(agentActualWithdrawal))
	} else {
		t.AuditMap[agentId].AddToQueue(float64(agentScore)*0.10 != float64(agentActualWithdrawal))
	}
}

func (t *Team2AoA) GetAuditCost(commonPool int) int {
	if commonPool < 5 {
		return 2
	}
	return 5 + ((commonPool - 5) / 5)
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
		if voteMap[vote.VotedForID] > 4 {
			return vote.VotedForID
		}
	}
	return uuid.Nil // Explicitly return uuid.Nil for "no result"
}

func (t *Team2AoA) GetWithdrawalOrder(agentIDs []uuid.UUID) []uuid.UUID {
	// Seed the random number generator to ensure different shuffles each time
	rand.NewSource(time.Now().UnixNano())

	// Create a copy of the agentIDs to avoid modifying the original list
	shuffledAgents := make([]uuid.UUID, len(agentIDs))
	copy(shuffledAgents, agentIDs)

	// Shuffle the agent list
	rand.Shuffle(len(shuffledAgents), func(i, j int) {
		shuffledAgents[i], shuffledAgents[j] = shuffledAgents[j], shuffledAgents[i]
	})

	return shuffledAgents
}

func (t *Team2AoA) RunAoAStuff() {}

func CreateTeam2AoA(auditDuration int) IArticlesOfAssociation {
	return &Team2AoA{
		AuditMap:   make(map[uuid.UUID]*AuditQueue),
		OffenceMap: make(map[uuid.UUID]int),
	}
}
