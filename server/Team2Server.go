package environmentServer

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/ADimoska/SOMASExtended/agents"
	"github.com/ADimoska/SOMASExtended/common"
	"github.com/google/uuid"
)

/*
 * In case the leader is caught cheating, they will be deposed,
 * this calls a vote for the leader among agents on the current team.
 * TODO: convert this to a borda vote maybe? ->
 * Would need ordered preference of votes
 */
func (cs *EnvironmentServer) DeposeLeader(teamId uuid.UUID) {
	agentsInTeam := cs.GetAgentsInTeam(teamId)
	if len(agentsInTeam) <= 0 {
		log.Fatal("Team can't have non-positive agent count")
	}

	votes := make(map[uuid.UUID]int)
	var maxVotes int
	var candidates []uuid.UUID

	for _, agentId := range agentsInTeam {
		agent := cs.GetAgentMap()[agentId]
		leaderVote := agent.(*agents.Team2Agent).GetLeaderVote()
		votedFor := leaderVote.VotedForID

		votes[votedFor]++
		voteCount := votes[votedFor]

		// In case of new maximum, create a new tie-break array
		if voteCount > maxVotes {
			maxVotes = voteCount
			candidates = []uuid.UUID{votedFor}
		} else if voteCount == maxVotes {
			candidates = append(candidates, votedFor)
			// Update the old tie-break array
		}
	}

	// Handle tie by selecting randomly
	var selectedLeader uuid.UUID
	if len(candidates) > 1 {
		selectedLeader = candidates[rand.Intn(len(candidates))]
	} else if len(candidates) == 1 {
		selectedLeader = candidates[0]
	}

	if len(candidates) == 0 {
		log.Fatal("No candidate selected!")
	}

	team := cs.Teams[teamId]
	team.TeamAoA.(*common.Team2AoA).SetLeader(selectedLeader)
}

/*
 * For the leader to override what a punished agent is rolling at that point
 */
func (cs *EnvironmentServer) OverrideAgentRolls(agentId uuid.UUID, leaderId uuid.UUID) {
	controlled := cs.GetAgentMap()[agentId]
	leader := cs.GetAgentMap()[leaderId]
	currentScore, accumulatedScore := controlled.GetTrueScore(), 0
	prevRoll := -1
	rounds := 0

	rollingComplete := false

	for !rollingComplete {
		stickDecision := leader.StickOrAgainFor(agentId, accumulatedScore, prevRoll)
		if stickDecision > 0 {
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
	score := 0
	for i := 0; i < 3; i++ {
		score += rand.Intn(6) + 1
	}
	return score
}
