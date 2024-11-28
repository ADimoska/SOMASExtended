package aoa

import "github.com/google/uuid"

type Vote struct {
	IsVote     int
	VoterID    uuid.UUID
	VotedForID uuid.UUID
}

type IArticlesOfAssociation interface {
	GetExpectedContribution(agentId uuid.UUID, agentScore int) int
	GetExpectedWithdrawal(agentId uuid.UUID, agentScore int) int
	GetAuditCost(commonPool int) int
	GetVoteResult(votes []Vote) uuid.UUID
	GetContributionAuditResult(agentId uuid.UUID) bool
	GetWithdrawalAuditResult(agentId uuid.UUID) bool

	SetContributionAuditResult(agentId uuid.UUID, agentScore int, agentActualContribution int, agentStatedContribution int)
	SetWithdrawalAuditResult(agentId uuid.UUID, agentScore int, agentActualWithdrawal int, agentStatedWithdrawal int)
}

func CreateVote(isVote int, voterId uuid.UUID, votedForId uuid.UUID) Vote {
	return Vote{
		IsVote:     isVote,
		VoterID:    voterId,
		VotedForID: votedForId,
	}
}
