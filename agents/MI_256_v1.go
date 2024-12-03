package agents

import (
	"log"
	"math/rand"

	common "github.com/ADimoska/SOMASExtended/common"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
)

type MI_256_v1 struct {
	*ExtendedAgent
}

// constructor for MI_256_v1
func Team4_CreateAgent(funcs agent.IExposedServerFunctions[common.IExtendedAgent], agentConfig AgentConfig) *MI_256_v1 {
	mi_256 := &MI_256_v1{
		ExtendedAgent: GetBaseAgents(funcs, agentConfig),
	}
	mi_256.trueSomasTeamID = 4 // IMPORTANT: add your team number here!
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

// ----------------------- State Helpers -----------------------
// TODO: add helper functions for managing / using internal states
