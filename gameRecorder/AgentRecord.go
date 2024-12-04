package gameRecorder

import (
	"log"

	"github.com/google/uuid"
)

// AgentRecord is a record of an agent's state at a given turn
type AgentRecord struct {
	// basic info fields
	TurnNumber      int
	IterationNumber int
	AgentID         uuid.UUID
	TrueSomasTeamID int    // SOMAS team number, e.g., Team 4
	AgentType       string // New field for agent type

	// turn-specific fields
	IsAlive            bool
	Score              int
	Contribution       int
	StatedContribution int
	Withdrawal         int
	StatedWithdrawal   int

	TeamID uuid.UUID
}

// NewAgentRecord creates a new instance of AgentRecord
func NewAgentRecord(agentID uuid.UUID, trueSomasTeamID int, agentType string, score int, contribution int, statedContribution int, withdrawal int, statedWithdrawal int, teamID uuid.UUID) AgentRecord {
	return AgentRecord{
		AgentID:            agentID,
		TrueSomasTeamID:    trueSomasTeamID,
		AgentType:          agentType, // Set the new field
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
	// log.Printf("Agent ID: %v\n", ar.AgentID)
	if !ar.IsAlive {
		log.Printf("[DEAD] ")
	}
	log.Printf("Agent Score: %v\n", ar.Score)
	// log.Printf("Agent Contribution: %v\n", ar.agent.GetActualContribution(ar.agent))
	// log.Printf("Agent Stated Contribution: %v\n", ar.agent.GetStatedContribution(ar.agent))
	// log.Printf("Agent Withdrawal: %v\n", ar.agent.GetActualWithdrawal(ar.agent))
	// log.Printf("Agent Stated Withdrawal: %v\n", ar.agent.GetStatedWithdrawal(ar.agent))
}
