package agents

import (
	"fmt"
	"log"
	"math"
	"math/rand"

	// "math/rand"

	common "github.com/ADimoska/SOMASExtended/common"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
)

// this is the third tier of composition - embed the extended agent and add 'user specific' fields
type Team6Agent struct {
	*ExtendedAgent
	rank                        bool
	AgentTurnScore              int
	TeamTrust                   map[uuid.UUID]float64
	strikeCount                 map[uuid.UUID]int
	statedContribution          map[uuid.UUID]int
	statedWithdrawal            map[uuid.UUID]int
	commonPoolEstimate          int
	WithdrawalAuditCount        int
	ContributionAuditCount      int
	ContributionSuccessfulCount int
	Selfishness                 float64
	Trust                       float64
	Greed                       float64
	AveragePersonalDiceRoll     float64 //this is used in common.Team6AoA in getting expected contribution, it is the score earned this turn
}

func Team6_CreateAgent(funcs agent.IExposedServerFunctions[common.IExtendedAgent], agentConfig AgentConfig) *Team6Agent {
	extendedAgent := GetBaseAgents(funcs, agentConfig)
	extendedAgent.TrueSomasTeamID = 6
	extendedAgent.AoARanking = []int{6, 5, 4, 3, 2, 1}

	return &Team6Agent{
		ExtendedAgent:           extendedAgent,
		rank:                    false,
		TeamTrust:               make(map[uuid.UUID]float64),
		strikeCount:             make(map[uuid.UUID]int),
		statedContribution:      make(map[uuid.UUID]int),
		statedWithdrawal:        make(map[uuid.UUID]int),
		commonPoolEstimate:      0,
		Selfishness:             rand.Float64(), //initialised randomly
		Trust:                   0.5,
		Greed:                   rand.Float64(),
		AgentTurnScore:          0, //initialised to zero
		AveragePersonalDiceRoll: 0.0,
	}

}

// ---------- TRUST SCORE SYSTEM ----------
func (mi *Team6Agent) SetTeamMemberTrust(agentID uuid.UUID) {
	mi.TeamTrust[agentID] = 0.5
}

func (mi *Team6Agent) SetAgentContributionAuditResult(agentID uuid.UUID, result bool) {
	if _, exists := mi.TeamTrust[agentID]; !exists {
		mi.SetTeamMemberTrust(agentID)
	}
	if result {
		mi.ContributionSuccessfulCount++
		mi.TeamTrust[agentID] -= 0.1
	}
	mi.ContributionAuditCount++
}

func (mi *Team6Agent) SetAgentWithdrawalAuditResult(agentID uuid.UUID, result bool) {
	if result {
		mi.WithdrawalAuditCount++
		mi.TeamTrust[agentID] -= 0.3
	}

}

func (mi *Team6Agent) GetTotalSuccessfulAudits() int {
	return mi.ContributionAuditCount + mi.WithdrawalAuditCount
}

// ---------- DECISION TO STICK  ----------

func (mi *Team6Agent) StickOrAgain(accumulatedScore int, prevRoll int) bool {
	baseValue := 10.0                            // This is the base value that the agent considers to stick
	stickThreshold := baseValue * (1 + mi.Greed) // Threshold determined by baseValue and greed level

	// If the agent's turn score reaches or exceeds the stick threshold, they will stick
	if float64(mi.AgentTurnScore) >= stickThreshold {
		fmt.Printf("(Team 6) Agent %s decides to STICK with score %.2f (Greed: %.2f, Threshold: %.2f)\n", mi.GetID(), float64(mi.AgentTurnScore), mi.Greed, stickThreshold)
		return true
	}

	// Otherwise, they decide to roll again
	fmt.Printf("(Team 6) Agent %s decides to ROLL AGAIN (Current score: %d, Threshold: %.2f)\n", mi.GetID(), mi.AgentTurnScore, stickThreshold)
	return false
}

func randomWeightedChoice(weights []float64) int {
	totalWeight := 0.0
	for _, weight := range weights {
		totalWeight += weight
	}

	randValue := rand.Float64() * totalWeight
	for i, weight := range weights {
		if randValue < weight {
			return i
		}
		randValue -= weight
	}
	return len(weights) - 1 // Fallback in case of rounding errors
}

// ---------- CONTRIBUTION, WITHDRAWAL AND ASSOCIATED AUDITING ----------

func (mi *Team6Agent) GetActualContribution(instance common.IExtendedAgent) int {
	if mi.TeamID == (uuid.UUID{}) {
		log.Printf("Agent %s does not have a team, skipping contribution...\n", mi.GetID())
		return 0
	}

	team := mi.Server.GetTeam(mi.GetID())
	aoa := team.TeamAoA
	aoaExpectedContribution := aoa.GetExpectedContribution(mi.GetID(), mi.GetTrueScore())

	// // Check if the AoA is of type *common.Team6AoA
	// if _, ok := aoa.(*common.Team6AoA); ok {
	// 	// If it is our team's AoA, adjust the contribution by investAmount
	// 	aoaExpectedContribution += (mi.GetTrueScore() - aoaExpectedContribution) / 2
	// }

	return aoaExpectedContribution
}

func (mi *Team6Agent) GetStatedContribution(instance common.IExtendedAgent) int {

	if mi.TeamID == (uuid.UUID{}) {
		log.Printf("Agent %s does not have a team, skipping contribution...\n", mi.GetID())
		return 0
	}
	team := mi.Server.GetTeam(mi.GetID())
	aoa := team.TeamAoA
	aoaExpectedContribution := aoa.GetExpectedContribution(mi.GetID(), mi.GetTrueScore())
	return aoaExpectedContribution
}

func (mi *Team6Agent) HandleContributionMessage(msg *common.ContributionMessage) {
	mi.ExtendedAgent.HandleContributionMessage(msg)
	mi.statedContribution[msg.GetSender()] = msg.StatedAmount
	mi.commonPoolEstimate += msg.StatedAmount
}

func (mi *Team6Agent) GetContributionAuditVote() common.Vote {
	// Get list of uuids in our team
	var agentsInTeam []uuid.UUID = mi.Server.GetAgentsInTeam(mi.TeamID)
	// Vote for the weakest link
	var weakestLink uuid.UUID
	var lowestContribution int = math.MaxInt
	for _, agentID := range agentsInTeam {
		if mi.statedContribution[agentID] < lowestContribution {
			lowestContribution = mi.statedContribution[agentID]
			weakestLink = agentID
		}
	}
	return common.CreateVote(1, mi.GetID(), weakestLink)
}

func (mi *Team6Agent) UpdateGreed() {
	// Update greed value based on Current Agent Score and Average Personal Dice Roll
	if mi.AgentTurnScore > 0 && mi.AveragePersonalDiceRoll > 0 {
		// Example logic: Greed increases when the agent's average dice roll is higher
		// and the current agent score is higher, indicating more risk-taking behavior
		mi.Greed = (float64(mi.AgentTurnScore) / 100.0) + (mi.AveragePersonalDiceRoll / 6.0)

		// Ensure greed is within bounds [0.0, 1.0]
		if mi.Greed > 1.0 {
			mi.Greed = 1.0
		} else if mi.Greed < 0.0 {
			mi.Greed = 0.0
		}
	}

	fmt.Printf("Agent %s greed updated to %.2f based on current agent score and average personal dice roll\n", mi.GetID(), mi.Greed)
}

func (mi *Team6Agent) UpdateSelfishness() {
	team := mi.Server.GetTeam(mi.GetID())
	if team == nil {
		fmt.Printf("[WARNING] Agent %s has no team, setting selfishness to default 0.5\n", mi.GetID())
		mi.Selfishness = 0.5
		return
	}

	// Update selfishness based on Agent Dice Roll (AveragePersonalDiceRoll), Team Common Pool, and Trust Level
	teamCommonPool := mi.Server.GetTeamCommonPool(mi.GetTeamID())

	if teamCommonPool > 0 && mi.AveragePersonalDiceRoll > 0 {
		// Example logic: Selfishness increases when trust is lower, the average dice roll is higher,
		// and there is a higher amount of common resources in the team pool.
		selfishnessFactor := (1.0 - mi.Trust) + (mi.AveragePersonalDiceRoll / 6.0) + (float64(teamCommonPool) / 200.0)

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

	fmt.Printf("Agent %s selfishness updated to %.2f based on dice roll, team common pool, and trust level\n", mi.GetID(), mi.Selfishness)
}

func (mi *Team6Agent) GetActualWithdrawal(instance common.IExtendedAgent) int {
	if mi.TeamID == (uuid.UUID{}) {
		log.Printf("Agent %s does not have a team, skipping withdrawal...\n", mi.GetID())
		return 0
	}
	commonPool := mi.Server.GetTeam(mi.GetID()).GetCommonPool()
	aoa := mi.Server.GetTeam(mi.GetID()).TeamAoA
	aoaExpectedWithdrawal := aoa.GetExpectedWithdrawal(mi.GetID(), mi.GetTrueScore(), commonPool)
	withdrawalOptions := []int{aoaExpectedWithdrawal, aoaExpectedWithdrawal + mi.GetActualContribution(instance)}
	var p_caught float64
	if mi.ContributionAuditCount > 0 {
		p_caught = float64(mi.ContributionSuccessfulCount) / float64(mi.ContributionAuditCount)
	} else {
		p_caught = 0.0
	}
	weights := []float64{1 - p_caught, p_caught}
	return withdrawalOptions[randomWeightedChoice(weights)]

}

func (mi *Team6Agent) GetStatedWithdrawal(instance common.IExtendedAgent) int {
	if mi.TeamID == (uuid.UUID{}) {
		log.Printf("Agent %s does not have a team, skipping withdrawal...\n", mi.GetID())
		return 0
	}
	commonPool := mi.Server.GetTeam(mi.GetID()).GetCommonPool()
	aoa := mi.Server.GetTeam(mi.GetID()).TeamAoA
	aoaExpectedWithdrawal := aoa.GetExpectedWithdrawal(mi.GetID(), mi.GetTrueScore(), commonPool)
	statedWithdrawal := min(mi.GetActualWithdrawal(instance), aoaExpectedWithdrawal)
	return statedWithdrawal
}

func (mi *Team6Agent) HandleWithdrawalMessage(msg *common.WithdrawalMessage) {
	mi.ExtendedAgent.HandleWithdrawalMessage(msg) // Call extendedagent version to enable logging

	// updating our agents "mind":

	// store this agents stated withdrawal in our map for use in deciding who to audit
	mi.statedWithdrawal[msg.GetSender()] = msg.StatedAmount
	// decrement the common pool estimate by the stated amount
	mi.commonPoolEstimate -= msg.StatedAmount
}

func (mi *Team6Agent) GetWithdrawalAuditVote() common.Vote {
	var voteOutAgent uuid.UUID
	lowestTrust := math.MaxFloat64

	for agentID, trustValue := range mi.TeamTrust {
		if agentID != mi.GetID() && trustValue < lowestTrust {
			lowestTrust = trustValue
			voteOutAgent = agentID
		}
	}

	if voteOutAgent == uuid.Nil {
		return common.CreateVote(0, mi.GetID(), uuid.Nil)
	}

	return common.CreateVote(1, mi.GetID(), voteOutAgent)
}
