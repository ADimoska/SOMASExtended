package common

import (
	"container/list"
	"github.com/google/uuid"
)

type Team1AoA struct {
	auditResult       map[uuid.UUID]*list.List
	ranking           map[uuid.UUID]int
	withdrawalPerRank map[int]int
}

func (t *Team1AoA) ResetAuditMap() {
	t.auditResult = make(map[uuid.UUID]*list.List)
}

func (t *Team1AoA) GetExpectedContribution(agentId uuid.UUID, agentScore int) int {
	return 0
}

func (t *Team1AoA) SetContributionAuditResult(agentId uuid.UUID, agentScore int, agentActualContribution int, agentStatedContribution int) {
	t.auditResult[agentId].PushBack(agentStatedContribution > agentActualContribution)

	// Just for our AoA we are updating rank based on stated
	t.ranking[agentId] += (agentStatedContribution / 5) // Plus 1 rank every 5 points?

	// Cap agent rank at 4
	if t.ranking[agentId] > 4 {
		t.ranking[agentId] = 4
	}

}

func (t *Team1AoA) GetExpectedWithdrawal(agentId uuid.UUID, agentScore int, commonPool int) int {
	k := t.ranking[agentId]
	totalInRank := t.getTotalInRank(k)
	percentage := t.withdrawalPerRank[k]
	expectedWithdrawal := (commonPool * (percentage)) / (totalInRank * 100)
	return expectedWithdrawal
}

func (t *Team1AoA) SetWithdrawalAuditResult(agentId uuid.UUID, agentScore int, agentActualWithdrawal int, agentStatedWithdrawal int, commonPool int) {
	t.auditResult[agentId].PushBack((agentActualWithdrawal > agentStatedWithdrawal) || (agentActualWithdrawal > t.GetExpectedWithdrawal(agentId, agentScore, commonPool)))
}

func (t *Team1AoA) GetAuditCost(commonPool int) int {
	return 5
}

func (t *Team1AoA) GetVoteResult(votes []Vote) uuid.UUID {
	// Count total votes
	totalVotes := 0
	voteMap := make(map[uuid.UUID]int)
	highestVotes := -1
	highestVotedID := uuid.Nil
	for _, vote := range votes {
		totalVotes += vote.IsVote
		if vote.IsVote == 1 { // Should agents who didnt want to vote, get a vote if majority wants to?
			voteMap[vote.VotedForID]++
		}
		// Check if this ID has the highest votes
		if voteMap[vote.VotedForID] > highestVotes {
			highestVotedID = vote.VotedForID
			highestVotes = voteMap[vote.VotedForID]
		}
	}
	if totalVotes <= 0 {
		return uuid.Nil // Majority does not want to vote
	}
	return highestVotedID
}

func (t *Team1AoA) getTotalInRank(k int) int {
	total := 0
	for _, rank := range t.ranking {
		if rank == k {
			total++
		}
	}
	return total
}

func (t *Team1AoA) GetContributionAuditResult(agentId uuid.UUID) bool {
	return t.auditResult[agentId].Back().Value.(int) == 1
}

func (t *Team1AoA) GetWithdrawalAuditResult(agentId uuid.UUID) bool {
	return t.auditResult[agentId].Back().Value.(int) == 1
}

func (t *Team1AoA) GetWithdrawalOrder(agentIDs []uuid.UUID) []uuid.UUID {
	return agentIDs
}

func CreateTeam1AoA(team *Team) IArticlesOfAssociation {
	withdrawalPerRank := make(map[int]int)
	withdrawalPerRank[0] = 5
	withdrawalPerRank[1] = 5
	withdrawalPerRank[2] = 10
	withdrawalPerRank[3] = 20
	withdrawalPerRank[4] = 40

	auditResult := make(map[uuid.UUID]*list.List)
	ranking := make(map[uuid.UUID]int)
	for _, agent := range team.Agents {
		auditResult[agent] = list.New()
		ranking[agent] = 0
	}

	return &Team1AoA{
		auditResult:       auditResult,
		ranking:           ranking,
		withdrawalPerRank: withdrawalPerRank,
	}
}
