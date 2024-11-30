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


