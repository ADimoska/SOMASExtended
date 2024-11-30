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
	// get agent profiles and update with assigned roles
	var agentIDs []uuid.UUID = t2a.server.GetAgentsInTeam(t2a.teamID);
	//TODO set the role of the agent - idk how to do this yet - some sort of agent profile where there is a role field
	return 0 // TODO Return updated agent profiles
}

// Decide who is leader and who isnt, do this with an integar flage. Key = {1: leader, 2: general, 3: citizen, 4: police}
func (t2a *Team2Agent) AllocateRank() {
	//TODO get turn, iteration and agent ids from server - call the server functions
	// Get the current turn and iteration
	currentTurn bool = t2a.common.GetLastRound()
	var highestTrustScore int = 0; // start with lowest poss trust score
	// Get the list of all agent UUIDs
	var agentIDs []uuid.UUID = t2a.server.GetAgentsInTeam(t2a.teamID);
	// int currentIteration = t2a.server.GetCurrentIteration(t2a.teamID)
	// // Check if it is the first turn of the game
	if currentTurn == 0 {
			// Randomly select a leader from the list of agents
		leaderIndex := rand.Intn(len(agentIDs))
		// Aassign id as leader
		leaderID := agentIDs[leaderIndex]
		// Assign the leader role
		t2a.AssignRole(leaderID, 1) // Role 1 represents the leader
		// Assign the citizen role to all other agents
		for _, agentID := range agentIDs {
			if agentID != leaderID {
				t2a.AssignRole(agentID, 3) // Role 3 represents citizens
			}
		}
	} else {
		// Agents vote leader in based on trust scores
		for _, agentID := range agentsInTeam {
			agentTrustScore := t2a.trustScore[agentID];

			if agentTrustScore > highestTrustScore {
				highestAgent = agentID;
				highestTrustScore = agentTrustScore;
			}
		}
		if  0 <= highestTrustScore <= 10{ 
			return common.CreateVote(1, t2a.GetID(), highestAgent);
		}else{
			return common.CreateVote(0, t2a.GetID(), uuid.Nil); // abstain cuz no trust score exists or is too high so gone wrong somehwere
		}
	}
}


