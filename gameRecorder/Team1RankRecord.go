package gameRecorder

import "github.com/google/uuid"

type Team1RankRecord struct {
	// basic info fields
	TurnNumber      int
	IterationNumber int
	Boundaries0     int
	Boundaries1     int
	Boundaries2     int
	Boundaries3     int
	Boundaries4     int
	TeamID          uuid.UUID
}

func NewTeam1RankRecord(turnNumber int, iterationNumber int, TeamID uuid.UUID, boundaries [5]int) Team1RankRecord {
	return Team1RankRecord{
		TurnNumber:      turnNumber,
		IterationNumber: iterationNumber,
		Boundaries0:     boundaries[0],
		Boundaries1:     boundaries[1],
		Boundaries2:     boundaries[2],
		Boundaries3:     boundaries[3],
		Boundaries4:     boundaries[4],
		TeamID:          TeamID,
	}
}
