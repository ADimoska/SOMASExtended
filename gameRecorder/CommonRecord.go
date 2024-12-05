package gameRecorder

// "SOMAS_Extended/common"

// AgentRecord is a record of an agent's state at a given turn
type CommonRecord struct {
	// basic info fields
	TurnNumber      int
	IterationNumber int

	Threshold              int  // current threshold set by server
	ThresholdAppliedInTurn bool // whether the threshold was applied in the current turn
}

func NewCommonRecord(turnNumber int, iterationNumber int, threshold int, thresholdAppliedInTurn bool) CommonRecord {
	return CommonRecord{
		TurnNumber:             turnNumber,
		IterationNumber:        iterationNumber,
		Threshold:              threshold,
		ThresholdAppliedInTurn: thresholdAppliedInTurn,
	}
}
