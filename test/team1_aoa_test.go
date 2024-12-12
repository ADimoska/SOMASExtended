package main

/*
* Code to test the AoA functionality for Team 1
 */

import (
	"bou.ke/monkey"
	agents "github.com/ADimoska/SOMASExtended/agents"
	"github.com/ADimoska/SOMASExtended/common"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"reflect"
	"testing"
)

func TestCondorcetWinner(t *testing.T) {
	serv, agentIDs := CreateTestServer()

	// Remove all team 4 agents, their use of the MI_256 agent prevents us from
	// being able to monkey-patch anything. Honestly might create issues down
	// the line as well.
	for _, agent := range serv.GetAgentMap() {
		if agent.GetTrueSomasTeamID() == 4 {
			serv.RemoveAgent(agent)
		}
	}

	// Force AoA to team 1
	teamID := serv.CreateAndInitTeamWithAgents(agentIDs)
	team := serv.GetTeamFromTeamID(teamID)
	team.TeamAoA = common.CreateTeam1AoA(team)

	/* Mock function to overwrite the voting of an agent. This particular
	 * function simulates a random vote. Note that this rarely produces a
	 * Condorcet winner but it is difficult to override the functions of
	 * different instances of the same agent. Monkeypatching just replaces that
	 * method for all agents. */
	randomAgentVote := func(mi *agents.ExtendedAgent, msg *common.Team1BoundaryBallotRequestMessage) {
		// Randomise the rankings
		ranking := [3]int{1, 2, 3}
		rand.Shuffle(len(ranking), func(i, j int) { ranking[i], ranking[j] = ranking[j], ranking[i] })

		resp := &common.Team1BoundaryBallotResponseMessage{
			BaseMessage:      mi.CreateBaseMessage(),
			RankedCandidates: ranking,
		}
		mi.SendSynchronousMessage(resp, msg.GetSender())
	}

	monkey.PatchInstanceMethod(reflect.TypeOf(&agents.ExtendedAgent{}), "Team1_BoundaryBallotRequestHandler", randomAgentVote)
	defer monkey.UnpatchAll()

	testAgents := team.TeamAoA.(*common.Team1AoA).SelectNChairs(agentIDs, 2)

	res1 := serv.GetAgentMap()[testAgents[0]].Team1_AgreeRankBoundaries()
	res2 := serv.GetAgentMap()[testAgents[1]].Team1_AgreeRankBoundaries()
	assert.Equal(t, res1, res2)
}
