package agents

import (
	"fmt"
	"math"
	// "math/rand"

	common "github.com/ADimoska/SOMASExtended/common"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
)

// this is the third tier of composition - embed the extended agent and add 'user specific' fields
type Team2Agent struct {
	*ExtendedAgent
	rank               bool
	trustScore         map[uuid.UUID]int
	strikeCount        map[uuid.UUID]int
	statedWithdrawal   map[uuid.UUID]int
	statedContribution map[uuid.UUID]int
	thresholdBounds    []int
	commonPoolEstimate int
}

// constructor for team2agent - initialised as all followers
func Team2_CreateAgent(funcs agent.IExposedServerFunctions[common.IExtendedAgent], agentConfig AgentConfig) *Team2Agent {
	return &Team2Agent{
		ExtendedAgent: GetBaseAgents(funcs, agentConfig), rank: false, trustScore: make(map[uuid.UUID]int), strikeCount: make(map[uuid.UUID]int), thresholdBounds: make([]int, 2),
	}
}

// Part 1: Specialised Agent Strategy Functions

// ---------- TRUST SCORE SYSTEM ----------

func (t2a *Team2Agent) SetTrustScore(id uuid.UUID) {
	if _, exists := t2a.trustScore[id]; !exists {
		t2a.trustScore[id] = 70
	}
}

// Overall function to update one agents trust score for other agents
func (t2a *Team2Agent) UpdateTrustScore(agentID uuid.UUID, eventType string, strikeCount int) {
	auditContributionResult := t2a.Server.GetTeam(agentID).TeamAoA.GetContributionAuditResult(agentID) //fix
	auditWithdrawalResult := t2a.Server.GetTeam(agentID).TeamAoA.GetWithdrawalAuditResult(agentID)
	switch eventType {
	case "strike":
		if auditContributionResult || auditWithdrawalResult {
			t2a.ApplyStrike(agentID)
		}
	case "notAudited":
		if !auditContributionResult || !auditWithdrawalResult {
			// If the target agent was not audited
			t2a.ApplyNotAudited(agentID)
		}
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

// update if agent not audited for that round
func (t2a *Team2Agent) ApplyNotAudited(agentID uuid.UUID) {
	if t2a.trustScore == nil {
		t2a.SetTrustScore(agentID)
	}
	// Update trust score based on not being audited
	t2a.trustScore[agentID] += 2
}

// ----------- RANKING SYSTEM ----------

func (t2a *Team2Agent) GetLeaderVote() common.Vote {
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

	highestTrustScore := 0 // record the highest trust score of an agent
	invitationList := []uuid.UUID{}
	// Iterate through all agents
	for _, agentInfo := range agentInfoList {
		agentUUID := agentInfo.AgentUUID
		// Initialize trust score map if it hasn't been initialized yet
		if t2a.trustScore[agentUUID] == 0 {
			t2a.SetTrustScore(agentUUID)
		}

		// Skip if it's our own ID
		if agentUUID == t2a.GetID() {
			continue
		}

		// Get current trust score for this agent
		trustScore := t2a.trustScore[agentUUID]

		// // Check if we're a leader and they're not
		// if t2a.rank {
		// }

		// Choose agent with highest trust score
		if trustScore > highestTrustScore {
			invitationList = append(invitationList, agentUUID)
			highestTrustScore = trustScore
		}

	}

	// agent at the end of the list will be the agent with the highest trust score
	lenInviteList := len(invitationList)
	if lenInviteList == 0 {
		return []uuid.UUID{}
	}
	chosenAgent := invitationList[lenInviteList-1]
	return []uuid.UUID{chosenAgent}
}

func (t2a *Team2Agent) HandleTeamFormationMessage(msg *common.TeamFormationMessage) {
	fmt.Printf("Agent %s received team forming invitation from %s\n", t2a.GetID(), msg.GetSender())

	// Already in a team - reject invitation
	if t2a.TeamID != (uuid.UUID{}) {
		if t2a.VerboseLevel > 6 {
			fmt.Printf("Agent %s rejected invitation from %s - already in team %v\n",
				t2a.GetID(), msg.GetSender(), t2a.TeamID)
		}
		return
	}

	sender := msg.GetSender()
	// Set the trust score if there is no previous record of this agent
	if _, ok := t2a.trustScore[sender]; !ok {
		t2a.SetTrustScore(sender)
	}

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
		fmt.Printf("Agent %s rejected invitation from %s - already in team %v\n",
			t2a.GetID(), msg.GetSender(), t2a.TeamID)
	}
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

// Function to determine the expected value of the next re-roll
func (t2a *Team2Agent) CalculateExpectedValue(prevRoll int) float64 {
    if prevRoll == 0 { // First roll of the iteration
        return 10.5 // Average value of 3 dice rolls with a uniform distribution
    }

    totalProbability := 0.0

    // Probabilities of sums with 3 dice (precomputed distribution of 3d6 outcomes)
    probabilities := map[int]float64{
        3:  1.0 / 216, 4:  3.0 / 216, 5:  6.0 / 216, 6:  10.0 / 216,
        7:  15.0 / 216, 8:  21.0 / 216, 9:  25.0 / 216, 10: 27.0 / 216,
        11: 27.0 / 216, 12: 25.0 / 216, 13: 21.0 / 216, 14: 15.0 / 216,
        15: 10.0 / 216, 16: 6.0 / 216, 17: 3.0 / 216, 18: 1.0 / 216,
    }

    sumWeightedOutcomes := 0.0 // Sum of the weighted likeliness of outcomes where a bust does not occur

    // Only consider outcomes greater than prevRoll
    for outcome := t2a.LastScore + 1; outcome <= 18; outcome++ {
        prob := probabilities[outcome]
        sumWeightedOutcomes += float64(outcome) * prob
        totalProbability += prob
    }

    expectedValue := 0.0

    // Normalize to determine expected value
    if totalProbability > 0 {
        expectedValue = sumWeightedOutcomes / totalProbability;
    }

    return expectedValue
}

// Function to determine risk tolerance which determines how risk averse or risky agent should be 
// Risk tolerance is based on current common pool size and trust scores
func (t2a* Team2Agent) DetermineRiskTolerance() float64 {
    // Current, actual common pool size
    actualCommonPoolSize := float64(t2a.Server.GetTeam(t2a.GetID()).GetCommonPool())

    // Current team size
    agentCount := 0
    for range t2a.Server.GetAgentsInTeam(t2a.TeamID) {
        agentCount += 1
    }

    // Ensure that each agent has at least 20 points (change accordingly) to withdraw from common pool
    minAgentWithdrawl := 20.0

    // Determine ideal common pool size based on minAgentWithdrawl
    idealCommonPoolSize := minAgentWithdrawl * float64(agentCount)

	// If very high common pool size then agent can be risk averse so riskTolerance is lower.
    // If very low common pool size then agent must be more risky so riskTolerance is higher.
    riskToleranceFromPoolSize := 1.0 - min((actualCommonPoolSize / idealCommonPoolSize), 1.0) 

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

    // Overall risk tolerance is equal parts: riskToleranceFromPoolSize, riskToleranceFromTrust
    return (riskToleranceFromPoolSize + riskToleranceFromTrust) / 2.0
}

// Objective of StickOrAgain is to maximize score after n turns for each agent
func (t2a *Team2Agent) StickOrAgain(accumulatedScore int, prevRoll int) bool {

    // Calculate the expected value of the current roll
    expectedValue := t2a.CalculateExpectedValue(prevRoll)

	// Determine agent risk tolerance
    riskTolerance := t2a.DetermineRiskTolerance()

	// Scale the expected value with risk tolerance
	// If high risk tolerance then more likely to re-roll
	// If low risk tolerance less likely to re-roll
    if (expectedValue * riskTolerance) > float64(prevRoll) {
        return false // Re-roll
    }
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
        return initialThresholdGuess; // No valid threshold guess for now
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

func (t2a *Team2Agent) DecideContribution() int {
	
	switch aoa := t2a.Server.GetTeam(t2a.GetID()).TeamAoA.(type) {
		case *common.Team2AoA:
			// under our aoa contribute as defined in the aoa.
			aoaExpectedContribution := aoa.GetExpectedContribution(t2a.GetID(), t2a.GetTrueScore())
			// double check if score in agent is sufficient (this should be handled by AoA though)
			if t2a.GetTrueScore() < aoaExpectedContribution {
				return t2a.GetTrueScore() // give all score if less than expected
			}
			return aoaExpectedContribution
		default:
			// TODO: under other aoas, follow the trsut score system. leaders and followers act differently.
			return 0
	}
}

func (t2a *Team2Agent) GetStatedContribution(instance common.IExtendedAgent) int {
	// Currently, stated contribution matches actual contribution
	// can edit this in the future to lie
	statedContribution := instance.DecideContribution()
	return statedContribution
}

func (t2a *Team2Agent) HandleContributionMessage(msg *common.ContributionMessage) {
	// TODO: Adjust suspicion based on the contribution of this agent and the AoA

	// Call the underlying function
	// fmt.Println("Overriding contribution message!")
	t2a.ExtendedAgent.HandleContributionMessage(msg) // Enables logging

	// increment the common pool estimate by the stated amount
	t2a.commonPoolEstimate += msg.StatedAmount
}

func (t2a *Team2Agent) GetContributionAuditVote() common.Vote {
	// 1: Setup

	// experiment with these;
	auditThreshold := 50       // decision to audit based on if an agents trust score is lower than this
	suspicionFactor := 20      // how much we lower everyone's trust scores if there is a discrepancy.
	discrepancyThreshold := 40 // if discrepancy between stated and actual common pool is greater than this, lower trust scores.

	// get list of uuids in our team
	var agentsInTeam []uuid.UUID = t2a.Server.GetAgentsInTeam(t2a.TeamID)

	// 2: Main logic

	// get the actual size of common pool post contributions, and the supposed size based on what agents have stated about their contributions.
	// compare them to find the discrepancy.
	var actualCommonPoolSize = t2a.Server.GetTeam(t2a.GetID()).GetCommonPool()
	var discrepancy int = t2a.commonPoolEstimate - actualCommonPoolSize

	// after finding discrepancy, reset common pool estimate to the actual size of the common pool in preparation for withdrawal stage
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

func (t2a *Team2Agent) DecideWithdrawal() int {
	// MVP: contribute exactly as defined in AoA

	// if we have a an aoa (expected case) ...
	if t2a.Server.GetTeam(t2a.GetID()).TeamAoA != nil {
		// double check if score in agent is sufficient (this should be handled by AoA though)
		commonPool := t2a.Server.GetTeam(t2a.GetID()).GetCommonPool()
		aoaExpectedWithdrawal := t2a.Server.GetTeam(t2a.GetID()).TeamAoA.GetExpectedWithdrawal(t2a.GetID(), t2a.GetTrueScore(), commonPool)
		if commonPool < aoaExpectedWithdrawal {
			return commonPool
		}
		return aoaExpectedWithdrawal
	} else {
		if t2a.VerboseLevel > 6 {
			fmt.Printf("[WARNING] Agent %s has no AoA, withdrawing 0\n", t2a.GetID())
		}
		return 0
	}
}

func (t2a *Team2Agent) GetStatedWithdrawal(instance common.IExtendedAgent) int {
	// Currently, stated withdrawal matches actual withdrawal
	// can edit this in the future to lie
	statedWithdrawal := instance.DecideWithdrawal()
	return statedWithdrawal
}

func (t2a *Team2Agent) HandleWithdrawalMessage(msg *common.WithdrawalMessage) {
	// TODO: Adjust suspicion based on the withdrawal by this agent, the AoA

	// fmt.Println("Overriding withdrawal message!")
	t2a.ExtendedAgent.HandleWithdrawalMessage(msg)

	// decrement the common pool estimate by the stated amount
	t2a.commonPoolEstimate -= msg.StatedAmount
}

func (t2a *Team2Agent) GetWithdrawalAuditVote() common.Vote {
	// 1: Setup

	// experiment with these;
	auditThreshold := 50       // decision to audit based on if an agents trust score is lower than this
	suspicionFactor := 20      // how much we lower everyone's trust scores if there is a discrepancy.
	discrepancyThreshold := 40 // if discrepancy between stated and actual common pool is greater than this, lower trust scores.

	// get list of uuids in our team
	var agentsInTeam []uuid.UUID = t2a.Server.GetAgentsInTeam(t2a.TeamID)

	// 2: Main logic

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

// 	fmt.Printf("Agent %s decided to contribute: %d (Performance: %s)\n",
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

// 	fmt.Printf("Agent %s decided to withdraw: %d (AoA: %d, Performance: %s, Trust: %.2f, TeamSize: %d)\n",
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
