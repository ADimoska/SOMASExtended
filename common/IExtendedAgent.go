package common

import (
	gameRecorder "github.com/ADimoska/SOMASExtended/gameRecorder"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/message"
)

type IExtendedAgent interface {
	agent.IAgent[IExtendedAgent]

	// Getters
	GetTeamID() uuid.UUID
	GetLastTeamID() uuid.UUID
	GetAgentType() string
	GetTrueScore() int
	GetTeamRanking() []uuid.UUID
	// Functions that involve strategic decisions
	StartTeamForming(instance IExtendedAgent, agentInfoList []ExposedAgentInfo)
	StartRollingDice(instance IExtendedAgent)
	// Setters
	SetTeamID(teamID uuid.UUID)
	SetTrueScore(score int)
	SetAgentContributionAuditResult(agentID uuid.UUID, result bool)
	SetAgentWithdrawalAuditResult(agentID uuid.UUID, result bool)
	SetTeamRanking(teamRanking []uuid.UUID)
	DecideStick()
	DecideRollAgain()

	// Strategic decisions (functions that each team can implement their own)
	// NOTE: Any function calling these should have a parameter of type IExtendedAgent (instance IExtendedAgent)
	DecideTeamForming(agentInfoList []ExposedAgentInfo) []uuid.UUID
	StickOrAgain(accumulatedScore int, prevRoll int) bool
	VoteOnAgentEntry(candidateID uuid.UUID) bool
	StickOrAgainFor(agentId uuid.UUID, accumulatedScore int, prevRoll int) int
	GetActualContribution(instance IExtendedAgent) int
	GetStatedContribution(instance IExtendedAgent) int
	GetActualWithdrawal(instance IExtendedAgent) int
	GetStatedWithdrawal(instance IExtendedAgent) int


	// Messaging functions
	HandleTeamFormationMessage(msg *TeamFormationMessage)
	HandleScoreReportMessage(msg *ScoreReportMessage)
	HandleWithdrawalMessage(msg *WithdrawalMessage)
	BroadcastSyncMessageToTeam(msg message.IMessage[IExtendedAgent])
	HandleContributionMessage(msg *ContributionMessage)
	HandleAgentOpinionRequestMessage(msg *AgentOpinionRequestMessage)
	HandleAgentOpinionResponseMessage(msg *AgentOpinionResponseMessage)
	StateContributionToTeam(instance IExtendedAgent)
	StateWithdrawalToTeam(instance IExtendedAgent)

	// Info
	GetExposedInfo() ExposedAgentInfo
	CreateScoreReportMessage() *ScoreReportMessage
	CreateContributionMessage(statedAmount int) *ContributionMessage
	CreateWithdrawalMessage(statedAmount int) *WithdrawalMessage
	CreateAgentOpinionRequestMessage(agentID uuid.UUID) *AgentOpinionRequestMessage
	CreateAgentOpinionResponseMessage(agentID uuid.UUID, opinion int) *AgentOpinionResponseMessage
	LogSelfInfo()
	GetAoARanking() []int
	SetAoARanking(Preferences []int)
	GetContributionAuditVote() Vote
	GetWithdrawalAuditVote() Vote
	GetTrueSomasTeamID() int

	// Data Recording
	RecordAgentStatus(instance IExtendedAgent) gameRecorder.AgentRecord

	// Team 1 specific functions
	Team1_ChairUpdateRanks(rankMap map[uuid.UUID]int) map[uuid.UUID]int
	Team1_VoteOnRankBoundaries(initialBoundaries [5]int) [5]int
}
