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
	GetTrueScore() int
	GetTeamRanking() []uuid.UUID
	GetName() int

	// Functions that involve strategic decisions
	StartTeamForming(instance IExtendedAgent, agentInfoList []ExposedAgentInfo)
	StartRollingDice(instance IExtendedAgent)
	GetActualContribution(instance IExtendedAgent) int
	GetActualWithdrawal(instance IExtendedAgent) int
	GetStatedContribution(instance IExtendedAgent) int
	GetStatedWithdrawal(instance IExtendedAgent) int
	GetLeaveOpinion(agentID uuid.UUID) bool

	// Setters
	SetName(name int)
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

	// Team4 AoA Functions
	Team4_GetRankUpVote() map[uuid.UUID]int
	Team4_GetConfession() bool
	Team4_GetProposedWithdrawalVote() map[uuid.UUID]int

	Team4_GetProposedWithdrawal(instance IExtendedAgent) int
	Team4_ProposeWithdrawal() int

	Team4_StateProposalToTeam()
	Team4_CreateProposedWithdrawalMessage(statedAmount int) *Team4_ProposedWithdrawalMessage
	Team4_HandleProposedWithdrawalMessage(msg *Team4_ProposedWithdrawalMessage)

	Team4_StateConfessionToTeam()
	Team4_CreateConfessionMessage(confession bool) *Team4_ConfessionMessage
	Team4_HandleConfessionMessage(msg *Team4_ConfessionMessage)
	Team4_GetPunishmentVoteMap() map[int]int
	// Data Recording
	RecordAgentStatus(instance IExtendedAgent) gameRecorder.AgentRecord

	// Team 1AoA specific functions
	Team1_ChairUpdateRanks(rankMap map[uuid.UUID]int) map[uuid.UUID]int
	Team1_AgreeRankBoundaries() [5]int
	Team1_BoundaryProposalRequestHandler(msg *Team1RankBoundaryRequestMessage)
	Team1_BoundaryProposalResponseHandler(msg *Team1RankBoundaryResponseMessage)
	Team1_BoundaryBallotRequestHandler(msg *Team1BoundaryBallotRequestMessage)
	Team1_BoundaryBallotResponseHandler(msg *Team1BoundaryBallotResponseMessage)

	// Team 2 specific functions
	Team2_GetLeaderVote() Vote
}
