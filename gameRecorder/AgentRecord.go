package gameRecorder

import (
	"fmt"

	"github.com/google/uuid"
)

// AgentRecord is a record of an agent's state at a given turn
type AgentRecord struct {
	// basic info fields
	TurnNumber      int
	IterationNumber int
	AgentID         uuid.UUID
	TrueSomasTeamID int // SOMAS team number, e.g. Team 4

	// turn-specific fields
	IsAlive            bool
	Score              int
	Contribution       int
	StatedContribution int
	Withdrawal         int
	StatedWithdrawal   int

	TeamID uuid.UUID
}

func NewAgentRecord(agentID uuid.UUID, trueSomasTeamID int, score int, contribution int, statedContribution int, withdrawal int, statedWithdrawal int, teamID uuid.UUID) AgentRecord {
	return AgentRecord{
		AgentID:            agentID,
		TrueSomasTeamID:    trueSomasTeamID,
		Score:              score,
		Contribution:       contribution,
		StatedContribution: statedContribution,
		Withdrawal:         withdrawal,
		StatedWithdrawal:   statedWithdrawal,
		TeamID:             teamID,
	}
}

func NewTeamRecord(teamID uuid.UUID) TeamRecord {
	return TeamRecord{
		TeamID: teamID,
	}
}

func (ar *AgentRecord) DebugPrint() {
	// fmt.Printf("Agent ID: %v\n", ar.AgentID)
	if !ar.IsAlive {
		fmt.Printf("[DEAD] ")
	}
	fmt.Printf("Agent Score: %v\n", ar.Score)
	// fmt.Printf("Agent Contribution: %v\n", ar.agent.GetActualContribution(ar.agent))
	// fmt.Printf("Agent Stated Contribution: %v\n", ar.agent.GetStatedContribution(ar.agent))
	// fmt.Printf("Agent Withdrawal: %v\n", ar.agent.GetActualWithdrawal(ar.agent))
	// fmt.Printf("Agent Stated Withdrawal: %v\n", ar.agent.GetStatedWithdrawal(ar.agent))
}
