package common

import (
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
)

type IExtendedAgent interface {
	agent.IAgent[IExtendedAgent]

	// Getters
	GetTeamID() uuid.UUID
	GetLastTeamID() uuid.UUID
	GetTrueScore() int
	GetActualContribution() int
	GetActualWithdrawal(currentPoolValue int) int
	GetStatedContribution() int
	GetStatedWithdrawal() int

	// Setters
	SetTeamID(teamID uuid.UUID)
	SetTrueScore(score int)
	StartRollingDice()
	StickOrAgain() bool
	DecideStick()
	DecideRollAgain()

	// team forming
	StartTeamForming(agentInfoList []ExposedAgentInfo)

	// Messaging functions
	HandleTeamFormationMessage(msg *TeamFormationMessage)
	HandleScoreReportMessage(msg *ScoreReportMessage)
	HandleWithdrawalMessage(msg *WithdrawalMessage)
	HandleContributionMessage(msg *ContributionMessage)

	// Info
	GetExposedInfo() ExposedAgentInfo
	LogSelfInfo()
	GetAoARanking() []int
	SetAoARanking(Preferences []int)
}
