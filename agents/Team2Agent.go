package agents

import (
	"SOMAS_Extended/common"
	"math"
	"math/rand"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
)

// this is the third tier of composition - embed the extended agent and add 'user specific' fields
type Team2Agent struct {
	*ExtendedAgent
	rank            map[uuid.UUID]string // Map agent UUID ot rank
	trustScore      map[uuid.UUID]int
	thresholdBounds []int
}

// constructor for Fascist Agent - not sure exaftly whats going on here
func Team2_CreateAgent(funcs agent.IExposedServerFunctions[common.IExtendedAgent], agentConfig AgentConfig) *FascistAgent {
	return &Team2Agent{
		ExtendedAgent: GetBaseAgents(funcs, agentConfig),
	}
}

// ----- 2.1 Decision to send or accept a team invitiation -----

func (t2a *Team2Agent) DecideTeamForming(agentInfoList []common.ExposedAgentInfo) []uuid.UUID {

}

// ----- 2.2 Decision to stick -----

func (t2a *Team2Agent) StickorAgain() {

}

// ----- 2.3 Decision to cheat / not cooperate

func (t2a *Team2Agent) DecideContribution() {

}

func (t2a *Team2Agent) DecideWithdrawal() {

}

// 2.4 ----- Decision to Audit Someone

func (t2a *Team2Agent) GetContributionAuditVote() common.Vote {

	// experiment with this; it is our threshold to decide to audit
	var auditThreshold int = 5

	// get list of uuids in our team
	var agentsInTeam []uuid.UUID = t2a.server.GetAgentsInTeam(t2a.teamID)

	var lowestTrustScore int = math.MaxInt // Start with the maximum possible int
	var lowestAgent uuid.UUID

	// iterate over our team, finding the agent with the lowest trust score
	for _, agentID := range agentsInTeam {
		agentTrustScore := t2a.trustScore[agentID]

		if agentTrustScore < lowestTrustScore {
			lowestAgent = agentID
			lowestTrustScore = agentTrustScore
		}
	}

	// if the lowest agent is below the threshold, submit a vote for them
	if lowestTrustScore < auditThreshold {
		// 1 means vote for audit of this person
		return common.CreateVote(1, t2a.GetID(), lowestAgent)
	} else {
		// 0 means abstain / no preference
		return common.CreateVote(0, t2a.GetID(), uuid.Nil)
	}

}

func (t2a *Team2Agent) GetWithdrawalAuditVote() common.Vote {
	// experiment with this; it is our threshold to decide to audit
	var auditThreshold int = 5

	// get list of uuids in our team
	var agentsInTeam []uuid.UUID = t2a.server.GetAgentsInTeam(t2a.teamID)

	var lowestTrustScore int = math.MaxInt // Start with the maximum possible int
	var lowestAgent uuid.UUID

	// iterate over our team, finding the agent with the lowest trust score
	for _, agentID := range agentsInTeam {
		agentTrustScore := t2a.trustScore[agentID]

		if agentTrustScore < lowestTrustScore {
			lowestAgent = agentID
			lowestTrustScore = agentTrustScore
		}
	}

	// if the lowest agent is below the threshold, submit a vote for them
	if lowestTrustScore < auditThreshold {
		// 1 means vote for audit of this person
		return common.CreateVote(1, t2a.GetID(), lowestAgent)
	} else {
		// 0 means abstain / no preference
		return common.CreateVote(0, t2a.GetID(), uuid.Nil)
	}

}

// /////////// ----------------------RANKING SYSTEM---------------------- /////////////

// AssignRole assigns a specific role to an agent based on their UUID
func (t2a *Team2Agent) AssignRole(agentID uuid.UUID, role string) string {
	// currentTurn := t2a.Team2AoA.GetLastRound() // TODO fix the way i call this
	currentTurn := false
	agentIDs := t2a.server.GetAgentsInTeam(t2a.teamID)

	// Ensure rankMap is initialized
	if t2a.rank == nil {
		t2a.rank = make(map[uuid.UUID]string)
	}

	if !currentTurn {
		// Check if the agent already has a role assigned
		if t2a.rank[agentID] != "" {
			// If the agent is already assigned a role, return the current role
			return t2a.rank[agentID]
		} else {
			// If no role is assigned, assign the new role
			t2a.rank[agentID] = role
		}
	} else {
		// Get uuid from vote result
		// voteResult := t2a.Team2AoA.GetVoteResult(t2a.server.GetVotes()) //TODO fix the way i call this
		voteResult := uuid.Nil
		// If the agent is the new leader, assign the leader role
		t2a.rank[voteResult] = "Leader"

		// Assign citizen roles to all other agents
		for _, agentID := range agentIDs {
			if agentID != voteResult {
				t2a.rank[agentID] = "Citizen"
			}
		}
	}

	return role // Return the newly assigned role
}

// AllocateRank decides roles and assigns them based on the current game state and votes
func (t2a *Team2Agent) AllocateRank(votes []common.Vote) common.Vote {
	// Get the current turn
	// currentTurn := t2a.Team2AoA.GetLastRound() // TODO fix the way i call this
	currentTurn := false
	var highestTrustScore int = math.MinInt // Lowest possible int for comparison
	var highestAgent uuid.UUID

	// Get the list of all agent UUIDs
	agentIDs := t2a.server.GetAgentsInTeam(t2a.teamID)

	if len(agentIDs) == 0 {
		// If no agents are found, abstain from voting
		return common.Vote{IsVote: 0, VoterID: t2a.GetID(), VotedForID: uuid.Nil}
	}

	// First turn: Randomly assign roles
	if !currentTurn {
		// Randomly select a leader
		leaderIndex := rand.Intn(len(agentIDs))
		leaderID := agentIDs[leaderIndex]
		// Assign leader role
		t2a.AssignRole(leaderID, "Leader")

		// Assign citizen roles to all other agents
		for _, agentID := range agentIDs {
			if agentID != leaderID {
				t2a.AssignRole(agentID, "Citizen")
			}
		}
		// Return a vote indicating the leader
		return common.Vote{IsVote: 1, VoterID: t2a.GetID(), VotedForID: leaderID}
	} else {
		// Subsequent turns: Use trust scores to assign roles
		for _, agentID := range agentIDs {
			agentTrustScore := t2a.trustScore[agentID] // Trust score map must be initialized
			if agentTrustScore > highestTrustScore {
				highestAgent = agentID
				highestTrustScore = agentTrustScore
			}
		}

		// If no valid trust scores are found, abstain
		if highestTrustScore <= 0 {
			return common.Vote{IsVote: 0, VoterID: t2a.GetID(), VotedForID: uuid.Nil}
		}

		// Return a vote indicating the new leader
		return common.Vote{IsVote: 1, VoterID: t2a.GetID(), VotedForID: highestAgent}

	}
}
