package environmentServer

import (
	"log"
	"math/rand"
	"sync"
	"time"

	gameRecorder "github.com/ADimoska/SOMASExtended/gameRecorder"
	"github.com/google/uuid"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/server"

	common "github.com/ADimoska/SOMASExtended/common"
)

type EnvironmentServer struct {
	*server.BaseServer[common.IExtendedAgent]
	Teams map[uuid.UUID]*common.Team

	teamsMutex    sync.RWMutex
	agentInfoList []common.ExposedAgentInfo

	roundScoreThreshold int
	deadAgents          []common.IExtendedAgent
	orphanPool          OrphanPoolType

	// data recorder
	DataRecorder *gameRecorder.ServerDataRecorder

	// server internal state
	turn           int
	iteration      int
	thresholdTurns int
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (cs *EnvironmentServer) RunTurn(i, j int) {
	log.Printf("\n\nIteration %v, Turn %v, current agent count: %v\n", i, j, len(cs.GetAgentMap()))

	// Go over the list of all agents and add orphans to the orphan pool if
	// they are not already there
	cs.PickUpOrphans()

	// Attempt to allocate the orphans to their preferred teams
	cs.AllocateOrphans()

	cs.turn = j

	cs.teamsMutex.Lock()
	// defer cs.teamsMutex.Unlock()

	for _, team := range cs.Teams {
		log.Println("\nRunning turn for team ", team.TeamID)
		// Sum of contributions from all agents in the team for this turn
		agentContributionsTotal := 0
		for _, agentID := range team.Agents {
			agent := cs.GetAgentMap()[agentID]
			if agent.GetTeamID() == uuid.Nil || cs.IsAgentDead(agentID) {
				continue
			}
			// Override agent rolls for testing purposes
			// agentList := []uuid.UUID{agentID}
			// cs.OverrideAgentRolls(agentID, agentList, 1)
			agent.StartRollingDice(agent)
			agentActualContribution := agent.GetActualContribution(agent)
			agentContributionsTotal += agentActualContribution
			agentStatedContribution := agent.GetStatedContribution(agent)

			agent.StateContributionToTeam(agent)
			agentScore := agent.GetTrueScore()
			// Update audit result for this agent
			team.TeamAoA.SetContributionAuditResult(agentID, agentScore, agentActualContribution, agentStatedContribution)
			agent.SetTrueScore(agentScore - agentActualContribution)
		}

		// Update common pool with total contribution from this team
		// 	Agents do not get to see the common pool before deciding their contribution
		//  Different to the withdrawal phase!
		team.SetCommonPool(team.GetCommonPool() + agentContributionsTotal)

		// Initiate Contribution Audit vote
		contributionAuditVotes := []common.Vote{}
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

		orderedAgents := team.TeamAoA.GetWithdrawalOrder(team.Agents)
		commonPoolBefore := team.GetCommonPool()
		for _, agentID := range orderedAgents {
			agent := cs.GetAgentMap()[agentID]
			if cs.IsAgentDead(agentID) {
				continue
			}

			// Pass the current pool value to agent's methods
			currentPool := team.GetCommonPool()
			agentActualWithdrawal := agent.GetActualWithdrawal(agent)
			if agentActualWithdrawal > currentPool {
				agentActualWithdrawal = currentPool // Ensure withdrawal does not exceed available pool
			}
			agentStatedWithdrawal := agent.GetStatedWithdrawal(agent)

			agentScore := agent.GetTrueScore()
			// Update audit result for this agent
			team.TeamAoA.SetWithdrawalAuditResult(agentID, agentScore, agentActualWithdrawal, agentStatedWithdrawal, commonPoolBefore)
			agent.SetTrueScore(agentScore + agentActualWithdrawal)

			// Update the common pool after each withdrawal so agents can see the updated pool before deciding their withdrawal.
			//  Different to the contribution phase!
			team.SetCommonPool(currentPool - agentActualWithdrawal)
			log.Printf("[server] Agent %v withdrew %v. Remaining pool: %v\n", agentID, agentActualWithdrawal, team.GetCommonPool())
		}

		stateWithdrawOrder := make([]uuid.UUID, len(team.Agents))
		copy(stateWithdrawOrder, team.Agents)
		// Shuffle the order of agents to broadcast withdrawal amounts
		rand.Shuffle(len(stateWithdrawOrder), func(i, j int) {
			stateWithdrawOrder[i], stateWithdrawOrder[j] = stateWithdrawOrder[j], stateWithdrawOrder[i]
		})

		for _, agentId := range stateWithdrawOrder {
			agent := cs.GetAgentMap()[agentId]
			if agent.GetTeamID() == uuid.Nil || cs.IsAgentDead(agentId) {
				continue
			}
			agent.StateWithdrawalToTeam(agent)
		}

		// Initiate Withdrawal Audit vote
		withdrawalAuditVotes := []common.Vote{}
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

	// check if threshold turn

	cs.teamsMutex.Unlock()

	if cs.turn%cs.thresholdTurns == 0 && cs.turn > 1 {
		cs.ApplyThreshold()
	}

	cs.teamsMutex.Lock()

	// record data
	cs.RecordTurnInfo()
	cs.teamsMutex.Unlock()
}

func (cs *EnvironmentServer) RunStartOfIteration(iteration int) {
	log.Printf("--------Start of iteration %v---------\n", iteration)

	cs.iteration = iteration

	// record data
	cs.DataRecorder.RecordNewIteration()

	// Initialise random threshold
	cs.createNewRoundScoreThreshold()

	// Revive all dead agents
	cs.reviveDeadAgents()

	// reset all agents (make sure their score starts at 0)
	cs.ResetAgents()

	// start team forming
	cs.StartAgentTeamForming()

	// take votes at team level and allocate Strategy.
	cs.allocateAoAs()
}

// Allocate AoAs (Articles of Association) to teams using the Borda Count voting method.
// Each team member ranks up to 6 AoAs, assigning weighted votes (1st choice = 6 points, 2nd = 5 points, ..., 6th = 1 point).
// The AoA with the highest total weight is selected for the team. If there's a tie, a random AoA among the tied ones is chosen.
func (cs *EnvironmentServer) allocateAoAs() {
	// Process AoA allocation for each team
	for _, team := range cs.Teams {
		// Array to store Borda Count totals for each AoA (1-6). Index 0 is unused for simplicity.
		aoaVoteWeights := make([]int, 7) // aoaVoteWeights[1] to aoaVoteWeights[6]

		// Process votes from all team members
		for _, agent := range team.Agents {

			// Get the agent's ranked preferences for AoAs (maximum of 6 ranked choices)
			agentAoARanking := cs.GetAgentMap()[agent].GetAoARanking()

			// Assign Borda Count weights to the agent's AoA rankings
			for position, aoa := range agentAoARanking {
				// Calculate weight: 1st choice = 6 points, 2nd = 5, ..., 6th = 1
				weight := 6 - position
				if weight < 1 {
					weight = 1 // Minimum weight is 1
				}

				// Add weight to the corresponding AoA if it's within the valid range (1-6)
				if aoa >= 1 && aoa <= 6 {
					aoaVoteWeights[aoa] += weight
				}
			}
		}

		// Determine the maximum weight across all AoAs
		maxWeight := -1
		for aoa := 1; aoa <= 6; aoa++ {
			if aoaVoteWeights[aoa] > maxWeight {
				maxWeight = aoaVoteWeights[aoa]
			}
		}

		// Identify all AoAs that are tied with the maximum weight
		var tiedAoAs []int
		for aoa := 1; aoa <= 6; aoa++ {
			if aoaVoteWeights[aoa] == maxWeight {
				tiedAoAs = append(tiedAoAs, aoa)
			}
		}

		// Resolve ties by selecting a random AoA from the tied candidates
		var selectedAoA int
		if len(tiedAoAs) == 1 {
			selectedAoA = tiedAoAs[0] // Single AoA with the highest weight
		} else if len(tiedAoAs) > 1 {
			randomIndex := rand.Intn(len(tiedAoAs)) // Random choice among tied AoAs
			selectedAoA = tiedAoAs[randomIndex]
		} else {
			selectedAoA = 1 + rand.Intn(6) // No valid votes, choose random AoA [1-6]
		}

		// Assign the selected AoA to the team's strategy based on its value
		switch selectedAoA {
		case 1:
			team.TeamAoA = common.CreateTeam2AoA(team, uuid.Nil, 5)
		case 2:
			team.TeamAoA = common.CreateTeam2AoA(team, uuid.Nil, 5)
		case 3:
			team.TeamAoA = common.CreateTeam2AoA(team, uuid.Nil, 5)
		case 4:
			team.TeamAoA = common.CreateTeam2AoA(team, uuid.Nil, 5)
		default:
			// Default AoA assignment if no valid preference is found
			team.TeamAoA = common.CreateFixedAoA(1)
		}

		// Update the team's AoA allocation in the EnvironmentServer
		cs.Teams[team.TeamID] = team
		log.Println("Team", team.TeamID, "has been assigned AoA", selectedAoA)
	}
}

func (cs *EnvironmentServer) RunEndOfIteration(int) {
	// for _, agent := range cs.GetAgentMap() {
	// 	cs.killAgentBelowThreshold(agent.GetID())
	// }
}

// custom override (what why this is called later then start iteration...)
func (cs *EnvironmentServer) Start() {
	// steal method from package...
	cs.BaseServer.Start()
}

// custom init that gets called earlier
func (cs *EnvironmentServer) Init(turnsForThreshold int) {
	cs.DataRecorder = gameRecorder.CreateRecorder()
	cs.thresholdTurns = turnsForThreshold
}

func (cs *EnvironmentServer) reviveDeadAgents() {
	for _, agent := range cs.deadAgents {
		log.Printf("[server] Agent %v is being revived\n", agent.GetID())
		agent.SetTrueScore(0) // new agents start with a score of 0
		cs.AddAgent(agent)    // re-add the agent to the server map
	}

	// Clear the slice
	cs.deadAgents = cs.deadAgents[:0]
}

// debug log printing
func (cs *EnvironmentServer) LogAgentStatus() {
	// log agent count, and their scores
	log.Printf("Agent count: %v\n", len(cs.GetAgentMap()))
	for _, agent := range cs.GetAgentMap() {
		agent.LogSelfInfo()
	}
	for _, agent := range cs.deadAgents {
		log.Printf("Agent %v is dead\n", agent.GetID())
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

		log.Println(shortAgentId, " Wants to join : ", shortTeamIds)
	}
}

// pretty logging to show all team status
func (cs *EnvironmentServer) LogTeamStatus() {
	log.Println("\n------------- [server] Team status -------------")
	for _, team := range cs.Teams {
		log.Printf("Team %v: %v\n", team.TeamID, team.Agents)
	}
	// Log agents with no team
	for _, agent := range cs.GetAgentMap() {
		if agent.GetTeamID() == uuid.Nil {
			log.Printf("Agent %v has no team\n", agent.GetID())
		}
	}
	// Log dead agents
	for _, agent := range cs.deadAgents {
		log.Printf("Agent %v is dead, last team: %v\n", agent.GetID(), agent.GetLastTeamID())
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
func (cs *EnvironmentServer) createNewRoundScoreThreshold() {
	// random one between 10 to 20 (TODO)
	cs.roundScoreThreshold = rand.Intn(10) + 10
	log.Printf("[server] New round score threshold: %v\n", cs.roundScoreThreshold)
}

// check agent score
func (cs *EnvironmentServer) killAgentBelowThreshold(agentID uuid.UUID) int {
	agent := cs.GetAgentMap()[agentID]
	score := agent.GetTrueScore()
	if score < cs.roundScoreThreshold {
		cs.killAgent(agentID)
	}
	return score
}

// kill agent
func (cs *EnvironmentServer) killAgent(agentID uuid.UUID) {
	agent := cs.GetAgentMap()[agentID]

	// Remove the agent from the team
	if teamID := agent.GetTeamID(); teamID != uuid.Nil {
		// cs.teamsMutex.Lock()
		// defer cs.teamsMutex.Unlock()

		team := cs.Teams[teamID]
		// check if team exists (patch fix - TODO check the root of the error)
		if team == nil {
			log.Printf("[server] Team %v does not exist\n", teamID)
		} else {
			for i, id := range team.Agents {
				if id == agentID {
					// Remove agent from the team
					team.Agents = append(team.Agents[:i], team.Agents[i+1:]...)
					cs.Teams[teamID] = team
					// Set the team of the agent to Nil
					agent.SetTeamID(uuid.Nil)
					break
				}
			}
		}
	}

	// Add the agent to the dead agent list and remove it from the server's agent map
	cs.deadAgents = append(cs.deadAgents, agent)
	cs.RemoveAgent(agent)
	log.Printf("[server] Agent %v killed\n", agentID)
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
	cs.Teams = make(map[uuid.UUID]*common.Team)
	cs.teamsMutex.Unlock()

	// Get updated agent info and let agents form teams
	agentInfo := cs.UpdateAndGetAgentExposedInfo()

	log.Printf("------------- [server] Starting team formation -------------\n\n")

	// Launch team formation for each agent
	for _, agent := range cs.GetAgentMap() {
		agent.StartTeamForming(agent, agentInfo)
	}

	// print team status
	cs.LogTeamStatus()
}

func (cs *EnvironmentServer) CreateTeam() {
	cs.Teams = make(map[uuid.UUID]*common.Team)
}

func (cs *EnvironmentServer) AddAgentToTeam(agentID uuid.UUID, teamID uuid.UUID) {
	cs.teamsMutex.Lock()
	defer cs.teamsMutex.Unlock()

	// Check if agent is already in this team
	team, exists := cs.Teams[teamID]
	if !exists {
		log.Printf("[server] Team %v does not exist\n", teamID)
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
	// cs.teamsMutex.RLock()
	// defer cs.teamsMutex.RUnlock()
	return cs.Teams[teamID].Agents
}

func (cs *EnvironmentServer) CheckAgentAlreadyInTeam(agentID uuid.UUID) bool {
	cs.teamsMutex.RLock()
	defer cs.teamsMutex.RUnlock()

	for _, team := range cs.Teams {
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
			log.Printf("[server] Agent %v is already in a team\n", agentID)
			return uuid.UUID{}
		}
	}

	// Generate team ID first
	teamID := uuid.New()

	// Protect map write with mutex
	cs.teamsMutex.Lock()
	cs.Teams[teamID] = common.NewTeam(teamID)
	cs.teamsMutex.Unlock()

	// Update each agent's team ID
	for _, agentID := range agentIDs {
		if agent, exists := cs.GetAgentMap()[agentID]; exists {
			agent.SetTeamID(teamID)
			cs.AddAgentToTeam(agentID, teamID)
		}
	}

	log.Printf("[server] Created team %v with agents %v\n", teamID, agentIDs)
	return teamID
}

// agent get team
func (cs *EnvironmentServer) GetTeam(agentID uuid.UUID) *common.Team {
	// cs.teamsMutex.RLock()
	// defer cs.teamsMutex.RUnlock()
	return cs.Teams[cs.GetAgentMap()[agentID].GetTeamID()]
}

// Get team from team ID, mostly for testing.
func (cs *EnvironmentServer) GetTeamFromTeamID(teamID uuid.UUID) *common.Team {
	return cs.Teams[teamID]
}

// reset all agents (preserve memory but clears scores)
func (cs *EnvironmentServer) ResetAgents() {
	for _, agent := range cs.GetAgentMap() {
		agent.SetTrueScore(0)
		agent.SetTeamID(uuid.UUID{})
	}
}

func (cs *EnvironmentServer) ApplyThreshold() {
	for _, team := range cs.Teams {
		team.SetCommonPool(0)
		for _, agentID := range team.Agents {
			if !cs.IsAgentDead(agentID) {
				cs.killAgentBelowThreshold(agentID)
			}
			if agent := cs.GetAgentMap()[agentID]; agent != nil {
				agent.SetTrueScore(0)
			}
		}
	}
}

func (cs *EnvironmentServer) RecordTurnInfo() {

	// agent information
	agentRecords := []gameRecorder.AgentRecord{}
	for _, agent := range cs.GetAgentMap() {
		newAgentRecord := agent.RecordAgentStatus(agent)
		newAgentRecord.IsAlive = true
		agentRecords = append(agentRecords, newAgentRecord)
	}

	for _, agent := range cs.deadAgents {
		newAgentRecord := agent.RecordAgentStatus(agent)
		newAgentRecord.IsAlive = false
		agentRecords = append(agentRecords, newAgentRecord)
	}

	teamRecords := []gameRecorder.TeamRecord{}
	for _, team := range cs.Teams {
		newTeamRecord := gameRecorder.NewTeamRecord(team.TeamID)
		teamRecords = append(teamRecords, newTeamRecord)
	}

	cs.DataRecorder.RecordNewTurn(agentRecords, teamRecords)
}
