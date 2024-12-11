package common

import (
	// TODO: should it be structured this way?

	"github.com/google/uuid"

	gameRecorder "github.com/ADimoska/SOMASExtended/gameRecorder"
)

type Team struct {
	TeamID         uuid.UUID
	Agents         []uuid.UUID
	TeamAoA        IArticlesOfAssociation
	TeamAoAID      int
	commonPool     int
	knownThreshold int  // current threshold set by server
	validThreshold bool // flag if the threshold has been updated this turn
}

func (team *Team) GetCommonPool() int {
	return team.commonPool
}

func (team *Team) SetCommonPool(amount int) {
	team.commonPool = amount
}

func (team *Team) RemoveAgent(agentID uuid.UUID) {
	for i, a := range team.Agents {
		if a == agentID {
			team.Agents = append(team.Agents[:i], team.Agents[i+1:]...)
			break
		}
	}
}

// constructor: NewTeam creates a new Team with a unique TeamID and initializes other fields as blank.
func NewTeam(teamID uuid.UUID) *Team {
	teamAoA := CreateFixedAoA(1)
	return &Team{
		TeamID:     teamID,        // Generate a unique TeamID
		commonPool: 0,             // Initialize commonPool to 0
		Agents:     []uuid.UUID{}, // Initialize an empty slice of agent UUIDs
		TeamAoA:    teamAoA,       // Initialize strategy as 0
	}
}

/**
* Set the known threshold so that agents can adapt their behaviour based on
* this. AGENTS PLEASE DON'T CALL THIS - in an ideal world we would use
* pre-signed certificates to know that only the server updated this but our
* code is not prioritising security.
 */
func (team *Team) SetKnownThreshold(threshold int) {
	team.knownThreshold = threshold
	team.validThreshold = true
}

// Same as above, @agents please don't call this
func (team *Team) InvalidateThreshold() {
	team.validThreshold = false
}

// Return the known threshold and the flag that determines if its valid or not
func (team *Team) GetKnownThreshold() (int, bool) {
	return team.knownThreshold, team.validThreshold
}

// --------- Recording Functions ---------
func (team *Team) RecordTeamStatus() gameRecorder.TeamRecord {
	return gameRecorder.NewTeamRecord(team.TeamID)
}
