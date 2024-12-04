package agents

/* Contains functions relevant to agent functionality when the AoA is Team1AoA */

import (
	"github.com/google/uuid"
	"log"

	common "github.com/ADimoska/SOMASExtended/common"
)

func (mi *ExtendedAgent) Team1_ChairUpdateRanks(currentRanking map[uuid.UUID]int) map[uuid.UUID]int {
	// Chair iterates through existing rank map in team
	// and gets the new ranks of the agents in the team
	// according to AoA function
	newRanking := make(map[uuid.UUID]int)
	for agentUUID := range currentRanking {
		newRank := mi.Server.GetTeam(agentUUID).TeamAoA.(*common.Team1AoA).GetAgentNewRank(agentUUID)
		newRanking[agentUUID] = newRank
	}

	// Returns a map of agent UUIDs to new Rank (int)
	return newRanking
}

/**
* Agrees the rank boundaries as a social decision. This is called as part of
* pre-roll logic, using various helper functions below. This function will only
* be called by an elected chair
 */
func (mi *ExtendedAgent) Team1_AgreeRankBoundaries() [5]int {
	// Step 1. Gather boundary proposals
	mi.team1_GatherRankBoundaryProposals()

	// Step 2. Generate candidates using statistical analysis of individual
	// agent proposals, providing 3 options to choose from
	cands := mi.team1_GenerateRankBoundaryCandidates()

	// Step 3. Conduct a vote on the candidates
	elected := mi.team1_VoteOnRankBoundaries(cands)

	return elected
}

/**
* Go through all the agents, and ask them what they would like the rank
* boundaries to be. Their returned value is completely up to the agent
* strategy.
 */
func (mi *ExtendedAgent) team1_GatherRankBoundaryProposals() {

	// Same request for all agents
	req := &common.Team1RankBoundaryRequestMessage{
		BaseMessage: mi.CreateBaseMessage(),
	}

	// Clear temp variable - this is just the location that the chair will add
	// all the data to as it comes into requests
	mi.team1RankBoundaryProposals = mi.team1RankBoundaryProposals[0:]

	// Iterate over all agents and ask them for their proposals. We do not
	// store who each vote came from to enforce anonymity
	for _, agentID := range mi.Server.GetAgentsInTeam(mi.TeamID) {
		mi.SendSynchronousMessage(req, agentID)
	}
}

/**
* Generate the candidates based off the proposals expressed by the agents
 */
func (mi *ExtendedAgent) team1_GenerateRankBoundaryCandidates() [3][5]int {
	return [3][5]int{{1, 2, 3, 4, 5}, {1, 2, 3, 4, 5}, {1, 2, 3, 4, 5}}
}

/**
* Conduct a vote on the expressed candidates, and elect a winner
 */
func (mi *ExtendedAgent) team1_VoteOnRankBoundaries(cands [3][5]int) [5]int {
	// Default behaviour just returns the first candidate
	return cands[0]
}

/**
* BASE IMPLEMENTATION - Always returns the same boundaries.
* OVERRIDE - You get a say in what you think the rank boundaries should be!
 */
func (mi *ExtendedAgent) Team1_BoundaryProposalRequestHandler(msg *common.Team1RankBoundaryRequestMessage) {
	bounds := [5]int{10, 20, 30, 40, 50}

	resp := &common.Team1RankBoundaryResponseMessage{
		BaseMessage: mi.CreateBaseMessage(),
		Bounds:      bounds,
	}

	mi.SendSynchronousMessage(resp, msg.GetSender())
}

func (mi *ExtendedAgent) Team1_BoundaryProposalResponseHandler(msg *common.Team1RankBoundaryResponseMessage) {
	log.Printf("Chair %v received rank boundary proposal %v from %v", mi.GetID(), msg.Bounds, msg.GetSender())
	mi.team1RankBoundaryProposals = append(mi.team1RankBoundaryProposals, msg.Bounds)
	mi.SignalMessagingComplete()
}

// Returns the non-exported function for testing
func (mi *ExtendedAgent) TestableGatherFunc() func() {
	return mi.team1_GatherRankBoundaryProposals
}
