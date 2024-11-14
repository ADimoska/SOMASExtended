package baseDiceServer

import (
	baseDiceAgent "SOMASExtended/BaseDiceGame/BaseDiceAgent"
	common "SOMASExtended/BaseDiceGame/common"
	"fmt"

	baseServer "github.com/MattSScott/basePlatformSOMAS/v2/pkg/server"
	uuid "github.com/google/uuid"
)

// NOTES:
// Need the BaseDiceAgent to have a getter / setter functions for their team, and their score
// once this is implemented, change any instances of ag.team and ag.score, etc etc to the appropriate getter / setter func.

type IBaseDiceServer interface {
	baseServer.IServer[baseDiceAgent.IBaseDiceAgent]
	createServer(int, int, int, int, int) *IBaseDiceServer
	formTeams()
	voteforArticlesofAssociation()
	runTurn()
	manageResources()
	generateReport()
	audit()
	modifyRules()
	verifyThreshold()
}

type BaseDiceServer struct {
	*baseServer.BaseServer[baseDiceAgent.IBaseDiceAgent]
	teams     map[uuid.UUID]common.Team //map of team IDs to their corresponding Team struct.
	turns     int
	teamSize  int
	numAgents int
	rounds    int
	threshold int
}

// TEAM 2 METHODS BELOW

// TODO:
func (bds *BaseDiceServer) createServer(threshold, rounds, turns, teamSize, numAgents int) *IBaseDiceServer {

}

func (bds *BaseDiceServer) formTeams() {
	agents := bds.GetAgentMap()
	teamSize := bds.teamSize
	numOfAgents := bds.numAgents
	numTeams := numOfAgents / teamSize
	teamIDList := []uuid.UUID{}

	// Step 1: Create Teams

	// create [numTeams] Team structs, initialised each with a different TeamID, empty agent slice and empty strategy / commonpool.
	for i := 0; i < numTeams; i++ {
		//Create a new Team struct
		team := common.NewTeam()

		// fill out the mapping between teamID's and the team struct.
		bds.teams[team.TeamID] = team

		// keep a list of the team ids
		teamIDList = append(teamIDList, team.TeamID)

	}

	// Step 2: Assign each agent a team

	teamIndex := 0  // what teamID we are currently looking at
	agentCount := 0 // counts number of agents on a team

	// iterate over all agents, first adding the agent to their team struct, then populating the agent with their team struct.
	for _, ag := range agents {

		// find the teamID of the team we are currently working with
		currentTeamID := teamIDList[teamIndex]

		// append this agents uuid to the list of agents in their team struct.
		teamAgentList := bds.teams[currentTeamID].Agents
		teamAgentList = append(teamAgentList, ag.GetID())

		//assign agent the team represented by the current team id
		ag.team = bds.teams[currentTeamID] // TODO: use setter function

		//increment num of agents on the team
		agentCount++

		// if we have reached the team size, move on to the next team index and reset the counter.
		if agentCount == teamSize {
			teamIndex++
			agentCount = 0
		}
	}

}

// TODO:
func (bds *BaseDiceServer) voteforArticlesofAssociation() {

}

func (bds *BaseDiceServer) runTurn() {

	// Step 1: Get each agent to enter the Dice Roll loop and attain a score.
	for _, ag := range bds.GetAgentMap() {
		ag.RollDice(ag)
	}

	// Step 2: Agents make contribution to their team pool, and server redistributes based on team rules.
	bds.manageResources()

	// Step 3: Report Generation (and broadcast?)

	bds.generateReport()

	// Step 4: Run the Audit Process

	bds.audit()

	// Step 5: Run the Rule Mod Process

	bds.modifyRules()

}

func (bds *BaseDiceServer) manageResources() {

	// Stage 1: Contribution

	// iterate through agents and call on them to make their contribution to their teams common pool
	for _, ag := range bds.GetAgentMap() {
		agentTeam := bds.teams[ag.team.TeamID]
		agentTeam.CommonPool += ag.MakeContribution()
	}

	// Stage 2: Redistribution

	// iterate through the agents and give them part of their teams common pool, based on their teams strategy.
	for _, ag := range bds.GetAgentMap() {

		// determine this agents share of their teams common pool, given their teams strategy.

		//TODO: need to implement determineShare function, and use a getter function to retrieve that agent's team, and thus their commonpool / strat.
		shareOfPool := determineShare(ag.team.CommonPool, ag.team.Strategy)

		// increase their score by what they are given from the pool
		ag.score += shareOfPool //TODO: use setter function
	}

}

// TEAM 5 METHODS

// generateReport generates and returns a report for each agent, including team common pool and agent-specific history.
func (bds *BaseDiceServer) generateReport() []Report {
	var reports []Report // Slice to store reports for each agent

	for _, team := range bds.teams { // Iterate over each team
		teamCommonPool := team.CommonPool // Retrieve the current team common pool

		for agentID := range team.Agents { // Iterate over each agent in the team
			agent := bds.GetAgentMap()[agentID] // Retrieve the agent by ID

			// Call the agent's BroadcastReport method, which returns a report slice
			agentReports := agent.BroadcastReport(teamCommonPool)

			// Append the agent's report(s) to the main reports slice
			reports = append(reports, agentReports...)
		}
	}

	return reports // Return the slice of all reports generated
}

func (bds *BaseDiceServer) eliminateAgentsBelowThreshold() {
	// Create a slice to store the IDs of agents who will be eliminated
	var agentsToEliminate []uuid.UUID

	// Iterate over all agents in the game
	for id, agent := range bds.GetAgentMap() {
		// Retrieve the agent's current score
		agentScore := getAgentScore(agent)

		// Check if the agent's score is below the elimination threshold
		if agentScore < bds.threshold {
			// Add the agent's ID to the list of agents to eliminate
			agentsToEliminate = append(agentsToEliminate, id)

			// Notify that the agent will be eliminated
			fmt.Printf("Agent %s did not meet the threshold of %d and will be eliminated.\n", id, bds.threshold)
		}
	}

	// Remove each agent who did not meet the threshold from the game
	for _, id := range agentsToEliminate {
		removeAgent(id)
	}
}
func removeAgent(id uuid.UUID) {
	// Implement logic to remove the agent from the server's agent map
	delete(bds.GetAgentMap(), id)
	// Additional cleanup might be necessary
}

func (bds *BaseDiceServer) audit(agent *BaseDiceAgent, team *Team) bool {
	// Retrieve the contribution rule function and audit failure strategy
	contributionRuleFunction := team.ArticlesOfAssociation.ContributionRule // Function to calculate required contribution per round
	auditFailureStrategy := team.ArticlesOfAssociation.AuditFailureStrategy // Function to handle actions when audit fails
	auditCost := team.ArticlesOfAssociation.AuditCost                       // Retrieve audit cost from AoA

	// Iterate over each turn record in the agent’s memory
	for _, turnRecord := range agent.memory[agent.GetID()] {
		totalScore := turnRecord.TotalScore           // Total score rolled in this turn
		actualContribution := turnRecord.Contribution // Actual contribution made by the agent in this turn

		// Calculate expected contribution using the team strategy function
		expectedContribution := contributionRuleFunction(totalScore)

		// Check if actual contribution meets expected contribution
		if actualContribution < expectedContribution {
			// Audit fails: contribution does not meet the required amount
			team.CommonPool -= auditCost // Deduct audit cost from team’s common pool

			// Call the team’s audit failure strategy
			auditFailureStrategy(agent, team) // Execute team-defined actions on audit failure

			// Return true indicating the agent cheated
			return true
		}
	}

	// Return false if no failures found (agent did not cheat)
	return false
}
