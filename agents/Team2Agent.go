package agents

import (
	"SOMAS_Extended/common"
	"fmt"
	"math/rand"
	"math"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"

)


// this is the third tier of composition - embed the extended agent and add 'user specific' fields
type Team2Agent struct {
	*ExtendedAgent
	rank string
	trustScore map[uuid.UUID]int
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
		var auditThreshold int = 5;

		// get list of uuids in our team
		var agentsInTeam []uuid.UUID = t2a.server.GetAgentsInTeam(t2a.teamID);

		var lowestTrustScore int = math.MaxInt; // Start with the maximum possible int
		var lowestAgent uuid.UUID;

		// iterate over our team, finding the agent with the lowest trust score
		for _, agentID := range agentsInTeam {
			agentTrustScore := t2a.trustScore[agentID];

			if agentTrustScore < lowestTrustScore {
				lowestAgent = agentID;
				lowestTrustScore = agentTrustScore;
			}
		}

		// if the lowest agent is below the threshold, submit a vote for them
		if lowestTrustScore < auditThreshold {
			// 1 means vote for audit of this person
			return common.CreateVote(1, t2a.GetID(), lowestAgent);
		} else {
			// 0 means abstain / no preference
			return common.CreateVote(0, t2a.GetID(), uuid.Nil);
		}

}

func (t2a *Team2Agent) GetWithdrawalAuditVote() common.Vote { 
		// experiment with this; it is our threshold to decide to audit 
		var auditThreshold int = 5;

		// get list of uuids in our team
		var agentsInTeam []uuid.UUID = t2a.server.GetAgentsInTeam(t2a.teamID);

		var lowestTrustScore int = math.MaxInt; // Start with the maximum possible int
		var lowestAgent uuid.UUID;

		// iterate over our team, finding the agent with the lowest trust score
		for _, agentID := range agentsInTeam {
			agentTrustScore := t2a.trustScore[agentID];

			if agentTrustScore < lowestTrustScore {
				lowestAgent = agentID;
				lowestTrustScore = agentTrustScore;
			}
		}

		// if the lowest agent is below the threshold, submit a vote for them
		if lowestTrustScore < auditThreshold {
			// 1 means vote for audit of this person
			return common.CreateVote(1, t2a.GetID(), lowestAgent);
		} else {
			// 0 means abstain / no preference
			return common.CreateVote(0, t2a.GetID(), uuid.Nil);
		}

}

// /////////// ----------------------RANKING SYSTEM---------------------- /////////////
func (t2a *Team2Agent) AssignRole(agentID uuid.UUID, role int) int {
	// Check if trustScore map exists
	if t2a.trustScore == nil {
		t2a.trustScore = make(map[uuid.UUID]int)
	}
	// Update the role of the agent in trustScore map (or other structure if required)
	t2a.trustScore[agentID] = role
	return role // Return the role assigned for confirmation
}


// Decide who is leader and who isnt, do this with an integar flage. Key = {1: leader, 2: general, 3: citizen, 4: police}
func (t2a *Team2Agent) AllocateRank() common.Vote {
	// Get the current turn (assumes GetLastRound fetches the turn number)
	currentTurn := t2a.common.GetLastRound() // Implement correctly if needed
	var highestTrustScore int = math.MinInt
	var highestAgent uuid.UUID

	// Get the list of all agent UUIDs
	agentIDs := t2a.server.GetAgentsInTeam(t2a.teamID)

	if len(agentIDs) == 0 {
		// If no agents, return abstain vote
		return common.CreateVote(0, t2a.GetID(), uuid.Nil)
	}

	// Check if it's the first turn
	if currentTurn == 0 {
		// Randomly select a leader
		leaderIndex := rand.Intn(len(agentIDs))
		leaderID := agentIDs[leaderIndex]
		// Assign the leader role
		t2a.AssignRole(leaderID, 1) // Role 1 is Leader

		// Assign citizens to others
		for _, agentID := range agentIDs {
			if agentID != leaderID {
				t2a.AssignRole(agentID, 3) // Role 3 is Citizen
			}
		}
		// Return a vote indicating the new leader
		return common.CreateVote(1, t2a.GetID(), leaderID)
	} else {
		// Choose leader based on trust scores
		for _, agentID := range agentIDs {
			agentTrustScore := t2a.trustScore[agentID] // Assuming trustScore is initialized properly
			if agentTrustScore > highestTrustScore {
				highestAgent = agentID
				highestTrustScore = agentTrustScore
			}
		}

		// If no valid trust scores, abstain
		if highestTrustScore <= 0 {
			return common.CreateVote(0, t2a.GetID(), uuid.Nil)
		}

		// Return vote for the highest agent
		return common.CreateVote(1, t2a.GetID(), highestAgent)
	}
}
