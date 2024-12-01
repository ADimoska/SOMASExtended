package common

import (
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/message"
)

type IExtendedAgent interface {
	agent.IAgent[IExtendedAgent]

	// Getters
	GetTeamID() uuid.UUID
	GetLastTeamID() uuid.UUID
	GetTrueScore() int

	// Functions that involve strategic decisions
	StartTeamForming(instance IExtendedAgent, agentInfoList []ExposedAgentInfo)
	StartRollingDice(instance IExtendedAgent)
	GetActualContribution(instance IExtendedAgent) int
	GetActualWithdrawal(instance IExtendedAgent) int
	GetStatedContribution(instance IExtendedAgent) int
	GetStatedWithdrawal(instance IExtendedAgent) int

	// Setters
	SetTeamID(teamID uuid.UUID)
	SetTrueScore(score int)
	SetAgentContributionAuditResult(agentID uuid.UUID, result bool)
	SetAgentWithdrawalAuditResult(agentID uuid.UUID, result bool)
	DecideStick()
	DecideRollAgain()

	// Strategic decisions (functions that each team can implement their own)
	// NOTE: Any function calling these should have a parameter of type IExtendedAgent (instance IExtendedAgent)
	DecideTeamForming(agentInfoList []ExposedAgentInfo) []uuid.UUID
	StickOrAgain() bool
	DecideContribution() int
	DecideWithdrawal() int

	// Messaging functions
	HandleTeamFormationMessage(msg *TeamFormationMessage)
	HandleScoreReportMessage(msg *ScoreReportMessage)
	HandleWithdrawalMessage(msg *WithdrawalMessage)
	BroadcastSyncMessageToTeam(msg message.IMessage[IExtendedAgent])
	HandleContributionMessage(msg *ContributionMessage)

	// Info
	GetExposedInfo() ExposedAgentInfo
	CreateScoreReportMessage() *ScoreReportMessage
	CreateContributionMessage(statedAmount int, expectedAmount int) *ContributionMessage
	CreateWithdrawalMessage(statedAmount int, expectedAmount int) *WithdrawalMessage
	LogSelfInfo()
	GetAoARanking() []int
	SetAoARanking(Preferences []int)
	GetContributionAuditVote() Vote
	GetWithdrawalAuditVote() Vote
}
