package agents

import (
	"SOMAS_Extended/common"
	"fmt"
	"math"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
)

// this is the third tier of composition - embed the extended agent and add 'user specific' fields
type Team2Agent struct {
	*ExtendedAgent
	rank            bool
	trustScore      map[uuid.UUID]int
	strikeCount     map[uuid.UUID]int
	thresholdBounds []int
}

// constructor for team2agent - initialised as all followers
func Team2_CreateAgent(funcs agent.IExposedServerFunctions[common.IExtendedAgent], agentConfig AgentConfig) *Team2Agent {
	return &Team2Agent{
		ExtendedAgent: GetBaseAgents(funcs, agentConfig), rank: false, trustScore: make(map[uuid.UUID]int), strikeCount: make(map[uuid.UUID]int), thresholdBounds: make([]int, 2),
	}
}

// ----- 1.1 Decide Contribution and Withdrawal -----
func (t2a *Team2Agent) DecideContribution() int {

	// Get the current points in the common pool
	currentPoints := t2a.server.GetTeam(t2a.GetID()).GetCommonPool()
	// Get the expected contribution from the AOA - currently just ou rAoA but can change this to whichever AoA is currently in use
	expectedContribution := t2a.server.GetTeam(t2a.teamID).TeamAoA.(*common.Team2AoA).GetExpectedContribution(t2a.GetID(), currentPoints)
	// If the expected contribution is less than or equal to current points just contribute expected
	if expectedContribution <= currentPoints {
		return expectedContribution
	}

	// If the expected contribution is more than current points, contribute all current points
	return currentPoints
}

func (t2a *Team2Agent) DecideWithdrawal() int {
    // Get the current points in the common pool
    currentPoints := t2a.server.GetTeam(t2a.GetID()).GetCommonPool()
    // Get common pool size
	commonPoolSize := t2a.server.GetTeam(t2a.GetID()).GetCommonPool()
    // Get the expected withdrawal from the AOA
    expectedWithdrawal := t2a.server.GetTeam(t2a.teamID).TeamAoA.(*common.Team2AoA).GetExpectedWithdrawal(t2a.GetID(), currentPoints, commonPoolSize)
    
    // If the expected withdrawal is more than current points, withdraw all current points
    return expectedWithdrawal
}

// ----- 1.2 Trust Score Update -----
func (t *Team2Agent) SetTrustScore(id uuid.UUID) {
	if _, exists := t.trustScore[id]; !exists {
		t.trustScore[id] = 70
	}
}

// Overall function to update one agents trust score for other agents
// (can either implement like this or call functions underneath during each event)
func (t2a *Team2Agent) UpdateTrustScore(agentID uuid.UUID, eventType string, strikeCount int) {
	switch eventType {
	case "strike":
		// if agent is not cooperating:
		t2a.ApplyStrike(agentID) // from helper function

	case "notAudited":
		// If the target agent was not audited
		t2a.ApplyNotAudited(agentID)

	case "auditOther":
		// If the target agent audited another agent
		t2a.ApplyAuditOther(agentID)

	default:
		fmt.Println("Invalid event type")
	}
}

// update when not cooperating based on strikes
func (t2a *Team2Agent) ApplyStrike(agentID uuid.UUID) {
	if t2a.trustScore == nil {
		t2a.SetTrustScore(agentID)
	}

	t2a.strikeCount[agentID]++ // Increment the strike count for this agent
	var penalty int
	strikeCount := t2a.strikeCount[agentID]

	if strikeCount == 1 {
		penalty = 10
	}
	if strikeCount == 2 {
		penalty = 20
	}
	if strikeCount == 3 {
		penalty = 30
	} else {
		// should never reach this point
		penalty = 40
	}

	// Update trust score based on strike count
	t2a.trustScore[agentID] -= penalty
}

// update trust score if agent kicked out of other team -->>(can't implement yet?)
// func (t2a *Team2Agent) ApplyKickFromTeam(agentID uuid.UUID) {
//     // Update trust score based on being kicked from another team
//     t2a.trustScore[agentID] -= 5
// }

// update trust score based on audit vote (maybe exclude this as shouldn't know other agents votes?)
// func (t2a *Team2Agent) ApplyAuditVote(agentID uuid.UUID) {
// 	if t2a.trustScore == nil {
// 		t2a.SetTrustScore(agentID)
// 	}
// 	// Update trust score based on audit vote
// 	t2a.trustScore[agentID] -= 5
// }

// update if agent not audited for that round
func (t2a *Team2Agent) ApplyNotAudited(agentID uuid.UUID) {
	if t2a.trustScore == nil {
		t2a.SetTrustScore(agentID)
	}
	// Update trust score based on not being audited
	t2a.trustScore[agentID] += 2
}

// update if chooses to audit another agent
func (t2a *Team2Agent) ApplyAuditOther(agentID uuid.UUID) {
	if t2a.trustScore == nil {
		t2a.SetTrustScore(agentID)
	}
	// Update trust score based on auditing another agent
	t2a.trustScore[agentID] += 2
}

// ----- 2.1 Decision to send or accept a team invitiation -----
func (t2a *Team2Agent) DecideTeamForming(agentInfoList []common.ExposedAgentInfo) []uuid.UUID {
	// Initialize selected agents slice
	selectedAgents := make([]uuid.UUID, 0)

	// Set trust threshold for accepting/sending invitations
	trustThreshold := 7 // This can be adjusted based on desired behavior

	// Iterate through all agents
	for _, agentInfo := range agentInfoList {
		agentUUID := agentInfo.AgentUUID
		// Initialize trust score map if it hasn't been initialized yet
		if t2a.trustScore == nil {
			t2a.SetTrustScore(agentUUID)
		}

		// Skip if it's our own ID
		if agentUUID == t2a.GetID() {
			continue
		}

		// Get current trust score for this agent
		trustScore, _ := t2a.trustScore[agentUUID]

		// Check if we're a leader and they're not
		if t2a.rank {
			// Leaders are more selective and only accept followers with high trust
			if trustScore >= trustThreshold {
				selectedAgents = append(selectedAgents, agentUUID)
			}
			continue
		}

		// If we're not a leader (follower), be more open to invitations
		// Accept/send invitation if trust score is above threshold
		if trustScore > trustThreshold {
			selectedAgents = append(selectedAgents, agentUUID)
		}

	}

	return selectedAgents
}

// ----- 2.2 Decision to stick -----

func (t2a *Team2Agent) StickorAgain() {

}

// ----- 2.3 Decision to cheat / not cooperate

func (t2a *Team2Agent) DecideContributionCheating() {

}

func (t2a *Team2Agent) DecideWithdrawal() {

}

func (t2a *Team2Agent) GetStatedContribution() {

}

func (t2a *Team2Agent) GetStatedWithdrawal() {

}

// 2.4 ----- Decision to Audit Someone

func (t2a *Team2Agent) GetContributionAuditVote() common.Vote {

	// 1: Setup

	// experiment with these;
	var auditThreshold int = 5   // decision to audit based on if an agents trust score is lower than this
	var suspicionFactor int = 2  // how much we lower everyone's trust scores if there is a discrepancy.
	var discrepancyThreshold = 4 // if discrepancy between stated and actual common pool is greater than this, lower trust scores.

	// get list of uuids in our team
	var agentsInTeam []uuid.UUID = t2a.server.GetAgentsInTeam(t2a.teamID)

	// 2: Main logic

	// get the actual size of common pool post contributions, and the supposed size based on what agents have stated about their contributions.
	// compare them to find the discrepancy.
	var actualCommonPoolSize = t2a.server.GetTeam(t2a.GetID()).GetCommonPool()
	var supposedCommonPoolSize = 0
	for _, agentID := range agentsInTeam {
		// TODO:
		// get the agents stated contribution
		// increment the supposed common pool size.
	}
	var discrepancy int = supposedCommonPoolSize - actualCommonPoolSize

	// if there is a significant discrepancy, decrement all your teams trust scores by a suspicion factor.
	// then check to see if the least trusted agent in your team is below the threshold
	if discrepancy > discrepancyThreshold {

		// decrement all team trust scores
		for _, agentID := range agentsInTeam {
			t2a.trustScore[agentID] -= suspicionFactor
		}

		var lowestTrustScore int = math.MaxInt
		var lowestAgent uuid.UUID

		// find the agent with the lowest trust score.
		for _, agentID := range agentsInTeam {
			agentTrustScore := t2a.trustScore[agentID]

			if agentTrustScore < lowestTrustScore {
				lowestAgent = agentID
				lowestTrustScore = agentTrustScore
			}
		}

		// if the lowest agent is below the threshold, submit a vote for them
		// if they still aren't, abstain.
		if lowestTrustScore < auditThreshold {
			// 1 means vote for audit of this person
			return common.CreateVote(1, t2a.GetID(), lowestAgent)
		} else {
			// 0 means abstain / no preference
			return common.CreateVote(0, t2a.GetID(), uuid.Nil)
		}
	} else {
		// in this case there is no discrepancy this round, so prefer not audit (-1)
		return common.CreateVote(-1, t2a.GetID(), uuid.Nil)
	}

}

func (t2a *Team2Agent) GetWithdrawalAuditVote() common.Vote {
	// 1: Setup

	// experiment with these;
	var auditThreshold int = 5   // decision to audit based on if an agents trust score is lower than this
	var suspicionFactor int = 2  // how much we lower everyone's trust scores if there is a discrepancy.
	var discrepancyThreshold = 4 // if discrepancy between stated and actual common pool is greater than this, lower trust scores.

	// get list of uuids in our team
	var agentsInTeam []uuid.UUID = t2a.server.GetAgentsInTeam(t2a.teamID)

	// 2: Main logic

	// get the actual size of common pool after withdrawals, and the supposed size based on what agents have stated about their withdrawals.
	// compare them to find the discrepancy.
	var actualCommonPoolSize = t2a.server.GetTeam(t2a.GetID()).GetCommonPool()
	var supposedCommonPoolSize int // starts out as the size of the common pool before withdrawals
	for _, agentID := range agentsInTeam {
		// TODO:
		// get the agents stated withdrawal
		// take this away from the supposed common pool size.
	}

	var discrepancy int = supposedCommonPoolSize - actualCommonPoolSize

	// if there is a significant discrepancy, decrement all your teams trust scores by a suspicion factor.
	// then check to see if the least trusted agent in your team is below the threshold
	if discrepancy > discrepancyThreshold {

		// decrement all team trust scores
		for _, agentID := range agentsInTeam {
			t2a.trustScore[agentID] -= suspicionFactor
		}

		var lowestTrustScore int = math.MaxInt
		var lowestAgent uuid.UUID

		// find the agent with the lowest trust score.
		for _, agentID := range agentsInTeam {
			agentTrustScore := t2a.trustScore[agentID]

			if agentTrustScore < lowestTrustScore {
				lowestAgent = agentID
				lowestTrustScore = agentTrustScore
			}
		}

		// if the lowest agent is below the threshold, submit a vote for them
		// if they still aren't, abstain.
		if lowestTrustScore < auditThreshold {
			// 1 means vote for audit of this person
			return common.CreateVote(1, t2a.GetID(), lowestAgent)
		} else {
			// 0 means abstain / no preference
			return common.CreateVote(0, t2a.GetID(), uuid.Nil)
		}
	} else {
		// in this case there is no discrepancy this round, so prefer not audit (-1)
		return common.CreateVote(-1, t2a.GetID(), uuid.Nil)
	}

}

// /////////// ----------------------RANKING SYSTEM---------------------- /////////////

func (t2a *Team2Agent) GetLeaderVote() common.Vote {
	// Experiment with this - it is our threshold to decide leader worthiness
	var leaderThreshold int = 6

	// Get list of UUIDs in our team
	var agentsInTeam []uuid.UUID = t2a.server.GetAgentsInTeam(t2a.teamID)

	var highestTrustScore int = math.MinInt // Start with the minimum possible int
	var mostTrustedAgent uuid.UUID

	// Iterate over our team, finding the agent with the highest trust score
	for _, agentID := range agentsInTeam {
		agentTrustScore := t2a.trustScore[agentID]
		// Initialize trust score map if it hasn't been initialized yet
		if t2a.trustScore == nil {
			t2a.SetTrustScore(agentID)
		}

		if agentTrustScore > highestTrustScore {
			mostTrustedAgent = agentID
			highestTrustScore = agentTrustScore
		}
	}

	// If the most trusted agent is above the threshold, vote for them as leader
	if highestTrustScore > leaderThreshold {
		// 1 means vote for this agent as leader
		return common.CreateVote(1, t2a.GetID(), mostTrustedAgent)
	} else {
		// 0 means abstain / no preference
		return common.CreateVote(0, t2a.GetID(), uuid.Nil)
	}
}

func (t2a *Team2Agent) ToggleLeader() {
	t2a.rank = !t2a.rank
}

func (t2a *Team2Agent) GetRole() bool {
	return t2a.rank // If true, they are the leader
}
