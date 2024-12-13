package agents

import (
	"log"
	"math"

	common "github.com/ADimoska/SOMASExtended/common"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
)

type Team6Agent struct {
	*ExtendedAgent

	// Trust + reputation
	AgentTrust               map[uuid.UUID]float64 // Trust of other agents
	WithdrawalAuditCount     int
	WithdrawalSucessfulCount int

	// Game state tracking
	ExpectedCommonPool int
	AgentTurnScore     int

	// Personality traits
	Selfishness float64
	Greed       float64
}

func Team6_CreateAgent(funcs agent.IExposedServerFunctions[common.IExtendedAgent], agentConfig AgentConfig) *Team6Agent {
	extendedAgent := GetBaseAgents(funcs, agentConfig)
	extendedAgent.TrueSomasTeamID = 6
	extendedAgent.AoARanking = []int{6, 5, 4, 3, 2, 1}

	return &Team6Agent{
		ExtendedAgent:            extendedAgent,
		AgentTrust:               make(map[uuid.UUID]float64),
		WithdrawalAuditCount:     0,
		WithdrawalSucessfulCount: 0,

		ExpectedCommonPool: 0,
		AgentTurnScore:     0,

		Selfishness: 0.3,
		Greed:       0.6, // start higher greed = more points for voting power
	}

}

// ---------- TRUST SCORE SYSTEM ----------
func (a6 *Team6Agent) UpdateAgentTrust(agentID uuid.UUID, trustChange float64) {
	// If agent doesn't exist, initialize with 0.5 trust
	if _, exists := a6.AgentTrust[agentID]; !exists {
		a6.AgentTrust[agentID] = 0.5
	}

	// Update trust value
	a6.AgentTrust[agentID] += trustChange

	// Clamp trust value between 0 and 1
	if a6.AgentTrust[agentID] < 0 {
		a6.AgentTrust[agentID] = 0
	} else if a6.AgentTrust[agentID] > 1 {
		a6.AgentTrust[agentID] = 1
	}
}
func (a6 *Team6Agent) SetAgentContributionAuditResult(agentID uuid.UUID, result bool) {
	if result {
		a6.UpdateAgentTrust(agentID, -0.1)
	}
}

func (a6 *Team6Agent) SetAgentWithdrawalAuditResult(agentID uuid.UUID, result bool) {
	if result {
		a6.WithdrawalSucessfulCount++
		a6.UpdateAgentTrust(agentID, -0.3)
	}

	a6.WithdrawalAuditCount++
}

// ---------- DECISION TO STICK  ----------

func (a6 *Team6Agent) StickOrAgain(accumulatedScore int, prevRoll int) bool {
	// Calculate stick threshold based on agent's greed
	// Greed value influences how risk-taking an agent is:
	// - High greed means they will only stick once we reach higher score
	// - Low greed means they will stick at a lower score

	baseValue := 10.0                            // This is the base value that the agent considers to stick
	stickThreshold := baseValue * (1 + a6.Greed) // Threshold determined by baseValue and greed level

	// If the agent's turn score reaches or exceeds the stick threshold, they will stick
	willStick := accumulatedScore >= int(stickThreshold)

	return willStick
}

// ---------- CONTRIBUTION, WITHDRAWAL AND ASSOCIATED AUDITING ----------

func (a6 *Team6Agent) GetActualContribution(instance common.IExtendedAgent) int {
	if !a6.HasTeam() {
		log.Printf("Agent %s does not have a team, skipping contribution...\n", a6.GetID())
		return 0
	}

	team := a6.Server.GetTeam(a6.GetID())
	aoa := team.TeamAoA
	aoaExpectedContribution := aoa.GetExpectedContribution(a6.GetID(), a6.Score)

	contributionChoice := -1
	if a6.Selfishness >= 0.5 {
		// if this agent is relatively selfish
		// linearly map 0.5 to 1 selfishness to contribute between aoaExpected and 0
		contributionChoice = int(math.Floor(2 * (1.0 - a6.Selfishness) * float64(aoaExpectedContribution)))
	} else {
		// if this agent is relatively selfless
		// linearly map 0 to 0.5 selfishness to contribute between (whole of) TurnScore and aoaExpected
		selfishnessScaling := float64(aoaExpectedContribution-a6.AgentTurnScore) / (0.5)
		contributionChoice = int(math.Ceil(selfishnessScaling*a6.Selfishness + float64(a6.AgentTurnScore)))
	}

	return contributionChoice

}

func (a6 *Team6Agent) GetStatedContribution(instance common.IExtendedAgent) int {

	if !a6.HasTeam() {
		log.Printf("Agent %s does not have a team, skipping contribution...\n", a6.GetID())
		return 0
	}

	// Always state the expected contribution (even if cheating)
	team := a6.Server.GetTeam(a6.GetID())
	aoa := team.TeamAoA
	aoaExpectedContribution := aoa.GetExpectedContribution(a6.GetID(), a6.GetTrueScore())
	return aoaExpectedContribution
}

func (a6 *Team6Agent) HandleContributionMessage(msg *common.ContributionMessage) {
	a6.ExtendedAgent.HandleContributionMessage(msg)
	a6.ExpectedCommonPool += msg.StatedAmount
}

func (a6 *Team6Agent) GetContributionAuditVote() common.Vote {

	// Get list of uuids in our team
	var agentsInTeam []uuid.UUID = a6.Server.GetAgentsInTeam(a6.TeamID)
	actualPoolSize := a6.Server.GetTeamCommonPool(a6.TeamID)

	// if actual pool size is less than expected
	if actualPoolSize < a6.ExpectedCommonPool {
		// someone lied - trust in rest of team decreases
		for _, agentID := range agentsInTeam {
			a6.UpdateAgentTrust(agentID, -0.02)
		}

		// Vote for least trusted team member (with trust below default 0.5)
		var votedAgent uuid.UUID
		lowestTrust := 0.5

		for _, agentID := range agentsInTeam {
			if a6.AgentTrust[agentID] <= lowestTrust {
				lowestTrust = a6.AgentTrust[agentID]
				votedAgent = agentID
			}
		}
		a6.ExpectedCommonPool = actualPoolSize

		// Send out message to team members about our opinion
		// Multiple times to get them to agree with us - exploiting handle opinion
		// Maxim of Quantity
		opinion := a6.CreateAgentOpinionResponseMessage(votedAgent, int(lowestTrust*100))
		for i := 0; i < 5; i++ {
			a6.BroadcastSyncMessageToTeam(opinion)
		}

		return common.CreateVote(1, a6.GetID(), votedAgent)

	}

	// if actual pool size is equal or greater than expected
	a6.ExpectedCommonPool = actualPoolSize
	return common.CreateVote(0, a6.GetID(), uuid.Nil)

}

// Update greed value based on Current Agent Score and Average Personal Dice Roll
func (a6 *Team6Agent) UpdateGreed() {
	const (
		expectedAverage = 10.5  // Expected average for 3d6
		scoreThreshold  = 100.0 // Threshold score for greed adjustment
	)

	rollDeviation := float64(a6.AgentTurnScore) - expectedAverage
	// Negative deviation means lower roll, positive means higher
	greedChangeFromRoll := -0.05 * (rollDeviation / expectedAverage)

	scoreDeviation := scoreThreshold - float64(a6.Score)
	// Positive deviation means score is below threshold, negative means above
	normalizedScoreDeviation := scoreDeviation / scoreThreshold
	if normalizedScoreDeviation > 1.0 {
		normalizedScoreDeviation = 1.0
	} else if normalizedScoreDeviation < -1.0 {
		normalizedScoreDeviation = -1.0
	}

	// Lower score increases greed, higher score decreases it
	greedChangeFromScore := 0.05 * normalizedScoreDeviation

	newGreed := a6.Greed + greedChangeFromRoll + greedChangeFromScore

	// Clamp greed between 0.0 and 1.0
	a6.Greed = math.Max(0.0, math.Min(1.0, newGreed))

}

// Update selfishness based on Agent Dice Roll (AveragePersonalDiceRoll), Team Common Pool, and Trust Level of team members
func (a6 *Team6Agent) UpdateSelfishness() {

	if !a6.HasTeam() {
		return
	}

	// Estimate threshold to be 100 - hardcoded
	// while score below threshold, increase selfishness slightly
	const threshold = 100.0
	var scoreInfluence float64
	if a6.Score < threshold {
		dangerLevel := (float64(threshold - a6.Score)) / float64(threshold)
		scoreInfluence = dangerLevel * 0.1
	} else {
		safetyLevel := float64(a6.Score) - threshold
		scoreInfluence = -0.1 * math.Min(safetyLevel/10.0, 1.0) // Maximum decrease of 0.1
	}

	// average team trust
	// higher trust decreases selfishness
	agentsInTeam := a6.Server.GetAgentsInTeam(a6.TeamID)
	totalTrust := 0.0
	for _, agentID := range agentsInTeam {
		if agentID != a6.GetID() {
			totalTrust += a6.AgentTrust[agentID]
		}
	}
	avgTrust := totalTrust / float64(len(agentsInTeam))
	trustInfluence := -0.2 * avgTrust

	newSelfishness := a6.Selfishness + scoreInfluence + trustInfluence

	a6.Selfishness = math.Max(0.0, math.Min(0.9, newSelfishness))

}

func (a6 *Team6Agent) GetActualWithdrawal(instance common.IExtendedAgent) int {
	if !a6.HasTeam() {
		return 0
	}

	commonPool := a6.Server.GetTeam(a6.GetID()).GetCommonPool()
	aoa := a6.Server.GetTeam(a6.GetID()).TeamAoA
	aoaExpectedWithdrawal := aoa.GetExpectedWithdrawal(a6.GetID(), a6.GetTrueScore(), commonPool)

	withdrawalChoice := -1
	if a6.Selfishness < 0.5 {
		// If the agent is relatively selfless (low selfishness)
		// Withdraw less than the maximum allowed
		withdrawalChoice = int(math.Floor(float64(aoaExpectedWithdrawal) * (1 - a6.Selfishness)))
	} else {
		// If the agent is relatively selfish (high selfishness)
		// Withdraw more, mapped between AoA and maximum allowed
		withdrawalChoice = int(math.Ceil(float64(aoaExpectedWithdrawal) + a6.Selfishness*float64(commonPool-aoaExpectedWithdrawal)))
	}
	return withdrawalChoice

}

func (a6 *Team6Agent) GetStatedWithdrawal(instance common.IExtendedAgent) int {
	if !a6.HasTeam() {
		return 0
	}

	// Always state the expected withdrawal (even if cheating)
	team := a6.Server.GetTeam(a6.GetID())
	aoa := team.TeamAoA
	aoaExpectedWithdrawal := aoa.GetExpectedWithdrawal(a6.GetID(), a6.GetTrueScore(), team.GetCommonPool())
	return aoaExpectedWithdrawal
}

func (a6 *Team6Agent) HandleWithdrawalMessage(msg *common.WithdrawalMessage) {
	a6.ExtendedAgent.HandleWithdrawalMessage(msg)
	a6.ExpectedCommonPool -= msg.StatedAmount
}

func (a6 *Team6Agent) GetWithdrawalAuditVote() common.Vote {
	// End of turn, update greed + selfishness
	a6.UpdateSelfishness()
	a6.UpdateGreed()

	// Get list of uuids in our team
	var agentsInTeam []uuid.UUID = a6.Server.GetAgentsInTeam(a6.TeamID)
	actualPoolSize := a6.Server.GetTeamCommonPool(a6.TeamID)

	// if actual pool size is less than expected
	if actualPoolSize < a6.ExpectedCommonPool {
		// someone lied - trust in rest of team decreases
		for _, agentID := range agentsInTeam {
			a6.UpdateAgentTrust(agentID, -0.02)
		}

		// Vote for least trusted team member (with trust below default 0.5)
		var votedAgent uuid.UUID
		lowestTrust := 0.5

		for _, agentID := range agentsInTeam {
			if a6.AgentTrust[agentID] <= lowestTrust {
				lowestTrust = a6.AgentTrust[agentID]
				votedAgent = agentID
			}
		}
		a6.ExpectedCommonPool = actualPoolSize
		return common.CreateVote(1, a6.GetID(), votedAgent)

	}

	// if actual pool size is equal or greater than expected
	// increase trust in rest of team
	for _, agentID := range agentsInTeam {
		a6.UpdateAgentTrust(agentID, 0.01)
	}

	a6.ExpectedCommonPool = actualPoolSize
	return common.CreateVote(0, a6.GetID(), uuid.Nil)
}

func (a6 *Team6Agent) HandleTeamFormationMessage(msg *common.TeamFormationMessage) {
	// Already in a team - reject invitation
	if a6.HasTeam() {
		return
	}

	sender := msg.GetSender()

	if _, ok := a6.AgentTrust[sender]; !ok {
		a6.AgentTrust[sender] = 0.5
	}

	senderTrustValue := a6.AgentTrust[sender]
	if senderTrustValue >= 0.5 {
		// Handle team creation/joining based on sender's team status
		sender := msg.GetSender()
		if a6.Server.CheckAgentAlreadyInTeam(sender) {
			existingTeamID := a6.Server.AccessAgentByID(sender).GetTeamID()
			a6.joinExistingTeam(existingTeamID)
		} else {
			a6.createNewTeam(sender)
		}
	}

	// reset game state - joined new team
	a6.ExpectedCommonPool = 0
	a6.AgentTurnScore = 0
	a6.WithdrawalAuditCount = 0
	a6.WithdrawalSucessfulCount = 0
}
