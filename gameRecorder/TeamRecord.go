package gameRecorder

import (
	// "SOMAS_Extended/common"
	"github.com/google/uuid"
)

// AgentRecord is a record of an agent's state at a given turn
type TeamRecord struct {
	// basic info fields
	TurnNumber      int
	IterationNumber int
	TeamID          uuid.UUID

	// turn-specific fields
	TeamCommonPool int
	AgentsAlive    []uuid.UUID
	AgentsDead     []uuid.UUID
}
