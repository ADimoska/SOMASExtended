package common

import (
	"container/list"
	"github.com/google/uuid"
	"sort"
)

type Team1AoA struct {
	auditResult map[uuid.UUID]*list.List
	ranking     map[uuid.UUID]int
	threshold   int
}

func (t *Team1AoA) ResetAuditMap() {
	t.auditResult = make(map[uuid.UUID]*list.List)
}

func (t *Team1AoA) GetExpectedContribution(agentId uuid.UUID, agentScore int) int {
	return t.threshold // For now using threshold as minimum for all ranks, later have per rank minimums? But need to vote what is min?
}

func (t *Team1AoA) SetContributionAuditResult(agentId uuid.UUID, agentScore int, agentActualContribution int, agentStatedContribution int) {
	t.auditResult[agentId].PushBack((agentStatedContribution > agentActualContribution))

	// Just for our AoA we are updating rank based on stated
	t.ranking[agentId] += (agentStatedContribution / t.threshold) // Plus 1 rank every `threshold` points?

	// Cap agent rank at 5
	if t.ranking[agentId] > 5 {
		t.ranking[agentId] = 5
	}
	// Lower rank if insufficient contribution
	if agentActualContribution < t.threshold { // For now using threshold as minimum for all ranks, later have per rank minimums? But need to vote what is min?
		t.ranking[agentId] -= 1
	}
	if t.ranking[agentId] < 1 { // Ensure cant go below 1
		t.ranking[agentId] = 1
	}

}

func (t *Team1AoA) GetExpectedWithdrawal(agentId uuid.UUID, agentScore int, commonPool int) int {
	totalWeightedSum := 0
	for _, rank := range t.ranking {
		totalWeightedSum += rank
	}
	expectedWithdrawal := t.ranking[agentId] * (commonPool / (totalWeightedSum + 5)) // Weight 5 for pool?
	return expectedWithdrawal
}

func (t *Team1AoA) SetWithdrawalAuditResult(agentId uuid.UUID, agentScore int, agentActualWithdrawal int, agentStatedWithdrawal int, commonPool int) {
	t.auditResult[agentId].PushBack((agentActualWithdrawal > agentStatedWithdrawal) || (agentActualWithdrawal > t.GetExpectedWithdrawal(agentId, agentScore, commonPool)))
}

func (t *Team1AoA) GetAuditCost(commonPool int) int {
	// Need to get argument which agent being audited and then change cost?
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

func (t *Team1AoA) GetContributionAuditResult(agentId uuid.UUID) bool {
	return t.auditResult[agentId].Back().Value.(int) == 1
}

func (t *Team1AoA) GetWithdrawalAuditResult(agentId uuid.UUID) bool {
	return t.auditResult[agentId].Back().Value.(int) == 1
}

func (t *Team1AoA) GetWithdrawalOrder(agentIDs []uuid.UUID) []uuid.UUID {
	// Sort the agent based on their rank value in descending order
	sort.Slice(agentIDs, func(i, j int) bool {
		return t.ranking[agentIDs[i]] > t.ranking[agentIDs[j]]
	})
	return agentIDs
}

func CreateTeam1AoA(team *Team) IArticlesOfAssociation {
	auditResult := make(map[uuid.UUID]*list.List)
	ranking := make(map[uuid.UUID]int)
	for _, agent := range team.Agents {
		auditResult[agent] = list.New()
		ranking[agent] = 1
	}

	return &Team1AoA{
		auditResult: auditResult,
		ranking:     ranking,
		threshold:   5,
	}
}
