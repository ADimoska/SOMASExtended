package aoa

// import "github.com/google/uuid"
import (
	"container/list"

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
	AuditMap map[uuid.UUID]*AuditQueue
	OffenceMap map[uuid.UUID]int
	Leader uuid.UUID
}

func (t *Team2AoA) ResetAuditMap() {
	t.AuditMap = make(map[uuid.UUID]*AuditQueue)
}

func (t *Team2AoA) SetContributionResult(agentId uuid.UUID, agentScore int, agentContribution int) {
	t.AuditMap[agentId].AddToQueue(agentContribution != agentScore)
}

func (t *Team2AoA) SetWithdrawalResult(agentId uuid.UUID, agentScore int, agentWithdrawal int) {
	if agentId == t.Leader {
		t.AuditMap[agentId].AddToQueue(float64(agentScore) * 0.25 != float64(agentWithdrawal))
	} else {
		t.AuditMap[agentId].AddToQueue(float64(agentScore) * 0.10 != float64(agentWithdrawal))
	}
}

func (t *Team2AoA) GetAuditCost(commonPool int) int {
	if commonPool < 5 {
		return 2
	}
	return 5 + ((commonPool - 5)/5)
}

func (t *Team2AoA) GetVoteResult(votes []Vote) *uuid.UUID {
	voteMap := make(map[uuid.UUID]int)
	for _, vote := range votes {
		if vote.IsVote {
			if vote.VoterID == t.Leader {
				voteMap[vote.VotedForID] += 2
			} else {
				voteMap[vote.VotedForID]++
			}
		}
		if voteMap[vote.VotedForID] > 4 {
			return &vote.VotedForID
		}
	}
	return &uuid.Nil
}


