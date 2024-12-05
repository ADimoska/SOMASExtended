package common

import (
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/message"
	"github.com/google/uuid"
)

type TeamFormationMessage struct {
	message.BaseMessage
	AgentInfo ExposedAgentInfo
	Message   string
}

type ScoreReportMessage struct {
	message.BaseMessage
	TurnScore int
	Rerolls   int
}

type ContributionMessage struct {
	message.BaseMessage
	StatedAmount   int
	ExpectedAmount int
}

type WithdrawalMessage struct {
	message.BaseMessage
	StatedAmount   int
	ExpectedAmount int
}

type AgentOpinionRequestMessage struct {
	message.BaseMessage
	AgentID uuid.UUID
}

type AgentOpinionResponseMessage struct {
	message.BaseMessage
	AgentID      uuid.UUID
	AgentOpinion int
}

type Team1RankBoundaryRequestMessage struct {
	message.BaseMessage
	// Could possibly provide additional info to guide agent decision here
}

type Team1RankBoundaryResponseMessage struct {
	message.BaseMessage
	Bounds [5]int
}

type Team1BoundaryBallotRequestMessage struct {
	message.BaseMessage
	Candidates [3][5]int
}

type Team1BoundaryBallotResponseMessage struct {
	message.BaseMessage
	RankedCandidates [3]int
}

type Team4_ProposedWithdrawalMessage struct {
	message.BaseMessage
	StatedAmount   int
	ExpectedAmount int
}

type Team4_ConfessionMessage struct {
	message.BaseMessage
	Confession bool
}

func (msg *TeamFormationMessage) InvokeMessageHandler(agent IExtendedAgent) {
	agent.HandleTeamFormationMessage(msg)
}

func (msg *ScoreReportMessage) InvokeMessageHandler(agent IExtendedAgent) {
	agent.HandleScoreReportMessage(msg)
}

func (msg *ContributionMessage) InvokeMessageHandler(agent IExtendedAgent) {
	agent.HandleContributionMessage(msg)
}

func (msg *WithdrawalMessage) InvokeMessageHandler(agent IExtendedAgent) {
	agent.HandleWithdrawalMessage(msg)
}

func (msg *AgentOpinionRequestMessage) InvokeMessageHandler(agent IExtendedAgent) {
	agent.HandleAgentOpinionRequestMessage(msg)
}

func (msg *AgentOpinionResponseMessage) InvokeMessageHandler(agent IExtendedAgent) {
	agent.HandleAgentOpinionResponseMessage(msg)
}

func (msg *Team1RankBoundaryRequestMessage) InvokeMessageHandler(agent IExtendedAgent) {
	agent.Team1_BoundaryProposalRequestHandler(msg)
}

func (msg *Team1RankBoundaryResponseMessage) InvokeMessageHandler(agent IExtendedAgent) {
	agent.Team1_BoundaryProposalResponseHandler(msg)
}

func (msg *Team1BoundaryBallotRequestMessage) InvokeMessageHandler(agent IExtendedAgent) {
	agent.Team1_BoundaryBallotRequestHandler(msg)
}

func (msg *Team1BoundaryBallotResponseMessage) InvokeMessageHandler(agent IExtendedAgent) {
	agent.Team1_BoundaryBallotResponseHandler(msg)
}

func (msg *Team4_ProposedWithdrawalMessage) InvokeMessageHandler(agent IExtendedAgent) {
	agent.Team4_HandleProposedWithdrawalMessage(msg)
}

func (msg *Team4_ConfessionMessage) InvokeMessageHandler(agent IExtendedAgent) {
	agent.Team4_HandleConfessionMessage(msg)
}
