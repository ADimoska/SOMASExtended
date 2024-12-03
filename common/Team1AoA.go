package common

import (
	"container/list"
	// "errors"
	"log"
	"math/rand"
	"sort"

	// "github.com/ADimoska/SOMASExtended/agents"
	// "github.com/MattSScott/basePlatformSOMAS/v2/pkg/server"
	"github.com/google/uuid"
)

type Team1AoA struct {
	auditResult map[uuid.UUID]*list.List
	ranking     map[uuid.UUID]int
	threshold   [5]int
	agentLQueue map[uuid.UUID]*LeakyQueue
}

// LeakyQueue represents a queue with a fixed capacity.
type LeakyQueue struct {
	data     []int
	sum 	 int
	capacity int
}

// NewLeakyQueue initializes a new LeakyQueue with the specified capacity.
func NewLeakyQueue(capacity int) *LeakyQueue {
	if capacity <= 0 {
		panic("capacity must be greater than 0")
	}
	return &LeakyQueue{
		data:     make([]int, 0, capacity),
		sum: 	  0,
		capacity: capacity,
	}
}

// Push adds an element to the queue.
// If the queue exceeds its capacity, the oldest element is removed.
func (q *LeakyQueue) Push(value int) {
	if len(q.data) >= q.capacity {
		q.sum -= q.data[0]
		q.data = q.data[1:] // Remove the oldest element
	}
	q.sum += value
	q.data = append(q.data, value) // Add the new element
}

func (q *LeakyQueue) Sum() int{
	return q.sum
}



func (t *Team1AoA) ResetAuditMap() {
	t.auditResult = make(map[uuid.UUID]*list.List)
}

// TODO: Add functionality for expected contribution to change based on rank
func (t *Team1AoA) GetExpectedContribution(agentId uuid.UUID, agentScore int) int {
	return 1 // For now using threshold as minimum for all ranks, later have per rank minimums? But need to vote what is min?
}

func (t *Team1AoA) SetContributionAuditResult(agentId uuid.UUID, agentScore int, agentActualContribution int, agentStatedContribution int) {
	t.auditResult[agentId].PushBack((agentStatedContribution > agentActualContribution))

	// Update The LeakyQueue of agent
	t.agentLQueue[agentId].Push(agentStatedContribution);
}

func (t *Team1AoA) GetExpectedWithdrawal(agentId uuid.UUID, agentScore int, commonPool int) int {
	totalWeightedSum := 0
	for _, rank := range t.ranking {
		totalWeightedSum += rank
	}
	expectedWithdrawal := t.ranking[agentId] * (commonPool / (totalWeightedSum + 5)) // Weight 5 for pool?
	return expectedWithdrawal
}

func (t *Team1AoA) SetWithdrawalAuditResult(agentId uuid.UUID, agentScore int, agentActualWithdrawal int, agentStatedWithdrawal int, commonPool int) {
	t.auditResult[agentId].PushBack((agentActualWithdrawal > agentStatedWithdrawal) || (agentActualWithdrawal > t.GetExpectedWithdrawal(agentId, agentScore, commonPool)))
}

func (t *Team1AoA) GetAuditCost(commonPool int) int {
	// Need to get argument which agent being audited and then change cost?
	return 5
}

func (t *Team1AoA) GetVoteResult(votes []Vote) uuid.UUID {
	// Count total votes
	totalVotes := 0
	voteMap := make(map[uuid.UUID]int)
	highestVotes := -1
	highestVotedID := uuid.Nil
	for _, vote := range votes {
		totalVotes += vote.IsVote
		if vote.IsVote == 1 { // Should agents who didnt want to vote, get a vote if majority wants to?
			voteMap[vote.VotedForID]++
		}
		// Check if this ID has the highest votes
		if voteMap[vote.VotedForID] > highestVotes {
			highestVotedID = vote.VotedForID
			highestVotes = voteMap[vote.VotedForID]
		}
	}
	if totalVotes <= 0 {
		return uuid.Nil // Majority does not want to vote
	}
	return highestVotedID
}

func (t *Team1AoA) GetContributionAuditResult(agentId uuid.UUID) bool {
	return t.auditResult[agentId].Back().Value.(int) == 1
}

func (t *Team1AoA) GetWithdrawalAuditResult(agentId uuid.UUID) bool {
	return t.auditResult[agentId].Back().Value.(int) == 1
}

func (t *Team1AoA) GetWithdrawalOrder(agentIDs []uuid.UUID) []uuid.UUID {
	// Sort the agent based on their rank value in descending order
	sort.Slice(agentIDs, func(i, j int) bool {
		return t.ranking[agentIDs[i]] > t.ranking[agentIDs[j]]
	})
	return agentIDs
}

// WeightedRandomSelection selects one agent based on weights derived from ranks.
func (t *Team1AoA) WeightedRandomSelection(agentIds []uuid.UUID) uuid.UUID {
    if len(agentIds) == 0 {
        log.Fatal("No agents to select from")
    }

    totalWeight := 0
    for _, agentId := range agentIds {
        totalWeight += t.ranking[agentId]
    }
    if totalWeight == 0 {
        log.Fatal("All agents have 0 weight")
    }
    
    randomNumber := rand.Intn(totalWeight) + 1
    cumulativeWeight := 0
    for _, agentId := range agentIds {
        cumulativeWeight += t.ranking[agentId]
        if cumulativeWeight >= randomNumber {
            return agentId
        }
    }
    
    log.Fatal("Failed to select an agent")
    return uuid.Nil // This line will never be reached due to log.Fatal
}

// SelectNChairs selects n distinct agents to be chairs, with probability of selection based on rank.
func (t *Team1AoA) SelectNChairs(agentIds []uuid.UUID, n int) []uuid.UUID {
    if len(agentIds) < n {
        log.Fatal("not enough agents to select from")
    }

    selectedChairs := make([]uuid.UUID, 0, n)
    remainingAgents := make([]uuid.UUID, len(agentIds))
    copy(remainingAgents, agentIds)

    for i := 0; i < n; i++ {
        agent := t.WeightedRandomSelection(remainingAgents)
        selectedChairs = append(selectedChairs, agent)

        // Remove the selected agent from remainingAgents
        // Find the index of the selected agent
        index := -1
        for j, id := range remainingAgents {
            if id == agent {
                index = j
                break
            }
        }

        if index == -1 {
            log.Fatal("selected agent not found in remainingAgents")
        }

        // Remove the agent by swapping with the last element and truncating the slice
        remainingAgents[index] = remainingAgents[len(remainingAgents)-1]
        remainingAgents = remainingAgents[:len(remainingAgents)-1]
    }

    return selectedChairs
}


func (t *Team1AoA) RunPostContributionAoaLogic(team *Team, agentMap map[uuid.UUID]IExtendedAgent) {
	// Choose 2 chairs based on rank
	// call function for agents to vote on ranks
	// If the chairs decision do not match, then reduce rank by 1 of their score and give to common pool
	// Then repeat until two agents agree

	
	for i:=0; i<10; i++ {
		chairsAgree := false
		// listOfAgentRankVotes := make([]map[uuid.UUID]int, 0)
		
		// call function for agents to vote on ranks 
		// for _, agentId := range team.Agents {
		// 	agent := agentMap[agentId]
		// 	ranks := agent.Team1_GetTeamRanks()
		// 	listOfAgentRankVotes = append(listOfAgentRankVotes, ranks)
		// }

		chairs := t.SelectNChairs(team.Agents, 2)
		var prev map[uuid.UUID]int
		for _, chairId := range chairs {
			chair := agentMap[chairId]
			current := chair.Team1_ChairUpdateRanks(t.ranking)
			if prev != nil {
				if !mapsEqual(prev, current) {
					// Reduce rank of both chairs by 1
					for _, id := range chairs {
						t.ranking[id]--
						if t.ranking[id] < 1 {
							t.ranking[id] = 1
						}
					}
					chairsAgree = false
					break				
				}
			} else {
				prev = current
			}
			chairsAgree = true
		}
		// Chairs agree
		if chairsAgree {
			break
		}
		
	}

	// get the pointers to chairs and call countVotes function to get the result

	return

}

func mapsEqual(a, b map[uuid.UUID]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func (t *Team1AoA) GetAgentNewRank(agentId uuid.UUID) int {
	agentTotalContributions := t.agentLQueue[agentId].Sum()  // total stated contributions of this agent (over the last n turns)

	agentCurrentRank := t.ranking[agentId]
	// iterate from the highest rank to the lowest rank
	// return the rank if the total contribution is greater than or equal to the threshold
	newRank := 0
	for rank := len(t.threshold) - 1; rank >= 0; rank-- {
		threshold := t.threshold[rank]
		if agentTotalContributions >= threshold {
			newRank = rank + 1
		}
	}
	// Speed Limit to climb rank
	if newRank > agentCurrentRank+1 {
		newRank = agentCurrentRank + 1
	} else if newRank < agentCurrentRank-1 {
		newRank = agentCurrentRank - 1
	}

	// log.fatal("Agent total contribution is less than the minimum threshold")
	return newRank // or an appropriate default value or error code
}

func CreateTeam1AoA(team *Team) IArticlesOfAssociation {
	auditResult := make(map[uuid.UUID]*list.List)
	ranking := make(map[uuid.UUID]int)
	agentLQueue := make(map[uuid.UUID]*LeakyQueue)
	for _, agent := range team.Agents {
		auditResult[agent] = list.New()
		ranking[agent] = 1
		agentLQueue[agent] = NewLeakyQueue(5)
	}

	return &Team1AoA{
		auditResult: auditResult,
		ranking:     ranking,
		threshold:   [5]int{10, 20, 30, 40, 50},
		agentLQueue: agentLQueue,
	}
}
