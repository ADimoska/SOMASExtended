package agents

import (
	"SOMAS_Extended/common"
	"fmt"
	"math/rand"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
)

type Leadership struct {
	*ExtendedAgent
}

// constructor for Leadership
func Team2_CreateAgent(funcs agent.IExposedServerFunctions[common.IExtendedAgent], agentConfig AgentConfig) *Leadership {
	return &Leadership{
		ExtendedAgent: GetBaseAgents(funcs, agentConfig),
	}
}

// ----------------------- Strategies -----------------------
// Team-forming Strategy
func (mi *Leadership) DecideTeamForming(agentInfoList []common.ExposedAgentInfo) []uuid.UUID {
	fmt.Printf("Called overriden DecideTeamForming\n")
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

	// TODO: implement team forming logic
	// random choice from the invitation list
	rand.Shuffle(len(invitationList), func(i, j int) { invitationList[i], invitationList[j] = invitationList[j], invitationList[i] })
	chosenAgent := invitationList[0]

	// Return a slice containing the chosen agent
	return []uuid.UUID{chosenAgent}
}

// Dice Strategy
func (mi *Leadership) StickOrAgain() bool {
	fmt.Printf("Called overriden StickOrAgain\n")
	// TODO: implement dice strategy
	return true
}

// !!! NOTE: name and signature of functions below are subject to change by the infra team !!!

// Contribution Strategy
func (mi *Leadership) DecideContribution() int {
	// TODO: implement contribution strategy
	return 1
}

// Withdrawal Strategy
func (mi *Leadership) DecideWithdrawal() int {
	// TODO: implement contribution strategy
	return 1
}

// Audit Strategy
func (mi *Leadership) DecideAudit() bool {
	// TODO: implement audit strategy
	return true
}

// Punishment Strategy
func (mi *Leadership) DecidePunishment() int {
	// TODO: implement punishment strategy
	return 1
}

// /////////// ----------------------RANKING SYSTEM---------------------- /////////////
func (mi *Leadership) AssignRole(agentID string, role int) int {
	// get agent profiles and update with assigned roles
	//agent := mi.GetBaseAgents()
	//TODO set the role of the agent - idk how to do this yet - some sort of agent profile where there is a role field
	return 0 // TODO Return updated agent profiles
}

func (mi *Leadership) GetBaseAgents() *ExtendedAgent {
	// TODO Get agents from decided teams formed function
	return mi.ExtendedAgent
}

// Decide who is leader and who isnt, do this with an integar flage. Key = {1: leader, 2: general, 3: citizen, 4: police}
func (mi *Leadership) AllocateRank() {
	//TODO get turn, iteration and agent ids from server - call the server functions
	// Get the current turn and iteration
	currentTurn := mi.GetBaseAgents().GetCurrentTurn()
	currentIteration := mi.GetBaseAgents().GetCurrentIteration()
	// Get the list of all agent UUIDs
	agentIDs := mi.GetBaseAgents().GetAllAgentIDs()
	// Check if it is the first turn of the game
	if currentIteration == 0 && currentTurn == 0 {
		// Randomly select a leader from the list of agents
		leaderIndex := rand.Intn(len(agentIDs))
		// Aassign id as leader
		leaderID := agentIDs[leaderIndex]
		// Assign the leader role
		mi.AssignRole(leaderID, 1) // Role 1 represents the leader
		// Assign the citizen role to all other agents
		for _, agentID := range agentIDs {
			if agentID != leaderID {
				mi.AssignRole(agentID, 3) // Role 3 represents citizens
			}
		}
	}
	// else{
	// 	// Agents vote leader in based on trust scores
	// }
}

// ----------------------- State Helpers -----------------------
// TODO: add helper functions for managing / using internal states
