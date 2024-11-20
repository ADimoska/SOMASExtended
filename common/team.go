package common

import "github.com/google/uuid"

type Team struct {
	TeamID     uuid.UUID
	CommonPool int
	Agents     []uuid.UUID
	AuditResult map[uuid.UUID]bool // Default is false, which means if false then there is no deferral
	ArticlesOfAssociation *ArticlesOfAssociation
}

// constructor: NewTeam creates a new Team with a unique TeamID and initializes other fields as blank.
func NewTeam() Team {
	aoa := CreateArticlesOfAssociation(CreateFixedContributionRule(10), CreateFixedWithdrawalRule(10), CreateFixedAuditCost(10), CreateFixedPunishment(10))
	return Team{
		TeamID:     uuid.New(),             // Generate a unique TeamID
		CommonPool: 0,                      // Initialize commonPool to 0
		Agents:     []uuid.UUID{},          // Initialize an empty slice of agent UUIDs
		AuditResult: map[uuid.UUID]bool{},  // Initialize an empty map of agentID -> bool
		ArticlesOfAssociation: aoa,         // Initialize strategy as 0
	}
}

func (team *Team) SetContributionResult(agentID uuid.UUID, agentScore int, agentActualContribution int) {
	agentExpectedContribution := team.ArticlesOfAssociation.contributionRule.GetExpectedContributionAmount(agentScore)
	if agentActualContribution != agentExpectedContribution {
		team.AuditResult[agentID] = team.AuditResult[agentID] || true // There is a deferral
	}
}

func (team *Team) SetWithdrawalResult(agentID uuid.UUID, agentScore int, agentActualWithdrawal int) {
	agentExpectedWithdrawal := team.ArticlesOfAssociation.withdrawalRule.GetExpectedWithdrawalAmount(agentScore)
	if agentActualWithdrawal != agentExpectedWithdrawal {
		team.AuditResult[agentID] = team.AuditResult[agentID] || true // There is a deferral
	}
}

func(team *Team) ResetAuditResult() {
	team.AuditResult = map[uuid.UUID]bool{}
}
