package common

import (
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/message"
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
	StatedAmount int
}

type WithdrawalMessage struct {
	message.BaseMessage
	StatedAmount int
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
