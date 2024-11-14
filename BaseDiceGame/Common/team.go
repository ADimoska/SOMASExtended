package common

import "github.com/google/uuid"


type Team struct{
	TeamID uuid.UUID
	CommonPool int
	Agents []uuid.UUID
	Strategy int
}

// constructor: NewTeam creates a new Team with a unique TeamID and initializes other fields as blank.
func NewTeam() Team {
	return Team{
		TeamID:     uuid.New(),  // Generate a unique TeamID
		CommonPool: 0,           // Initialize commonPool to 0
		Agents:     []uuid.UUID{}, // Initialize an empty slice of agent UUIDs
		Strategy:   0,            // Initialize strategy as 0
	}
}


