package common

import (
	// TODO: should it be structured this way?
	aoa "SOMAS_Extended/ArticlesOfAssociation"

	"github.com/google/uuid"
)

type Team struct {
	TeamID  uuid.UUID
	Agents  []uuid.UUID
	TeamAoA aoa.IArticlesOfAssociation

	commonPool int
}

func (team *Team) GetCommonPool() int {
	return team.commonPool
}

func (team *Team) SetCommonPool(amount int) {
	team.commonPool = amount
}

// constructor: NewTeam creates a new Team with a unique TeamID and initializes other fields as blank.
func NewTeam(teamID uuid.UUID) *Team {
	teamAoA := aoa.CreateFixedAoA()
	return &Team{
		TeamID:     teamID,        // Generate a unique TeamID
		commonPool: 0,             // Initialize commonPool to 0
		Agents:     []uuid.UUID{}, // Initialize an empty slice of agent UUIDs
		TeamAoA:    teamAoA,       // Initialize strategy as 0
	}
}
