package agents

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	"SOMAS_Extended/common"
	"SOMAS_Extended/messages"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
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
	AoARanking [6]int
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

// dev function
func (mi *ExtendedAgent) LogSelfInfo() {
	fmt.Printf("[Agent %s] score: %v\n", mi.GetID(), mi.score)
}

// ----------------------- Messaging functions -----------------------
func (mi *ExtendedAgent) CreateExtendedMessage() messages.ExtendedMessage {
	return messages.ExtendedMessage{
		BaseMessage: mi.CreateBaseMessage(),
		TeamID:      mi.teamID,
	}
}

// send a message of base type IExtendedMessage
func (mi *ExtendedAgent) SendPrivateMessage(receiver uuid.UUID, msg common.IExtendedMessage) {
	mi.SendMessage(msg, receiver)
}

// send a message to team (if teamID is not 0)
func (mi *ExtendedAgent) SendTeamMessage(msg common.IExtendedMessage) {
	if mi.teamID != (uuid.UUID{}) {
		mi.BroadcastMessage(msg) // todo in the team file
	} else if mi.verboseLevel > 6 {
		fmt.Printf("Agent %s is trying to send a team message, but has no team\n", mi.GetID())
	}
}

// broadcast a message of base type IExtendedMessage
func (mi *ExtendedAgent) SendMessageBroadcast(msg common.IExtendedMessage) {
	mi.BroadcastMessage(msg)
}

// A function that can receive any type of message
func (mi *ExtendedAgent) ReceiveMessage(msg any) {
	switch msg := msg.(type) {
	case *messages.TeamFormingInvitationMessage:
		fmt.Printf("Agent %s received team forming invitation from %s\n", mi.GetID(), msg.GetSender())
		debug_checkID := mi.teamID == (uuid.UUID{})
		fmt.Printf("debug_checkID: %v\n", debug_checkID)
		// Case 1: Neither agent has a team - create new team
		if mi.teamID == (uuid.UUID{}) && msg.GetTeamID() == (uuid.UUID{}) {
			fmt.Printf("Agent %s is creating a new team\n", mi.GetID())
			teamIDs := []uuid.UUID{mi.GetID(), msg.GetSender()}
			newTeamID := mi.server.CreateAndInitTeamWithAgents(teamIDs)
			fmt.Printf("newTeamID: %v\n", newTeamID)
			if newTeamID == (uuid.UUID{}) {
				if mi.verboseLevel > 6 {
					fmt.Printf("Agent %s failed to create a new team\n", mi.GetID())
				}
			} else {
				mi.teamID = newTeamID
				if mi.verboseLevel > 6 {
					fmt.Printf("Agent %s created a new team with ID %v\n", mi.GetID(), newTeamID)
				}
			}

			// Case 2: Sender has a team, receiver doesn't - join sender's team
		} else if mi.teamID == (uuid.UUID{}) && msg.GetTeamID() != (uuid.UUID{}) {
			mi.teamID = msg.GetTeamID()
			mi.server.AddAgentToTeam(mi.GetID(), msg.GetTeamID())
			if mi.verboseLevel > 6 {
				fmt.Printf("Agent %s joined team %v\n", mi.GetID(), msg.GetTeamID())
			}

			// Case 3: Already in a team - reject invitation
		} else if mi.teamID != (uuid.UUID{}) {
			if mi.verboseLevel > 6 {
				fmt.Printf("Agent %s rejected invitation from %s - already in team %v\n",
					mi.GetID(), msg.GetSender(), mi.teamID)
			}
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
		invitationMsg := &messages.TeamFormingInvitationMessage{
			ExtendedMessage: mi.CreateExtendedMessage(),
			AgentInfo:       mi.GetExposedInfo(),
			Message:         "Would you like to form a team?",
		}
		// Debug print to check message contents
		fmt.Printf("Sending invitation: sender=%v, teamID=%v, receiver=%v\n",
			mi.GetID(), invitationMsg.GetTeamID(), agentID)
		mi.SendPrivateMessage(agentID, invitationMsg)
	}
}

// SetTeamID assigns a new team ID to the agent
// Parameters:
//   - teamID: The UUID of the team to assign to this agent
func (mi *ExtendedAgent) SetTeamID(teamID uuid.UUID) {
	mi.teamID = teamID
}

// In RunStartOfIteration, the server loops through each agent in each team
// and sets the teams AoA by majority vote from the agents in that team.
func (mi *ExtendedAgent) SetAgentAoA() {
	mi.AoARanking = [0,0,0,0,0,0]
}
