package agents

import (
	"fmt"
	"log"
	"math/rand"

	common "github.com/ADimoska/SOMASExtended/common"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
)

type MI_256_v1 struct {
	*ExtendedAgent
}

// Constructor for MI_256_v1
func Team4_CreateAgent(funcs agent.IExposedServerFunctions[common.IExtendedAgent], agentConfig AgentConfig) *MI_256_v1 {
	mi_256 := &MI_256_v1{
		ExtendedAgent: GetBaseAgents(funcs, agentConfig),
	}
	mi_256.TrueSomasTeamID = 4 // IMPORTANT: add your team number here!
	return mi_256
}

// ----------------------- Strategies -----------------------
// Team-forming Strategy
func (mi *MI_256_v1) DecideTeamForming(agentInfoList []common.ExposedAgentInfo) []uuid.UUID {
	log.Printf("Called overriden DecideTeamForming\n")
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
func (mi *MI_256_v1) StickOrAgain(accumulatedScore int, prevRoll int) bool {
	log.Printf("Called overriden StickOrAgain\n")
	// TODO: implement dice strategy
	return true
}

// !!! NOTE: name and signature of functions below are subject to change by the infra team !!!

// Contribution Strategy
func (mi *MI_256_v1) DecideContribution() int {
	// TODO: implement contribution strategy
	return 1
}

// Withdrawal Strategy
func (mi *MI_256_v1) DecideWithdrawal() int {
	// TODO: implement contribution strategy
	return 1
}

// Audit Strategy
func (mi *MI_256_v1) DecideAudit() bool {
	// TODO: implement audit strategy
	return true
}

// Punishment Strategy
func (mi *MI_256_v1) DecidePunishment() int {
	// TODO: implement punishment strategy
	return 1
}

// Need to do something about agents own ID in here
func (mi *MI_256_v1) Team4_GetRankUpVote() map[uuid.UUID]int {
	log.Printf("Called overriden GetRankUpVote()")
	agentsInTeam := mi.Server.GetAgentsInTeam(mi.TeamID)
	rankUpVote := make(map[uuid.UUID]int)

	for _, agentId := range agentsInTeam {
		rankUpVote[agentId] = rand.Intn(2)
	}

	fmt.Println(rankUpVote)
	return rankUpVote
}

func (mi *MI_256_v1) Team4_GetConfession() bool {
	return true
}

func (mi *MI_256_v1) Team4_GetProposedWithdrawalVote() map[uuid.UUID]int {
	log.Printf("Called overriden GetProposedWithdrawalVote()")
	agentsInTeam := mi.Server.GetAgentsInTeam(mi.TeamID)
	proposedWithdrawals := make(map[uuid.UUID]int)

	for _, agentId := range agentsInTeam {
		proposedWithdrawals[agentId] = rand.Intn(2)
	}

	fmt.Println(proposedWithdrawals)
	return proposedWithdrawals
}

func (mi *MI_256_v1) GetWithdrawalAuditVote() common.Vote {
	log.Printf("Called overriden GetWithdrawalAuditVote()")

	// Get the agents in the team
	agentsInTeam := mi.Server.GetAgentsInTeam(mi.TeamID)

	// Check if the team has any agents
	if len(agentsInTeam) == 0 {
		return common.CreateVote(0, mi.GetID(), uuid.Nil)
	}

	firstAgentID := agentsInTeam[0]

	return common.CreateVote(1, mi.GetID(), firstAgentID)
}

func (mi *MI_256_v1) Team4_GetPunishmentVoteMap() map[int]int {
	punishmentVoteMap := make(map[int]int)

	for punishment := 0; punishment <= 4; punishment++ {
		punishmentVoteMap[punishment] = rand.Intn(5)
	}

	return punishmentVoteMap
}

// ----------------------- State Helpers -----------------------
// TODO: add helper functions for managing / using internal states
