package environmentServer

import (
	"log"

	"github.com/google/uuid"
)

/* Declare the orphan pool for keeping track of agents that are not currently
* part of a team. This maps agentID -> slice of teamIDs that agent wants to
* join. Note that the slice of teams is processed in order, so the agent should
* put the team it most wants to join at the start of the slice. */
type OrphanPoolType map[uuid.UUID][]uuid.UUID

// The percentage of agents that have to vote 'accept' in order for an orphan
// to be taken into a team
const MajorityVoteThreshold float32 = 0.7

/*
* Ask all the agents in a team if they would be willing to accept an orphan
* into the team. This function accepts a threshold, that is used to determine
* whether to grant entry or not. For example, a threshold of 0.7 means that at
* least 70% of agents in the team have to be willing to accept the orphan.
*
* There is no logic in this function to check for the case where the agent is
* already in the team, this is not the responsibility of this function. It
* should not happen if the orphan pool is correctly managed.
 */
func (cs *EnvironmentServer) RequestOrphanEntry(orphanID, teamID uuid.UUID, entryThreshold float32) bool {
	// Get the team and the current number of team members
	team := cs.GetTeamFromTeamID(teamID)
	agent_map := cs.GetAgentMap()

	num_members := len(team.Agents)
	total_votes := 0

	// For each agent in the team
	for _, agentID := range team.Agents {
		// Get their vote
		vote := agent_map[agentID].VoteOnAgentEntry(orphanID)
		// increment the total votes if they vote 'yes'
		if vote {
			total_votes++
		}
	}

	// Calculate the acceptance ratio and return 'yes' only if enough of the
	// team has voted to accept.
	acceptance := float32(total_votes) / float32(num_members)
	return (acceptance >= entryThreshold)
}

/*
* Go through the pool and attempt to allocate each of the orphans to a team,
* based on the preference they have expressed.
 */
func (cs *EnvironmentServer) AllocateOrphans() {
	agent_map := cs.GetAgentMap()

	// Create pool for keeping track of which orphans have not been allocated.
	// This is because we want to only keep the unallocated orphans in the
	// pool. This is the safer alternative to deleting from the pool as we are
	// iterating through it.
	unallocated := make(OrphanPoolType)

	// for each orphan currently in the pool / shelter
	for orphanID, teamsList := range cs.orphanPool {
		log.Printf("allocating %v\n", orphanID)
		var accepted = false
		// for each team that orphan wants to join
		for _, teamID := range teamsList {
			log.Printf("team id testing is %v\n", teamID)
			// Skip if already accepted into a team
			if accepted {
				break
			}

			// Otherwise attempt to join the team
			accepted = cs.RequestOrphanEntry(orphanID, teamID, MajorityVoteThreshold)
			// If the team has voted to accept the orphan
			if accepted {
				agent_map[orphanID].SetTeamID(teamID) // Update agent's knowledge of its team
				cs.AddAgentToTeam(orphanID, teamID)   // Update team's knowledge of its agents
				log.Printf("%v accepted by team %v !!\n", orphanID, teamID)
			}
			// Otherwise, continue to the next team in the preference list.
		}

		if !accepted {
			unallocated[orphanID] = teamsList // add to unallocated
			log.Printf("%v remains in the orphan pool after allocation...\n", orphanID)
		}
	}

	// Assign the unallocated pool as the new orphan pool.
	cs.orphanPool = unallocated
}

/*
* Go over all the agents in the agent map. If there is an agent that is not
* part of a team, then add it to the orphan pool. This allows the server to
* actively pick up agents that have been removed from a team or that have left.
* This prevents agents from having to tell the server to 'please put me in the
* orphan pool'.
*
* This will not break for dead agents, because dead agents should be in a
* separate map. (deadAgents)
 */
func (cs *EnvironmentServer) PickUpOrphans() {

	// Initialise the orphanPool map if it is nil
	if cs.orphanPool == nil {
		cs.orphanPool = make(OrphanPoolType, 0)
	}

	// sweep over all the agents in the server's agent map
	for agentID, agent := range cs.GetAgentMap() {

		// If the agent is not part of a team
		if agent.GetTeamID() == uuid.Nil {
			// if the agent does not belong to a team, and is not in the orphan
			// pool already, then add it to the orphan pool.
			_, exists := cs.orphanPool[agentID]

			// Extract the preferences from the agent, and update them. We do
			// this even for orphans that are already in the pool because we want
			// them to be able to update their preferences on which teams they
			// would like to join
			log.Printf("testing %v\n", agentID)
			cs.orphanPool[agentID] = agent.GetTeamRanking()

			for _, x := range cs.orphanPool[agentID] {
				log.Printf("wants to join %v\n", x)
			}

			if !exists {
				log.Printf("%v was added to the orphan pool \n", agentID)
			}
		}
	}
}
