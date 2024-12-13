package agents

import (
	// "fmt"
	"log"
	"strconv"

	"github.com/google/uuid"

	"github.com/ADimoska/SOMASExtended/common"
	"github.com/ADimoska/SOMASExtended/gameRecorder"
	baseAgent "github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
)

type AgentScoreInfo struct {
	TurnScore int
	Rerolls   int
}

type AgentContributionInfo struct {
	ContributionStated   int
	ContributionExpected int
}

type AgentWithdrawalInfo struct {
	WithdrawalStated   int
	WithdrawalExpected int
}

type AgentMemory struct {
	honestyScore *common.LeakyQueue

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

	// Rational(Value: 0): Agents who always state what they actually contributed.
	// Withdraw as per their expected withdrawal.
	Rational = iota

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

func (a1 *Team1Agent) AmountToNextRank() int {
	teamAoA, ok := a1.Server.GetTeam(a1.GetID()).TeamAoA.(*common.Team1AoA)
	if !ok {
		// If unable to access Team1AoA, just return 0 - this shouldn't happen
		return 0
	}

	currentRank := teamAoA.GetAgentRank(a1.GetID())
	thresholds := teamAoA.GetRankThresholds()

	// Check if the agent is already at the highest rank
	if currentRank+1 >= len(thresholds) {
		// Already at the highest rank, no "next rank" exists
		return 0
	}

	// Calculate the amount to the next rank
	amountToNextRank := thresholds[currentRank+1] - a1.Score
	return max(amountToNextRank, 0)
}

func (a1 *Team1Agent) GetLatestStatedContributions() int {
	// Fetch the memory for this agent using its ID
	agentMemory, exists := a1.memory[a1.GetID()]
	if !exists {
		return 0 // No memory found for this agent
	}

	// Restrict to relevant contributions based on LastContributionCount
	relevantContributions := agentMemory.historyContribution[:agentMemory.LastContributionCount]

	// Handle cases where there are fewer than 5 contributions
	startIndex := len(relevantContributions) - 5
	if startIndex < 0 {
		startIndex = 0
	}

	// Extract the latest 5 contributions and calculate their sum
	latestContributions := relevantContributions[startIndex:]

	total := 0
	for _, contrib := range latestContributions {
		total += contrib.ContributionStated
	}

	count := len(latestContributions)
	if count == 0 {
		return 0
	}

	return total / count // Integer division
}

func (a1 *Team1Agent) GetActualContribution(instance common.IExtendedAgent) int {
	if a1.HasTeam() {
		actualContribution := 0
		avg_last_5_contributions := a1.GetLatestStatedContributions()
		switch a1.agentType {
		case Rational, CheatLongTerm:
			//if threshold known - try to rise up a rank, without dying
			knownThreshold, ok := a1.Server.GetTeam(a1.GetID()).GetKnownThreshold()
			if ok {
				if a1.AmountToNextRank()-(4*avg_last_5_contributions) < (a1.Score - knownThreshold) {
					actualContribution = a1.AmountToNextRank() - (4 * avg_last_5_contributions)
				} else if avg_last_5_contributions < (a1.Score - knownThreshold) {
					//then contribute last n contributions
					actualContribution = avg_last_5_contributions
				} else {
					actualContribution = a1.Score
				}
			} else {
				//if threshold unknown - contribute max(check difference to next rank, 30% of savings)
				actualContribution = int(max(float64(a1.AmountToNextRank()-(4*avg_last_5_contributions)), (0.3 * float64(a1.Score))))
				if actualContribution > a1.Score {
					actualContribution = a1.Score
				}
			}
			return actualContribution
		case CheatShortTerm:
			// Doesn't Contribute anything
			return 0
		default:
			return actualContribution
		}
	} else {
		// log.Println("Agent does not have a team")
		return 0
	}
}

func (a1 *Team1Agent) GetActualWithdrawal(instance common.IExtendedAgent) int {
	if a1.HasTeam() {
		commonPool := a1.Server.GetTeam(a1.GetID()).GetCommonPool()
		aoaExpectedWithdrawal := a1.Server.GetTeam(a1.GetID()).TeamAoA.GetExpectedWithdrawal(a1.GetID(), a1.Score, commonPool)
		currentRank := 0

		decision := 0
		switch a1.agentType {
		case Rational:
			decision = aoaExpectedWithdrawal
		case CheatLongTerm:
			// Perform type assertion to get Team1AoA
			teamAoA, ok := a1.Server.GetTeam(a1.GetID()).TeamAoA.(*common.Team1AoA)
			if ok {
				currentRank = teamAoA.GetAgentRank(a1.GetID())
				if currentRank > 1 {
					// Agent has risen up a rank, start over-withdrawing
					withdrawalAmount := aoaExpectedWithdrawal + cheat_amount // Over-withdraw by 3 if possible to
					if withdrawalAmount > commonPool {
						withdrawalAmount = aoaExpectedWithdrawal //doesn't take whole pool to avoid getting caught
					}
					decision = withdrawalAmount
				} else {
					decision = aoaExpectedWithdrawal
				}
			} else {
				decision = aoaExpectedWithdrawal
			}
		case CheatShortTerm:
			// Over-withdraw regardless of rank
			withdrawalAmount := aoaExpectedWithdrawal + cheat_amount
			if withdrawalAmount > commonPool { //takes whatever is left in pool if withdrawalAmount is too much
				withdrawalAmount = commonPool
			}
			decision = withdrawalAmount
		default:
			decision = aoaExpectedWithdrawal
		}
		/* If the threshold is known (this only occurs in some games), then just
		   withdraw the minimum you need to survive. */
		knownThreshold, ok := a1.Server.GetTeam(a1.GetID()).GetKnownThreshold()
		if ok {
			survival := int(max(float64(knownThreshold)-float64(a1.Score), 0.0))
			return int(max(float64(decision), float64(survival)))
		}
		return decision
	} else {
		// log.Println("Agent does not have a team")
		return 0
	}
}

func (a1 *Team1Agent) GetStatedContribution(instance common.IExtendedAgent) int {
	if a1.HasTeam() {
		actualContribution := instance.GetActualContribution(instance)
		switch a1.agentType {
		case Rational, CheatLongTerm:
			return actualContribution
		case CheatShortTerm:
			_, ok := a1.Server.GetTeam(a1.GetID()).TeamAoA.(*common.Team1AoA)
			if !ok {
				// If unable to access Team1AoA, just use actual contribution with some fixed cheating value
				return actualContribution + overstate_contribution
			}
			//State what they would have contributed to climb the next rank (but didn't actually do)
			statedContribution := a1.AmountToNextRank()
			return statedContribution
		default:
			return actualContribution
		}

	} else {
		return 0
	}
}

func (a1 *Team1Agent) GetStatedWithdrawal(instance common.IExtendedAgent) int {
	actualWithdrawal := instance.GetActualWithdrawal(instance)
	switch a1.agentType {
	case Rational, CheatLongTerm:
		return actualWithdrawal
	case CheatShortTerm:
		// Understate the withdrawal by fixed amount = 3
		statedWithdrawal := actualWithdrawal - cheat_amount //TO_CHECK: Are we happy with this?
		if statedWithdrawal < 0 {
			statedWithdrawal = min_stated_withdrawal // = 1
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
		currentRank := teamAoA.GetAgentRank(a1.GetID())
		memoryEntry := a1.memory[a1.GetID()]
		return currentRank > 1 && len(memoryEntry.historyWithdrawal) > 0
	} else {
		// log.Println("Agent does not have a team")
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

	if a1.agentType == Rational {
		team := a1.Server.GetTeam(a1.GetID())
		agentsInTeam := team.Agents
		minHonestyScore := 0
		agentToAudit := uuid.Nil
		for _, agentID := range agentsInTeam {
			if agentMemory, exists := a1.memory[agentID]; exists {
				currHonestyScore := agentMemory.honestyScore.Sum()
				if currHonestyScore < 0 && currHonestyScore < minHonestyScore {
					minHonestyScore = currHonestyScore
					agentToAudit = agentID
				}
			}
		}

		if minHonestyScore < 0 && agentToAudit != uuid.Nil {
			return common.CreateVote(1, a1.GetID(), agentToAudit)
		}

	}

	// Rational agent logic
	if a1.agentType == Rational || (a1.agentType == CheatLongTerm && !a1.hasClimbedRankAndWithdrawn()) {

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

	// Rational agent logic
	if a1.agentType == Rational || (a1.agentType == CheatLongTerm && !a1.hasClimbedRankAndWithdrawn()) {
		var suspectID uuid.UUID
		highestDiscrepancy := 0

		if a1.agentType == Rational {
			team := a1.Server.GetTeam(a1.GetID())
			agentsInTeam := team.Agents
			minHonestyScore := 0
			agentToAudit := uuid.Nil
			for _, agentID := range agentsInTeam {
				if agentMemory, exists := a1.memory[agentID]; exists {
					currHonestyScore := agentMemory.honestyScore.Sum()
					if currHonestyScore < 0 && currHonestyScore < minHonestyScore {
						minHonestyScore = currHonestyScore
						agentToAudit = agentID
					}
				}
			}

			if minHonestyScore < 0 && agentToAudit != uuid.Nil {
				return common.CreateVote(1, a1.GetID(), agentToAudit)
			}

		}

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

func (a1 *Team1Agent) AddAgentToMemory(agentID uuid.UUID, honestyScoreLength int) {
	// Add agent to memory if not already present
	if _, exists := a1.memory[agentID]; !exists {
		a1.memory[agentID] = AgentMemory{
			honestyScore: common.NewLeakyQueue(5),

			// TODO: Do the other fields need to be initialized?
			// No??? should exists as nil slices that can be appended to

		}
	}

}

func (a1 *Team1Agent) SetAgentContributionAuditResult(agentID uuid.UUID, result bool) {
	// Update the memory of the agent who was audited

	// check that the agent has been added to the memory
	// if not, add the agent to the memory
	if _, exists := a1.memory[agentID]; !exists {
		a1.AddAgentToMemory(agentID, 5)
	}

	if result {
		// Agent was dishonest
		// check that agent is in memory
		a1.memory[agentID].honestyScore.Push(-1)
	} else {
		// agent was honest
		a1.memory[agentID].honestyScore.Push(1)
	}
}

func (a1 *Team1Agent) SetAgentWithdrawalAuditResult(agentID uuid.UUID, result bool) {
	// Update the memory of the agent who was audited

	// check that the agent has been added to the memory
	// if not, add the agent to the memory
	if _, exists := a1.memory[agentID]; !exists {
		a1.AddAgentToMemory(agentID, 5)
	}

	if result {
		// Agent was dishonest
		// check that agent is in memory
		a1.memory[agentID].honestyScore.Push(-1)
	} else {
		// agent was honest
		a1.memory[agentID].honestyScore.Push(1)
	}
}

func (a1 *Team1Agent) VoteOnAgentEntry(candidateID uuid.UUID) bool {
	// Look at the honesty map of an agent
	// If the agent has a negative score, they are dishonest
	// If the agent has a positive score, they are honest
	value, exists := a1.memory[candidateID]

	if !exists {
		return true
	}

	switch a1.agentType {
	case Rational:
		return value.honestyScore.Sum() > 0
	default:
		return true

	}
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

	// check that the agent has been intialised in the memory
	if _, exists := mi.memory[msg.GetSender()]; !exists {
		mi.AddAgentToMemory(msg.GetSender(), 5)
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

	// check that the agent has been intialised in the memory
	if _, exists := mi.memory[msg.GetSender()]; !exists {
		mi.AddAgentToMemory(msg.GetSender(), 5)
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

	// check that the agent has been intialised in the memory
	if _, exists := mi.memory[msg.GetSender()]; !exists {
		mi.AddAgentToMemory(msg.GetSender(), 5)
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

// ----------------------- Data Recording Functions -----------------------
func (mi *Team1Agent) RecordAgentStatus(instance common.IExtendedAgent) gameRecorder.AgentRecord {

	specialNote := "-1"
	if mi.HasTeam() {
		teamAoA := mi.Server.GetTeam(instance.GetID()).TeamAoA
		switch teamAoA := teamAoA.(type) {
		case *common.Team1AoA:
			specialNote = "Rank: " + strconv.Itoa(teamAoA.GetAgentRank(instance.GetID()))
		}

	}

	record := gameRecorder.NewAgentRecord(
		instance.GetID(),
		instance.GetTrueSomasTeamID(),
		instance.GetTrueScore(),
		instance.GetStatedContribution(instance),
		instance.GetActualContribution(instance),
		instance.GetActualWithdrawal(instance),
		instance.GetStatedWithdrawal(instance),
		instance.GetTeamID(),
		specialNote,
	)
	return record
}
