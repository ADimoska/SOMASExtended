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
