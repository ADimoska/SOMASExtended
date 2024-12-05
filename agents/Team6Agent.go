package agents

import (
	"SOMAS_Extended/common"
	"fmt"
	"math/rand"
	"math"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
)

type Team6_Agent struct {
	*ExtendedAgent
	TeamTrust           	map[uuid.UUID]float64 	//this used in deciding auditing votes, and team choosing
	
	Selfishness             float64               	//a value from 0 (least selfish) to 1 (most selfish) which affects agent strategies
	AgentTurnScore          int                    	//this is used in common.Team6AoA in getting expected contribution, it is the score earned this turn
	Trust                   float64
	Greed                   float64
	AveragePersonalDiceRoll float64					//this is used in common.Team6AoA in getting expected contribution, it is the score earned this turn
	ContributionSuccessCount int
	WithdrawalSuccessCount	 int
}

// constructor for Team6_Agent
func Team6_CreateAgent(funcs agent.IExposedServerFunctions[common.IExtendedAgent], agentConfig AgentConfig) *Team6_Agent {
	teamTrust := make(map[uuid.UUID]float64)

	// Initialize trust levels for team members (default: 0.5)
	for _, teamMember := range funcs.GetTeamMembers(agentConfig.AgentID) {
		if teamMember != agentConfig.AgentID { // Exclude self
			teamTrust[teamMember] = 0.5
		}
	}

	return &Team6_Agent{
		ExtendedAgent: 			GetBaseAgents(funcs, agentConfig),
		TeamTrust:				make(map[uuid.UUID]*float64)
		Selfishness: 			 rand.Float64(),						//initialised randomly
		//Reputation:				0.5,								//initialised to neutral reputation
		Trust:                   0.5,
		Greed:                   rand.Float64(),
		AgentTurnScore:          0, //initialised to zero
		AveragePersonalDiceRoll: 0.0,

	}
}
func (mi *Team6_Agent) SetAgentContributionAuditResult(agentID uuid.UUID, result bool) bool {
	if result {
		mi.ContributionSuccessCount++
		mi.TeamTrust[agentID] -= 0.1
	}
	return result
}

func (mi *Team6_Agent) SetAgentWithdrawalAuditResult(agentID uuid.UUID, result bool) bool {
	if result {
		mi.WithdrawalSuccessCount++
		mi.TeamTrust[agentID] -= 0.3
	}
	return result
}

func (mi *Team6_Agent) GetTotalSuccessfulAudits() int {
	return mi.ContributionSuccessCount + mi.WithdrawalSuccessCount
}

func (mi *Team6_Agent) UpdateTrust() {
	// Calculate trust based on total successful audits
	successfulAudits := mi.GetTotalSuccessfulAudits()

	// Incremental trust update
	if successfulAudits > 0 {
		trustIncrement := 0.1 // Define how much each successful audit increases trust
		mi.Trust += float64(successfulAudits) * -trustIncrement

		// Ensure trust does not exceed 1.0
		if mi.Trust > 1.0 {
			mi.Trust = 1.0
		}
	} 
	if mi.Trust < 0.0 {
		mi.Trust = 0.0
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

// Update trust for a specific team member
func (mi *Team6_Agent) UpdateAllTeamMembersTrust() {
	// Iterate over each agent in the TeamTrust map
	for agentID := range mi.TeamTrust {
		// Temporarily set the current agent's ID to mimic updating their trust
		mi.Trust = mi.TeamTrust[agentID]

		// Apply the existing UpdateTrust logic
		mi.UpdateTrust()

		// Save the updated trust back to the TeamTrust map
		mi.TeamTrust[agentID] = mi.Trust
	}
}

func (mi *Team6_Agent) GetTurnScore() int {
	return mi.AgentTurnScore
}

// ----------------------- Strategies -----------------------

// so this is all from team 4 strategy stuff: it is up to us to implement the strategies unique to the agents in our team

// Team-forming Strategy
func (mi *Team6_Agent) DecideTeamForming(agentInfoList []common.ExposedAgentInfo) []uuid.UUID {
	// fmt.Printf("Called overriden DecideTeamForming\n")
	// invitationList := []uuid.UUID{}
	// for _, agentInfo := range agentInfoList {
	// 	// exclude the agent itself
	// 	if agentInfo.AgentUUID == mi.GetID() {
	// 		continue
	// 	}
	// 	if agentInfo.AgentTeamID == (uuid.UUID{}) {
	// 		invitationList = append(invitationList, agentInfo.AgentUUID)
	// 	}
	// }

	// // TODO: implement team forming logic
	// // random choice from the invitation list
	// rand.Shuffle(len(invitationList), func(i, j int) { invitationList[i], invitationList[j] = invitationList[j], invitationList[i] })
	// chosenAgent := invitationList[0]

	// // Return a slice containing the chosen agent
	// return []uuid.UUID{chosenAgent}
}

// Contribution Strategy - HERE WE CAN DEFINE HOW SELFISHNESS / REPUTATION WILL HAVE AN EFFECT ON AN AGENTS STRATEGY
func (mi *Team6_Agent) GetActualContribution() int {
	
	// if this agent is in a team
	if mi.server.GetTeam(mi.GetID()).TeamAoA != nil {
		// calculate expected contribution according to AoAs
		aoaExpectedContribution := mi.server.GetTeam(mi.GetID()).TeamAoA.GetExpectedContribution(mi.GetID(), mi.GetTurnScore())	

		// HERE IS AN EXAMPLE OF HOW SELFISHNESS COULD WORK
		// Note: aoa expected contribution is defined as a fraction of TurnScore, 
		// So we don't need to worry about weird behaviour of trying to contribute more than is scored in a turn
		contributionChoice := -1
		contributionFraction := float64(aoaExpectedContribution/mi.GetTurnScore())

		if mi.Selfishness >= 0.5	{					
			// if this agent is relatively selfish
			// linearly map 0.5 to 1 selfishness to contribute between aoaExpected and 0
			contributionChoice = math.floor(2 * (1-mi.Selfishness) * aoaExpectedContribution)		
		} else { 											
			// if this agent is relatively selfless
			// linearly map 0 to 0.5 selfishness to contribute between (whole of) TurnScore and aoaExpected
			selfishnessScaling := (aoaExpectedContribution-mi.GetTurnScore())/(0.5)
			contributionChoice = math.ceil(selfishnessScaling * selfishness + mi.GetTurnScore())
		}

		if contributionChoice != -1 {
			// so if this is not -1, then no errors have occured and all is bueno
			return contributionChoice
		}
	} else {	
		// if this agent is not in a team
		if mi.verboseLevel > 6 {
			// should not happen!
			fmt.Printf("[WARNING] Agent %s has no AoA, contributing 0\n", mi.GetID())
		}
		return 0
	}
}

// Withdrawal Strategy - THIS NEEDS TO BE DEFINED, IN A SIMILAR WAY TO HOW CONTRIBUTION STRAT MIGHT USE SELFISHNESS AND REPUTATION
func (mi *Team6_Agent) GetActualWithdrawal() int {
	team := mi.Server.GetTeam(mi.GetID())
	if team != nil && team.TeamAoA != nil {
		// Calculate expected withdrawal according to AoAs
		aoaExpectedWithdrawal := team.TeamAoA.GetExpectedWithdrawal(mi.GetID(), mi.Server.GetTeamCommonPool())

		// Reversed logic for determining withdrawal based on selfishness
		withdrawalChoice := -1

		if mi.Selfishness < 0.5 {
			// If the agent is relatively selfless (low selfishness)
			// Withdraw less than the maximum allowed
			withdrawalChoice = int(math.Floor(float64(aoaExpectedWithdrawal) * (1 - mi.Selfishness)))
		} else {
			// If the agent is relatively selfish (high selfishness)
			// Withdraw more, mapped between AoA and maximum allowed
			withdrawalChoice = int(math.Ceil(float64(aoaExpectedWithdrawal) + mi.Selfishness*float64(mi.Server.GetTeamCommonPool()-aoaExpectedWithdrawal)))
		}

		if withdrawalChoice != -1 {
			// If this is not -1, then no errors have occurred and all is well
			return withdrawalChoice
		}
	} else {
		// If this agent is not in a team
		if mi.verboseLevel > 6 {
			// Should not happen!
			fmt.Printf("[WARNING] Agent %s has no AoA, withdrawing 0\n", mi.GetID())
		}
		return 0
	}
	return 0
}

// Audit Strategy
func (mi *Team6_Agent) DecideAudit() bool {
	// TODO: implement audit strategy
	return true
}


// Dice Strategy - HERE WE CAN MESS WITH HOW RISKY OR NOT WE WANT OUR AGENTS TO BE
func (mi *Team6_Agent) StickOrAgain() bool {
	// Calculate stick threshold based on agent's greed
	// Greed value influences how risk-taking an agent is:
	// - High greed means they will only stick at a higher score
	// - Low greed means they will stick at a lower score

	baseValue := 10.0                            // This is the base value that the agent considers to stick
	stickThreshold := baseValue * (1 + mi.Greed) // Threshold determined by baseValue and greed level

	// If the agent's turn score reaches or exceeds the stick threshold, they will stick
	if float64(mi.AgentTurnScore) >= stickThreshold {
		fmt.Printf("Agent %s decides to STICK with score %.2f (Greed: %.2f, Threshold: %.2f)\n", mi.GetID(), float64(mi.AgentTurnScore), mi.Greed, stickThreshold)
		return true
	}

	// Otherwise, they decide to roll again
	fmt.Printf("Agent %s decides to ROLL AGAIN (Current score: %d, Threshold: %.2f)\n", mi.GetID(), mi.AgentTurnScore, stickThreshold)
	return false
}

/*
Provide agentId for memory, current accumulated score
(to see if above or below predicted threshold for common pool contribution)
And previous roll in case relevant
*/
// I'M NOT SURE HOW THIS IS DIFFERENT FROM STICKORAGAIN, BUT IS PROBABLY ALSO PART OF STRATEGY
func (mi *Team6_Agent) StickOrAgainFor(agentId uuid.UUID, accumulatedScore int, prevRoll int) int {
	// random chance, to simulate what is already implemented
	return rand.Intn(2)
}

// Agent returns their preference for an audit on contribution
// 0: No preference
// 1: Prefer audit
// -1: Prefer no audit
// WE NEED TO IMPLEMENT AGENT STRATEGY HERE TO DECIDE WHO TO VOTE FOR
// - MAYBE WHOEVER IS LOWEST IN OPINION MATRIX?
// - CAN AN AGENT BEING MONITORED BE AUDITED AS WELL?
func (mi *ExtendedAgent) GetContributionAuditVote() common.Vote {
	var voteOutAgent uuid.UUID
	lowestTrust := math.MaxFloat64

	for agentID, trustValue := range mi.OpinionVector {
		agentID != mi.GetID() && trustValue < lowestTrust {
			lowestTrust = *trustValue
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
func (mi *ExtendedAgent) GetWithdrawalAuditVote() common.Vote {
	var voteOutAgent uuid.UUID
	lowestTrust := math.MaxFloat64

	for agentID, trustValue := range mi.OpinionVector {
		agentID != mi.GetID() && trustValue < lowestTrust {
			lowestTrust = *trustValue
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

func (mi *Team6_Agent) UpdateSelfishness() {
	team := mi.Server.GetTeam(mi.GetID())
	if team == nil {
		fmt.Printf("[WARNING] Agent %s has no team, setting selfishness to default 0.5\n", mi.GetID())
		mi.Selfishness = 0.5
		return
	}

	// Update selfishness based on Agent Dice Roll (AveragePersonalDiceRoll), Team Common Pool, and Trust Level
	teamCommonPool := mi.Server.GetTeamCommonPool()

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

//Gives a successful audit result 
func (mi *Team6_Agent) SetAgentContributionAuditResult(agentID uuid.UUID, result bool) bool {
    if result {
        mi.ContributionSuccessCount++
		mi.TeamTrust[agentID] -= 0.1
    }
    return result
}

func (mi *Team6_Agent) SetAgentWithdrawalAuditResult(agentID uuid.UUID, result bool) bool {
    if result {
        mi.WithdrawalSuccessCount++ // Increment the withdrawal success counter
		mi.TeamTrust[agentID] -= 0.3
    }
    return result
}

func (mi *Team6_Agent) GetTotalSuccessfulAudits() int {
    return mi.ContributionSuccessCount + mi.WithdrawalSuccessCount
}


func (mi *Team6_Agent) HandleWithdrawalMessage(msg *WithdrawalMessage)
func (mi *Team6_Agent) HandleContributionMessage(msg *ContributionMessage)

func (mi *Team6_Agent) HandleAgentOpinionRequestMessage(msg *AgentOpinionRequestMessage)
func (mi *Team6_Agent) HandleAgentOpinionResponseMessage(msg *AgentOpinionResponseMessage)


// mi.CreateAgentOpinionResponseMessage(agentID uuid.UUID, opinion int)
// 
