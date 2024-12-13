package agents

import (
	"math"

	common "github.com/ADimoska/SOMASExtended/common"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
)

type Team6_Agent struct {
	*ExtendedAgent
	TeamTrust map[uuid.UUID]float64 //this used in deciding auditing votes, and team choosing

	Selfishness float64 //Value from 0 to 1 (most selfish) which affects agent contribution / withdrawal
	Greed       float64 //Value from 0 to 1 (most greedy) which affects dice rolling strategy

	AgentTurnScore          int
	AveragePersonalDiceRoll float64
	ExpectedTeamPool        int
}

// constructor for Team6_Agent
func Team6_CreateAgent(funcs agent.IExposedServerFunctions[common.IExtendedAgent], agentConfig AgentConfig) *Team6_Agent {

	team6 := &Team6_Agent{
		ExtendedAgent:           GetBaseAgents(funcs, agentConfig),
		TeamTrust:               make(map[uuid.UUID]float64),
		Selfishness:             0.3, //start low selfishness (=> contribute more / withdraw less)
		Greed:                   0.7, //start high greed
		AgentTurnScore:          0,
		AveragePersonalDiceRoll: 0.0,
		ExpectedTeamPool:        0,
	}

	team6.TrueSomasTeamID = 6
	return team6
}

func (mi *Team6_Agent) SetAgentContributionAuditResult(agentID uuid.UUID, result bool) {
	if result {
		mi.TeamTrust[agentID] -= 0.1
	}
}

func (mi *Team6_Agent) SetAgentWithdrawalAuditResult(agentID uuid.UUID, result bool) {
	if result {
		mi.TeamTrust[agentID] -= 0.3
	}
}

// Set trust for a specific team member
func (mi *Team6_Agent) SetTeamMemberTrust(agentID uuid.UUID, trust float64) {
	if _, exists := mi.TeamTrust[agentID]; exists {
		mi.TeamTrust[agentID] = trust
	}
}

// Get trust for a specific team member
func (mi *Team6_Agent) GetTeamMemberTrust(agentID uuid.UUID) (float64, bool) {
	trust, exists := mi.TeamTrust[agentID]
	return trust, exists
}

func (mi *Team6_Agent) GetTurnScore() int {
	return mi.AgentTurnScore
}

// ----------------------- Strategies -----------------------

// so this is all from team 4 strategy stuff: it is up to us to implement the strategies unique to the agents in our team

// Team-forming Strategy
func (mi *Team6_Agent) DecideTeamForming(agentInfoList []common.ExposedAgentInfo) []uuid.UUID {
	return mi.ExtendedAgent.DecideTeamForming(agentInfoList)
}

// Contribution Strategy - HERE WE CAN DEFINE HOW SELFISHNESS / REPUTATION WILL HAVE AN EFFECT ON AN AGENTS STRATEGY
func (mi *Team6_Agent) GetActualContribution(instance common.IExtendedAgent) int {

	if mi.HasTeam() {

		team := mi.Server.GetTeam(mi.GetID())
		// calculate expected contribution according to AoAs
		aoaExpectedContribution := team.TeamAoA.GetExpectedContribution(mi.GetID(), mi.Score)

		// HERE IS AN EXAMPLE OF HOW SELFISHNESS COULD WORK
		// Note: aoa expected contribution is defined as a fraction of TurnScore,
		// So we don't need to worry about weird behaviour of trying to contribute more than is scored in a turn
		contributionChoice := -1

		if mi.Selfishness >= 0.5 {
			// if this agent is relatively selfish
			// linearly map 0.5 to 1 selfishness to contribute between aoaExpected and 0
			contributionChoice = int(math.Floor(2 * (1.0 - mi.Selfishness) * float64(aoaExpectedContribution)))
		} else {
			// if this agent is relatively selfless
			// linearly map 0 to 0.5 selfishness to contribute between (whole of) TurnScore and aoaExpected
			selfishnessScaling := float64(aoaExpectedContribution-mi.GetTurnScore()) / (0.5)
			contributionChoice = int(math.Ceil(selfishnessScaling*mi.Selfishness + float64(mi.GetTurnScore())))
		}

		return contributionChoice
	} else {
		// If this agent is not in a team
		return 0
	}
}

// Withdrawal Strategy - THIS NEEDS TO BE DEFINED, IN A SIMILAR WAY TO HOW CONTRIBUTION STRAT MIGHT USE SELFISHNESS AND REPUTATION
func (mi *Team6_Agent) GetActualWithdrawal(instance common.IExtendedAgent) int {
	if mi.HasTeam() {

		team := mi.Server.GetTeam(mi.GetID())
		teamCommonPool := team.GetCommonPool()

		// Calculate expected withdrawal according to AoAs
		aoaExpectedWithdrawal := team.TeamAoA.GetExpectedWithdrawal(mi.GetID(), mi.AgentTurnScore, teamCommonPool)

		// Reversed logic for determining withdrawal based on selfishness
		withdrawalChoice := -1

		if mi.Selfishness < 0.5 {
			// If the agent is relatively selfless (low selfishness)
			// Withdraw less than the maximum allowed
			withdrawalChoice = int(math.Floor(float64(aoaExpectedWithdrawal) * (1 - mi.Selfishness)))
		} else {
			// If the agent is relatively selfish (high selfishness)
			// Withdraw more, mapped between AoA and maximum allowed
			withdrawalChoice = int(math.Ceil(float64(aoaExpectedWithdrawal) + mi.Selfishness*float64(teamCommonPool-aoaExpectedWithdrawal)))
		}
		return withdrawalChoice

	} else {
		return 0
	}
}

// Dice Strategy
func (mi *Team6_Agent) StickOrAgain(accumulatedScore int, prevRoll int) bool {
	// Calculate stick threshold based on agent's greed
	// Greed value influences how risk-taking an agent is:
	// - High greed means they will only stick once we reach higher score
	// - Low greed means they will stick at a lower score

	baseValue := 10.0                            // This is the base value that the agent considers to stick
	stickThreshold := baseValue * (1 + mi.Greed) // Threshold determined by baseValue and greed level

	// If the agent's turn score reaches or exceeds the stick threshold, they will stick
	return float64(accumulatedScore) >= stickThreshold
}

// Agent returns their preference for an audit on contribution
// 0: No preference
// 1: Prefer audit
// -1: Prefer no audit
func (mi *Team6_Agent) GetContributionAuditVote() common.Vote {
	var voteOutAgent uuid.UUID
	lowestTrust := 0.5

	for agentID, trustValue := range mi.TeamTrust {
		if agentID != mi.GetID() && trustValue <= lowestTrust {
			lowestTrust = trustValue
			voteOutAgent = agentID
		}
	}

	if voteOutAgent == uuid.Nil {
		return common.CreateVote(0, mi.GetID(), uuid.Nil)
	}

	return common.CreateVote(1, mi.GetID(), voteOutAgent)
}

// Agent returns their preference for an audit on withdrawal
// 0: No preference
// 1: Prefer audit
// -1: Prefer no audit
// SAME DEAL HERE
func (mi *Team6_Agent) GetWithdrawalAuditVote() common.Vote {
	var voteOutAgent uuid.UUID
	lowestTrust := 0.5

	for agentID, trustValue := range mi.TeamTrust {
		if agentID != mi.GetID() && trustValue <= lowestTrust {
			lowestTrust = trustValue
			voteOutAgent = agentID
		}
	}

	if voteOutAgent == uuid.Nil {
		return common.CreateVote(0, mi.GetID(), uuid.Nil)
	}

	return common.CreateVote(1, mi.GetID(), voteOutAgent)
}

func (mi *Team6_Agent) UpdateGreed() {
	// Update greed value based on Current Agent Score and Average Personal Dice Roll
	if mi.AgentTurnScore > 0 && mi.AveragePersonalDiceRoll > 0 {
		// Greed increases when the agent's average dice roll is higher
		// and the current agent score is higher, indicating more risk-taking behavior
		mi.Greed = (float64(mi.AgentTurnScore) / 100.0) + (mi.AveragePersonalDiceRoll / 6.0)

		// Ensure greed is within bounds [0.0, 1.0]
		if mi.Greed > 1.0 {
			mi.Greed = 1.0
		} else if mi.Greed < 0.0 {
			mi.Greed = 0.0
		}
	}
}

func (mi *Team6_Agent) UpdateSelfishness() {
	team := mi.Server.GetTeam(mi.GetID())
	if team == nil {
		mi.Selfishness = 0.5
		return
	}

	// Update selfishness based on Agent Dice Roll (AveragePersonalDiceRoll), Team Common Pool, and Trust Level
	teamCommonPool := mi.Server.GetTeamCommonPool(mi.TeamID)

	if teamCommonPool > 0 && mi.AveragePersonalDiceRoll > 0 {
		// Selfishness increases when team trust is lower, the average dice roll is higher,
		// and there is a higher amount of common resources in the team pool.

		// Calculate average team trust
		agentsInTeam := mi.Server.GetAgentsInTeam(mi.TeamID)
		totalTrust := 0.0
		numAgents := 0

		for _, agentID := range agentsInTeam {
			if agentID == mi.GetID() {
				continue // Skip ourselves
			}
			if trust, exists := mi.TeamTrust[agentID]; exists {
				totalTrust += trust
				numAgents++
			}
		}

		// Calculate average team trust (default to 0.5 if no other agents)
		teamTrust := 0.5
		if numAgents > 0 {
			teamTrust = totalTrust / float64(numAgents)
		}

		selfishnessFactor := (1.0 - teamTrust) + (mi.AveragePersonalDiceRoll / 6.0) + (float64(teamCommonPool) / 200.0)

		// Scale selfishness to be between [0.0, 1.0]
		mi.Selfishness = selfishnessFactor / 3.0

		// Ensure selfishness is within bounds [0.0, 1.0]
		if mi.Selfishness > 1.0 {
			mi.Selfishness = 1.0
		} else if mi.Selfishness < 0.0 {
			mi.Selfishness = 0.0
		}
	} else {
		// If no data available, default to a neutral selfishness value
		mi.Selfishness = 0.5
	}
}

func (mi *Team6_Agent) HandleTeamFormationMessage(msg *common.TeamFormationMessage) {
	// Already in a team - reject invitation
	if mi.TeamID != (uuid.UUID{}) {
		return
	}

	sender := msg.GetSender()

	if _, ok := mi.TeamTrust[sender]; !ok {
		mi.TeamTrust[sender] = 0.5
	}

	senderTrustValue := mi.TeamTrust[sender]
	if senderTrustValue > 0.5 {
		// Handle team creation/joining based on sender's team status
		sender := msg.GetSender()
		if mi.Server.CheckAgentAlreadyInTeam(sender) {
			existingTeamID := mi.Server.AccessAgentByID(sender).GetTeamID()
			mi.joinExistingTeam(existingTeamID)
		} else {
			mi.createNewTeam(sender)
		}
	}
}

func (mi *Team6_Agent) HandleWithdrawalMessage(msg *common.WithdrawalMessage) {
	mi.ExpectedTeamPool -= msg.StatedAmount
}

func (mi *Team6_Agent) HandleContributionMessage(msg *common.ContributionMessage) {
	mi.ExpectedTeamPool += msg.StatedAmount
}

func (mi *Team6_Agent) HandleAgentOpinionResponseMessage(msg *common.AgentOpinionResponseMessage) {

}
