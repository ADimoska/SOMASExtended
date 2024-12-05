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
	OpinionVector 	map[uuid.UUID]*float64  //this used in deciding auditing votes, and team choosing
	Selfishness 	float64					//a value from 0 (least selfish) to 1 (most selfish) which affects agent strategies
	Reputation 		float64					//a value which guesses how selfish other agents believe this agent to be
	AgentTurnScore  int						//this is used in common.Team6AoA in getting expected contribution, it is the score earned this turn
}

// constructor for Team6_Agent
func Team6_CreateAgent(funcs agent.IExposedServerFunctions[common.IExtendedAgent], agentConfig AgentConfig) *Team6_Agent {
	return &Team6_Agent{
		ExtendedAgent: 			GetBaseAgents(funcs, agentConfig),
		OpinionVector:			make(map[uuid.UUID]*float64)
		Selfishness: 			rand.Float64(),						//initialised randomly
		Reputation:				0.5,								//initialised to neutral reputation
		AgentTurnScore:			0,									//initialised to zero

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
func (mi *Team6_Agent) DecideContribution() int {
	
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
func (mi *Team6_Agent) DecideWithdrawal() int {
	// TODO: implement contribution strategy

// USE THIS FOR REFERENCE
// // Decide the withdrawal amount based on AoA and current pool size
// func (mi *ExtendedAgent) DecideWithdrawal() int {
// 	if mi.server.GetTeam(mi.GetID()).TeamAoA != nil {
// 		// double check if score in agent is sufficient (this should be handled by AoA though)
// 		commonPool := mi.server.GetTeam(mi.GetID()).GetCommonPool()
// 		aoaExpectedWithdrawal := mi.server.GetTeam(mi.GetID()).TeamAoA.GetExpectedWithdrawal(mi.GetID(), mi.GetTrueScore(), commonPool)
// 		if commonPool < aoaExpectedWithdrawal {
// 			return commonPool
// 		}
// 		return aoaExpectedWithdrawal
// 	} else {
// 		if mi.verboseLevel > 6 {
// 			fmt.Printf("[WARNING] Agent %s has no AoA, withdrawing 0\n", mi.GetID())
// 		}
// 		return 0
// 	}
// }
	return 1
}



// Audit Strategy
func (mi *Team6_Agent) DecideAudit() bool {
	// TODO: implement audit strategy
	return true
}

// Punishment Strategy
func (mi *Team6_Agent) DecidePunishment() int {
	// TODO: implement punishment strategy
	return 1
}

// Dice Strategy - HERE WE CAN MESS WITH HOW RISKY OR NOT WE WANT OUR AGENTS TO BE
func (mi *Team6_Agent) StickOrAgain() bool {
	// fmt.Printf("Called overriden StickOrAgain\n")
	// // TODO: implement dice strategy
	// return true
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

// get the agent's stated contribution to the common pool
// TODO: the value returned by this should be broadcasted to the team via a message
// This function MUST return the same value when called multiple times in the same turn
func (mi *Team6_Agent) GetStatedContribution(instance common.IExtendedAgent) int {
	// Hardcoded stated
	// TODO: Implement actual strategy based off selfishness, reputation, etc
	statedContribution := instance.GetActualContribution(instance)
	return statedContribution
}

// The value returned by this should be broadcasted to the team via a message
// This function MUST return the same value when called multiple times in the same turn
func (mi *Team6_Agent) GetStatedWithdrawal(instance common.IExtendedAgent) int {
	// Currently, assume stated withdrawal matches actual withdrawal
	// TODO: Implement Agent Strategy behaviour
	return instance.DecideWithdrawal()
}

// Agent returns their preference for an audit on contribution
// 0: No preference
// 1: Prefer audit
// -1: Prefer no audit
// WE NEED TO IMPLEMENT AGENT STRATEGY HERE TO DECIDE WHO TO VOTE FOR
// - MAYBE WHOEVER IS LOWEST IN OPINION MATRIX?
// - CAN AN AGENT BEING MONITORED BE AUDITED AS WELL?
func (mi *ExtendedAgent) GetContributionAuditVote() common.Vote {
	return common.CreateVote(0, mi.GetID(), uuid.Nil)
}

// Agent returns their preference for an audit on withdrawal
// 0: No preference
// 1: Prefer audit
// -1: Prefer no audit
// SAME DEAL HERE
func (mi *ExtendedAgent) GetWithdrawalAuditVote() common.Vote {
	return common.CreateVote(0, mi.GetID(), uuid.Nil)
}

// WE NEED A SPECIAL IMPLEMENTATION OF THIS FOR TEAM6AGENTS TO UPDATE AGENTTURNSCORE VALUE
// other than that, nothing is changed from its implementation in ExtendedAgent.go
func (mi *Team6_Agent) StartRollingDice(instance common.IExtendedAgent) {
	if mi.verboseLevel > 10 {
		fmt.Printf("%s is rolling the Dice\n", mi.GetID())
	}
	if mi.verboseLevel > 9 {
		fmt.Println("---------------------")
	}
	// TODO: implement the logic in environment, do a random of 3d6 now with 50% chance to stick
	mi.lastScore = -1
	rounds := 1
	turnScore := 0

	willStick := false

	// loop until not stick
	for !willStick {
		// debug add score directly
		currentScore := Debug_RollDice()

		// check if currentScore is higher than lastScore
		if currentScore > mi.lastScore {
			turnScore += currentScore
			mi.lastScore = currentScore
			willStick = instance.StickOrAgain()
			if willStick {
				mi.DecideStick()
				break
			}
			mi.DecideRollAgain()
		} else {
			// burst, lose all turn score
			if mi.verboseLevel > 4 {
				fmt.Printf("%s **BURSTED!** round: %v, current score: %v\n", mi.GetID(), rounds, currentScore)
			}
			turnScore = 0
			break
		}

		rounds++
	}

	// add turn score to total score
	mi.AgentTurnScore = turnScore 
	mi.score += turnScore

	if mi.verboseLevel > 4 {
		fmt.Printf("%s's turn score: %v, total score: %v\n", mi.GetID(), turnScore, mi.score)
	}
}