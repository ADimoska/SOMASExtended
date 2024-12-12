package main

/*
* Code to test the functionality of the orphan pool, which deals with agents
* that are not currently part of a team re-joining teams in subsequent turns.
 */

import (
	agents "github.com/ADimoska/SOMASExtended/agents"
	common "github.com/ADimoska/SOMASExtended/common"
	envServer "github.com/ADimoska/SOMASExtended/server"
	baseServer "github.com/MattSScott/basePlatformSOMAS/v2/pkg/server"
	"reflect"
	"testing" // built-in go testing package
	"time"

	"bou.ke/monkey"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert" // assert package, easier to
)

/*
* Return a test server environment
 */
func CreateTestServer() (*envServer.EnvironmentServer, []uuid.UUID) {
	// Default test config
	agentConfig := agents.AgentConfig{
		InitScore:    0,
		VerboseLevel: 10,
	}

	serv := &envServer.EnvironmentServer{
		// note: the zero turn is used for team forming
		BaseServer: baseServer.CreateBaseServer[common.IExtendedAgent](2, 3, 1000*time.Millisecond, 10),
		Teams:      make(map[uuid.UUID]*common.Team),
	}
	serv.SetGameRunner(serv)

	const numAgents int = 10

	agentPopulation := []common.IExtendedAgent{}
	for i := 0; i < numAgents; i++ {
		// agentPopulation = append(agentPopulation, agents.Team4_CreateAgent(serv, agentConfig))
		agentPopulation = append(agentPopulation, agents.GetBaseAgents(serv, agentConfig))
		agentPopulation = append(agentPopulation, agents.Create_Team1Agent(serv, agentConfig, 0))
		agentPopulation = append(agentPopulation, agents.Team2_CreateAgent(serv, agentConfig))
		agentPopulation = append(agentPopulation, agents.Team3_CreateAgent(serv, agentConfig))
		// Add other teams' agents here
	}

	for _, agent := range agentPopulation {
		serv.AddAgent(agent)
	}

	// Extract the list of agent IDs
	agentIDs := make([]uuid.UUID, 0)
	for id := range serv.GetAgentMap() {
		agentIDs = append(agentIDs, id)
	}

	return serv, agentIDs
}

/* Define Mock functions for the VoteOnAgentEntry function. These will override
* the base implementation to test different voting logic. Yes, monkeypatching
* is not particularly safe practice with Go but seeing as there is literally no
* way to override the base platform during the game instantiation without
* putting test code inside the base platform, this is the best I can think of.
* The fact that it is so hard to override the base platfrom from outside the
* base platform is an issue we will probably encounter later. */

/*
* Behaviour 1 - Override the VoteOnAgentEntry function to always vote 'no'
 */
func mockVoteAlwaysFalse(mi *agents.ExtendedAgent, candidateID uuid.UUID) bool {
	return false
}

/*
* Allocation as it occurs on the BasePlatform, where the VoteOnAgentEntry()
* function returns true for every candidate ID
 */
func TestAlwaysAccept(t *testing.T) {
	// Default Test Configuration
	serv, agentIDs := CreateTestServer()

	teamID := serv.CreateAndInitTeamWithAgents(agentIDs)
	agents := serv.GetAgentsInTeam(teamID)
	assert.Equal(t, agentIDs, agents)

	// Go through all agents, and ask the team it is already in if it will
	// accept it. Yes this should technically never be asked, this is a base
	// case.
	for agentID, agent := range serv.GetAgentMap() {
		teamID := agent.GetTeamID()
		accepted := serv.RequestOrphanEntry(agentID, teamID, 1.00)
		assert.Equal(t, true, accepted)
	}
}

/*
* Test if all the orphans are rejected when all agents vote 'false' to accept
 */
func TestAlwaysReject(t *testing.T) {
	// Create a test server and put all the agents in the same team
	serv, agentIDs := CreateTestServer()
	serv.CreateAndInitTeamWithAgents(agentIDs)

	// Monkey-path the voting method to always reject
	monkey.PatchInstanceMethod(reflect.TypeOf(&agents.ExtendedAgent{}), "VoteOnAgentEntry", mockVoteAlwaysFalse)
	defer monkey.UnpatchAll()

	// Go through all the agents, every agent should be rejected.
	for agentID, agent := range serv.GetAgentMap() {
		teamID := agent.GetTeamID()
		accepted := serv.RequestOrphanEntry(agentID, teamID, 1.00)
		assert.Equal(t, false, accepted)
	}
}

/*
* Test basic team-based voting logic. In this example, agents will vote to
* 'accept' an orphan if that 'orphan' is already in that team. In theory this
* would not happen since an orphan should not be in that team. Just a basic
* unit test
 */
func TestAcceptIfInCurrentTeam(t *testing.T) {
	// Create a test server and put all the agents in the same team
	serv, agentIDs := CreateTestServer()

	// Create two sub-teams, each with half the agents.
	numAgents := len(agentIDs)
	team1ID := serv.CreateAndInitTeamWithAgents(agentIDs[:(numAgents / 2)])
	team2ID := serv.CreateAndInitTeamWithAgents(agentIDs[(numAgents / 2):])

	/*
	 * Behaviour 2 - Override the VoteOnAgentEntry function to vote yes only if the
	 * agent is already in the current team. In order to be able to check what
	 * team we're in, we define this as a closure to have access to the server
	 * (you can't access the agent's private members inside the mock function
	 * here...
	 */
	mockVoteBasedOnTeam := func(mi *agents.ExtendedAgent, candidateID uuid.UUID) bool {
		myTeam := mi.GetTeamID()
		candidateTeam := serv.GetTeam(candidateID).TeamID
		return (candidateTeam == myTeam)
	}

	// Monkey-path the voting method to only accept an agent if it is already in that team
	monkey.PatchInstanceMethod(reflect.TypeOf(&agents.ExtendedAgent{}), "VoteOnAgentEntry", mockVoteBasedOnTeam)
	defer monkey.UnpatchAll()

	// Go through all the agents. The agent should be accepted if it is already
	// in that team, otherwise it should be rejected.
	for agentID, agent := range serv.GetAgentMap() {
		teamID := agent.GetTeamID()

		// Try to enter team1
		accepted := serv.RequestOrphanEntry(agentID, team1ID, 1.00)
		assert.Equal(t, (teamID == team1ID), accepted)

		// Try to enter team2
		accepted = serv.RequestOrphanEntry(agentID, team2ID, 1.00)
		assert.Equal(t, (teamID == team2ID), accepted)
	}
}

/*
* Test that the server can allocate orphans to a team
 */
func TestAllocateOrphans(t *testing.T) {
	// Create the test server
	serv, agentIDs := CreateTestServer()
	agent_map := serv.GetAgentMap()

	// Allocate all agents but the first two to a team
	orphan1 := agentIDs[0]
	orphan2 := agentIDs[1]
	teamID := serv.CreateAndInitTeamWithAgents(agentIDs[2:])

	team := serv.GetTeamFromTeamID(teamID)

	team.TeamAoAID = 1
	// Set the preferences of these agents to the team ID
	agent_map[orphan1].SetAoARanking([]int{1})
	agent_map[orphan2].SetAoARanking([]int{1})

	// Sweep and add these orphans to the orphan pool (this function would
	// normally be called inside RunTurn())
	serv.PickUpOrphans()

	// Attempt to allocate the orphans (also called in RunTurn)
	serv.AllocateOrphans()

	// The default VoteOnAgentEntry method just returns 'true', so all orphans
	// should be accepted by the team.

	// Both agents are associated with that team
	assert.Equal(t, teamID, serv.GetAgentMap()[agentIDs[0]].GetTeamID())
	assert.Equal(t, teamID, serv.GetAgentMap()[agentIDs[1]].GetTeamID())
	// All agents in the game are now in one team
	assert.Equal(t, len(agentIDs), len(serv.GetTeamFromTeamID(teamID).Agents))
}
