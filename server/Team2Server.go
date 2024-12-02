package environmentServer

import (
	"fmt"
	"github.com/google/uuid"
	"math/rand"
)

/*
 * In case the leader is caught cheating, they will be deposed,
 * this calls a vote for the leader among agents on the current team.
 */
func (cs *EnvironmentServer) DeposeLeader() {

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
