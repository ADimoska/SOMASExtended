package environmentServer

import (
	"fmt"
	"github.com/google/uuid"
	"math/rand"
	"sync"
	"time"

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
}

func (cs *EnvironmentServer) RunTurn(i, j int) {
	fmt.Printf("\n\nIteration %v, Turn %v, current agent count: %v\n", i, j, len(cs.GetAgentMap()))

	// Go over the list of all agents and add orphans to the orphan pool if
	// they are not already there
	cs.PickUpOrphans()

	// Attempt to allocate the orphans to their preferred teams
	cs.AllocateOrphans()

	cs.teamsMutex.Lock()
	defer cs.teamsMutex.Unlock()

	for _, team := range cs.Teams {
		fmt.Println("\nRunning turn for team ", team.TeamID)
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

			agent.StateContributionToTeam()
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
		for _, agentID := range orderedAgents {
			agent := cs.GetAgentMap()[agentID]
			if agent.GetTeamID() == uuid.Nil || cs.IsAgentDead(agentID) {
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
			team.TeamAoA.SetWithdrawalAuditResult(agentID, agentScore, agentActualWithdrawal, agentStatedWithdrawal, team.GetCommonPool())
			agent.SetTrueScore(agentScore + agentActualWithdrawal)

			// Update the common pool after each withdrawal so agents can see the updated pool before deciding their withdrawal.
			//  Different to the contribution phase!
			team.SetCommonPool(currentPool - agentActualWithdrawal)
			fmt.Printf("[server] Agent %v withdrew %v. Remaining pool: %v\n", agentID, agentActualWithdrawal, team.GetCommonPool())
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
			agent.StateWithdrawalToTeam()
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
}

func (cs *EnvironmentServer) RunStartOfIteration(iteration int) {
	fmt.Printf("--------Start of iteration %v---------\n", iteration)

	// Initialise random threshold
	cs.createNewRoundScoreThreshold()

	// Revive all dead agents
	cs.reviveDeadAgents()

	// start team forming
	cs.StartAgentTeamForming()

	// take votes at team level and allocate Strategy.
	cs.allocateAoAs()
}

// Allocate AoA based on team votes;
// for each member in team, count vote for AoA and then take majority (?) vote
// assign majority vote back to team struct (team.Strategy)
func (cs *EnvironmentServer) allocateAoAs() {
	// var leader = uuid.Nil
	// Iterate over each team
	for _, team := range cs.Teams {
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
		var i int = 0
		for _, agent := range team.Agents {
			i += 1
			if i == 1{
				leader := agent
				// Update the team's strategy
				switch preference {
				case 0:
					team.TeamAoA = common.CreateTeam2AoA(team, leader)
				case 1:
					team.TeamAoA = common.CreateTeam2AoA(team , leader)
				case 2:
					team.TeamAoA = common.CreateTeam2AoA(team , leader)
				case 3:
					team.TeamAoA = common.CreateTeam2AoA(team , leader)
				case 4:
					team.TeamAoA = common.CreateTeam2AoA(team , leader)
				default:
					team.TeamAoA = common.CreateTeam2AoA(team , leader)
				}

			}
		}
		// // Update the team's strategy
		// switch preference {
		// case 0:
		// 	team.TeamAoA = common.CreateTeam2AoA(team, leader)
		// case 1:
		// 	team.TeamAoA = common.CreateTeam2AoA()
		// case 2:
		// 	team.TeamAoA = common.CreateTeam2AoA()
		// case 3:
		// 	team.TeamAoA = common.CreateTeam2AoA()
		// case 4:
		// 	team.TeamAoA = common.CreateTeam2AoA()
		// default:
		// 	team.TeamAoA = common.CreateTeam2AoA()
		// }

		cs.Teams[team.TeamID] = team
	}
}

func (cs *EnvironmentServer) RunEndOfIteration(int) {
	for _, agent := range cs.GetAgentMap() {
		cs.killAgentBelowThreshold(agent.GetID())
	}
}

// custom override
func (cs *EnvironmentServer) Start() {
	// steal method from package...
	cs.BaseServer.Start()

	// TODO
}

func (cs *EnvironmentServer) reviveDeadAgents() {
	for _, agent := range cs.deadAgents {
		fmt.Printf("[server] Agent %v is being revived\n", agent.GetID())
		agent.SetTrueScore(0) // new agents start with a score of 0
		cs.AddAgent(agent)    // re-add the agent to the server map
	}

	// Clear the slice
	cs.deadAgents = cs.deadAgents[:0]
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
	for _, team := range cs.Teams {
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
func (cs *EnvironmentServer) createNewRoundScoreThreshold() {
	// random one between 10 to 20 (TODO)
	cs.roundScoreThreshold = rand.Intn(10) + 10
	fmt.Printf("[server] New round score threshold: %v\n", cs.roundScoreThreshold)
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
		cs.teamsMutex.Lock()
		team := cs.Teams[teamID]
		for i, id := range team.Agents {
			if id == agentID {
				// Remove agent from the team
				team.Agents = append(team.Agents[:i], team.Agents[i+1:]...)
				cs.Teams[teamID] = team
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
	cs.Teams = make(map[uuid.UUID]*common.Team)
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
	cs.Teams = make(map[uuid.UUID]*common.Team)
}

func (cs *EnvironmentServer) AddAgentToTeam(agentID uuid.UUID, teamID uuid.UUID) {
	cs.teamsMutex.Lock()
	defer cs.teamsMutex.Unlock()

	// Check if agent is already in this team
	team, exists := cs.Teams[teamID]
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
			fmt.Printf("[server] Agent %v is already in a team\n", agentID)
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

	fmt.Printf("[server] Created team %v with agents %v\n", teamID, agentIDs)
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

// Possibly needs to look at what team/AoA is being used to tally up the votes
func (cs *EnvironmentServer) overrideAgentRolls(agentId uuid.UUID, controllerIds []uuid.UUID, stickThreshold int) {
	controlled := cs.GetAgentMap()[agentId]
	currentScore := controlled.GetTrueScore()

	accumulatedScore := 0
	rounds := 1
	prevRoll := -1

	rollingComplete := false

	for !rollingComplete {
		// AoAs can change how many stick decisions are needed here
		numStickDecisions := 0
		// The agents responsible for making the stick or again decision
		for _, controllerId := range controllerIds {
			controller := cs.GetAgentMap()[controllerId]
			numStickDecisions += controller.StickOrAgainFor(agentId, accumulatedScore, prevRoll)
		}

		if numStickDecisions >= stickThreshold {
			rollingComplete = true
			fmt.Printf("%s decided to [STICK], score accumulated: %v", agentId, accumulatedScore)
			break
		}

		if rounds > 1 {
			fmt.Printf("%s decided to [CONTINUE ROLLING], previous roll: %v", agentId, prevRoll)
		}

		currentRoll := generateScore()
		fmt.Printf("%s rolled: %v\n this turn", agentId, currentRoll)
		if currentRoll <= prevRoll {
			// Gone bust, so reset the accumulated score and break out of the loop
			accumulatedScore = 0
			fmt.Printf("%s **[HAS GONE BUST!]** round: %v, current score: %v\n", agentId, rounds, currentScore)
			break
		}

		accumulatedScore += currentRoll
		prevRoll = currentRoll
		rounds++
	}
	// In case the agent has gone bust, this does nothing
	controlled.SetTrueScore(currentScore + accumulatedScore)
	// Log the updated score
	fmt.Printf("%s turn score: %v, total score: %v\n", agentId, accumulatedScore, controlled.GetTrueScore())
}

func generateScore() int {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	score := 0
	for i := 0; i < 3; i++ {
		score += rand.Intn(6) + 1
	}
	return score
}
