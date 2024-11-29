package agents

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	common "SOMAS_Extended/common"

	// TODO:

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/message"
)

type ExtendedAgent struct {
	*agent.BaseAgent[common.IExtendedAgent]
	server common.IServer
	score  int
	teamID uuid.UUID

	// private
	lastScore int

	// debug
	verboseLevel int

	// AoA vote
	AoARanking []int

	LastTeamID uuid.UUID // Tracks the last team the agent was part of
}

type AgentConfig struct {
	InitScore    int
	VerboseLevel int
}

func GetBaseAgents(funcs agent.IExposedServerFunctions[common.IExtendedAgent], configParam AgentConfig) *ExtendedAgent {
	return &ExtendedAgent{
		BaseAgent:    agent.CreateBaseAgent(funcs),
		server:       funcs.(common.IServer), // Type assert the server functions to IServer interface
		score:        configParam.InitScore,
		verboseLevel: configParam.VerboseLevel,
		AoARanking:   []int{3, 2, 1, 0},
	}
}

// ----------------------- Interface implementation -----------------------

// Get the agent's current team ID
func (mi *ExtendedAgent) GetTeamID() uuid.UUID {
	return mi.teamID
}

// Get the agent's last team ID
func (mi *ExtendedAgent) GetLastTeamID() uuid.UUID {
	return mi.LastTeamID
}

// Get the agent's current score
// Can only be called by the server (otherwise other agents will see their true score)
func (mi *ExtendedAgent) GetTrueScore() int {
	return mi.score
}

// Setter for the server to call, in order to set the true score for this agent
func (mi *ExtendedAgent) SetTrueScore(score int) {
	mi.score = score
}

// custom function: ask for rolling the dice
func (mi *ExtendedAgent) StartRollingDice(instance common.IExtendedAgent) {
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
	mi.score += turnScore

	if mi.verboseLevel > 4 {
		fmt.Printf("%s's turn score: %v, total score: %v\n", mi.GetID(), turnScore, mi.score)
	}
}

// stick or again
func (mi *ExtendedAgent) StickOrAgain() bool {
	// if mi.verboseLevel > 8 {
	// 	fmt.Printf("%s is deciding to stick or again\n", mi.GetID())
	// }
	decision := Debug_StickOrAgainJudgement()
	return decision
}

// decide to stick
func (mi *ExtendedAgent) DecideStick() {
	if mi.verboseLevel > 6 {
		fmt.Printf("%s decides to [STICK], last score: %v\n", mi.GetID(), mi.lastScore)
	}
}

// decide to roll again
func (mi *ExtendedAgent) DecideRollAgain() {
	if mi.verboseLevel > 6 {
		fmt.Printf("%s decides to ROLL AGAIN, last score: %v\n", mi.GetID(), mi.lastScore)
	}
}

// TODO: TO BE IMPLEMENTED BY TEAM'S AGENT
// get the agent's actual contribution to the common pool
// This function MUST return the same value when called multiple times in the same turn
func (mi *ExtendedAgent) GetActualContribution(instance common.IExtendedAgent) int {
	if mi.HasTeam() {
		contribution := instance.DecideContribution()
		if mi.verboseLevel > 6 {
			fmt.Printf("%s is contributing %d to the common pool and thinks the common pool size is %d\n", mi.GetID(), contribution, mi.server.GetTeam(mi.GetID()).GetCommonPool())
		}
		return contribution
	} else {
		if mi.verboseLevel > 6 {
			fmt.Printf("%s has no team, skipping contribution\n", mi.GetID())
		}
		return 0
	}
}

func (mi *ExtendedAgent) DecideContribution() int {
	// MVP: contribute exactly as defined in AoA
	if mi.server.GetTeam(mi.GetID()).TeamAoA != nil {
		aoaExpectedContribution := mi.server.GetTeam(mi.GetID()).TeamAoA.GetExpectedContribution(mi.GetID(), mi.GetTrueScore())
		// double check if score in agent is sufficient (this should be handled by AoA though)
		if mi.GetTrueScore() < aoaExpectedContribution {
			return mi.GetTrueScore() // give all score if less than expected
		}
		return aoaExpectedContribution
	} else {
		if mi.verboseLevel > 6 {
			// should not happen!
			fmt.Printf("[WARNING] Agent %s has no AoA, contributing 0\n", mi.GetID())
		}
		return 0
	}
}

// get the agent's stated contribution to the common pool
// TODO: the value returned by this should be broadcasted to the team via a message
// This function MUST return the same value when called multiple times in the same turn
func (mi *ExtendedAgent) GetStatedContribution(instance common.IExtendedAgent) int {
	// Hardcoded stated
	statedContribution := instance.GetActualContribution(instance)
	return statedContribution
}

// make withdrawal from common pool
func (mi *ExtendedAgent) GetActualWithdrawal(instance common.IExtendedAgent) int {
	currentPool := mi.server.GetTeam(mi.GetID()).GetCommonPool()
	withdrawal := instance.DecideWithdrawal()
	fmt.Printf("%s is withdrawing %d from the common pool of size %d\n", mi.GetID(), withdrawal, currentPool)
	return withdrawal
}

// The value returned by this should be broadcasted to the team via a message
// This function MUST return the same value when called multiple times in the same turn
func (mi *ExtendedAgent) GetStatedWithdrawal(instance common.IExtendedAgent) int {
	// Currently, assume stated withdrawal matches actual withdrawal
	return instance.DecideWithdrawal()
}

// Decide the withdrawal amount based on AoA and current pool size
func (mi *ExtendedAgent) DecideWithdrawal() int {
	if mi.server.GetTeam(mi.GetID()).TeamAoA != nil {
		// double check if score in agent is sufficient (this should be handled by AoA though)
		commonPool := mi.server.GetTeam(mi.GetID()).GetCommonPool()
		aoaExpectedWithdrawal := mi.server.GetTeam(mi.GetID()).TeamAoA.GetExpectedWithdrawal(mi.GetID(), mi.GetTrueScore(), commonPool)
		if commonPool < aoaExpectedWithdrawal {
			return commonPool
		}
		return aoaExpectedWithdrawal
	} else {
		if mi.verboseLevel > 6 {
			fmt.Printf("[WARNING] Agent %s has no AoA, withdrawing 0\n", mi.GetID())
		}
		return 0
	}
}

/*
Provide agentId for memory, current accumulated score
(to see if above or below predicted threshold for common pool contribution)
And previous roll in case relevant
*/
func (mi *ExtendedAgent) StickOrAgainFor(agentId uuid.UUID, accumulatedScore int, prevRoll int) int {
	// random chance, to simulate what is already implemented
	return rand.Intn(2)
}

// dev function
func (mi *ExtendedAgent) LogSelfInfo() {
	fmt.Printf("[Agent %s] score: %v\n", mi.GetID(), mi.score)
}

// Agent returns their preference for an audit on contribution
// 0: No preference
// 1: Prefer audit
// -1: Prefer no audit
func (mi *ExtendedAgent) GetContributionAuditVote() common.Vote {
	return common.CreateVote(0, mi.GetID(), uuid.Nil)
}

// Agent returns their preference for an audit on withdrawal
// 0: No preference
// 1: Prefer audit
// -1: Prefer no audit
func (mi *ExtendedAgent) GetWithdrawalAuditVote() common.Vote {
	return common.CreateVote(0, mi.GetID(), uuid.Nil)
}

func (mi *ExtendedAgent) SetAgentContributionAuditResult(agentID uuid.UUID, result bool) {}

func (mi *ExtendedAgent) SetAgentWithdrawalAuditResult(agentID uuid.UUID, result bool) {}

// ----Withdrawal------- Messaging functions -----------------------

func (mi *ExtendedAgent) HandleTeamFormationMessage(msg *common.TeamFormationMessage) {
	fmt.Printf("Agent %s received team forming invitation from %s\n", mi.GetID(), msg.GetSender())

	// Already in a team - reject invitation
	if mi.teamID != (uuid.UUID{}) {
		if mi.verboseLevel > 6 {
			fmt.Printf("Agent %s rejected invitation from %s - already in team %v\n",
				mi.GetID(), msg.GetSender(), mi.teamID)
		}
		return
	}

	// Handle team creation/joining based on sender's team status
	sender := msg.GetSender()
	if mi.server.CheckAgentAlreadyInTeam(sender) {
		existingTeamID := mi.server.AccessAgentByID(sender).GetTeamID()
		mi.joinExistingTeam(existingTeamID)
	} else {
		mi.createNewTeam(sender)
	}
}

func (mi *ExtendedAgent) HandleContributionMessage(msg *common.ContributionMessage) {
	if mi.verboseLevel > 8 {
		fmt.Printf("Agent %s received contribution notification from %s: amount=%d\n",
			mi.GetID(), msg.GetSender(), msg.StatedAmount)
	}

	// Team's agent should implement logic to store or process the reported contribution amount as desired
}

func (mi *ExtendedAgent) HandleScoreReportMessage(msg *common.ScoreReportMessage) {
	if mi.verboseLevel > 8 {
		fmt.Printf("Agent %s received score report from %s: score=%d\n",
			mi.GetID(), msg.GetSender(), msg.TurnScore)
	}

	// Team's agent should implement logic to store or process score of other agents as desired
}

func (mi *ExtendedAgent) HandleWithdrawalMessage(msg *common.WithdrawalMessage) {
	if mi.verboseLevel > 8 {
		fmt.Printf("Agent %s received withdrawal notification from %s: amount=%d\n",
			mi.GetID(), msg.GetSender(), msg.StatedAmount)
	}

	// Team's agent should implement logic to store or process the reported withdrawal amount as desired
}

func (mi *ExtendedAgent) BroadcastSyncMessageToTeam(msg message.IMessage[common.IExtendedAgent]) {
	// Send message to all team members synchronously
	agentsInTeam := mi.server.GetAgentsInTeam(mi.teamID)
	for _, agentID := range agentsInTeam {
		if agentID != mi.GetID() {
			mi.SendSynchronousMessage(msg, agentID)
		}
	}
}

// ----------------------- Info functions -----------------------
func (mi *ExtendedAgent) GetExposedInfo() common.ExposedAgentInfo {
	return common.ExposedAgentInfo{
		AgentUUID:   mi.GetID(),
		AgentTeamID: mi.teamID,
	}
}

func (mi *ExtendedAgent) CreateScoreReportMessage() *common.ScoreReportMessage {
	return &common.ScoreReportMessage{
		BaseMessage: mi.CreateBaseMessage(),
		TurnScore:   mi.lastScore,
	}
}

func (mi *ExtendedAgent) CreateContributionMessage(statedAmount int, expectedAmount int) *common.ContributionMessage {
	return &common.ContributionMessage{
		BaseMessage:    mi.CreateBaseMessage(),
		StatedAmount:   statedAmount,
		ExpectedAmount: expectedAmount,
	}
}

func (mi *ExtendedAgent) CreateWithdrawalMessage(statedAmount int, expectedAmount int) *common.WithdrawalMessage {
	return &common.WithdrawalMessage{
		BaseMessage:    mi.CreateBaseMessage(),
		StatedAmount:   statedAmount,
		ExpectedAmount: expectedAmount,
	}
}

// ----------------------- Debug functions -----------------------

func Debug_RollDice() int {
	// row 3d6
	total := 0
	for i := 0; i < 3; i++ {
		total += rand.Intn(6) + 1
	}
	return total
}

func Debug_StickOrAgainJudgement() bool {
	// 50% chance to stick
	return rand.Intn(2) == 0
}

// ----------------------- Team forming functions -----------------------
func (mi *ExtendedAgent) StartTeamForming(instance common.IExtendedAgent, agentInfoList []common.ExposedAgentInfo) {
	// TODO: implement team forming logic
	if mi.verboseLevel > 6 {
		fmt.Printf("%s is starting team formation\n", mi.GetID())
	}

	chosenAgents := instance.DecideTeamForming(agentInfoList)
	mi.SendTeamFormingInvitation(chosenAgents)
	mi.SignalMessagingComplete()
}

func (mi *ExtendedAgent) DecideTeamForming(agentInfoList []common.ExposedAgentInfo) []uuid.UUID {
	invitationList := []uuid.UUID{}
	for _, agentInfo := range agentInfoList {
		// exclude the agent itself
		if agentInfo.AgentUUID == mi.GetID() {
			continue
		}
		if agentInfo.AgentTeamID == (uuid.UUID{}) {
			invitationList = append(invitationList, agentInfo.AgentUUID)
		}
	}

	// random choice from the invitation list
	rand.Shuffle(len(invitationList), func(i, j int) { invitationList[i], invitationList[j] = invitationList[j], invitationList[i] })
	if len(invitationList) == 0 {
		return []uuid.UUID{}
	}
	chosenAgent := invitationList[0]

	// Return a slice containing the chosen agent
	return []uuid.UUID{chosenAgent}
}

func (mi *ExtendedAgent) SendTeamFormingInvitation(agentIDs []uuid.UUID) {
	for _, agentID := range agentIDs {
		invitationMsg := &common.TeamFormationMessage{
			BaseMessage: mi.CreateBaseMessage(),
			AgentInfo:   mi.GetExposedInfo(),
			Message:     "Would you like to form a team?",
		}
		// Debug print to check message contents
		fmt.Printf("Sending invitation: sender=%v, teamID=%v, receiver=%v\n", mi.GetID(), mi.teamID, agentID)
		mi.SendMessage(invitationMsg, agentID)
	}
}

func (mi *ExtendedAgent) createNewTeam(senderID uuid.UUID) {
	fmt.Printf("Agent %s is creating a new team\n", mi.GetID())
	teamIDs := []uuid.UUID{mi.GetID(), senderID}
	newTeamID := mi.server.CreateAndInitTeamWithAgents(teamIDs)

	if newTeamID == (uuid.UUID{}) {
		if mi.verboseLevel > 6 {
			fmt.Printf("Agent %s failed to create a new team\n", mi.GetID())
		}
		return
	}

	mi.teamID = newTeamID
	if mi.verboseLevel > 6 {
		fmt.Printf("Agent %s created a new team with ID %v\n", mi.GetID(), newTeamID)
	}
}

func (mi *ExtendedAgent) joinExistingTeam(teamID uuid.UUID) {
	mi.teamID = teamID
	mi.server.AddAgentToTeam(mi.GetID(), teamID)
	if mi.verboseLevel > 6 {
		fmt.Printf("Agent %s joined team %v\n", mi.GetID(), teamID)
	}
}

// SetTeamID assigns a new team ID to the agent
// Parameters:
//   - teamID: The UUID of the team to assign to this agent
func (mi *ExtendedAgent) SetTeamID(teamID uuid.UUID) {
	// Store the previous team ID
	mi.LastTeamID = mi.teamID
	mi.teamID = teamID
}

func (mi *ExtendedAgent) HasTeam() bool {
	return mi.teamID != (uuid.UUID{})
}

// In RunStartOfIteration, the server loops through each agent in each team
// and sets the teams AoA by majority vote from the agents in that team.
func (mi *ExtendedAgent) SetAoARanking(Preferences []int) {
	mi.AoARanking = Preferences
}

func (mi *ExtendedAgent) GetAoARanking() []int {
	return mi.AoARanking
}

func (mi *ExtendedAgent) DecideLeaveTeam() bool {
	// Generate a random number between 0 and 9
	decision := rand.Intn(10)

	// If the random number is 0, the agent decides to leave (10% chance)
	if decision == 0 {
		if mi.verboseLevel > 6 {
			fmt.Printf("Agent %s decided to leave the team (10%% probability triggered)\n", mi.GetID())
		}
		return true
	}

	// Otherwise, the agent stays in the team
	if mi.verboseLevel > 6 {
		fmt.Printf("Agent %s decided to stay in the team\n", mi.GetID())
	}
	return false
}
