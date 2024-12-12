package agents

import (
	"fmt"
	"math"
	"math/rand"

	common "github.com/ADimoska/SOMASExtended/common"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
)

type MI_256_v1 struct {
	*ExtendedAgent

	//character matrix:
	chaoticness int // from 1 to 3, 3 being most chaotic
	evilness    int // from 1 to 3, 3 being most evil

	//opinions
	affinity map[uuid.UUID]int // has opinions for each agent in the game
	mood     int               // starting from 0

	affinityChange map[uuid.UUID]int // the total change in affinity this turn

	// store of other character's states (what the id of the agents in the team )
	teamAgentsDeclaredRolls        map[uuid.UUID]int
	teamAgentsDeclaredContribution map[uuid.UUID]int
	teamAgentsDeclaredWithdraw     map[uuid.UUID]int
	teamAgentsExpectedScore        map[uuid.UUID]int

	teamAgentsExpectedContribution map[uuid.UUID]int
	teamAgentsExpectedWithdraw     map[uuid.UUID]int

	// store of the other character's states during an audit
	lastAuditTarget  uuid.UUID
	lastVotes        map[uuid.UUID]bool
	lastAuditStarter uuid.UUID
	lastAuditResult  bool

	// TODO: internal states

	// Information I need
	last_common_pool int

	//AOA parameterss
	AOAOpinion              map[uuid.UUID]int
	AoAExpectedContribution int
	AoAExpectedWithdrawal   int
	AoAAuditCost            int
	AoAPunishment           int

	isAoAContributionFixed bool
	isAoAWithdrawalFixed   bool

	//Intended Withdrawal and Contribution
	IntendedWithdrawal   int
	declaredWithdrawal   int
	intendedContribution int
	declaredcontribution int

	isThereCheatWithdrawal   bool
	cheatWithdrawalDiff      int
	isThereCheatContribution bool
	cheatContributeDiff      int
	haveIlied                bool
	IcaughtLying             bool

	lastThreshold int
	lastTurnScore int
}

func (mi *MI_256_v1) print_alignment() {
	fmt.Println(mi.GetID(), " has been created. Chaoticness:", mi.chaoticness, "Evilness:", mi.evilness)
}

// constructor for MI_256_v1
func Team4_CreateAgent(funcs agent.IExposedServerFunctions[common.IExtendedAgent], agentConfig AgentConfig) *MI_256_v1 {
	mi_256 := &MI_256_v1{
		ExtendedAgent: GetBaseAgents(funcs, agentConfig),
	}
	mi_256.TrueSomasTeamID = 4 // IMPORTANT: add your team number here!
	mi_256.RandomizeCharacter()
	mi_256.teamAgentsDeclaredRolls = make(map[uuid.UUID]int)
	mi_256.teamAgentsDeclaredContribution = make(map[uuid.UUID]int)
	mi_256.teamAgentsDeclaredWithdraw = make(map[uuid.UUID]int)
	mi_256.teamAgentsExpectedScore = make(map[uuid.UUID]int)
	mi_256.teamAgentsExpectedContribution = make(map[uuid.UUID]int)
	mi_256.teamAgentsExpectedWithdraw = make(map[uuid.UUID]int)
	mi_256.affinity = make(map[uuid.UUID]int)
	mi_256.affinityChange = make(map[uuid.UUID]int)

	fmt.Println(mi_256.GetID(), " has been created. Chaoticness:", mi_256.chaoticness, "Evilness:", mi_256.evilness)
	return mi_256
}

// ----------------------- Function Override -----------------------
func (mi *MI_256_v1) SetAgentContributionAuditResult(agentID uuid.UUID, result bool) {
	mi.lastAuditTarget = agentID
	mi.lastAuditResult = result
}

func (mi *MI_256_v1) SetAgentWithdrawalAuditResult(agentID uuid.UUID, result bool) {
	mi.lastAuditTarget = agentID
	mi.lastAuditResult = result
}

// ----functions to calculate the data of other team members------------
func (mi *MI_256_v1) UpdateTeamDeclaredContribution() {
	for _, agent := range mi.Server.GetTeam(mi.GetID()).Agents {
		mi.teamAgentsDeclaredContribution[agent] = mi.GetStatedContribution(mi.Server.AccessAgentByID(agent))
		// numAgent += 1
	}
}
func (mi *MI_256_v1) UpdateTeamDeclaredWithdrawal() {
	for _, agent := range mi.Server.GetTeam(mi.GetID()).Agents {
		mi.teamAgentsDeclaredWithdraw[agent] = mi.GetStatedWithdrawal(mi.Server.AccessAgentByID(agent))
		// numAgent += 1
	}
}

func (mi *MI_256_v1) Team4_UpdateStateAfterContribution() {
	mi.UpdateTeamDeclaredContribution()
	mi.UpdateAffinityAfterContribute()

}
func (mi *MI_256_v1) Team4_UpdateStateAfterWithdrawal() {
	mi.UpdateTeamDeclaredWithdrawal()
	mi.UpdateAffinityAfterWithdraw()

}
func (mi *MI_256_v1) Team4_UpdateStateAfterContributionAudit() {
	mi.UpdateAffinityAfterVote()
	mi.UpdateAffinityAfterAudit()

}
func (mi *MI_256_v1) Team4_UpdateStateAfterWithdrawalAudit() {
	mi.UpdateAffinityAfterVote()
	mi.UpdateAffinityAfterAudit()
	mi.UpdateMoodAfterAuditionEnd()

}
func (mi *MI_256_v1) Team4_UpdateStateAfterRoll() {
	mi.UpdateMoodAfterRoll()

}
func (mi *MI_256_v1) Team4_UpdateStateTurnend() {
	mi.UpdateMoodAfterRoundEnd()
	mi.lastTurnScore = mi.Score

}

//------functions to calculated the expected AOA contributions and stuff for all agents -------------------------------

func (mi *MI_256_v1) CalcAOAContibution() int {

	// mi.AoAExpectedContribution = int(0.5 * float64(mi.Score))

	mi.AoAExpectedContribution = mi.Server.GetTeam(mi.GetID()).TeamAoA.GetExpectedContribution(mi.GetID(), mi.GetTrueScore())
	fmt.Println(mi.GetID(), " the expected contribution is:", mi.AoAExpectedContribution)
	mi.isAoAContributionFixed = true
	return mi.AoAExpectedContribution

}
func (mi *MI_256_v1) CalcAOAWithdrawal() int {
	// common_pool := mi.Server.GetTeam(mi.GetID()).GetCommonPool()

	// mi.AoAExpectedWithdrawal = int(common_pool / (len(mi.Server.GetTeam(mi.GetID()).Agents) + 1))
	mi.AoAExpectedWithdrawal = mi.Server.GetTeam(mi.GetID()).TeamAoA.GetExpectedWithdrawal(mi.GetID(), mi.GetTrueScore(), mi.Server.GetTeamCommonPool(mi.GetTeamID()))

	fmt.Println(mi.GetID(), " the expected withdrawal is:", mi.AoAExpectedWithdrawal)
	mi.isAoAWithdrawalFixed = true

	return mi.AoAExpectedWithdrawal
}
func (mi *MI_256_v1) CalcAOAAuditCost() {

}
func (mi *MI_256_v1) CalcAOAPunishment() {

}
func (mi *MI_256_v1) CalcAOAOpinion() {

}
func (mi *MI_256_v1) SetAoARanking(Preferences []int) {
	mi.AoARanking = Preferences
}

// ----------------------- Strategies -----------------------

func (mi *MI_256_v1) AnyoneCheatedAfterContribute() {
	common_pool := mi.Server.GetTeam(mi.GetID()).GetCommonPool()
	change := common_pool - mi.last_common_pool
	mi.last_common_pool = common_pool
	sum := 0
	for _, agent := range mi.Server.GetTeam(mi.GetID()).Agents {
		sum += mi.teamAgentsDeclaredContribution[agent]
		// numAgent += 1
	}
	if sum != change {
		mi.isThereCheatContribution = true
	} else {
		mi.isThereCheatContribution = false
	}
	mi.cheatContributeDiff = sum - change

	// get common pool before and after contribution, compare with the total declared contribution, to see if there is a cheat this round

}
func (mi *MI_256_v1) AnyoneCheatedAfterWithdrawal() {
	common_pool := mi.Server.GetTeam(mi.GetID()).GetCommonPool()
	change := common_pool - mi.last_common_pool
	mi.last_common_pool = common_pool
	sum := 0
	for _, agent := range mi.Server.GetTeam(mi.GetID()).Agents {
		sum += mi.teamAgentsDeclaredWithdraw[agent]
		// numAgent += 1
	}
	if sum != change {
		mi.isThereCheatWithdrawal = true
	} else {
		mi.isThereCheatWithdrawal = false
	}
	mi.cheatWithdrawalDiff = sum - change

}

// // Team-forming Strategy
// func (mi *MI_256_v1) DecideTeamForming(agentInfoList []common.ExposedAgentInfo) []uuid.UUID {
// 	invitationList := []uuid.UUID{}
// 	for _, agentInfo := range agentInfoList {
// 		// exclude the agent itself
// 		if agentInfo.AgentUUID == mi.GetID() {
// 			continue
// 		}
// 		if agentInfo.AgentTeamID == (uuid.UUID{}) {
// 			invitationList = append(invitationList, agentInfo.AgentUUID)
// 		}

// 	}

// 	// TODO: implement team forming logic
// 	// random choice from the invitation list
// 	rand.Shuffle(len(invitationList), func(i, j int) { invitationList[i], invitationList[j] = invitationList[j], invitationList[i] })
// 	chosenAgent := invitationList[0]

// 	// Return a slice containing the chosen agent
// 	return []uuid.UUID{chosenAgent}
// }

// Dice Strategy
func (mi *MI_256_v1) StickOrAgain(accumulatedScore int, prevRoll int) bool {

	// fmt.Printf("Called overriden StickOrAgain\n")

	// the higher the mood, the more risky the roll dice strategy will be

	threshLow := 9
	threshMid := 12
	threshHigh := 14
	moodthresh := 10
	if mi.mood > moodthresh { // be greedy
		if mi.LastScore < threshHigh {
			return false
		} else {
			return true
		}
	} else if (-moodthresh < mi.mood) && (mi.mood < moodthresh) {
		if mi.LastScore < threshMid {
			return false
		} else {
			return true
		}
	} else {
		if mi.LastScore < threshLow {
			return false
		} else {
			return true
		}
	}

}

// !!! NOTE: name and signature of functions below are subject to change by the infra team !!!

// definition of Evilness: The more he would value his own benifit over others.
// definition of Chaoticness: The willingness to take risk to breach the AoA
// definition of Mood: The willingness to take actions which diverges from abosolute neutral
// Contribution Strategy
func (mi *MI_256_v1) DecideContribution() int {
	mi.CalcAOAContibution()
	if mi.Score == 0 || mi.AoAExpectedContribution == 0 {
		mi.intendedContribution = 0
		mi.declaredcontribution = 0
		return 0

	}

	//parameters: AoAExpectedContribution, Mood, Lawfulness,Evilness，  CurrentScore
	mi.intendedContribution = 0
	contribute_percentage := 0.0
	// CurrentScore = mi.Score

	mood_modifier := float64(1.0 / 100.0 * float64(mi.mood))
	neutral_mean := (float64(mi.AoAExpectedContribution) / float64(min(mi.Score, 2*mi.AoAExpectedContribution))) //normalize from 0-1
	//we model evil and good with different standard deviations, with chaotic being high standard deviation,
	lawful_evil_mean := neutral_mean * 5 / 8
	neutral_evil_mean := neutral_mean * 2 / 4
	neutral_good_mean := (1-neutral_mean)/2 + neutral_mean
	lawful_good_mean := float64(7.0 / 8)
	neutral_standard_deviation := float64(1.0 / 12)
	lawful_standard_deviation := 1.0 / 16.0
	chaotic_standard_deviation := 1.0 / 8
	chaotic_good_standard_deviation := 3.0 / 8 * (1 - neutral_mean)
	chaotic_evil_standard_deviation := 3.0 / 8 * neutral_mean
	neutral_good_standard_deviation := 1.0 / 8 * (1 - neutral_mean)
	neutral_evil_standard_deviation := 1.0 / 8 * neutral_mean
	lawful_good_standard_deviation := 1.0 / 16 * (1 - neutral_mean)
	lawful_evil_standard_deviation := 1.0 / 16 * neutral_mean
	// fmt.Println(mi.GetID(), " neutral_mean", neutral_mean)
	// fmt.Println(mi.GetID(), " lawful_evil_mean", lawful_evil_mean)
	// fmt.Println(mi.GetID(), " neutral_evil_mean", neutral_evil_mean)

	Apply_mood := func(contribute_percentage float64) float64 {
		if contribute_percentage > float64(neutral_mean) {
			return contribute_percentage + mood_modifier
		} else {
			return contribute_percentage - mood_modifier
		}
	}

	// if the agent is a lawful agent, he would tend to not break the AoA
	// if the agent is a good agent, he would tend to contribute more than he should
	// absolute neutral: donating as much as the AoA asks.

	// we set a linear specturm for the strategies and calculate where we would end up in this scale
	//
	// Each personality would have a mean on that scale, and would have standard deviations based on the chaoticness

	if mi.evilness == 2 { // if agent has neutral evilness, use neutral mean
		if mi.chaoticness == 1 { // if lawful neutral

			contribute_percentage = (rand.NormFloat64() * float64(lawful_standard_deviation)) + float64(neutral_mean)

		} else if mi.chaoticness == 2 { // abs neutral
			contribute_percentage = (rand.NormFloat64() * float64(neutral_standard_deviation)) + float64(neutral_mean)

		} else if mi.chaoticness == 3 { //chaotic neutral
			contribute_percentage = (rand.NormFloat64() * float64(chaotic_standard_deviation)) + float64(neutral_mean)

		}

	} else if mi.evilness == 1 { // if agent is good
		if mi.chaoticness == 1 { // if lawful good

			contribute_percentage = (rand.NormFloat64() * float64(lawful_good_standard_deviation)) + float64(lawful_good_mean)

		} else if mi.chaoticness == 2 { // neutral good
			contribute_percentage = (rand.NormFloat64() * float64(neutral_good_standard_deviation)) + float64(neutral_good_mean)

		} else if mi.chaoticness == 3 { //chaotic good
			contribute_percentage = (rand.NormFloat64() * float64(chaotic_good_standard_deviation)) + float64(neutral_good_mean)

		}

	} else if mi.evilness == 3 { // if agent is evil
		if mi.chaoticness == 1 { // if lawful evil

			contribute_percentage = (rand.NormFloat64() * float64(lawful_evil_standard_deviation)) + float64(lawful_evil_mean)

		} else if mi.chaoticness == 2 { // neutral evil
			contribute_percentage = (rand.NormFloat64() * float64(neutral_evil_standard_deviation)) + float64(neutral_evil_mean)

		} else if mi.chaoticness == 3 { //chaotic evil
			contribute_percentage = (rand.NormFloat64() * float64(chaotic_evil_standard_deviation)) + float64(neutral_evil_mean)

		}
	}
	contribute_percentage = Apply_mood(contribute_percentage)
	fmt.Println(mi.GetID(), " contribution percentage", contribute_percentage)
	mi.print_alignment()
	mi.intendedContribution = min(max(int(math.Round(float64(contribute_percentage)*float64(min(mi.Score, 2*mi.AoAExpectedContribution)))), 0), mi.Score)

	// how much to declare:
	// if you contributed less, there is no point to lie ( if audition checks against the expected contribution)
	if mi.isAoAContributionFixed {
		mi.declaredcontribution = max(mi.intendedContribution, mi.AoAExpectedContribution)
	} else {
		if mi.intendedContribution > 0 {
			mi.declaredcontribution = mi.intendedContribution
		} else {
			mi.declaredcontribution = rand.Intn(11) + 2
		}
	}
	if mi.intendedContribution < mi.AoAExpectedContribution {
		mi.haveIlied = true
	}

	return mi.intendedContribution
}

func (mi *MI_256_v1) GetActualContribution(instance common.IExtendedAgent) int {
	if mi.HasTeam() {
		mi.intendedContribution = mi.DecideContribution()
		return mi.intendedContribution
	} else {
		if mi.VerboseLevel > 6 {
		}
		return 0
	}
}

func (mi *MI_256_v1) GetStatedContribution(instance common.IExtendedAgent) int {
	return mi.declaredcontribution
}

// Withdrawal Strategy
func (mi *MI_256_v1) DecideWithdrawal(upperbound int) int {
	fmt.Print("deciding withdrawal")
	mi.CalcAOAWithdrawal()
	if mi.AoAExpectedWithdrawal == 0 {
		mi.IntendedWithdrawal = 0
		mi.declaredWithdrawal = 0
		return 0
	}

	mi.IntendedWithdrawal = 0
	Withdrawal_percentage := 0.0
	// CurrentScore = mi.Score

	mood_modifier := float64(1.0 / 100.0 * float64(mi.mood))
	// we scale from 0-2*AoAWithdrawal
	neutral_mean := float64(float64(mi.AoAExpectedWithdrawal) / float64((upperbound))) //normalize from 0-1
	//we model evil and good with different standard deviations, with chaotic being high standard deviation,
	lawful_evil_mean := neutral_mean * 9 / 8
	neutral_evil_mean := neutral_mean/4 + neutral_mean
	neutral_good_mean := neutral_mean * 7.0 / 8
	lawful_good_mean := neutral_mean * 3.0 / 4
	neutral_standard_deviation := float64(1.0 / 12)
	lawful_standard_deviation := 1.0 / 16.0
	chaotic_standard_deviation := 1.0 / 8
	chaotic_good_standard_deviation := 2.0 / 8 * (1 - neutral_mean)
	chaotic_evil_standard_deviation := 2.0 / 8 * neutral_mean
	neutral_good_standard_deviation := 1.0 / 8 * (1 - neutral_mean)
	neutral_evil_standard_deviation := 1.0 / 8 * neutral_mean
	lawful_good_standard_deviation := 1.0 / 16 * (1 - neutral_mean)
	lawful_evil_standard_deviation := 1.0 / 8 * neutral_mean

	Apply_mood := func(Withdrawal_percentage float64) float64 {
		if Withdrawal_percentage > float64(neutral_mean) {
			return Withdrawal_percentage + mood_modifier
		} else {
			return Withdrawal_percentage - mood_modifier
		}
	}
	if mi.evilness == 2 { // if agent has neutral evilness, use neutral mean
		if mi.chaoticness == 1 { // if lawful neutral

			Withdrawal_percentage = (rand.NormFloat64() * float64(lawful_standard_deviation)) + float64(neutral_mean)

		} else if mi.chaoticness == 2 { // abs neutral
			Withdrawal_percentage = (rand.NormFloat64() * float64(neutral_standard_deviation)) + float64(neutral_mean)

		} else if mi.chaoticness == 3 { //chaotic neutral
			Withdrawal_percentage = (rand.NormFloat64() * float64(chaotic_standard_deviation)) + float64(neutral_mean)

		}

	} else if mi.evilness == 1 { // if agent is good
		if mi.chaoticness == 1 { // if lawful good

			Withdrawal_percentage = (rand.NormFloat64() * float64(lawful_good_standard_deviation)) + float64(lawful_good_mean)

		} else if mi.chaoticness == 2 { // neutral good
			Withdrawal_percentage = (rand.NormFloat64() * float64(neutral_good_standard_deviation)) + float64(neutral_good_mean)

		} else if mi.chaoticness == 3 { //chaotic good
			Withdrawal_percentage = (rand.NormFloat64() * float64(chaotic_good_standard_deviation)) + float64(neutral_good_mean)

		}

	} else if mi.evilness == 3 { // if agent is evil
		if mi.chaoticness == 1 { // if lawful evil

			Withdrawal_percentage = (rand.NormFloat64() * float64(lawful_evil_standard_deviation)) + float64(lawful_evil_mean)

		} else if mi.chaoticness == 2 { // neutral evil
			Withdrawal_percentage = (rand.NormFloat64() * float64(neutral_evil_standard_deviation)) + float64(neutral_evil_mean)

		} else if mi.chaoticness == 3 { //chaotic evil
			Withdrawal_percentage = (rand.NormFloat64() * float64(chaotic_evil_standard_deviation)) + float64(neutral_evil_mean)

		}
	}
	Withdrawal_percentage = Apply_mood(Withdrawal_percentage)
	fmt.Println(mi.GetID(), " withdrawal percentage", Withdrawal_percentage)
	mi.print_alignment()
	mi.IntendedWithdrawal = max(int(math.Round(float64(Withdrawal_percentage)*float64(upperbound))), 0)

	// how much to declare:
	// if you withdrawed more, there is no point to lie ( if audition checks against the expected )
	if mi.isAoAWithdrawalFixed {
		mi.declaredWithdrawal = min(mi.IntendedWithdrawal, mi.AoAExpectedWithdrawal)
	} else {
		mi.declaredWithdrawal = mi.IntendedWithdrawal
	}
	if mi.IntendedWithdrawal > mi.AoAExpectedContribution {
		mi.haveIlied = true
	}

	// TODO: implement contribution strategy
	return mi.IntendedWithdrawal
}

func (mi *MI_256_v1) GetActualWithdrawal(instance common.IExtendedAgent) int {
	// first check if the agent has a team
	if !mi.HasTeam() {
		return 0
	}
	commonPool := mi.Server.GetTeam(mi.GetID()).GetCommonPool()
	mi.AoAExpectedWithdrawal = mi.CalcAOAWithdrawal()
	if mi.Score < mi.lastThreshold+5 {
		mi.IntendedWithdrawal = mi.DecideWithdrawal((commonPool))
	} else {
		mi.IntendedWithdrawal = mi.DecideWithdrawal((mi.AoAExpectedWithdrawal * 2))
	}
	// commonPool := mi.Server.GetTeam(mi.GetID()).GetCommonPool()
	// withdrawal := mi.Server.GetTeam(mi.GetID()).TeamAoA.GetExpectedWithdrawal(mi.GetID(), mi.GetTrueScore(), commonPool)
	// if commonPool < withdrawal {
	// 	withdrawal = commonPool
	// }

	return mi.IntendedWithdrawal
}

func (mi *MI_256_v1) GetStatedWithdrawal(instance common.IExtendedAgent) int {
	return mi.declaredWithdrawal
}

// Audit Strategy
func (mi *MI_256_v1) BuildVotemap(IsHarmfulIntent bool) map[uuid.UUID]float64 {
	// you decide who to audit based on your alignment, if there is a cheat happened, and your affinity of other agents, and the net affinity change this round
	// alignment calculates the likelyhood of you initiating an audit
	// cheat greatly modifies the likelyhood, if there is cheating detected, then there will be a greater chance of auditing
	// affinity factors in the suspision,  if someone is declaring large contributions when there is a cheat, taking small sums when there are loads missing from the pool
	mood_modifier := float64(1.0 / 100.0 * float64(mi.mood))
	Apply_mood := func(audit_percentage, neutral_mean float64) float64 {
		if audit_percentage > float64(neutral_mean) {
			return audit_percentage + mood_modifier
		} else {
			return audit_percentage - mood_modifier
		}
	}
	auditmap := make(map[uuid.UUID]float64)

	var abs_neutral_mean, neutral_evil_mean, neutral_good_mean, lawful_good_mean, lawful_evil_mean, lawful_neutral_mean, chaotic_neutral_mean, chaotic_evil_mean, chaotic_good_mean float64
	var chaotic_neutral_standard_deviation, chaotic_good_standard_deviation, chaotic_evil_standard_deviation, neutral_good_standard_deviation, neutral_evil_standard_deviation, abs_neutral_standard_deviation, lawful_good_standard_deviation, lawful_evil_standard_deviation, lawful_neutral_standard_deviation float64

	if IsHarmfulIntent {

		if !(mi.isThereCheatWithdrawal || mi.isThereCheatContribution) { // when no cheating happens

			//we model evil and good with different standard deviations, with chaotic being high standard deviation,
			abs_neutral_mean = 0.1
			neutral_evil_mean = 0.15
			neutral_good_mean = 0.05

			lawful_good_mean = 0.05
			lawful_evil_mean = 0.1
			lawful_neutral_mean = 0.075

			chaotic_neutral_mean = 0.15
			chaotic_evil_mean = 0.2
			chaotic_good_mean = 0.1

			chaotic_neutral_standard_deviation = 0.07
			chaotic_good_standard_deviation = 0.05
			chaotic_evil_standard_deviation = 0.1

			neutral_good_standard_deviation = 0.05
			neutral_evil_standard_deviation = 0.05
			abs_neutral_standard_deviation = 0.05

			lawful_good_standard_deviation = 0.025
			lawful_evil_standard_deviation = 0.025
			lawful_neutral_standard_deviation = 0.025

		} else { // when there is a cheat
			abs_neutral_mean = 0.5
			neutral_evil_mean = 0.75
			neutral_good_mean = 0.25

			lawful_good_mean = 0.1
			lawful_evil_mean = 0.6
			lawful_neutral_mean = 0.5

			chaotic_neutral_mean = 0.5
			chaotic_evil_mean = 0.85
			chaotic_good_mean = 0.25

			chaotic_neutral_standard_deviation = 0.2
			chaotic_good_standard_deviation = 0.2
			chaotic_evil_standard_deviation = 0.15

			neutral_good_standard_deviation = 0.15
			neutral_evil_standard_deviation = 0.15
			abs_neutral_standard_deviation = 0.15

			lawful_good_standard_deviation = 0.1
			lawful_evil_standard_deviation = 0.1
			lawful_neutral_standard_deviation = 0.1

		}
	} else {
		// if this vote is to help them rank up, get more resource and etc.
		if !(mi.isThereCheatWithdrawal || mi.isThereCheatContribution) { // when no cheating happens

			//we model evil and good with different standard deviations, with chaotic being high standard deviation,
			abs_neutral_mean = 0.4
			neutral_evil_mean = 0.15
			neutral_good_mean = 0.4

			lawful_good_mean = 0.8
			lawful_evil_mean = 0.2
			lawful_neutral_mean = 0.4

			chaotic_neutral_mean = 0.4
			chaotic_evil_mean = 0.15
			chaotic_good_mean = 0.8

			chaotic_neutral_standard_deviation = 0.2
			chaotic_good_standard_deviation = 0.2
			chaotic_evil_standard_deviation = 0.2

			neutral_good_standard_deviation = 0.1
			neutral_evil_standard_deviation = 0.1
			abs_neutral_standard_deviation = 0.1

			lawful_good_standard_deviation = 0.05
			lawful_evil_standard_deviation = 0.05
			lawful_neutral_standard_deviation = 0.05

		} else { // when there is a cheat
			abs_neutral_mean = 0.25
			neutral_evil_mean = 0.1
			neutral_good_mean = 0.4

			lawful_good_mean = 0.5
			lawful_evil_mean = 0.1
			lawful_neutral_mean = 0.25

			chaotic_neutral_mean = 0.25
			chaotic_evil_mean = 0.1
			chaotic_good_mean = 0.4

			chaotic_neutral_standard_deviation = 0.2
			chaotic_good_standard_deviation = 0.2
			chaotic_evil_standard_deviation = 0.2

			neutral_good_standard_deviation = 0.1
			neutral_evil_standard_deviation = 0.1
			abs_neutral_standard_deviation = 0.1

			lawful_good_standard_deviation = 0.05
			lawful_evil_standard_deviation = 0.05
			lawful_neutral_standard_deviation = 0.05

		}
	}

	for _, agent := range mi.Server.GetTeam(mi.GetID()).Agents {
		audit_percentage := 0.0

		// we need to genereate a probability for each agent
		if mi.evilness == 2 { // if agent has neutral evilness, use neutral mean
			if mi.chaoticness == 1 { // if lawful neutral

				audit_percentage = (rand.NormFloat64() * float64(lawful_neutral_standard_deviation)) + float64(lawful_neutral_mean)

			} else if mi.chaoticness == 2 { // abs neutral
				audit_percentage = (rand.NormFloat64() * float64(abs_neutral_standard_deviation)) + float64(abs_neutral_mean)

			} else if mi.chaoticness == 3 { //chaotic neutral
				audit_percentage = (rand.NormFloat64() * float64(chaotic_neutral_standard_deviation)) + float64(chaotic_neutral_mean)

			}

		} else if mi.evilness == 1 { // if agent is good
			if mi.chaoticness == 1 { // if lawful good

				audit_percentage = (rand.NormFloat64() * float64(lawful_good_standard_deviation)) + float64(lawful_good_mean)

			} else if mi.chaoticness == 2 { // neutral good
				audit_percentage = (rand.NormFloat64() * float64(neutral_good_standard_deviation)) + float64(neutral_good_mean)

			} else if mi.chaoticness == 3 { //chaotic good
				audit_percentage = (rand.NormFloat64() * float64(chaotic_good_standard_deviation)) + float64(chaotic_good_mean)

			}

		} else if mi.evilness == 3 { // if agent is evil
			if mi.chaoticness == 1 { // if lawful evil

				audit_percentage = (rand.NormFloat64() * float64(lawful_evil_standard_deviation)) + float64(lawful_evil_mean)

			} else if mi.chaoticness == 2 { // neutral evil
				audit_percentage = (rand.NormFloat64() * float64(neutral_evil_standard_deviation)) + float64(neutral_evil_mean)

			} else if mi.chaoticness == 3 { //chaotic evil
				audit_percentage = (rand.NormFloat64() * float64(chaotic_evil_standard_deviation)) + float64(chaotic_evil_mean)

			}
		}
		// we apply affinity, it will shift the mean(just the percentage) based on the affinity of that agnet
		affinity_modifier := 0.03
		ignore_thresh := 10
		if mi.affinity[agent] > ignore_thresh {
			audit_percentage -= math.Sqrt(float64(mi.affinity[agent]-10)) * affinity_modifier
		} else if mi.affinity[agent] < -ignore_thresh {
			audit_percentage -= math.Sqrt(float64(mi.affinity[agent]+10)) * affinity_modifier
		}

		Apply_mood(audit_percentage, abs_neutral_mean)
		auditmap[agent] = audit_percentage

	}

	return auditmap

}

func (mi *MI_256_v1) GetWithdrawalAuditVote() common.Vote {
	// fmt.Println("audit starts")
	/*vote happens at many occasions:

	when approving to rank up
	when approving withdrawal
	voting for a leader(maybe)s
	voting for a AoA
	voting for audition
	*/
	// TODO: implement vote strategy
	votemap := mi.BuildVotemap(true)
	var max_agent uuid.UUID
	max_audit_perentage := 0.0
	for agent, audit_percentage := range votemap {
		if audit_percentage > float64(max_audit_perentage) {
			max_agent = agent
			max_audit_perentage = audit_percentage
		}
	}
	var vote common.Vote
	random := rand.Float64()
	if random <= max_audit_perentage {
		vote = common.CreateVote(1, mi.GetID(), max_agent)

	} else if random >= max_audit_perentage+(1-max_audit_perentage)/2 {
		vote = common.CreateVote(-1, mi.GetID(), max_agent)
	} else {
		// if in this range, abstain
		vote = common.CreateVote(0, mi.GetID(), max_agent)
	}
	fmt.Println(mi.GetID(), " audit vote", vote)
	return vote

}

func (mi *MI_256_v1) Team4_GetRankUpVote() map[uuid.UUID]int {
	// log.Printf("Called overriden GetRankUpVote()")
	votePercentmap := mi.BuildVotemap(false)
	votemap := make(map[uuid.UUID]int)
	for agent, percentage := range votePercentmap {
		random := rand.Float64()
		if random <= percentage {
			votemap[agent] = 1
		} else if random >= percentage+(1-percentage)/2 {
			votemap[agent] = -1
		} else {
			votemap[agent] = 0
			// if in this range, abstain

		}
	}
	return votemap
}

func (mi *MI_256_v1) Team4_GetConfession() bool {
	chance := rand.Intn(2)
	if chance != 0 {
		return true
	}
	return false
}

func (mi *MI_256_v1) Team4_GetProposedWithdrawalVote() map[uuid.UUID]int {
	// log.Printf("Called overriden GetProposedWithdrawalVote()")
	votePercentmap := mi.BuildVotemap(false)
	votemap := make(map[uuid.UUID]int)
	for agent, percentage := range votePercentmap {
		random := rand.Float64()
		if random <= percentage {
			votemap[agent] = 1
		} else if random >= percentage+(1-percentage)/2 {
			votemap[agent] = -1
		} else {
			votemap[agent] = 0
			// if in this range, abstain

		}
	}
	return votemap
}

// ----------------------- State Helpers -----------------------
// TODO: add helper functions for managing / using internal states

//get common pool resource

// mi.Server.GetTeam(mi.GetID()).GetCommonPool()

// //get ids of people in my team
// mi.Server.GetTeam(mi.GetID()).Agents

// //get the voting status (to implement)

// mi.Server.GetTeam(mi.GetID()).

// ---------------------------------------------------------------
func (mi *MI_256_v1) RandomizeCharacter() {
	mi.chaoticness = rand.Intn(3) + 1

	mi.evilness = rand.Intn(3) + 1
	mi.haveIlied = false
	mi.Initialize_opninions()
	mi.AoARanking = []int{4, 1, 2, 3, 4, 5}
	mi.SetAoARanking(mi.AoARanking)
}

// ----------- functions that update character opinions ---------------------------------

func (mi *MI_256_v1) Initialize_opninions() {
	mi.affinity = make(map[uuid.UUID]int)
	for _, agent := range mi.Server.UpdateAndGetAgentExposedInfo() {
		mi.affinity[agent.AgentUUID] = 0
		mi.affinityChange[agent.AgentUUID] = 0
	}
	// needs to change
	mi.mood = 0

}

func (mi *MI_256_v1) Team4_UpdateStateStartTurn() {
	// overwrite if your agent need to update internal state at this stage.

	// this function updates the agent states every start of turn, refreshing states if needed
	fmt.Println(mi.GetID(), " mood this turn", mi.mood)
	mi.UpdateMoodTurnStart()
	mi.haveIlied = false
	for key := range mi.affinityChange {
		mi.affinityChange[key] = 0
	}
	mi.isThereCheatContribution = false
	mi.isThereCheatWithdrawal = false
	if mi.Score < mi.lastTurnScore {
		mi.lastThreshold = mi.lastTurnScore - mi.Score
	}

}

/*
Mood decides wether the agent want to be more active
Mood needs to be updated at the following timepoints:

1. At the start of turn, where a random Urge is generated
2. Dice rolls, wether if you have gone bust or not
3. Comparing the expected Score of every agent after round end to you
4. If you get audited,

*/

func (mi *MI_256_v1) UpdateMoodTurnStart() {
	//generate random float from -1 to 1 and multiply by chaoticness

	randomUrge := (rand.Float64()*2 - 1) * float64(mi.chaoticness)
	randomUrge = math.Round(randomUrge)
	mi.mood += int(randomUrge)
}
func (mi *MI_256_v1) UpdateMoodAfterRoll() {

	bustMoodModifier := 1
	// if the roll has gone bust
	if mi.LastScore == 0 {
		mi.mood += bustMoodModifier * (mi.chaoticness - 2)

	}
}

func (mi *MI_256_v1) UpdateMoodAfterRoundEnd() {
	numAgent := 0
	contributedSum, withdrawalSum := 0, 0
	for _, agent := range mi.Server.GetTeam(mi.GetID()).Agents {
		contributedSum += mi.teamAgentsDeclaredContribution[agent]
		withdrawalSum += mi.teamAgentsDeclaredWithdraw[agent]
		numAgent += 1

	}
	averageNetgain := float64((withdrawalSum - contributedSum) / numAgent)
	personalNetgain := mi.IntendedWithdrawal - mi.intendedContribution

	//comparing the agent's performance to your own performance
	if float64(personalNetgain) < (averageNetgain) {
		teamPerformanceModifier := 3
		// change if you are not doing very well
		mi.mood += teamPerformanceModifier * (mi.chaoticness - 2)
	} else if float64(personalNetgain) > (averageNetgain) {
		teamPerformanceModifier := 1
		//else you feel confident about your current mood
		mi.mood += teamPerformanceModifier * (mi.chaoticness)

	}
	// if lying and not caught, be more adventurous, else no.
	lyingModifier := 1
	if mi.haveIlied && mi.IcaughtLying {
		// mood decreases
		mi.mood -= lyingModifier * mi.chaoticness
	} else if mi.haveIlied && !mi.IcaughtLying {
		mi.mood += lyingModifier * mi.chaoticness
	}
	// you calculate everyone's declared net gain,

}
func (mi *MI_256_v1) UpdateMoodAfterAuditionEnd() {
	// if you get audited and has been caught lying, then you probably want to be more greedy to survive
	if mi.IcaughtLying {
		mi.mood = 3 * mi.chaoticness
	}

}

/*
Affinity decides the likelyhood of our Agent will support other agent's decisions

Affinity should be updated when:
1. They declared their contributions and Withdrawals. The Amount
2. When they voted to audit you
3. When someone has failed an audit
4. When agents propose their Ideologies.
*/
func (mi *MI_256_v1) UpdateAffinityAoAProposal() {
	// need to see all of the AoA in detail
}

func (mi *MI_256_v1) UpdateAffinityAfterContribute() {
	//this function is called
	// we need to value if the declared contribution is fair, we do not know what everyone rolled, we only know their declared contribution against expected
	// apart from the fairness, there is also the satisfaction. Good people would tend to "understand" if they are not contributing as much, while evil people would want people to donate more
	// and of course, both sides would love people that donates more, more for good people, less for evil people because their is a cheat probability

	// contructing a socially fair structure:
	// simple, meeting the AoA Expected Contribution==fair

	// if we do not know their expected contributions however,  we could measure the average of all the declared contributions, and use that as agent expected.

	sum := 0
	numAgent := 0
	for _, agent := range mi.Server.GetTeam(mi.GetID()).Agents {
		sum += mi.teamAgentsDeclaredContribution[agent]
		numAgent += 1

	}
	agentExpected := sum / numAgent
	for _, agent := range mi.Server.GetTeam(mi.GetID()).Agents {

		if mi.isAoAContributionFixed {
			agentExpected = mi.teamAgentsDeclaredContribution[agent]
		}
		affinityChange := 0.0
		agentDeclared := mi.teamAgentsDeclaredContribution[agent]
		// fairness, if they contributed more or less than expected, less take note, more don't do anything
		fairnessThresh := 0.5
		if agentDeclared < agentExpected {
			affinityChange -= float64(agentExpected-agentDeclared) * (fairnessThresh)
		}

		if !mi.isThereCheatContribution {

			// satisfaction if there are no cheats, so we value people highly
			satisfactionThresh := 0.5
			if agentDeclared < agentExpected {
				amount_less := float64(agentExpected - agentDeclared)
				affinityChange -= (satisfactionThresh) * float64(mi.evilness-1) * math.Sqrt(amount_less)
			} else {
				amount_more := -float64(agentExpected - agentDeclared)
				affinityChange += (satisfactionThresh) * float64(3-(mi.evilness-1)) * math.Sqrt(amount_more)
			}
		} else { // if there are cheats happening, we would like to be more spectical
			// good people would still think they are good, but with less modifier, neutral people hold thought, and evil would decrease if they say contributed more (suspision)
			satisfactionThresh := 0.5
			if agentDeclared < agentExpected {
				//if you contribute less than expected, no change
				amount_less := float64(agentExpected - agentDeclared)
				affinityChange -= (satisfactionThresh) * float64(mi.evilness-1) * math.Sqrt(amount_less)
			} else {
				// if you donate more or equal to, we suspect you
				amount_more := -float64(agentExpected - agentDeclared)
				affinityChange += (satisfactionThresh) * float64(-(mi.evilness - 2)) * math.Sqrt(amount_more)
			}
		}

		mi.affinity[agent] += int(affinityChange)

		mi.affinityChange[agent] += int(affinityChange)

	}

}
func (mi *MI_256_v1) UpdateAffinityAfterWithdraw() {
	//similar to contribution, there withdrawing same amount is fair, and satisfaction comes into play
	// if there is no set distribution, we would assume the avarage amount in the pot would be a fair number
	common_pool := mi.Server.GetTeam(mi.GetID()).GetCommonPool()
	agentExpected := int(common_pool / (len(mi.Server.GetTeam(mi.GetID()).Agents) + 1))

	for _, agent := range mi.Server.GetTeam(mi.GetID()).Agents {
		if mi.isAoAWithdrawalFixed {
			agentExpected = mi.teamAgentsExpectedWithdraw[agent]
		}

		agentDeclared := mi.teamAgentsDeclaredWithdraw[agent]
		affinityChange := 0.0
		// fairness, if they withdrawed more or less than expected, less take note, more don't do anything
		fairnessThresh := 0.5
		if agentDeclared > agentExpected {
			affinityChange += float64(agentExpected-agentDeclared) * (fairnessThresh)
		}
		// satisfaction ( not too important as if they miss they will cheat anyways so you can't tell)

		if !mi.isThereCheatWithdrawal {

			satisfactionThresh := 0.5
			if agentDeclared < agentExpected {
				amount_less := float64(agentExpected - agentDeclared)
				affinityChange += (satisfactionThresh) * float64(mi.evilness-1) * math.Sqrt(amount_less)
			} else {
				amount_more := -float64(agentExpected - agentDeclared)
				affinityChange -= (satisfactionThresh) * float64(3-(mi.evilness-1)) * math.Sqrt(amount_more)
			}
		} else {
			// if there are cheat, then we would be more critical on whoever says they took equal or less
			satisfactionThresh := 0.5
			if agentDeclared < agentExpected {
				amount_less := float64(agentExpected - agentDeclared)
				affinityChange += (satisfactionThresh) * float64(-(mi.evilness - 2)) * math.Sqrt(amount_less)
			} else {
				amount_more := -float64(agentExpected - agentDeclared)
				affinityChange -= (satisfactionThresh) * float64(3-(mi.evilness-1)) * math.Sqrt(amount_more)
			}

		}
		mi.affinity[agent] += int(affinityChange)
		mi.affinityChange[agent] += int(affinityChange)

	}

}
func (mi *MI_256_v1) UpdateAffinityAfterVote() {
	// whenever a vote happens, if the vote is targeting you, someone you really dislike, someone you really like, you would change affinity based on people's votes.

	if mi.lastAuditTarget == mi.GetID() { // if the target agent is yourself:
		//then you would really not like the guy that targeted you, and not like whoever voted yes to vote you out
		for _, agent := range mi.Server.GetTeam(mi.GetID()).Agents {
			affinityChange := 0
			if mi.lastAuditStarter == agent {
				affinityChange -= 5
			}
			voteAgainstAffinityChange := 3
			voteForAffinityChange := 1
			if mi.lastVotes[agent] {
				//affninity changes based on the chaoticness, the more chaotic, the more you cannot stand the person
				affinityChange -= voteAgainstAffinityChange * mi.chaoticness
			} else {
				affinityChange += voteForAffinityChange * mi.chaoticness
			}
			mi.affinity[agent] += affinityChange
		}

	} else if mi.affinity[mi.lastAuditTarget] <= -10 { // if the vote is against someone you really dislike, you gain little change for people agreeing to it, but dislike people who disagree with it
		for _, agent := range mi.Server.GetTeam(mi.GetID()).Agents {
			affinityChange := 0
			if mi.lastAuditStarter == agent {

				affinityChange += 2
			}

			voteAgainstAffinityChange := 2
			voteForAffinityChange := 1
			if mi.lastVotes[agent] {
				//affninity changes based on the chaoticness, the more chaotic, the more you cannot stand the person
				affinityChange += voteForAffinityChange * mi.chaoticness
			} else {
				affinityChange -= voteAgainstAffinityChange * mi.chaoticness
			}
			mi.affinity[agent] += affinityChange
		}
	} else if mi.affinity[mi.lastAuditTarget] <= -10 { // if the vote is against someone you really like, you gain little dislike for people agreeing to it, but like people who disagree with it
		for _, agent := range mi.Server.GetTeam(mi.GetID()).Agents {
			affinityChange := 0
			if mi.lastAuditStarter == agent {
				if mi.isThereCheatWithdrawal || mi.isThereCheatContribution {
					affinityChange -= 2
				} else {
					affinityChange -= 3 // you would get more mad if there seems to be no cheating this round
				}

			}
			voteAgainstAffinityChange := 0
			voteForAffinityChange := 0
			if mi.isThereCheatWithdrawal || mi.isThereCheatContribution {
				voteAgainstAffinityChange = 1
				voteForAffinityChange = 2
			} else {
				voteAgainstAffinityChange = 1
				voteForAffinityChange = 3
			}

			if mi.lastVotes[agent] {
				//affninity changes based on the chaoticness, the more chaotic, the more you cannot stand the person
				affinityChange -= voteForAffinityChange * mi.chaoticness
			} else {
				affinityChange += voteAgainstAffinityChange * mi.chaoticness
			}
			mi.affinity[agent] += affinityChange
		}
	}

}

func (mi *MI_256_v1) UpdateAffinityAfterAudit() {
	//if someone fails or succeeds an audit, then you would gain impression of them
	affinityChange := 0
	if mi.lastAuditResult {
		// if the audit is successful, then you would like the person less
		affinityChange = -1
	} else {
		// if the audit is unsuccessful, then you would like the person more
		affinityChange = 1
	}
	mi.affinity[mi.lastAuditTarget] += 2 * affinityChange * mi.chaoticness
	mi.affinity[mi.lastAuditStarter] -= affinityChange * mi.chaoticness
}

// ----------------------- Helper Functions -----------------------
func GetAgentTeamAoA(mi *MI_256_v1) common.IArticlesOfAssociation {
	return mi.Server.GetTeam(mi.GetID()).TeamAoA
}

func (mi *MI_256_v1) Team4_ProposeWithdrawal() int {
	// first check if the agent has a team
	if !mi.HasTeam() {
		return 0
	}
	mi.DecideWithdrawal(mi.Server.GetTeam(mi.GetID()).GetCommonPool())
	return mi.IntendedWithdrawal
}
func (mi *MI_256_v1) Team4_GetPunishmentVoteMap() map[int]int {
	punishmentVoteMap := make(map[int]int)

	for punishment := 0; punishment <= 4; punishment++ {
		punishmentVoteMap[punishment] = min(4, rand.Intn(5)+max(mi.evilness-2, 0))
	}

	return punishmentVoteMap
}
