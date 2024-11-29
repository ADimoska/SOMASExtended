package gameRecorder

import (
	"fmt"
)

// --------- General External Functions ---------
func Log(message string) {
	fmt.Println(message)
}

type TurnRecord struct {
	TurnNumber      int
	IterationNumber int
	AgentRecords    []AgentRecord
	TeamRecords     []TeamRecord
}

// turn record constructor
func NewTurnRecord(turnNumber int, iterationNumber int) TurnRecord {
	return TurnRecord{
		TurnNumber:      turnNumber,
		IterationNumber: iterationNumber,
	}
}

// --------- Server Recording Functions ---------
type ServerDataRecorder struct {
	TurnRecords []TurnRecord // where all our info is stored!

	currentIteration int
	currentTurn      int
}

func (sdr *ServerDataRecorder) GetCurrentTurnRecord() *TurnRecord {
	return &sdr.TurnRecords[len(sdr.TurnRecords)-1]
}

func CreateRecorder() *ServerDataRecorder {
	return &ServerDataRecorder{
		TurnRecords:      []TurnRecord{},
		currentIteration: -1, // to start from 0
		currentTurn:      -1,
	}
}

func (sdr *ServerDataRecorder) RecordNewIteration() {
	sdr.currentIteration += 1
	sdr.currentTurn = 0

	// create new turn record
	sdr.TurnRecords = append(sdr.TurnRecords, NewTurnRecord(sdr.currentTurn, sdr.currentIteration))
}

// func (sdr *ServerDataRecorder) RecordNewTurn() {
// 	sdr.currentTurn += 1

// 	// create new turn record
// 	sdr.TurnRecords = append(sdr.TurnRecords, NewTurnRecord(sdr.currentTurn, sdr.currentIteration))
// }

func (sdr *ServerDataRecorder) RecordNewTurn(agentRecords []AgentRecord, teamRecords []TeamRecord) {
	sdr.currentTurn += 1
	sdr.TurnRecords = append(sdr.TurnRecords, NewTurnRecord(sdr.currentTurn, sdr.currentIteration))

	sdr.TurnRecords[len(sdr.TurnRecords)-1].AgentRecords = agentRecords
	sdr.TurnRecords[len(sdr.TurnRecords)-1].TeamRecords = teamRecords

	// for _, agent := range serv.GetAgentMap() {
	// 	sdr.TurnRecords[len(sdr.TurnRecords)-1].AgentRecords = append(sdr.TurnRecords[len(sdr.TurnRecords)-1].AgentRecords, NewAgentRecord(agent))
	// }

	// for _, agent := range serv.GetDeadAgents() {
	// 	sdr.TurnRecords[len(sdr.TurnRecords)-1].AgentRecords = append(sdr.TurnRecords[len(sdr.TurnRecords)-1].AgentRecords, NewAgentRecord(agent))
	// }

	// agentRecord := AgentRecord{
	// 	// TurnNumber:      serv.currentTurnNumber,
	// 	// IterationNumber: serv.currentIterationNumber,
	// 	// AgentID:         agent.AgentID,
	// 	// TrueSomasTeamID: 0, // TODO

	// 	// IsAlive:            !serv.IsAgentDead(agent.AgentID),
	// 	// Score:              agent.GetTrueScore(),
	// 	// Contribution:       agent.GetActualContribution(agent),
	// 	// StatedContribution: agent.GetStatedContribution(agent),
	// 	// Withdrawal:         agent.GetActualWithdrawal(agent),
	// 	// StatedWithdrawal:   agent.GetStatedWithdrawal(agent),

	// 	// TeamID: agent.teamID,

	// 	agent: agent,
	// }

	//sdr.TurnRecords[len(sdr.TurnRecords)-1].AgentRecords = append(sdr.TurnRecords[len(sdr.TurnRecords)-1].AgentRecords, agentRecord)
}

func (sdr *ServerDataRecorder) GamePlaybackSummary() {
	fmt.Printf("\n\nGamePlaybackSummary - playing %v turn records\n", len(sdr.TurnRecords))
	for _, turnRecord := range sdr.TurnRecords {
		fmt.Printf("\nIteration %v, Turn %v:\n", turnRecord.IterationNumber, turnRecord.TurnNumber)
		for _, agentRecord := range turnRecord.AgentRecords {
			agentRecord.DebugPrint()
		}
	}

	// Create the HTML visualization
	CreatePlaybackHTML(sdr)
}
