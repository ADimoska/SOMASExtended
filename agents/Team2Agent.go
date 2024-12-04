package agents

import (
	"log"
	"math"
	"sort"

	// "math/rand"

	common "github.com/ADimoska/SOMASExtended/common"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
)

// this is the third tier of composition - embed the extended agent and add 'user specific' fields
type Team2Agent struct {
	*ExtendedAgent
	rank               bool
	AgentType          string
	trustScore         map[uuid.UUID]int
	strikeCount        map[uuid.UUID]int
	statedContribution map[uuid.UUID]int
	statedWithdrawal   map[uuid.UUID]int
	thresholdBounds    []int
	commonPoolEstimate int
}

// constructor for team2agent - initialised as all followers
func Team2_CreateAgent(funcs agent.IExposedServerFunctions[common.IExtendedAgent], agentConfig AgentConfig) *Team2Agent {
	extendedAgent := GetBaseAgents(funcs, agentConfig)
	extendedAgent.TrueSomasTeamID = 2   // Our true team ID
	extendedAgent.AoARanking = []int{2} // just ours for now. TODO: CHANGE WHEN WE KNOW OTHER AOAs

	return &Team2Agent{
		ExtendedAgent:      extendedAgent,
		rank:               false,
		trustScore:         make(map[uuid.UUID]int),
		strikeCount:        make(map[uuid.UUID]int),
		AgentType:          "Team2Agent",
		statedContribution: make(map[uuid.UUID]int),
		statedWithdrawal:   make(map[uuid.UUID]int),
		thresholdBounds:    make([]int, 2),
		commonPoolEstimate: 0,
	}
}

// Part 1: Specialised Agent Strategy Functions

// ---------- TRUST SCORE SYSTEM ----------
func (t2a *Team2Agent) SetTrustScore(agentID uuid.UUID) {
	// Initialize trust score for this agent
	t2a.trustScore[agentID] = 70
}

func (t2a *Team2Agent) getAverageTeamTrustScore(teamID uuid.UUID) int {
	totalTrustScore := 0
	agentsInTeam := t2a.Server.GetAgentsInTeam(teamID)

	for _, agentID := range agentsInTeam {

		if _, exists := t2a.trustScore[agentID]; !exists {
			t2a.SetTrustScore(agentID)
		}

		totalTrustScore += t2a.trustScore[agentID]
	}

	numAgentsinTeam := len(agentsInTeam)
	if numAgentsinTeam == 0 {
		log.Printf("Error: No agents in team %v\n", teamID)
		return 0
	}
	averageTrustScore := totalTrustScore / numAgentsinTeam

	return averageTrustScore
}

func (t2a *Team2Agent) SetAgentContributionAuditResult(agentID uuid.UUID, result bool) {
	//apply strike and decrease trust score for agent audited
	// Check if the trust score for this agent exists; if not, initialize it to 70
	if _, exists := t2a.trustScore[agentID]; !exists {
		t2a.SetTrustScore(agentID)
	}
	var penalty int

	//increasing audit result for everyone else
	agentsInTeam := t2a.Server.GetAgentsInTeam(t2a.TeamID)

	if result {
		t2a.strikeCount[agentID]++ // Increment the strike count for this agent

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

		for _, agent := range agentsInTeam {
			if agent != agentID {
				t2a.trustScore[agent] += 10
			}
		}
	}

	// TO-DO: increase trust score for agents that were not audited

}

func (t2a *Team2Agent) SetAgentWithdrawalAuditResult(agentID uuid.UUID, result bool) {
	//apply strike and decrease trust score for agent audited
	// Check if the trust score for this agent exists; if not, initialize it to 70
	if _, exists := t2a.trustScore[agentID]; !exists {
		t2a.SetTrustScore(agentID)
	}
	var penalty int
	agentsInTeam := t2a.Server.GetAgentsInTeam(t2a.TeamID)

	if result {
		t2a.strikeCount[agentID]++ // Increment the strike count for this agent

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

		for _, agent := range agentsInTeam {
			if agent != agentID {
				t2a.trustScore[agent] += 10
			}
		}
	}

	// TO-DO: increase trust score for agents that were not audited
}

// ----------- RANKING SYSTEM ----------

func (t2a *Team2Agent) Team2_GetLeaderVote() common.Vote {
	// Experiment with this - it is our threshold to decide leader worthiness
	var leaderThreshold int = 60

	// Get list of UUIDs in our team
	var agentsInTeam []uuid.UUID = t2a.Server.GetAgentsInTeam(t2a.TeamID)

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

// Part 2: Core Game Flow Functions

// ---------- TEAM FORMING ----------
func (t2a *Team2Agent) DecideTeamForming(agentInfoList []common.ExposedAgentInfo) []uuid.UUID {

	// trustThreshold := 10 // invite agents who are above this threshold
	// invitationList := []uuid.UUID{}
	// // Iterate through all agents and add those with
	// for _, agentInfo := range agentInfoList {
	// 	agentUUID := agentInfo.AgentUUID
	// 	// Initialize trust score map if it hasn't been initialized yet
	// 	if t2a.trustScore[agentUUID] == 0 {
	// 		t2a.SetTrustScore(agentUUID)
	// 	}

	// 	// Skip if it's our own ID
	// 	if agentUUID == t2a.GetID() {
	// 		continue
	// 	}

	// 	// Get current trust score for this agent
	// 	trustScore := t2a.trustScore[agentUUID]

	// 	// If agent is above threshold, append to invite list
	// 	if trustScore > trustThreshold {
	// 		invitationList = append(invitationList, agentUUID)
	// 	}
	// }

	// return invitationList
	return t2a.ExtendedAgent.DecideTeamForming(agentInfoList)
}

func (t2a *Team2Agent) HandleTeamFormationMessage(msg *common.TeamFormationMessage) {
	log.Printf("Agent %s received team forming invitation from %s\n", t2a.GetID(), msg.GetSender())

	// Already in a team - reject invitation
	if t2a.TeamID != (uuid.UUID{}) {
		if t2a.VerboseLevel > 6 {
			log.Printf("Agent %s rejected invitation from %s - already in team %v\n",
				t2a.GetID(), msg.GetSender(), t2a.TeamID)
		}
		return
	}

	sender := msg.GetSender()
	// Set the trust score if there is no previous record of this agent
	if _, ok := t2a.trustScore[sender]; !ok {
		t2a.SetTrustScore(sender)
	}

	t2a.sendOpinionMessages(sender, 3) // Ask our top 3 most trusted agent about their opinions of our current agent

	// Get the sender's trust score
	senderTrustScore := t2a.trustScore[sender]

	if senderTrustScore > 60 {
		// Handle team creation/joining based on sender's team status
		sender := msg.GetSender()
		if t2a.Server.CheckAgentAlreadyInTeam(sender) {
			existingTeamID := t2a.Server.AccessAgentByID(sender).GetTeamID()
			t2a.joinExistingTeam(existingTeamID)
		} else {
			t2a.createNewTeam(sender)
		}
	} else {
		log.Printf("Agent %s rejected invitation from %s - already in team %v\n",
			t2a.GetID(), msg.GetSender(), t2a.TeamID)
	}
}

func (t2a *Team2Agent) HandleAgentOpinionRequestMessage(msg *common.AgentOpinionRequestMessage) {
	// Respond to opinion requests from other agents
	agentID, trustScore := msg.AgentID, 0

	if _, ok := t2a.trustScore[agentID]; ok {
		t2a.SetTrustScore(agentID)
		return // Ignore if you don't have an opinion, but set it in the map
	}

	trustScore = t2a.trustScore[agentID]
	agentOpinionResponseMessage := t2a.CreateAgentOpinionResponseMessage(agentID, trustScore)

	t2a.SendMessage(agentOpinionResponseMessage, msg.GetSender()) // Respond to the person asking with a trust score, asynchronously
}

func (t2a *Team2Agent) HandleAgentOpinionResponseMessage(msg *common.AgentOpinionResponseMessage) {
	agentID, trustScore := msg.AgentID, msg.AgentOpinion

	if _, ok := t2a.trustScore[agentID]; !ok {
		t2a.SetTrustScore(agentID)
		return
	}

	// Update trust score to be the weighted average of the two opinions
	t2a.trustScore[agentID] = ((trustScore * 2) + t2a.trustScore[agentID]) / 3
}

// ---------- VOTE ON ORPHANS ----------

func (t2a *Team2Agent) VoteOnAgentEntry(candidateID uuid.UUID) bool {
	// Return true to accept them, false to not accept them.

	acceptOrphanThreshold := 20 // low as we want to accept orphans.

	if t2a.trustScore[candidateID] > acceptOrphanThreshold {
		return true
	} else {
		return false
	}
}

// ---------- DECISION TO STICK  ----------

// func (t2a *Team2Agent) StickOrAgainFor(agentId uuid.UUID, accumulatedScore int, prevRoll int) int {
// 	return 0
// }

// Function to retrieve ID and Score of all dead agents in team
func (t2a *Team2Agent) GetDeadTeammates() []struct {
	AgentID uuid.UUID
	Score   int
} {
	// Slice to store dead agents' information
	deadTeammates := make([]struct {
		AgentID uuid.UUID
		Score   int
	}, 0)

	// Get the IDs of agents in the same team
	for _, agentID := range t2a.Server.GetAgentsInTeam(t2a.TeamID) {
		// Skip the agent is the current agent or if the agent is not dead
		if t2a.GetID() != agentID && t2a.Server.IsAgentDead(agentID) {
			// Get the score of the dead agent
			score := t2a.Server.GetAgentKilledScore(agentID)

			// Append the agent's ID and score to the result slice
			deadTeammates = append(deadTeammates, struct {
				AgentID uuid.UUID
				Score   int
			}{
				AgentID: agentID,
				Score:   score,
			})
		}
	}

	return deadTeammates
}

// Function to determine the probability of improvement of the next re-roll compared to previous roll
func (t2a *Team2Agent) probabilityOfImprovement(prevRoll int) float64 {
	if prevRoll == 0 { // First roll of the iteration so guaranteed probability of improvement
		return 1
	}

	// Probabilities of sums with 3 dice (precomputed distribution of 3d6 outcomes)
	probabilities := map[int]float64{
		3: 1.0 / 216, 4: 3.0 / 216, 5: 6.0 / 216, 6: 10.0 / 216,
		7: 15.0 / 216, 8: 21.0 / 216, 9: 25.0 / 216, 10: 27.0 / 216,
		11: 27.0 / 216, 12: 25.0 / 216, 13: 21.0 / 216, 14: 15.0 / 216,
		15: 10.0 / 216, 16: 6.0 / 216, 17: 3.0 / 216, 18: 1.0 / 216,
	}

	// Calculate the cumulative probability of rolling higher than `prevRoll`
	cumulativeProbability := 0.0
	for outcome := prevRoll + 1; outcome <= 18; outcome++ {
		cumulativeProbability += probabilities[outcome]
	}

	return cumulativeProbability

}

// Function to determine risk tolerance which determines how risk averse or risky agent should be
// Risk tolerance is based on trust scores and accumulated score up till current roll
func (t2a *Team2Agent) DetermineRiskTolerance(accumulatedScore int) float64 {

	// Current team size
	agentCount := 0
	for range t2a.Server.GetAgentsInTeam(t2a.TeamID) {
		agentCount += 1
	}

	// Determine risk tolerance from trust scores of other agents in the team
	totalTrust := 0
	for _, score := range t2a.trustScore {
		totalTrust += score
	}

	// Scale the average trust to between 0 - 1
	averageScaledTrust := (float64(totalTrust) / float64(agentCount)) / 100.0

	// If very high trust score for other agents then less likely agents will cheat so agent does NOT need to over-compensate to the common pool so can be risk averse so riskTolerance is lower.
	// If very low trust score for other agents then highly likely agents will cheat so agent needs to over-compensate the common pool so must be risky so riskTolerance is higher.
	riskToleranceFromTrust := 1.0 - averageScaledTrust

	// Use accumulatedScore to determine risk tolerance
	// If high accumulated score then more risk averse
	// If low accumulated score can be more risky
	riskToleranceFromAccumScore := 1.0 / (float64(accumulatedScore) / 10.0)

	// Overall risk tolerance is equal parts: riskToleranceFromTrust, riskToleranceFromAccumScore
	return (riskToleranceFromTrust + riskToleranceFromAccumScore) / 2.0
}

// Objective of StickOrAgain is to maximize score after n turns for each agent
func (t2a *Team2Agent) StickOrAgain(accumulatedScore int, prevRoll int) bool {

	log.Printf("*****Total Score before deciding to re-roll or stick: %d\n", accumulatedScore)

	log.Printf("*****Prev Roll: %d\n", prevRoll)

	// Determine cumulative probability of improvement
	cumulativeProbability := t2a.probabilityOfImprovement(prevRoll)
	log.Printf("*****Cumulative Probability of Improvement: %.2f\n", cumulativeProbability)

	log.Printf("*****Rank is: %t\n", t2a.rank) // true is leader, false is citizen

	// Determine agent risk tolerance (lower riskTolerance value indicates more risky, higher indicates more risk averse)
	riskTolerance := t2a.DetermineRiskTolerance(accumulatedScore)
	log.Printf("*****Risk Tolerance: %.2f\n", riskTolerance)

	if t2a.rank { // Leader is very risky and has a fixed riskTolerance of 0.8
		riskTolerance = 0.8
		threshold := float64(prevRoll) * (1.0 - riskTolerance)
		if (cumulativeProbability * 18) > threshold {
			log.Printf("*****Decision: Re-roll\n")
			return false // Re-roll
		}
	} else { // Citizens have risk given by a dynamic riskTolerance
		// Scale prevRoll value with risk tolerance to give threshold
		// If high risk tolerance then lower threshold hence likely to re-roll
		// If low risk tolerance then higher threshold hence less likely to re-roll
		threshold := float64(prevRoll) * (1.0 - riskTolerance)
		log.Printf("*****Citizen threshold: %d\n", threshold)
		log.Printf("*****Cumulative Probability * 18: %d\n", (cumulativeProbability * 18))
		if (cumulativeProbability * 18) > threshold {
			log.Printf("*****Decision: Re-roll\n")
			return false // Re-roll
		}
	}

	log.Printf("*****Decision: Stick\n")
	return true // Stick
}

// Function guesses threshold based on highest dead agent score and current agent score
// Current agent score must be before contribution or withdrawal to common pool
// This ensures current agent (which is alive) has score > threshold in current iteration
// Hence call ThresholdGuessStrategy after application of threshold (before contribution/withdrawal to comomon pool)

// Usage: Can use ThresholdGuessStrategy when contributing or withdrawing from common pool to inform how much to contribute/withdraw
// If current agent's score is significantly above threshold guess then can contribute more
// If current agent's score is significantly below threshold guess then can withdraw more
func (t2a *Team2Agent) ThresholdGuessStrategy() int {
	// If no dead teammates, no guess can be made
	// Initially assume threshold is very high this ensures all agents try to contribute little
	// Ensures agents can survive until threshold is applied and dead agents can be used to guess threshold from
	initialThresholdGuess := 10000

	deadTeammates := t2a.GetDeadTeammates()

	if len(deadTeammates) == 0 {
		return initialThresholdGuess // No valid threshold guess for now
	}

	// Find the highest score among dead teammates
	maxDeadScore := 0
	for _, deadID := range deadTeammates {
		if deadID.Score > maxDeadScore {
			maxDeadScore = deadID.Score
		}
	}

	// Use current agent true score
	agentAliveScore := t2a.GetTrueScore()

	// Calculate the new threshold guess by taking the midpoint of maxDeadScore and agentAliveScore
	// Add a margin of error to ensure threshold guess is above forecasted threshold
	marginOfError := 10

	thresholdGuess := ((maxDeadScore + agentAliveScore) / 2) + marginOfError
	return thresholdGuess
}

// ---------- CONTRIBUTION, WITHDRAWAL AND ASSOCIATED AUDITING ----------

func (t2a *Team2Agent) GetActualContribution(instance common.IExtendedAgent) int {
	// dependent on:
	// 1. team AoA
	// 2. average team trust score

	if t2a.TeamID == (uuid.UUID{}) {
		log.Printf("Agent %s does not have a team, skipping contribution...\n", t2a.GetID())
		return 0
	}

	switch aoa := t2a.Server.GetTeam(t2a.GetID()).TeamAoA.(type) {
	case *common.Team2AoA:
		// under our own AoA, for now we just return what is expected of us.

		// get the contribution we are expected to make
		aoaExpectedContribution := aoa.GetExpectedContribution(t2a.GetID(), t2a.GetTrueScore())

		// if we have less than the expected, just contribute whats left
		if t2a.GetTrueScore() < aoaExpectedContribution {
			return t2a.GetTrueScore() // give all score if less than expected
		}

		return aoaExpectedContribution

	default:
		// under other aoas, adapt based on the average team trust score

		// get the contribution we are expected to make
		aoaExpectedContribution := aoa.GetExpectedContribution(t2a.GetID(), t2a.GetTrueScore())

		// if we have less than the expected, just contribute whats left
		if t2a.GetTrueScore() < aoaExpectedContribution {
			return t2a.GetTrueScore() // give all score if less than expected
		}

		// otherwise, look at the average team trust score and base contribution decision on this.
		if t2a.getAverageTeamTrustScore(t2a.TeamID) > 60 {
			return aoaExpectedContribution
		} else {
			return int(0.8 * float64(aoaExpectedContribution)) // contribute less if we don't trust our team very much.
		}
	}
}

// func (t2a *Team2Agent) GetStatedContribution(instance common.IExtendedAgent) int {
// 	// Currently, stated contribution matches actual contribution
// 	// can edit this in the future to lie
// 	statedContribution := instance.DecideContribution()
// 	return statedContribution
// }

func (t2a *Team2Agent) HandleContributionMessage(msg *common.ContributionMessage) {
	t2a.ExtendedAgent.HandleContributionMessage(msg) // Call extendedagent version to enable logging

	// updating our agents "mind":

	// store this agents stated contribution in our map for use in deciding who to audit
	t2a.statedContribution[msg.GetSender()] = msg.StatedAmount
	// increment the common pool estimate by the stated amount
	t2a.commonPoolEstimate += msg.StatedAmount
}

func (t2a *Team2Agent) GetContributionAuditVote() common.Vote {
	// 1: Setup

	// experiment with these;
	auditThreshold := 50       // decision to audit based on if an agents trust score is lower than this
	suspicionFactor := 2       // how much we lower everyone's trust scores if there is a discrepancy.
	discrepancyThreshold := 10 // if discrepancy between stated and actual common pool is greater than this, lower trust scores.

	// get list of uuids in our team
	var agentsInTeam []uuid.UUID = t2a.Server.GetAgentsInTeam(t2a.TeamID)

	// 2: Main logic

	// Step 1: Check if there is someone obvious to audit based on stated contributions
	for _, agentID := range agentsInTeam {

		// if somones stated contribution is ridiculously high.
		if t2a.statedContribution[agentID] > 30 {
			return common.CreateVote(1, t2a.GetID(), agentID)
		}

		// if somones stated contribution is ridiculously low.
		if t2a.statedContribution[agentID] < 2 {
			return common.CreateVote(1, t2a.GetID(), agentID)
		}

	}

	// Step 2: If there is no one obvious to audit based on stated contributions, then:
	// get the actual size of common pool post contributions, and the supposed size based on what agents have stated about their contributions.
	// compare them to find the discrepancy.
	var actualCommonPoolSize = t2a.Server.GetTeam(t2a.GetID()).GetCommonPool()
	var discrepancy int = t2a.commonPoolEstimate - actualCommonPoolSize

	// after finding discrepancy, set our common pool estimate to the actual size of the common pool in preparation for withdrawal stage
	t2a.commonPoolEstimate = actualCommonPoolSize

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

// ----------

func (t2a *Team2Agent) GetActualWithdrawal(instance common.IExtendedAgent) int {
	// dependent on:
	// 1. team AoA
	// 2. average team trust score

	if t2a.TeamID == (uuid.UUID{}) {
		log.Printf("Agent %s does not have a team, skipping withdrawal...\n", t2a.GetID())
		return 0
	}

	commonPool := t2a.Server.GetTeam(t2a.GetID()).GetCommonPool()

	switch aoa := t2a.Server.GetTeam(t2a.GetID()).TeamAoA.(type) {
	case *common.Team2AoA:
		// under our own AoA, for now we just withdraw what is expected of us.

		aoaExpectedWithdrawal := aoa.GetExpectedWithdrawal(t2a.GetID(), t2a.GetTrueScore(), commonPool)
		if commonPool < aoaExpectedWithdrawal {
			return commonPool
		}
		return aoaExpectedWithdrawal

	default:
		// under other aoas, adapt based on the average team trust score

		aoaExpectedWithdrawal := aoa.GetExpectedWithdrawal(t2a.GetID(), t2a.GetTrueScore(), commonPool)
		if commonPool < aoaExpectedWithdrawal {
			return commonPool
		}

		// otherwise, look at the average team trust score and base withdrawal decision on this.
		if t2a.getAverageTeamTrustScore(t2a.TeamID) > 60 {
			return aoaExpectedWithdrawal
		} else {
			return int(1.2 * float64(aoaExpectedWithdrawal)) // withdraw more if our team is untrustworthy (cover ourselves)
		}
	}
}

// func (t2a *Team2Agent) GetStatedWithdrawal(instance common.IExtendedAgent) int {
// 	// Currently, stated withdrawal matches actual withdrawal
// 	// can edit this in the future to lie
// 	statedWithdrawal := instance.DecideWithdrawal()
// 	return statedWithdrawal
// }

func (t2a *Team2Agent) HandleWithdrawalMessage(msg *common.WithdrawalMessage) {
	t2a.ExtendedAgent.HandleWithdrawalMessage(msg) // Call extendedagent version to enable logging

	// updating our agents "mind":

	// store this agents stated withdrawal in our map for use in deciding who to audit
	t2a.statedWithdrawal[msg.GetSender()] = msg.StatedAmount
	// decrement the common pool estimate by the stated amount
	t2a.commonPoolEstimate -= msg.StatedAmount
}

func (t2a *Team2Agent) GetWithdrawalAuditVote() common.Vote {
	// 1: Setup

	// experiment with these;
	auditThreshold := 50       // decision to audit based on if an agents trust score is lower than this
	suspicionFactor := 2       // how much we lower everyone's trust scores if there is a discrepancy.
	discrepancyThreshold := 10 // if discrepancy between stated and actual common pool is greater than this, lower trust scores.

	// get list of uuids in our team
	var agentsInTeam []uuid.UUID = t2a.Server.GetAgentsInTeam(t2a.TeamID)

	// 2: Main logic

	// Step 1: Check if there is someone obvious to audit based on stated withdrawals
	for _, agentID := range agentsInTeam {

		// if someones stated withdrawal is ridiculously high. (agent may be stupid and greedy)
		if t2a.statedWithdrawal[agentID] > 30 {
			return common.CreateVote(1, t2a.GetID(), agentID)
		}

		// if somones stated withdrawal is ridiculously low. (agent is likely trying to decieve)
		if t2a.statedWithdrawal[agentID] < 2 {
			return common.CreateVote(1, t2a.GetID(), agentID)
		}

	}

	// get the actual size of common pool after withdrawals, and the supposed size based on what agents have stated about their withdrawals.
	// compare them to find the discrepancy.
	var actualCommonPoolSize = t2a.Server.GetTeam(t2a.GetID()).GetCommonPool()
	var discrepancy int = t2a.commonPoolEstimate - actualCommonPoolSize

	// reset to commonpoolestimate after withdrawal
	t2a.commonPoolEstimate = t2a.Server.GetTeam(t2a.GetID()).GetCommonPool()

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

// Send opinion messages to the top n most trusted agents
func (t2a *Team2Agent) sendOpinionMessages(agentID uuid.UUID, numAgents int) {
	// Convert to different form for ease of sorting
	type entry struct {
		Key   uuid.UUID
		Value int
	}

	entries := make([]entry, 0, len(t2a.trustScore))

	for id, score := range t2a.trustScore {
		entries = append(entries, entry{Key: id, Value: score})
	}

	// Sort in decreasing order of trust score (most trusted first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Value > entries[j].Value
	})

	if numAgents > len(entries) {
		numAgents = len(entries)
	}

	// Create an opinion request for the given agent
	agentOpinionMessage := t2a.CreateAgentOpinionRequestMessage(agentID)

	for i := 0; i < numAgents; i++ {
		receiver := entries[i].Key
		if receiver == agentID {
			continue // Don't ask an agent for its opinion of itself
		}
		t2a.SendMessage(agentOpinionMessage, entries[i].Key)
	}
}

func (t2a *Team2Agent) GetTeamRanking() []uuid.UUID {
	log.Println("Team 2 Team ranking called!")

	teamIDs := t2a.Server.GetTeamIDs()
	ranking := make(map[uuid.UUID]int)

	// Calculate composite scores for each team
	for _, teamID := range teamIDs {
		commonPool := t2a.Server.GetTeamCommonPool(teamID)
		trustScore := t2a.getAverageTeamTrustScore(teamID)
		if trustScore == 0 {
			continue // Skip teams with no trust score -> Likely means they are empty or very bad in general
		}
		ranking[teamID] = int((float64(commonPool)*2.0 + float64(trustScore)) / 3.0)
	}

	// Convert the map to a slice of key-value pairs
	type teamScore struct {
		ID    uuid.UUID
		Score int
	}
	var teamScores []teamScore
	for id, score := range ranking {
		teamScores = append(teamScores, teamScore{ID: id, Score: score})
	}

	// Sort the slice by composite score in descending order
	sort.Slice(teamScores, func(i, j int) bool {
		return teamScores[i].Score > teamScores[j].Score
	})

	// Extract sorted team IDs
	sortedTeamIDs := make([]uuid.UUID, len(teamScores))
	for i, ts := range teamScores {
		sortedTeamIDs[i] = ts.ID
	}

	log.Println("Agent ", t2a.GetID(), " has ranked teams: ", sortedTeamIDs, " for the orphan pool")

	return sortedTeamIDs
}

// ---------- MISC TO INCORPORATE ----------

// func (t2a *Team2Agent) DecideContribution() int {
// 	// Get AoA expected contribution
// 	agentID := t2a.GetID()
// 	agentScore := t2a.trustScore[agentID]
// 	aoaContribution := t2a.Server.GetTeam(t2a.TeamID).TeamAoA.(*common.Team2AoA).GetExpectedContribution(t2a.GetID(), agentScore)

// 	// Evaluate performance
// 	// performance := t2a.EvaluatePerformance(5) // Evaluate over the last 5 rounds
// 	performance := "Great" // Performance over the last 5 rounds
// 	// Adjust contribution based on performance
// 	contribution := aoaContribution
// 	switch performance {
// 	case "Great":
// 		contribution = aoaContribution // Full contribution
// 	case "Average":
// 		if WeightedRandom("Average") {
// 			contribution = int(float64(aoaContribution) * 0.8) // 20% reduction
// 		}
// 	case "Bad":
// 		if WeightedRandom("Bad") {
// 			contribution = int(float64(aoaContribution) * 0.5) // 50% reduction
// 		}
// 	case "Terrible":
// 		if WeightedRandom("Terrible") {
// 			contribution = int(float64(aoaContribution) * 0.2) // 80% reduction
// 		}
// 	}

// 	// Ensure contribution is non-negative
// 	if contribution < 0 {
// 		contribution = 0
// 	}

// 	log.Printf("Agent %s decided to contribute: %d (Performance: %s)\n",
// 		t2a.GetID(), contribution, performance)

// 	return contribution
// }

// func (t2a *Team2Agent) DecideWithdrawal() int {
// 	// Agent-specific variables
// 	agentID := t2a.GetID()
// 	agentScore := t2a.score
// 	commonPool := t2a.Server.GetTeam(t2a.GetID()).GetCommonPool()

// 	// Expected withdrawal from AoA
// 	aoaWithdrawal := t2a.Server.GetTeam(t2a.TeamID).TeamAoA.(*common.Team2AoA).GetExpectedWithdrawal(t2a.GetID(), agentScore, commonPool)
// 	// performance := t2a.EvaluatePerformance(5) // Evaluate over the last 5 rounds
// 	// Evaluate performance
// 	performance := "Great" // Performance over the last 5 rounds

// 	// Base withdrawal starts from AoA expectation
// 	baseWithdrawal := aoaWithdrawal
// 	switch performance {
// 	case "Great":
// 		baseWithdrawal = int(float64(aoaWithdrawal) * 0.5) // Withdraw less if performing well
// 	case "Bad":
// 		baseWithdrawal = int(float64(aoaWithdrawal) * 1.2) // Withdraw more if performing poorly
// 	case "Terrible":
// 		baseWithdrawal = int(float64(aoaWithdrawal) * 1.75) // Withdraw much more if struggling
// 	}

// 	// Adjust for trust level
// 	trust := t2a.trustScore[agentID]
// 	trustModifier := 1.0
// 	if trust > 7 {
// 		trustModifier = 0.9 // High trust => more cooperative
// 	} else if trust < 3 {
// 		trustModifier = 1.2 // Low trust => more selfish
// 	}

// 	// Adjust for team size
// 	teamAgents := t2a.Server.GetAgentsInTeam(t2a.TeamID) // Get agents in the team
// 	teamSize := len(teamAgents)                          // Calculate team size
// 	teamSizeModifier := 1.0                              // Default modifier

// 	if teamSize > 5 {
// 		teamSizeModifier = 0.8 // Larger teams => scale down withdrawal
// 	} else if teamSize <= 3 {
// 		teamSizeModifier = 1.2 // Smaller teams => scale up withdrawal
// 	}

// 	// Calculate final withdrawal amount
// 	finalWithdrawal := int(float64(baseWithdrawal) * trustModifier * teamSizeModifier)

// 	log.Printf("Agent %s decided to withdraw: %d (AoA: %d, Performance: %s, Trust: %.2f, TeamSize: %d)\n",
// 		agentID, finalWithdrawal, aoaWithdrawal, performance, trust, teamSize)

// 	return finalWithdrawal
// }

// // EvaluatePerformance calculates the agent's performance relative to the team
// func (t2a *Team2Agent) EvaluatePerformance(rounds int) string {
// 	// Calculate the agent's average performance
// 	agentID := t2a.GetID()
// 	agentTotal := 0
// 	for i := len(t2a.rollHistory[agentID]) - 1; i >= 0 && i >= len(t2a.rollHistory[agentID])-rounds; i-- {
// 		agentTotal += t2a.rollHistory[agentID][i]
// 	}
// 	agentAvg := float64(agentTotal) / float64(rounds)

// 	// Calculate the team's overall average performance
// 	teamTotal, totalRounds := 0, 0
// 	for _, history := range t2a.rollHistory {
// 		for i := len(history) - 1; i >= 0 && i >= len(history)-rounds; i-- {
// 			teamTotal += history[i]
// 			totalRounds++
// 		}
// 	}
// 	teamAvg := float64(teamTotal) / float64(totalRounds)

// 	// Categorize performance
// 	if agentAvg > teamAvg*1.2 {
// 		return "Great"
// 	} else if agentAvg >= teamAvg*0.8 {
// 		return "Average"
// 	} else if agentAvg >= teamAvg*0.5 {
// 		return "Bad"
// 	} else {
// 		return "Terrible"
// 	}
// }

// // WeightedRandom returns true if the agent decides to reduce their contribution
// func WeightedRandom(category string) bool {
// 	probabilities := map[string]float64{
// 		"Great":    0.01, // 0% chance to reduce
// 		"Average":  0.1,  // 20% chance to reduce
// 		"Bad":      0.25, // 60% chance to reduce
// 		"Terrible": 0.5,  // 90% chance to reduce
// 	}
// 	randVal := rand.Float64()
// 	return randVal < probabilities[category]
// }
