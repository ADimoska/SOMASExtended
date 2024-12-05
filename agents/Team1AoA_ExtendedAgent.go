package agents

/* Contains functions relevant to agent functionality when the AoA is Team1AoA */

import (
	"log"
	"math"
	"sort"

	"github.com/google/uuid"

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
	mi.team1RankBoundaryProposals = mi.team1RankBoundaryProposals[:0]

	// Iterate over all agents and ask them for their proposals. We do not
	// store who each vote came from to enforce anonymity
	for _, agentID := range mi.Server.GetAgentsInTeam(mi.TeamID) {
		mi.SendSynchronousMessage(req, agentID)
	}
}

/**
* Generate the candidates based off the proposals provided by the agents
 */
func (mi *ExtendedAgent) team1_GenerateRankBoundaryCandidates() [3][5]int {
	// This function will take the values stored in team1RankBoundaryProposals
	// (which should have been collected just before this).
	if mi.team1RankBoundaryProposals == nil {
		log.Fatal("Tried generating candidates without asking for agent proposals! (nil pointer)")
		return [3][5]int{}
	}

	// Transpose the proposals matrix - This converts arrays of proposals of
	// each agent to arrays of proposals for each rank
	rows := len(mi.team1RankBoundaryProposals)
	cols := 5 // for 5 ranks

	// pre-allocate
	boundaries := [5][]int{}
	for i := 0; i < len(boundaries); i++ {
		boundaries[i] = make([]int, rows)
	}

	// fill
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			boundaries[j][i] = mi.team1RankBoundaryProposals[i][j]
		}
	}

	lower_candidate := [5]int{} // lower quartile of each bound
	avg_candidate := [5]int{}   // median of each bound
	upper_candidate := [5]int{} // upper quartile of each bound

	// Go through each rank boundary, and add it to the respective candidate
	for pos, b := range boundaries {
		q1, q2, q3 := CalculateQuartiles(b)
		lower_candidate[pos] = q1
		avg_candidate[pos] = q2
		upper_candidate[pos] = q3
	}

	/* There is a rare edge case here where the median for a higher quartile is
	 * lower than one for a lower quartile, if you have unprogressive or
	 * bad-actor agents. We account for this here by strictly enforcing that the
	 * options are non-decreasing. */
	for i := 1; i < 5; i++ {
		if prev := lower_candidate[i-1]; prev > lower_candidate[i] {
			lower_candidate[i] = lower_candidate[i-1]
		}
		if prev := avg_candidate[i-1]; prev > avg_candidate[i] {
			avg_candidate[i] = avg_candidate[i-1]
		}
		if prev := upper_candidate[i-1]; prev > upper_candidate[i] {
			upper_candidate[i] = upper_candidate[i-1]
		}
	}

	cands := [3][5]int{lower_candidate, avg_candidate, upper_candidate}
	return cands
}

/**
* Conduct a vote on the expressed candidates, and elect the Condorcet winner
 */
func (mi *ExtendedAgent) team1_VoteOnRankBoundaries(cands [3][5]int) [5]int {

	// Ask all candidates to vote on their preferences
	req := &common.Team1BoundaryBallotRequestMessage{
		BaseMessage: mi.CreateBaseMessage(),
		Candidates:  cands,
	}

	mi.team1Ballots = mi.team1Ballots[:0] // clear previous votes

	// Collect all the ballots and store them
	for _, agentID := range mi.Server.GetAgentsInTeam(mi.TeamID) {
		mi.SendSynchronousMessage(req, agentID)
	}

	// Compute pairwise relations for Condorcet winner algorithm
	pairwise := [3][3]int{
		{0, 0, 0},
		{0, 0, 0},
		{0, 0, 0},
	}

	for _, vote := range mi.team1Ballots {
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				if vote[i] > vote[j] {
					pairwise[i][j]++
				}
			}
		}
	}

	condorcet := -1

	// Check for a Condorcet winner
	for i := 0; i < 3; i++ {
		winner := true
		for j := 0; j < 3; j++ {
			if i != j && pairwise[i][j] <= pairwise[j][i] {
				winner = false
				break
			}
		}
		if winner {
			condorcet = i
			break
		}
	}

	/* If there is no condorcet winner, we return the median. This is deemed
	 * acceptable by voluntary association - Using a mean here could be
	 * influenced heavily by outliers */
	if condorcet != -1 {
		log.Printf("Chair %v identified Condorcet winner %v", mi.GetID(), cands[condorcet])
		return cands[condorcet]
	} else {
		// No condorcet winner, return the median.
		log.Printf("Chair %v could not compute a Condorcet winner. Defaulting to median %v", mi.GetID(), cands[1])
		return cands[1] // median
	}
}

/**
* Calculate the median, upper and lower quartiles for a range of data. This is
* used for generating candidates based on individual rank boundaries provided
* by agents. Calculations are done based on floats but results are rounded to
* integers before being returned.
 */
func CalculateQuartiles(values []int) (q1, q2, q3 int) {
	n := len(values)
	sort.Ints(values)

	// Calculate each quartile
	q1 = calculateMedian(values[:n/2]) // lower quartile
	q2 = calculateMedian(values)       // median

	/* Upper quartile needs a little more thought, since you need to account
	   for the case where the size is even. This is not a problem for the lower
	   quartile because n/2 gets floored down if it is a float. But in this case
	   we need to think whether we consider the median or not. */

	if n%2 == 0 {
		q3 = calculateMedian(values[n/2:])
	} else {
		q3 = calculateMedian(values[n/2+1:])
	}

	return
}

/**
* Calculate the median of a set of values. Returns an integer, rounding to the
* nearest whole number (not truncation)
 */
func calculateMedian(sortedValues []int) int {
	n := len(sortedValues)
	var median float64 = 0.0
	if n%2 == 0 {
		median = float64(sortedValues[n/2-1]+sortedValues[n/2]) / 2.0
	} else {
		median = float64(sortedValues[n/2])
	}
	return int(math.Round(median))
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

/**
* BASE IMPLEMENTATION - Always vote for the median, then the lower quartile,
* then the upper quartile.
* OVERRIDE - For the 3 candidates provided, give your preference for each one.
* Use the integers 1, 2, and 3 to represent the 1st, 2nd and 3rd quartiles
* (they will be provided in order by the chair)
 */
func (mi *ExtendedAgent) Team1_BoundaryBallotRequestHandler(msg *common.Team1BoundaryBallotRequestMessage) {
	resp := &common.Team1BoundaryBallotResponseMessage{
		BaseMessage:      mi.CreateBaseMessage(),
		RankedCandidates: [3]int{2, 1, 3},
	}
	mi.SendSynchronousMessage(resp, msg.GetSender())
}

func (mi *ExtendedAgent) Team1_BoundaryProposalResponseHandler(msg *common.Team1RankBoundaryResponseMessage) {
	log.Printf("Chair %v received rank boundary proposal %v from %v", mi.GetID(), msg.Bounds, msg.GetSender())
	mi.team1RankBoundaryProposals = append(mi.team1RankBoundaryProposals, msg.Bounds)
	mi.SignalMessagingComplete()
}

func (mi *ExtendedAgent) Team1_BoundaryBallotResponseHandler(msg *common.Team1BoundaryBallotResponseMessage) {
	log.Printf("Chair %v received ballot %v from %v", mi.GetID(), msg.RankedCandidates, msg.GetSender())
	mi.team1Ballots = append(mi.team1Ballots, msg.RankedCandidates)
	mi.SignalMessagingComplete()
}

// Test functions expose non-exported functions for testing purposes
func (mi *ExtendedAgent) TestableGatherFunc() func() {
	return mi.team1_GatherRankBoundaryProposals
}

func (mi *ExtendedAgent) TestableCandidateGenFunc(proposals [][5]int) func() [3][5]int {
	mi.team1RankBoundaryProposals = proposals
	return mi.team1_GenerateRankBoundaryCandidates
}
