package agents

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	"SOMAS_Extended/common"

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

// Getter
func (mi *ExtendedAgent) GetTeamID() uuid.UUID {
	return mi.teamID
}

// Can only be called by the server (otherwise other agents will see their true score)
func (mi *ExtendedAgent) GetTrueScore() int {
	return mi.score
}

// Setter for the server to call, in order to set the true score for this agent
func (mi *ExtendedAgent) SetTrueScore(score int) {
	mi.score = score
}

// custom function: ask for rolling the dice
func (mi *ExtendedAgent) StartRollingDice() {
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
			willStick = mi.StickOrAgain()
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

// make a contribution to the common pool
func (mi *ExtendedAgent) ContributeToCommonPool() int {
	if mi.HasTeam() {
		contribution := mi.DecideSelfContribution()
		if mi.verboseLevel > 6 {
			fmt.Printf("%s is contributing %d to the common pool\n", mi.GetID(), contribution)
		}
		return contribution
	} else {
		if mi.verboseLevel > 6 {
			fmt.Printf("%s has no team, skipping contribution\n", mi.GetID())
		}
		return 0
	}
}

func (mi *ExtendedAgent) DecideSelfContribution() int {
	// MVP: contribute exactly as defined in AoA
	if mi.server.GetTeam(mi.GetID()).TeamAoA != nil {
		aoaExpectedContribution := mi.server.GetTeam(mi.GetID()).TeamAoA.GetContributionRule().GetExpectedContributionAmount(mi.GetTrueScore())
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

// make withdrawal from common pool
func (mi *ExtendedAgent) WithdrawFromCommonPool() int {
	fmt.Printf("%s is withdrawing from the common pool and thinks the common pool size is %d\n", mi.GetID(), mi.server.GetTeam(mi.GetID()).GetCommonPool())
	return 0
}

// dev function
func (mi *ExtendedAgent) LogSelfInfo() {
	fmt.Printf("[Agent %s] score: %v\n", mi.GetID(), mi.score)
}

// ----------------------- Messaging functions -----------------------

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
func (mi *ExtendedAgent) StartTeamForming(agentInfoList []common.ExposedAgentInfo) {
	// TODO: implement team forming logic
	if mi.verboseLevel > 6 {
		fmt.Printf("%s is starting team formation\n", mi.GetID())
	}

	chosenAgents := mi.DecideTeamForming(agentInfoList)
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
