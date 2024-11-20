package environmentServer

import (
	"SOMAS_Extended/agents"
	"SOMAS_Extended/common"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/server"
)

type EnvironmentServer struct {
	*server.BaseServer[common.IExtendedAgent]

	teamsMutex    sync.RWMutex
	agentInfoList []common.ExposedAgentInfo
	teams         map[uuid.UUID]common.Team

	roundScoreThreshold int
	deadAgents          []common.IExtendedAgent
}

// overrides that requires implementation
func (cs *EnvironmentServer) RunTurn(i, j int) {
	fmt.Printf("\nTurn %v, %v, current agent count: %v\n", i, j, len(cs.GetAgentMap()))

	if j == 0 {
		cs.StartAgentTeamForming()

	} else { // debug roll dice for agents
		for _, agent := range cs.GetAgentMap() {
			if !cs.IsAgentDead(agent.GetID()) { // only agents that are alive can roll dice
				agent.StartRollingDice()
			}
		}
	}
}

func (cs *EnvironmentServer) RunStartOfIteration(int) {
	fmt.Printf("--------Start of iteration %v---------\n", cs.GetIterations())
	cs.CreateNewRoundScoreThreshold()
	// start team forming

	// take votes at team level and allocate Strategy.
	cs.AllocateAoAs()
}

// Allocate AoA based on team votes;
// for each member in team, count vote for AoA and then take majority (?) vote
// assign majority vote back to team struct (team.Strategy)
func (cs *EnvironmentServer) AllocateAoAs(){
	// once teams assigned, gather AoA votes from each agent.
	for _, team := range cs.teams {
		// ranking cache for each team.
		voteSum = []
		for _, agent := range team.Agents {
			for aoa, vote := range agent.AoARanking{}
			// accumulate vote from each agent in team
			voteSum[aoa] += vote
		}
		// logic to check largest
		maxVote = 0
		for _, sum := range voteSum {
			if sum > maxVote{
				maxVote = sum
			}
		}
		// update teams strategy. 
		team.Strategy = maxVote
	}
}

func (cs *EnvironmentServer) RunEndOfIteration(int) {
	for _, agent := range cs.GetAgentMap() {
		cs.KillAgentBelowThreshold(agent.GetID())
	}
}

// custom override
func (cs *EnvironmentServer) Start() {
	// steal method from package...
	cs.BaseServer.Start()

	// TODO
}

// constructor
func MakeEnvServer(numAgent int, iterations int, turns int, maxDuration time.Duration, maxThread int, agentConfig agents.AgentConfig) *EnvironmentServer {
	serv := &EnvironmentServer{
		BaseServer: server.CreateBaseServer[common.IExtendedAgent](iterations, turns, maxDuration, maxThread),
		teams:      make(map[uuid.UUID]common.Team),
	}
	serv.SetGameRunner(serv)

	// create agents
	for i := 0; i < numAgent; i++ {
		agent := agents.GetBaseAgents(serv, agentConfig)
		serv.AddAgent(agent)
	}

	return serv
}

// debug log printing
func (cs *EnvironmentServer) LogAgentStatus() {
	// log agent count, and their scores
	fmt.Printf("Agent count: %v\n", len(cs.GetAgentMap()))
	for _, agent := range cs.GetAgentMap() {
		agent.LogSelfInfo()
	}
	for _, agent := range cs.deadAgents {
		fmt.Printf("Agent %v is dead\n", agent.GetID())
	}
}

// pretty logging to show all team status
func (cs *EnvironmentServer) LogTeamStatus() {
	for _, team := range cs.teams {
		fmt.Printf("Team %v: %v\n", team.TeamID, team.Agents)
	}
	// log agents that have no team
	for _, agent := range cs.GetAgentMap() {
		if agent.GetTeamID() == uuid.Nil {
			fmt.Printf("Agent %v has no team\n", agent.GetID())
		}
	}
}

func (cs *EnvironmentServer) UpdateAndGetAgentExposedInfo() []common.ExposedAgentInfo {
	// clear the list
	cs.agentInfoList = nil
	for _, agent := range cs.GetAgentMap() {
		cs.agentInfoList = append(cs.agentInfoList, agent.GetExposedInfo())
	}
	return cs.agentInfoList
}

// create a new round score threshold
func (cs *EnvironmentServer) CreateNewRoundScoreThreshold() {
	// random one between 10 to 20 (TODO)
	cs.roundScoreThreshold = rand.Intn(10) + 10
	fmt.Printf("[server] New round score threshold: %v\n", cs.roundScoreThreshold)
}

// check agent score
func (cs *EnvironmentServer) KillAgentBelowThreshold(agentID uuid.UUID) int {
	agent := cs.GetAgentMap()[agentID]
	score := agent.GetTrueScore()
	if score < cs.roundScoreThreshold {
		cs.KillAgent(agentID)
	}
	return score
}

// kill agent
func (cs *EnvironmentServer) KillAgent(agentID uuid.UUID) {
	cs.deadAgents = append(cs.deadAgents, cs.GetAgentMap()[agentID])
	cs.RemoveAgent(cs.GetAgentMap()[agentID])
	fmt.Printf("[server] Agent %v killed\n", agentID)
}

// is agent dead
func (cs *EnvironmentServer) IsAgentDead(agentID uuid.UUID) bool {
	for _, deadAgent := range cs.deadAgents {
		if deadAgent.GetID() == agentID {
			return true
		}
	}
	return false
}

// team forming

func (cs *EnvironmentServer) StartAgentTeamForming() {
	// Clear existing teams at the start of team formation
	cs.teamsMutex.Lock()
	cs.teams = make(map[uuid.UUID]common.Team)
	cs.teamsMutex.Unlock()

	// Get updated agent info and let agents form teams
	agentInfo := cs.UpdateAndGetAgentExposedInfo()

	fmt.Printf("------------- [server] Starting team formation -------------\n\n")

	// Launch team formation for each agent
	for _, agent := range cs.GetAgentMap() {
		agent.StartTeamForming(agentInfo)
	}
}

func (cs *EnvironmentServer) CreateTeam() {
	cs.teams = make(map[uuid.UUID]common.Team)
}

func (cs *EnvironmentServer) AddAgentToTeam(agentID uuid.UUID, teamID uuid.UUID) {
	cs.teamsMutex.Lock()
	defer cs.teamsMutex.Unlock()

	// Check if agent is already in this team
	team := cs.teams[teamID]
	for _, existingAgent := range team.Agents {
		if existingAgent == agentID {
			return // Skip if agent already exists
		}
	}

	team.Agents = append(team.Agents, agentID)
	cs.teams[teamID] = team
}

func (cs *EnvironmentServer) CheckAgentAlreadyInTeam(agentID uuid.UUID) bool {
	cs.teamsMutex.RLock()
	defer cs.teamsMutex.RUnlock()

	for _, team := range cs.teams {
		for _, agent := range team.Agents {
			if agent == agentID {
				return true
			}
		}
	}
	return false
}

func (cs *EnvironmentServer) CreateAndInitTeamWithAgents(agentIDs []uuid.UUID) uuid.UUID {
	// Skip if no agents provided
	if len(agentIDs) == 0 {
		return uuid.UUID{}
	}

	// check if any agent is already in a team
	for _, agentID := range agentIDs {
		if cs.CheckAgentAlreadyInTeam(agentID) {
			fmt.Printf("[server] Agent %v is already in a team\n", agentID)
			return uuid.UUID{}
		}
	}

	// Generate team ID first
	teamID := uuid.New()

	// Protect map write with mutex
	cs.teamsMutex.Lock()
	cs.teams[teamID] = common.Team{
		TeamID: teamID,
		Agents: agentIDs,
	}
	cs.teamsMutex.Unlock()

	// Update each agent's team ID
	for _, agentID := range agentIDs {
		if agent, exists := cs.GetAgentMap()[agentID]; exists {
			agent.SetTeamID(teamID)
		}
	}

	fmt.Printf("[server] Created team %v with agents %v\n", teamID, agentIDs)
	return teamID
}
