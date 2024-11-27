package aoa

import (
	"container/list"
	"github.com/google/uuid"
)


type Team1AoA struct {
	auditResult map[uuid.UUID]*list.List
	ranking map[uuid.UUID]int
	withdrawalPerRank map[int]int
	commonPool int
}

func (t *Team1AoA) ResetAuditMap() {
	t.auditResult = make(map[uuid.UUID]*list.List)
}

func (t *Team1AoA) GetExpectedContribution(agentId uuid.UUID, agentScore int) int {
	return 0;  // Did we confirm 0?
}

func (t *Team1AoA) SetContributionAuditResult(agentId uuid.UUID, agentScore int, agentActualContribution int, agentStatedContribution int) {
	t.auditResult[agentId].PushBack(agentStatedContribution > agentActualContribution)
}

func (t *Team1AoA) GetExpectedWithdrawal(agentId uuid.UUID, agentScore int) int {
	k := t.ranking[agentId]
	total := t.getTotalInRank(k)
	percentage := t.withdrawalPerRank[k]
	expectedWithdrawal := (t.commonPool * percentage)/total
	return expectedWithdrawal;
}

func (t *Team1AoA) SetWithdrawalAuditResult(agentId uuid.UUID, agentScore int, agentActualWithdrawal int, agentStatedWithdrawal int) {
	t.auditResult[agentId].PushBack((agentActualWithdrawal > agentStatedWithdrawal) || (agentActualWithdrawal > t.GetExpectedWithdrawal(agentId, agentScore)));
}

func (t *Team1AoA) GetAuditCost(commonPool int) int {
	return 5 // Constant cost?
}

func (t *Team1AoA) GetVoteResult(votes []Vote) *uuid.UUID {
	//ToDo
	// Couldnt find how vote works
	return &uuid.Nil
}

func (t *Team1AoA) getTotalInRank(k int) int{
	total := 0
	for _, rank := range t.ranking {
		if rank == k {
			total++;
		}
	}
	return total
}

func CreateTeam1AoA() IArticlesOfAssociation {
	withdrawalPerRank := make(map[int]int)
	withdrawalPerRank[0] = 5
	withdrawalPerRank[1] = 5
	withdrawalPerRank[2] = 10
	withdrawalPerRank[3] = 20
	withdrawalPerRank[4] = 40

	return &Team1AoA{
		auditResult: make(map[uuid.UUID]*list.List),
		ranking: make(map[uuid.UUID]int),
		withdrawalPerRank: withdrawalPerRank,
		commonPool: 0,	// how do we update common pool?
	}
}
