package agents

/* Contains functions relevant to agent functionality when the AoA is Team4AoA */

import (
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"

	common "github.com/ADimoska/SOMASExtended/common"
)

// ----------------------- Team 4 AoA Functions -----------------------

func (mi *ExtendedAgent) Team4_GetRankUpVote() map[uuid.UUID]int {
	return make(map[uuid.UUID]int)
}

func (mi *ExtendedAgent) Team4_GetConfession() bool {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return rng.Intn(2) == 1
}

func (mi *ExtendedAgent) Team4_StateConfessionToTeam() {
	// Broadcast contribution to team
	confession := mi.Team4_GetConfession()
	confessionMsg := mi.Team4_CreateConfessionMessage(confession)
	mi.BroadcastSyncMessageToTeam(confessionMsg)
}

func (mi *ExtendedAgent) Team4_HandleConfessionMessage(msg *common.Team4_ConfessionMessage) {
	if mi.VerboseLevel > 8 {
		if msg.Confession {
			log.Printf("Agent %s received confession notification from %s: I'm really sorry :(",
				mi.GetID(), msg.GetSender())
		} else {
			log.Printf("Agent %s received confession notification from %s: Noo! I'm innocent I swear!",
				mi.GetID(), msg.GetSender())
		}
	}
	// Team's agent should implement logic to store or process the reported proposed withdrawal amount as desired
}

// Get agents vote map for
func (mi *ExtendedAgent) Team4_GetProposedWithdrawalVote() map[uuid.UUID]int {
	log.Printf("Called overriden GetProposedWithdrawalVote()")
	agentsInTeam := mi.Server.GetAgentsInTeam(mi.TeamID)
	proposedWithdrawals := make(map[uuid.UUID]int)

	for _, agentId := range agentsInTeam {
		proposedWithdrawals[agentId] = rand.Intn(2)
	}

	log.Println(proposedWithdrawals)
	return proposedWithdrawals
}

func (mi *ExtendedAgent) Team4_GetProposedWithdrawal(instance common.IExtendedAgent) int {
	// first check if the agent has a team
	if !mi.HasTeam() {
		return 0
	}
	// Currently, assume stated withdrawal matches actual withdrawal
	return instance.Team4_ProposeWithdrawal()
}

func (mi *ExtendedAgent) Team4_StateProposalToTeam() {
	// Broadcast contribution to team
	proposedWithdrawal := mi.Team4_GetProposedWithdrawal(mi)
	proposalMsg := mi.Team4_CreateProposedWithdrawalMessage(proposedWithdrawal)
	mi.BroadcastSyncMessageToTeam(proposalMsg)
}

func (mi *ExtendedAgent) Team4_ProposeWithdrawal() int {
	// first check if the agent has a team
	if !mi.HasTeam() {
		return 0
	}
	if mi.Server.GetTeam(mi.GetID()).TeamAoA != nil {
		// double check if score in agent is sufficient (this should be handled by AoA though)
		commonPool := mi.Server.GetTeam(mi.GetID()).GetCommonPool()
		aoaExpectedWithdrawal := mi.Server.GetTeam(mi.GetID()).TeamAoA.GetExpectedWithdrawal(mi.GetID(), mi.GetTrueScore(), commonPool)

		// Seed the random generator
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))

		// Generate a random number between 0 and 10
		randomAddition := rng.Intn(11) // Intn(11) generates a number in [0, 10]
		// Add the random number to the expected withdrawal
		aoaExpectedWithdrawal += randomAddition
		if commonPool < aoaExpectedWithdrawal {
			return commonPool
		}
		return aoaExpectedWithdrawal
	} else {
		if mi.VerboseLevel > 6 {
			log.Printf("[WARNING] Agent %s has no AoA, withdrawing 0\n", mi.GetID())
		}
		return 0
	}
}

func (mi *ExtendedAgent) Team4_HandleProposedWithdrawalMessage(msg *common.Team4_ProposedWithdrawalMessage) {
	if mi.VerboseLevel > 8 {
		log.Printf("Agent %s received proposed withdrawal notification from %s: amount=%d\n",
			mi.GetID(), msg.GetSender(), msg.StatedAmount)
	}
	// Team's agent should implement logic to store or process the reported proposed withdrawal amount as desired
}

func (mi *ExtendedAgent) Team4_GetPunishmentVoteMap() map[int]int {
	punishmentVoteMap := make(map[int]int)

	for punishment := 0; punishment <= 4; punishment++ {
		punishmentVoteMap[punishment] = rand.Intn(5)
	}

	return punishmentVoteMap
}
