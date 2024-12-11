package agents

import (
	// "fmt"
	"log"

	"github.com/google/uuid"

	"github.com/ADimoska/SOMASExtended/common"
	baseAgent "github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
)

type AgentScoreInfo struct {
	TurnScore int
	Rerolls   int
}

type AgentContributionInfo struct {
	ContributionStated int
	ContributionExpected int
}

type AgentWithdrawalInfo struct {
	WithdrawalStated int
	WithdrawalExpected int
}


type AgentMemory struct {
	honestyScore int

	// Count to ensure that you only read the actual values, even if slice might be larger size with irrelvant entries
	// This is due to how append works: https://stackoverflow.com/questions/38543825/appending-one-element-to-nil-slice-increases-capacity-by-two
	LastContributionCount int
	LastWithdrawalCount   int
	LastScoreCount        int

	// Slice of all previous history
	historyContribution []AgentContributionInfo
	historyWithdrawal   []AgentWithdrawalInfo
	historyScore        []AgentScoreInfo // turnScore and rerolls
}

// AgentType is an enumeration of different agent behaviors.
// The underlying type is int.
type AgentType int

const (
	// iota automatically increments the value by 1 for each constant, starting from 0.

	// Honest (Value: 0): Agents who always state what they actually contributed.
	// Withdraw as per their expected withdrawal.
	Honest AgentType = iota

	// CheatLongTerm (Value: 1): Agents who always contribute honestly. After
	// rising in rank, they start withdrawing more than allowed.
	CheatLongTerm

	// CheatShortTerm (Value: 2): Agents who immediately start cheating. They
	// overstate their contributions and withdraw more than allowed.
	CheatShortTerm
)

type Team1Agent struct {
	*ExtendedAgent
	memory    map[uuid.UUID]AgentMemory
	agentType AgentType
}

const suspicious_contribution = 10 //suspicious contribution flag
const overstate_contribution = 10  //maximum contribution stated by cheater
const min_stated_withdrawal = 1    //minimum withdrawal stated by cheater
const cheat_amount = 3             //how much stated & actually contributed or withdrawn if cheating

func (a1 *Team1Agent) StickOrAgain(accumulatedScore int, prevRoll int) bool {
	exp := getExpectedGain(accumulatedScore, prevRoll)
	if exp < 2.0 {
		return true
	} else {
		return false
	}

}

func getExpectedGain(accumulatedScore, prevRoll int) float64 {
	lookup := make(map[int]int)
	for i := 1; i <= 6; i++ {
		for j := 1; j <= 6; j++ {
			for k := 1; k <= 6; k++ {
				sum := i + j + k
				lookup[sum]++
			}
		}
	}

	prob := make([]float64, 19)
	var totalCombinations int
	for _, v := range lookup {
		totalCombinations += v
	}
	for k, v := range lookup {
		prob[k] = float64(v) / float64(totalCombinations)
	}

	var pLoss, eLoss, eGain float64

	for i := 3; i <= prevRoll; i++ {
		pLoss += prob[i]
	}
	eLoss = pLoss * float64(accumulatedScore) * -1

	for i := prevRoll + 1; i < 19; i++ {
		eGain += prob[i] * float64(i)
	}

	return eGain + eLoss
}

func (a1 *Team1Agent) GetActualContribution(instance common.IExtendedAgent) int {
	if a1.HasTeam() {
		aoaExpectedContribution := a1.Server.GetTeam(a1.GetID()).TeamAoA.GetExpectedContribution(a1.GetID(), a1.Score)
		switch a1.agentType {
		case Honest, CheatLongTerm:
			return aoaExpectedContribution
		case CheatShortTerm:
			// Contribute less than expected
			contributedAmount := aoaExpectedContribution - cheat_amount
			if contributedAmount < 0 {
				contributedAmount = 0
			}
			return contributedAmount
		default:
			return aoaExpectedContribution
		}
	} else {
		log.Println("Agent does not have a team")
		return 0
	}
}

func (a1 *Team1Agent) GetActualWithdrawal(instance common.IExtendedAgent) int {
	if a1.HasTeam() {
		commonPool := a1.Server.GetTeam(a1.GetID()).GetCommonPool()
		aoaExpectedWithdrawal := a1.Server.GetTeam(a1.GetID()).TeamAoA.GetExpectedWithdrawal(a1.GetID(), a1.Score, commonPool)
		currentRank := 0
		switch a1.agentType {
		case Honest:
			return aoaExpectedWithdrawal
		case CheatLongTerm:
			// Perform type assertion to get Team1AoA
			teamAoA, ok := a1.Server.GetTeam(a1.GetID()).TeamAoA.(*common.Team1AoA)
			if ok {
				currentRank = teamAoA.GetAgentNewRank(a1.GetID())
				if currentRank > 1 {
					// Agent has risen up a rank, start over-withdrawing
					withdrawalAmount := aoaExpectedWithdrawal + cheat_amount // Over-withdraw by 3 if possible to
					if withdrawalAmount > commonPool {
						withdrawalAmount = aoaExpectedWithdrawal //doesn't take whole pool to avoid getting caught
					}
					return withdrawalAmount
				}
				return aoaExpectedWithdrawal
			}
			return aoaExpectedWithdrawal
		case CheatShortTerm:
			// Over-withdraw regardless of rank
			withdrawalAmount := aoaExpectedWithdrawal + cheat_amount
			if withdrawalAmount > commonPool { //takes whatever is left in pool if withdrawalAmount is too much
				withdrawalAmount = commonPool
			}
			return withdrawalAmount
		default:
			return aoaExpectedWithdrawal
		}
	} else {
		log.Println("Agent does not have a team")
		return 0
	}
}

func (a1 *Team1Agent) GetStatedContribution(instance common.IExtendedAgent) int {
	actualContribution := instance.GetActualContribution(instance)
	switch a1.agentType {
	case Honest, CheatLongTerm:
		return actualContribution
	case CheatShortTerm:
		// Overstate the contribution (hardcoded high values - can change later)
		statedContribution := actualContribution + cheat_amount
		if statedContribution > overstate_contribution {
			statedContribution = overstate_contribution
		}
		return statedContribution
	default:
		return actualContribution
	}
}

func (a1 *Team1Agent) GetStatedWithdrawal(instance common.IExtendedAgent) int {
	actualWithdrawal := instance.GetActualWithdrawal(instance)
	switch a1.agentType {
	case Honest, CheatLongTerm:
		return actualWithdrawal
	case CheatShortTerm:
		// Understate the withdrawal
		statedWithdrawal := actualWithdrawal - cheat_amount
		if statedWithdrawal < 0 {
			statedWithdrawal = min_stated_withdrawal
		}
		return statedWithdrawal
	default:
		return actualWithdrawal
	}
}

func (a *Team1Agent) GetAoARanking() []int {
	return []int{1, 2, 5}
}

func (a1 *Team1Agent) hasClimbedRankAndWithdrawn() bool {
	if a1.HasTeam() {
		// Access Team1AoA and check rank changes or over-withdrawals
		teamAoA, ok := a1.Server.GetTeam(a1.GetID()).TeamAoA.(*common.Team1AoA)
		if !ok {
			return false // If unable to access Team1AoA, assume no rank climb
		}
		currentRank := teamAoA.GetAgentNewRank(a1.GetID())
		memoryEntry := a1.memory[a1.GetID()]
		return currentRank > 1 && len(memoryEntry.historyWithdrawal) > 0
	} else {
		log.Println("Agent does not have a team")
		return false
	}
}

// Agent returns their preference for an audit on contribution
// 0: No preference
// 1: Prefer audit
// -1: Prefer no audit
func (a1 *Team1Agent) GetContributionAuditVote() common.Vote {
	// Short-term cheater never votes for audits
	if a1.agentType == CheatShortTerm {
		return common.CreateVote(-1, a1.GetID(), uuid.Nil) // No audit - doesn't want to get caught
	}

	// Honest agent logic
	if a1.agentType == Honest || (a1.agentType == CheatLongTerm && !a1.hasClimbedRankAndWithdrawn()) {
		var suspectID uuid.UUID
		highestStatedContribution := 0

		// Iterate over memory to find the agent with suspiciously high contributions
		// Can be improved by adding a check to compare true common pool value with stated contribution
		for agentID, memoryEntry := range a1.memory {
			// Limit by the last contributions
			relevantContributions := memoryEntry.historyContribution[:memoryEntry.LastContributionCount]
			for _, contribution := range relevantContributions {
				if contribution.ContributionStated > suspicious_contribution && contribution.ContributionStated > highestStatedContribution {
					highestStatedContribution = contribution.ContributionStated
					suspectID = agentID
				}
				// TODO Add functionality to check if stated contribution is lower than expected. 
			}
		}

		if suspectID != uuid.Nil {
			return common.CreateVote(1, a1.GetID(), suspectID) // Vote to audit the suspect
		}
		return common.CreateVote(0, a1.GetID(), uuid.Nil) // No preference if no suspect
	}

	// Long-term cheater avoiding audits if climbing ranks
	if a1.agentType == CheatLongTerm && a1.hasClimbedRankAndWithdrawn() {
		return common.CreateVote(-1, a1.GetID(), uuid.Nil) // Prefer no audit
	}

	return common.CreateVote(0, a1.GetID(), uuid.Nil) // Default: No preference
}

func (a1 *Team1Agent) GetWithdrawalAuditVote() common.Vote {
	// Short-term cheater never votes for audits
	if a1.agentType == CheatShortTerm {
		return common.CreateVote(-1, a1.GetID(), uuid.Nil) // No audit
	}

	// Honest agent logic
	if a1.agentType == Honest || (a1.agentType == CheatLongTerm && !a1.hasClimbedRankAndWithdrawn()) {
		var suspectID uuid.UUID
		highestDiscrepancy := 0

		// Iterate over memory to find the agent with the largest discrepancy
		for agentID, memoryEntry := range a1.memory {
			relevantWithdrawals := memoryEntry.historyWithdrawal[:memoryEntry.LastWithdrawalCount]
			for _, withdrawal := range relevantWithdrawals {
				discrepancy := withdrawal.WithdrawalExpected - withdrawal.WithdrawalStated //expected - stated
				if discrepancy > highestDiscrepancy {
					highestDiscrepancy = discrepancy
					suspectID = agentID
				}
			}
		}

		if suspectID != uuid.Nil {
			return common.CreateVote(1, a1.GetID(), suspectID) // Vote to audit the suspect
		}
		return common.CreateVote(0, a1.GetID(), uuid.Nil) // No preference if no suspect
	}

	// Long-term cheater avoiding audits if climbing ranks
	if a1.agentType == CheatLongTerm && a1.hasClimbedRankAndWithdrawn() {
		return common.CreateVote(-1, a1.GetID(), uuid.Nil) // Prefer no audit
	}

	return common.CreateVote(0, a1.GetID(), uuid.Nil) // Default: No preference
}

func Create_Team1Agent(funcs baseAgent.IExposedServerFunctions[common.IExtendedAgent], agentConfig AgentConfig, ag_type AgentType) *Team1Agent {
	return &Team1Agent{
		ExtendedAgent: GetBaseAgents(funcs, agentConfig),
		memory:        make(map[uuid.UUID]AgentMemory),
		agentType:     ag_type,
	}
}

// ----------------- Messaging functions -----------------------

func (mi *Team1Agent) HandleContributionMessage(msg *common.ContributionMessage) {
	if mi.VerboseLevel > 8 {
		log.Printf("Agent %s received contribution notification from %s: amount=%d\n",
			mi.GetID(), msg.GetSender(), msg.StatedAmount)
	}

	memoryEntry := mi.memory[msg.GetSender()]

	// Modify the historyContribution field
	memoryEntry.historyContribution = append(memoryEntry.historyContribution, AgentContributionInfo{	
		msg.StatedAmount,
		msg.ExpectedAmount,		
	})

	// Update Index
	memoryEntry.LastContributionCount++

	// Update the map with the modified entry
	mi.memory[msg.GetSender()] = memoryEntry
}

func (mi *Team1Agent) HandleScoreReportMessage(msg *common.ScoreReportMessage) {
	if mi.VerboseLevel > 8 {
		log.Printf("Agent %s received score report from %s: score=%d\n",
			mi.GetID(), msg.GetSender(), msg.TurnScore)
	}

	memoryEntry := mi.memory[msg.GetSender()]

	// Modify the historyContribution field
	memoryEntry.historyScore = append(memoryEntry.historyScore, AgentScoreInfo{
		msg.TurnScore,
		msg.Rerolls,
	})

	// Update Index
	memoryEntry.LastScoreCount++

	// Update the map with the modified entry
	mi.memory[msg.GetSender()] = memoryEntry
}

func (mi *Team1Agent) HandleWithdrawalMessage(msg *common.WithdrawalMessage) {
	if mi.VerboseLevel > 8 {
		log.Printf("Agent %s received withdrawal notification from %s: amount=%d\n",
			mi.GetID(), msg.GetSender(), msg.StatedAmount)
	}

	memoryEntry := mi.memory[msg.GetSender()]

	// Modify the historyContribution field
	memoryEntry.historyWithdrawal = append(memoryEntry.historyWithdrawal, AgentWithdrawalInfo{
		msg.StatedAmount,
		msg.ExpectedAmount,
	})

	// Update Index
	memoryEntry.LastWithdrawalCount++

	// Update the map with the modified entry
	mi.memory[msg.GetSender()] = memoryEntry
}

// Get true somas ID (team 1) for debug purposes
func (mi *Team1Agent) GetTrueSomasTeamID() int {
	return 1
}

// Get agent personality type for debug purposes
func (mi *Team1Agent) GetAgentType() int {
	return int(mi.agentType)
}
