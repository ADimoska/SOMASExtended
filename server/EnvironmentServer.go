package environmentServer

import (
	aoa "SOMAS_Extended/ArticlesOfAssociation"
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
	teams         map[uuid.UUID]*common.Team

	roundScoreThreshold int
	deadAgents          []common.IExtendedAgent
	orphanPool          OrphanPoolType

	// set of options for team strategies (agents rank these options)
	aoaMenu []aoa.IArticlesOfAssociation
}

// overrides that requires implementation
func (cs *EnvironmentServer) RunTurn(i, j int) {
	fmt.Printf("\n\nIteration %v, Turn %v, current agent count: %v\n", i, j, len(cs.GetAgentMap()))

	cs.teamsMutex.Lock()
	defer cs.teamsMutex.Unlock()

	// Agents roll dice and make their contributions for this turn
	for _, team := range cs.teams {
		fmt.Println("\nRunning turn for team ", team.TeamID)
		// Sum of contributions from all agents in the team for this turn
		agentContributionsTotal := 0
		for _, agentID := range team.Agents {
			agent := cs.GetAgentMap()[agentID]
			if agent.GetTeamID() == uuid.Nil {
				continue
			}
			if cs.IsAgentDead(agentID) {
				continue
			}
			agent.StartRollingDice(agent)
			agentActualContribution := agent.GetActualContribution(agent)
			agentContributionsTotal += agentActualContribution
			agentStatedContribution := agent.GetStatedContribution(agent)
			agentScore := agent.GetTrueScore()
			// Update audit result for this agent
			team.TeamAoA.SetContributionAuditResult(agentID, agentScore, agentActualContribution, agentStatedContribution)
			agent.SetTrueScore(agentScore - agentActualContribution)
		}

		// Update common pool with total contribution from this team
		team.SetCommonPool(team.GetCommonPool() + agentContributionsTotal)

		// Initiate Contribution Audit vote
		contributionAuditVotes := []aoa.Vote{}
		for _, agentID := range team.Agents {
			agent := cs.GetAgentMap()[agentID]
			vote := agent.GetContributionAuditVote()
			contributionAuditVotes = append(contributionAuditVotes, vote)
		}

		// Execute Contribution Audit if necessary
		if agentToAudit := team.TeamAoA.GetVoteResult(contributionAuditVotes); agentToAudit != uuid.Nil {
			auditResult := team.TeamAoA.GetContributionAuditResult(agentToAudit)
			for _, agentID := range team.Agents {
				agent := cs.GetAgentMap()[agentID]
				agent.SetAgentContributionAuditResult(agentToAudit, auditResult)
			}
		}

		// Sum of withdrawals from all agents in the team for this turn
		agentWithdrawalsTotal := 0
		// All agents withdraw from common pool for this turn
		for _, agentID := range team.Agents {
			agent := cs.GetAgentMap()[agentID]
			if agent.GetTeamID() == uuid.Nil {
				continue
			}
			if cs.IsAgentDead(agentID) {
				continue
			}
			agentActualWithdrawal := agent.GetActualWithdrawal(agent)
			agentWithdrawalsTotal += agentActualWithdrawal
			agentStatedWithdrawal := agent.GetStatedWithdrawal(agent)
			agentScore := agent.GetTrueScore()
			// Update audit result for this agent
			team.TeamAoA.SetWithdrawalAuditResult(agentID, agentScore, agentActualWithdrawal, agentStatedWithdrawal)
			agent.SetTrueScore(agentScore + agentActualWithdrawal)
		}
		// Update common pool with total withdrawal from this team
		// .. we only do this after all agents have withdrawn from the common pool
		team.SetCommonPool(team.GetCommonPool() - agentWithdrawalsTotal)

		// Initiate Withdrawal Audit vote
		withdrawalAuditVotes := []aoa.Vote{}
		for _, agentID := range team.Agents {
			agent := cs.GetAgentMap()[agentID]
			vote := agent.GetWithdrawalAuditVote()
			withdrawalAuditVotes = append(withdrawalAuditVotes, vote)
		}

		// Execute Withdrawal Audit if necessary
		if agentToAudit := team.TeamAoA.GetVoteResult(withdrawalAuditVotes); agentToAudit != uuid.Nil {
			auditResult := team.TeamAoA.GetWithdrawalAuditResult(agentToAudit)
			for _, agentID := range team.Agents {
				agent := cs.GetAgentMap()[agentID]
				agent.SetAgentWithdrawalAuditResult(agentToAudit, auditResult)
			}
		}
	}

	// TODO: Reallocate agents who left their teams during the turn
}

func (cs *EnvironmentServer) RunStartOfIteration(iteration int) {
	fmt.Printf("--------Start of iteration %v---------\n", iteration)

	// Initialise random threshold
	cs.CreateNewRoundScoreThreshold()

	// Revive all dead agents
	cs.ReviveDeadAgents()

	// start team forming
	cs.StartAgentTeamForming()

	// take votes at team level and allocate Strategy.
	cs.AllocateAoAs()
}

// Allocate AoA based on team votes;
// for each member in team, count vote for AoA and then take majority (?) vote
// assign majority vote back to team struct (team.Strategy)
func (cs *EnvironmentServer) AllocateAoAs() {
	// Iterate over each team
	for _, team := range cs.teams {
		// ranking cache for each team.
		var voteSum = []int{0, 0, 0, 0}
		for _, agent := range team.Agents {
			if cs.IsAgentDead(agent) {
				continue
			}
			for aoa, vote := range cs.GetAgentMap()[agent].GetAoARanking() {
				voteSum[aoa] += vote
			}
		}

		// Determine the preferred AoA based on the majority vote
		currentMax := 0
		preference := 0
		for aoa, voteCount := range voteSum {
			if voteCount > currentMax {
				currentMax = voteCount
				preference = aoa
			}
		}

		// Update the team's strategy
		team.TeamAoA = cs.aoaMenu[preference]
		cs.teams[team.TeamID] = team
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

func (cs *EnvironmentServer) ReviveDeadAgents() {
	for _, agent := range cs.deadAgents {
		fmt.Printf("[server] Agent %v is being revived\n", agent.GetID())
		agent.SetTrueScore(0) // new agents start with a score of 0
		cs.AddAgent(agent)    // re-add the agent to the server map
	}

	// Clear the slice
	cs.deadAgents = cs.deadAgents[:0]
}

// constructor
func MakeEnvServer(numAgent int, iterations int, turns int, maxDuration time.Duration, maxThread int, agentConfig agents.AgentConfig) *EnvironmentServer {
	serv := &EnvironmentServer{
		BaseServer: server.CreateBaseServer[common.IExtendedAgent](iterations, turns, maxDuration, maxThread),
		teams:      make(map[uuid.UUID]*common.Team),
	}
	serv.SetGameRunner(serv)

	// create agents
	// example: Base Agent & MI_256 from team 4

	// dummy agents (base agent)
	for i := 0; i < numAgent; i++ {
		base_agent := agents.GetBaseAgents(serv, agentConfig)
		serv.AddAgent(base_agent)

		// TEAM 1
		// TEAM 2
		// TEAM 3
		// TEAM 4
		// example: MI_256 from team 4
		team4_agent := agents.Team4_CreateAgent(serv, agentConfig)
		serv.AddAgent(team4_agent)
		// TEAM 5
		// TEAM 6
	}

	serv.aoaMenu = make([]aoa.IArticlesOfAssociation, 4)

	// for now, menu just has 4 choices of AoA. TBC.
	serv.aoaMenu[0] = aoa.CreateFixedAoA()

	serv.aoaMenu[1] = aoa.CreateFixedAoA()

	serv.aoaMenu[2] = aoa.CreateFixedAoA()

	serv.aoaMenu[3] = aoa.CreateFixedAoA()

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

/*
* Print the contents of the orphan pool. Careful as this will not necessarily
* print the elements in the order that you added them.
 */
func (cs *EnvironmentServer) PrintOrphanPool() {
	for i, v := range cs.orphanPool {
		// truncate the UUIDs to make it easier to read
		shortAgentId := i.String()[:8]
		shortTeamIds := make([]string, len(v))

		// go over all the teams in the wishlist and add to shortened IDs
		for _, teamID := range v {
			shortTeamIds = append(shortTeamIds, teamID.String()[:8])
		}

		fmt.Println(shortAgentId, " Wants to join : ", shortTeamIds)
	}
}

// pretty logging to show all team status
func (cs *EnvironmentServer) LogTeamStatus() {
	for _, team := range cs.teams {
		fmt.Printf("Team %v: %v\n", team.TeamID, team.Agents)
	}
	// Log agents with no team
	for _, agent := range cs.GetAgentMap() {
		if agent.GetTeamID() == uuid.Nil {
			fmt.Printf("Agent %v has no team\n", agent.GetID())
		}
	}
	// Log dead agents
	for _, agent := range cs.deadAgents {
		fmt.Printf("Agent %v is dead, last team: %v\n", agent.GetID(), agent.GetLastTeamID())
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
	agent := cs.GetAgentMap()[agentID]

	// Remove the agent from the team
	if teamID := agent.GetTeamID(); teamID != uuid.Nil {
		cs.teamsMutex.Lock()
		team := cs.teams[teamID]
		for i, id := range team.Agents {
			if id == agentID {
				// Remove agent from the team
				team.Agents = append(team.Agents[:i], team.Agents[i+1:]...)
				cs.teams[teamID] = team
				// Set the team of the agent to Nil !!!
				agent.SetTeamID(uuid.Nil)
				break
			}
		}
		cs.teamsMutex.Unlock()

		// Add the agent to the dead agent list and remove it from the server's agent map
		cs.deadAgents = append(cs.deadAgents, agent)
		cs.RemoveAgent(agent)
		fmt.Printf("[server] Agent %v killed\n", agentID)
	}
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
	cs.teams = make(map[uuid.UUID]*common.Team)
	cs.teamsMutex.Unlock()

	// Get updated agent info and let agents form teams
	agentInfo := cs.UpdateAndGetAgentExposedInfo()

	fmt.Printf("------------- [server] Starting team formation -------------\n\n")

	// Launch team formation for each agent
	for _, agent := range cs.GetAgentMap() {
		agent.StartTeamForming(agent, agentInfo)
	}
}

func (cs *EnvironmentServer) CreateTeam() {
	cs.teams = make(map[uuid.UUID]*common.Team)
}

func (cs *EnvironmentServer) AddAgentToTeam(agentID uuid.UUID, teamID uuid.UUID) {
	cs.teamsMutex.Lock()
	defer cs.teamsMutex.Unlock()

	// Check if agent is already in this team
	team, exists := cs.teams[teamID]
	if !exists {
		fmt.Printf("[server] Team %v does not exist\n", teamID)
		return
	}

	for _, existingAgent := range team.Agents {
		if existingAgent == agentID {
			return // Skip if agent already exists
		}
	}

	team.Agents = append(team.Agents, agentID)
}

func (cs *EnvironmentServer) GetAgentsInTeam(teamID uuid.UUID) []uuid.UUID {
	cs.teamsMutex.RLock()
	defer cs.teamsMutex.RUnlock()
	return cs.teams[teamID].Agents
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
	cs.teams[teamID] = common.NewTeam(teamID)
	cs.teamsMutex.Unlock()

	// Update each agent's team ID
	for _, agentID := range agentIDs {
		if agent, exists := cs.GetAgentMap()[agentID]; exists {
			agent.SetTeamID(teamID)
			cs.AddAgentToTeam(agentID, teamID)
		}
	}

	fmt.Printf("[server] Created team %v with agents %v\n", teamID, agentIDs)
	return teamID
}

// agent get team
func (cs *EnvironmentServer) GetTeam(agentID uuid.UUID) *common.Team {
	// cs.teamsMutex.RLock()
	// defer cs.teamsMutex.RUnlock()
	return cs.teams[cs.GetAgentMap()[agentID].GetTeamID()]
}

// Get team from team ID, mostly for testing. 
func (cs *EnvironmentServer) GetTeamFromTeamID(teamID uuid.UUID) *common.Team {
    return cs.teams[teamID]
}
