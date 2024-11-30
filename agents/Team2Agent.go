package agents

import (
	"SOMAS_Extended/common"
	"fmt"
	"math"
	"math/rand"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
)

// this is the third tier of composition - embed the extended agent and add 'user specific' fields
type Team2Agent struct {
	*ExtendedAgent
	rank            map[uuid.UUID]string // Map agent UUID ot rank
	trustScore      map[uuid.UUID]int
	strikeCount     map[uuid.UUID]int
	thresholdBounds []int
}

// constructor for Fascist Agent - not sure exaftly whats going on here
func Team2_CreateAgent(funcs agent.IExposedServerFunctions[common.IExtendedAgent], agentConfig AgentConfig) *Team2Agent {
	return &Team2Agent{
		ExtendedAgent: GetBaseAgents(funcs, agentConfig),
	}
}

// ----- 1.2 Trust Score Update -----

// update when not cooperating based on strikes
// update trust score if agent kicked out of other team (can't implement yet?)
// update if votes to audit different to you
// update if agent not audited for that round
// update depending on AoA votes
// update if chooses to audit another agent 

// Overall function to update one agents trust score for other agents
// (can either implement like this or call functions underneath during each event)
func (t2a *Team2Agent) UpdateTrustScore(agentID uuid.UUID, eventType string, strikeCount int) {
    switch eventType {
        case "strike":
            // if agent is not cooperating:
            t2a.ApplyStrike(agentID, strikeCount) // from helper function
        
        // case "kickedFromTeam":
        //     // If the target agent was kicked out of another team
        //     t2a.ApplyKickFromTeam(targetAgentID)
        
        case "auditVote":
            // If the target agent voted to audit you
            t2a.ApplyAuditVote(agentID)
        
        case "notAudited":
            // If the target agent was not audited
            t2a.ApplyNotAudited(agentID)
        
        case "AoAVote":
            // If the target agent voted in AoA
            t2a.ApplyAoAVote(agentID)
        
        case "auditOther":
            // If the target agent audited another agent
            t2a.ApplyAuditOther(agentID)
        
        default:
            fmt.Println("Invalid event type")
    }
}

// TO-DO: call this function when agent found to be not cooperating or in function above
func (t2a *Team2Agent) ApplyStrike(agentID uuid.UUID) {
    t2a.strikeCount[agentID]++ // Increment the strike count for this agent
    var penalty int
    strikeCount := t2a.strikeCount[agentID]
    
    if strikeCount == 1 {
        penalty = 10
    }
    if strikeCount == 2 {
        penalty = 20
    }
    if strikeCount == 3 {
        penalty = 30
    } else {
        penalty = 40
    }

    // Update trust score based on strike count
    t2a.trustScore[agentID] -= penalty

    fmt.Printf("Agent %s received a strike. Total strikes: %d, Penalty applied: %d, New trust score: %d\n",
        agentID, strikeCount, penalty, t2a.trustScore[agentID])
}

// func (t2a *Team2Agent) ApplyKickFromTeam(agentID uuid.UUID) {
//     // Update trust score based on being kicked from another team
//     t2a.trustScore[agentID] -= 5
// }

//TO-DO: get audit votes for other agents in this round
func (t2a *Team2Agent) ApplyAuditVote(agentID uuid.UUID) {
    // Update trust score based on audit vote
    t2a.trustScore[agentID] -= 5
}

//TO-DO: get audit information for other agents in this round
func (t2a *Team2Agent) ApplyNotAudited(agentID uuid.UUID) {
    // Update trust score based on not being audited
    t2a.trustScore[agentID] += 2
}

//TO-DO: figure out AoA vote system
func (t2a *Team2Agent) ApplyAoAVote(agentID uuid.UUID) {
    // Update trust score based on AoA vote
    var reward int
    switch t2a.server.GetAgentVote(agentID) {
    case 1:
        reward = 20
    case 2:
        reward = 10
    case 3:
        reward = 5
    case 4:
        reward = 0
    case 5:
        reward = -5
    case 6:
        reward = -10
    }
    t2a.trustScore[agentID] += reward
}

// TO-DO: call this function in audit vote functions below
func (t2a *Team2Agent) ApplyAuditOther(agentID uuid.UUID) {
    // Update trust score based on auditing another agent
    t2a.trustScore[agentID] += 2
}


// ----- 2.1 Decision to send or accept a team invitiation -----

func (t2a *Team2Agent) DecideTeamForming(agentInfoList []common.ExposedAgentInfo) []uuid.UUID {
    // Initialize selected agents slice
    selectedAgents := make([]uuid.UUID, 0)
    
    // Initialize trust score map if it hasn't been initialized yet
    if t2a.trustScore == nil {
        t2a.trustScore = make(map[uuid.UUID]int)
    }
    
    // Set trust threshold for accepting/sending invitations
    trustThreshold := 7 // This can be adjusted based on desired behavior
    
    // Iterate through all agents
    for _, agentInfo := range agentInfoList {
        // Use the full AgentUUID instead of just the ID
        agentUUID := agentInfo.AgentUUID
        
        // Skip if it's our own ID
        if agentUUID == t2a.GetID() {
            continue
        }
        
        // Get current trust score for this agent
        trustScore, exists := t2a.trustScore[agentUUID]
        if !exists {
            // If we haven't interacted with this agent before, set a neutral trust score
            trustScore = 5
            t2a.trustScore[agentUUID] = trustScore
        }
        
        // Check if we're a follower and they're a leader
        if t2a.rank != nil {
            ourRole, weHaveRole := t2a.rank[t2a.GetID()]
            theirRole, theyHaveRole := t2a.rank[agentUUID]
            
            // Followers always accept leader's invitation
            if weHaveRole && theyHaveRole && 
               ourRole == "Citizen" && theirRole == "Leader" {
                selectedAgents = append(selectedAgents, agentUUID)
                continue
            }
            
            // Leaders are more selective with followers
            if weHaveRole && theyHaveRole && 
               ourRole == "Leader" && theirRole == "Citizen" {
                // Only accept if they meet trust threshold
                if trustScore >= trustThreshold {
                    selectedAgents = append(selectedAgents, agentUUID)
                }
                continue
            }
        }
        
        // For agents without established roles or regular interactions
        // Accept/send invitation if trust score is above threshold
        if trustScore >= trustThreshold {
            selectedAgents = append(selectedAgents, agentUUID)
        }
    }
    
    return selectedAgents
}

// ----- 2.2 Decision to stick -----

func (t2a *Team2Agent) StickorAgain() {

}

// ----- 2.3 Decision to cheat / not cooperate

func (t2a *Team2Agent) DecideContribution() {

}

func (t2a *Team2Agent) DecideWithdrawal() {

}

// 2.4 ----- Decision to Audit Someone

func (t2a *Team2Agent) GetContributionAuditVote() common.Vote {

	// experiment with this; it is our threshold to decide to audit
	var auditThreshold int = 5

	// get list of uuids in our team
	var agentsInTeam []uuid.UUID = t2a.server.GetAgentsInTeam(t2a.teamID)

	var lowestTrustScore int = math.MaxInt // Start with the maximum possible int
	var lowestAgent uuid.UUID

	// iterate over our team, finding the agent with the lowest trust score
	for _, agentID := range agentsInTeam {
		agentTrustScore := t2a.trustScore[agentID]

		if agentTrustScore < lowestTrustScore {
			lowestAgent = agentID
			lowestTrustScore = agentTrustScore
		}
	}

	// if the lowest agent is below the threshold, submit a vote for them
	if lowestTrustScore < auditThreshold {
		// 1 means vote for audit of this person
		return common.CreateVote(1, t2a.GetID(), lowestAgent)
	} else {
		// 0 means abstain / no preference
		return common.CreateVote(0, t2a.GetID(), uuid.Nil)
	}

}

func (t2a *Team2Agent) GetWithdrawalAuditVote() common.Vote {
	// experiment with this; it is our threshold to decide to audit
	var auditThreshold int = 5

	// get list of uuids in our team
	var agentsInTeam []uuid.UUID = t2a.server.GetAgentsInTeam(t2a.teamID)

	var lowestTrustScore int = math.MaxInt // Start with the maximum possible int
	var lowestAgent uuid.UUID

	// iterate over our team, finding the agent with the lowest trust score
	for _, agentID := range agentsInTeam {
		agentTrustScore := t2a.trustScore[agentID]

		if agentTrustScore < lowestTrustScore {
			lowestAgent = agentID
			lowestTrustScore = agentTrustScore
		}
	}

	// if the lowest agent is below the threshold, submit a vote for them
	if lowestTrustScore < auditThreshold {
		// 1 means vote for audit of this person
		return common.CreateVote(1, t2a.GetID(), lowestAgent)
	} else {
		// 0 means abstain / no preference
		return common.CreateVote(0, t2a.GetID(), uuid.Nil)
	}

}

// /////////// ----------------------RANKING SYSTEM---------------------- /////////////

// AssignRole assigns a specific role to an agent based on their UUID
func (t2a *Team2Agent) AssignRole(agentID uuid.UUID, role string) string {
	// currentTurn := t2a.Team2AoA.GetLastRound() // TODO fix the way i call this
	currentTurn := false
	agentIDs := t2a.server.GetAgentsInTeam(t2a.teamID)

	// Ensure rankMap is initialized
	if t2a.rank == nil {
		t2a.rank = make(map[uuid.UUID]string)
	}

	if !currentTurn {
		// Check if the agent already has a role assigned
		if t2a.rank[agentID] != "" {
			// If the agent is already assigned a role, return the current role
			return t2a.rank[agentID]
		} else {
			// If no role is assigned, assign the new role
			t2a.rank[agentID] = role
		}
	} else {
		// Get uuid from vote result
		// voteResult := t2a.Team2AoA.GetVoteResult(t2a.server.GetVotes()) //TODO fix the way i call this
		voteResult := uuid.Nil
		// If the agent is the new leader, assign the leader role
		t2a.rank[voteResult] = "Leader"

		// Assign citizen roles to all other agents
		for _, agentID := range agentIDs {
			if agentID != voteResult {
				t2a.rank[agentID] = "Citizen"
			}
		}
	}

	return role // Return the newly assigned role
}

// AllocateRank decides roles and assigns them based on the current game state and votes
func (t2a *Team2Agent) AllocateRank(votes []common.Vote) common.Vote {
	// Get the current turn
	var highestTrustScore int = math.MinInt // Lowest possible int for comparison
	var highestAgent uuid.UUID

	// Get the list of all agent UUIDs
	agentIDs := t2a.server.GetAgentsInTeam(t2a.teamID)

	if len(agentIDs) == 0 {
		// If no agents are found, abstain from voting
		return common.Vote{IsVote: 0, VoterID: t2a.GetID(), VotedForID: uuid.Nil}
	}
	//get vote result from the server
	// vote result = t2a.Team2AoA.GetVoteResult(votes) //TODO fix the way i call this
	voteResult := uuid.Nil
	if !voteResult {
		//Randomly select a leader
		leaderIndex := rand.Intn(len(agentIDs))
		leaderID := agentIDs[leaderIndex]
		// Assign leader role
		t2a.AssignRole(leaderID, "Leader")

		// Assign citizen roles to all other agents
		for _, agentID := range agentIDs {
			if agentID != leaderID {
				t2a.AssignRole(agentID, "Citizen")
			}
	}else{
		t2a.AssignRole(voteResult, "Leader")
		// Assign citizen roles to all other agents
		for _, agentID := range agentIDs {
			if agentID != voteResult {
				t2a.AssignRole(agentID, "Citizen")
			}
		}
	}
	// // First turn: Randomly assign roles
	// if currentGame == 0 {
	// 	// Randomly select a leader
	// 	leaderIndex := rand.Intn(len(agentIDs))
	// 	leaderID := agentIDs[leaderIndex]
	// 	// Assign leader role
	// 	t2a.AssignRole(leaderID, "Leader")

	// 	// Assign citizen roles to all other agents
	// 	for _, agentID := range agentIDs {
	// 		if agentID != leaderID {
	// 			t2a.AssignRole(agentID, "Citizen")
	// 		}
	// 	}
	// 	// Return a vote indicating the leader
	// 	return common.Vote{IsVote: 1, VoterID: t2a.GetID(), VotedForID: leaderID}
	// } else {
	// 	// Subsequent turns: Use trust scores to assign roles
	// 	for _, agentID := range agentIDs {
	// 		agentTrustScore := t2a.trustScore[agentID] // Trust score map must be initialized
	// 		if agentTrustScore > highestTrustScore {
	// 			highestAgent = agentID
	// 			highestTrustScore = agentTrustScore
	// 		}
	// 	}

	// 	// If no valid trust scores are found, abstain
	// 	if highestTrustScore <= 0 {
	// 		return common.Vote{IsVote: 0, VoterID: t2a.GetID(), VotedForID: uuid.Nil}
	// 	}

	// 	// Return a vote indicating the new leader
	// 	return common.Vote{IsVote: 1, VoterID: t2a.GetID(), VotedForID: highestAgent}

	// }
}
