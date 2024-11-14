package baseDiceAgent

import (
	common "SOMASExtended/BaseDiceGame/common"
	rand "math/rand"

	baseAgent "github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	uuid "github.com/google/uuid"
)

type TurnRecord struct {
	RollResults  []int // Individual roll results
	TotalScore   int   // Final score for the turn
	Stuck        bool  // Whether the agent stuck or bust
	Bust         bool  // Whether the agent bust
	Contribution int   // How much they contributed to pool
}

type Report struct {
	AgentID     uuid.UUID
	TurnHistory []TurnRecord
	CommonPool  int
}

type BaseDiceAgent struct {
	*baseAgent.BaseAgent[IBaseDiceAgent]
	team   common.Team
	score  int
	memory map[uuid.UUID][]TurnRecord
}

type IBaseDiceAgent interface {
	baseAgent.IAgent[IBaseDiceAgent]
	RollDice(IBaseDiceAgent)
	MakeContribution() int
	BroadcastReport(int) []Report
	ProposeAudit() bool
	VoteForAudit() uuid.UUID
	ProposeAoAChange() bool
	VoteForNewAoA() int
	DoIStick(int, int) bool
}

func NewBaseDiceAgent(id uuid.UUID, team common.Team) *BaseDiceAgent {
	return &BaseDiceAgent{
		BaseAgent: baseAgent.NewBaseAgent[IBaseDiceAgent](id),
		team:      team,
		score:     0,
		memory:    make(map[uuid.UUID][]TurnRecord),
	}
}

func (agent *BaseDiceAgent) RollDice(specificAgent IBaseDiceAgent) {
	prev := 0
	total := 0
	stick := false
	bust := false
	rollResults := []int{}

	for !stick && !bust {
		r1, r2, r3 := (rand.Intn(6) + 1), (rand.Intn(6) + 1), (rand.Intn(6) + 1)
		score := r1 + r2 + r3

		if score > prev {
			total += score
			prev = score
			rollResults = append(rollResults, score)
			stick = specificAgent.DoIStick(total, prev)
		} else {
			bust = true
			score = 0
			rollResults = append(rollResults, score)
		}
	}

	// Create turn record
	turnRecord := TurnRecord{
		RollResults:  rollResults,
		TotalScore:   total,
		Stuck:        stick,
		Bust:         bust,
		Contribution: 0, // Will be updated when MakeContribution is called
	}

	// Store the turn record in memory
	agent.memory[agent.GetID()] = append(agent.memory[agent.GetID()], turnRecord)
	agent.score = total
}

func (agent *BaseDiceAgent) MakeContribution() int {
	currentScore := agent.score
	proposedContribution := agent.team.GetStrategy()

	validContribution := proposedContribution
	if proposedContribution > currentScore {
		validContribution = currentScore
	} else if proposedContribution < 0 {
		validContribution = 0
	}

	// Update the contribution in the latest turn record
	if len(agent.memory[agent.GetID()]) > 0 {
		lastIdx := len(agent.memory[agent.GetID()]) - 1
		agent.memory[agent.GetID()][lastIdx].Contribution = validContribution
	}

	agent.team.AddToPool(validContribution)
	return validContribution
}

func (agent *BaseDiceAgent) BroadcastReport(commonPool int) []Report {
	report := Report{
		AgentID:     agent.GetID(),
		TurnHistory: agent.memory[agent.GetID()],
		CommonPool:  commonPool,
	}

	return []Report{report}
}

func (agent *BaseDiceAgent) ProposeAudit() bool {
	return true
}

func (agent *BaseDiceAgent) VoteForAudit(vote bool) bool {
	// Simply returns the user's vote choice (true for yes, false for no)
	return vote
}

func ProposeAoAChange() bool {
	return true
}

func VoteForNewAoA() int {
	return 0
}
