package agents

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"

	"gonum.org/v1/gonum/mat"

	common "github.com/ADimoska/SOMASExtended/common"

	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
)

type Team3Agent struct {
	*ExtendedAgent

	RollHistory     []int          // Memory to store the results of each roll
	Bust            bool           // New field to track if the agent went bust
	TrainingHistory []TrainingData // New field to store training data for ML

	// Neural Network weights
	inputWeights  *mat.Dense
	outputWeights *mat.Dense
	hiddenBias    *mat.Dense // Add bias terms
	outputBias    *mat.Dense // Add bias terms
	learningRate  float64

	contributionLies    map[uuid.UUID]int
	withdrawalLies      map[uuid.UUID]int
	numberOfLies        map[uuid.UUID]int
	invitationResponses map[uuid.UUID]bool // true if accepted, false if rejected
	invitationsSent     map[uuid.UUID]bool // track if we've sent an invitation

	// Neural Network for cheating
	cheatNN              *NeuralNetwork
	cheatHistory         []CheatRecord
	lastCheatProbability float64
	wasAudited           bool
	wasCaughtCheating    bool
	cheatSuccessRate     float64
	totalAudits          int
	successfulCheats     int
}

// Structure to store weights
type NetworkWeights struct {
	InputWeights  [][]float64
	OutputWeights [][]float64
	HiddenBias    [][]float64
	OutputBias    [][]float64
}

// Save weights to file
func saveWeights(filename string, team3 *Team3Agent) error {
	// Convert matrices to slice format for JSON
	weights := NetworkWeights{
		InputWeights:  matrixToSlice(team3.inputWeights),
		OutputWeights: matrixToSlice(team3.outputWeights),
		HiddenBias:    matrixToSlice(team3.hiddenBias),
		OutputBias:    matrixToSlice(team3.outputBias),
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(weights, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(filename, data, 0644)
}

// Load weights from file
func loadWeights(filename string) (*NetworkWeights, error) {
	// Read file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON
	var weights NetworkWeights
	if err := json.Unmarshal(data, &weights); err != nil {
		return nil, err
	}

	return &weights, nil
}

// Helper function to convert mat.Dense to [][]float64
func matrixToSlice(m *mat.Dense) [][]float64 {
	rows, cols := m.Dims()
	slice := make([][]float64, rows)
	for i := 0; i < rows; i++ {
		slice[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			slice[i][j] = m.At(i, j)
		}
	}
	return slice
}

// Helper function to convert [][]float64 to mat.Dense
func sliceToMatrix(slice [][]float64) *mat.Dense {
	rows := len(slice)
	cols := len(slice[0])
	data := make([]float64, rows*cols)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			data[i*cols+j] = slice[i][j]
		}
	}
	return mat.NewDense(rows, cols, data)
}

// constructor for Team3Agent
func Team3_CreateAgent(funcs agent.IExposedServerFunctions[common.IExtendedAgent], agentConfig AgentConfig) *Team3Agent {
	team3 := &Team3Agent{

		ExtendedAgent:       GetBaseAgents(funcs, agentConfig),
		RollHistory:         []int{},
		TrainingHistory:     []TrainingData{},
		learningRate:        0.5,
		contributionLies:    make(map[uuid.UUID]int),
		withdrawalLies:      make(map[uuid.UUID]int),
		numberOfLies:        make(map[uuid.UUID]int),
		invitationResponses: make(map[uuid.UUID]bool),
		invitationsSent:     make(map[uuid.UUID]bool),
		cheatNN:             NewNeuralNetwork(4, 3), // 4 inputs, 3 hidden neurons
		cheatHistory:        make([]CheatRecord, 0),
	}
	team3.initializeNeuralNetworkStickRoll()

	team3.TrueSomasTeamID = 3 // IMPORTANT: add your team number here!
	return team3
}

// Neural Network initialization
func (team3 *Team3Agent) initializeNeuralNetworkStickRoll() {
	weightsFile := "neural_weights_stick_roll.json"

	// Try to load existing weights
	if weights, err := loadWeights(weightsFile); err == nil {
		// Use existing weights
		team3.inputWeights = sliceToMatrix(weights.InputWeights)
		team3.outputWeights = sliceToMatrix(weights.OutputWeights)
		team3.hiddenBias = sliceToMatrix(weights.HiddenBias)
		team3.outputBias = sliceToMatrix(weights.OutputBias)
	} else {
		// Initialize new weights if file doesn't exist
		team3.inputWeights = mat.NewDense(2, 36, nil)
		team3.outputWeights = mat.NewDense(36, 1, nil)
		team3.hiddenBias = mat.NewDense(1, 36, nil)
		team3.outputBias = mat.NewDense(1, 1, nil)

		// Random initialization
		for i := 0; i < 2; i++ {
			for j := 0; j < 36; j++ {
				team3.inputWeights.Set(i, j, rand.Float64()*2-1)
			}
		}
		for i := 0; i < 36; i++ {
			team3.outputWeights.Set(i, 0, rand.Float64()*2-1)
			team3.hiddenBias.Set(0, i, rand.Float64()*2-1)
		}
		team3.outputBias.Set(0, 0, rand.Float64()*2-1)
	}
}

// Sigmoid activation function
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// Neural network forward pass
func (team3 *Team3Agent) MLModelPredictStickRoll(currentScore, previousRoll int) bool {
	// Normalize inputs
	normalizedScore := float64(currentScore) / 100.0
	if normalizedScore > 1.0 {
		normalizedScore = 1.0
	}
	normalizedRoll := float64(previousRoll) / 18.0 // Three dice have max value of 18

	// Create input matrix
	input := mat.NewDense(1, 2, []float64{normalizedScore, normalizedRoll})

	// Hidden layer with bias
	hidden := mat.NewDense(1, 36, nil)
	hidden.Mul(input, team3.inputWeights)
	for i := 0; i < 36; i++ {
		val := hidden.At(0, i) + team3.hiddenBias.At(0, i)
		hidden.Set(0, i, sigmoid(val))
	}

	// Output layer with bias
	output := mat.NewDense(1, 1, nil)
	output.Mul(hidden, team3.outputWeights)
	finalOutput := sigmoid(output.At(0, 0) + team3.outputBias.At(0, 0))

	return finalOutput > 0.5
}

// ----------------------- Strategies -----------------------
// Team-forming Strategy
// func (team3 *Team3Agent) DecideTeamForming(agentInfoList []common.ExposedAgentInfo) []uuid.UUID {
// 	log.Printf("DecideTeamForming called for agent %s\n", team3.GetID())
// 	invitationList := []uuid.UUID{}
// 	for _, agentInfo := range agentInfoList {
// 		// Exclude the agent itself
// 		if agentInfo.AgentUUID == team3.GetID() {
// 			continue
// 		}
// 		// Check if the agent is not already in a team
// 		if agentInfo.AgentTeamID == (uuid.UUID{}) {
// 			invitationList = append(invitationList, agentInfo.AgentUUID)
// 		}
// 	}

// 	if len(invitationList) > 0 {
// 		// Randomly choose an agent to invite
// 		rand.Shuffle(len(invitationList), func(i, j int) {
// 			invitationList[i], invitationList[j] = invitationList[j], invitationList[i]
// 		})
// 		chosenAgent := invitationList[0]

// 		// Send the invitation
// 		team3.SendTeamFormingInvitation([]uuid.UUID{chosenAgent})

// 		// Log the chosen agent
// 		log.Printf("Agent %s sent invitation to %s\n", team3.GetID(), chosenAgent)

// 		// Return a slice containing the chosen agent
// 		return []uuid.UUID{chosenAgent}
// 	}

// 	log.Printf("No available agents to invite for agent %s\n", team3.GetID())
// 	return []uuid.UUID{}
// }

// Dice Strategy
func (team3 *Team3Agent) StickOrAgain(turn int, score int) bool {

	// Get the current score and previous roll
	currentScore := team3.Score
	previousRoll := 0
	if len(team3.RollHistory) > 0 {
		previousRoll = team3.RollHistory[len(team3.RollHistory)-1] // Get the last roll
	}

	// Use the ML model to decide whether to stick or roll again
	shouldStick := team3.MLModelPredictStickRoll(currentScore, previousRoll)

	// Store the result for retraining
	team3.StoreTrainingData(currentScore, previousRoll, shouldStick)

	// Debugging output for decision
	if shouldStick {
		fmt.Println("Decision: Stick")
	} else {
		fmt.Println("Decision: Roll Again")
	}

	return shouldStick
}

// Store training data for retraining the model
func (team3 *Team3Agent) StoreTrainingData(currentScore, previousRoll int, shouldStick bool) {
	trainingData := TrainingData{
		CurrentScore: currentScore,
		PreviousRoll: previousRoll,
		ShouldStick:  shouldStick,
		Reward:       0, // Will be updated after the turn
	}
	team3.TrainingHistory = append(team3.TrainingHistory, trainingData)
}

// Struct to hold training data
type TrainingData struct {
	CurrentScore int
	PreviousRoll int
	ShouldStick  bool
	Reward       float64 // Add reward field
}

// StartRollingDice custom function: ask for rolling the dice
func (team3 *Team3Agent) StartRollingDice(instance common.IExtendedAgent) {
	if team3.VerboseLevel > 10 {
		fmt.Printf("%s is rolling the Dice\n", team3.GetID())
	}
	if team3.VerboseLevel > 9 {
		fmt.Println("---------------------")
	}
	team3.LastScore = -1
	rounds := 1
	turnScore := 0

	willStick := false
	team3.Bust = false // Initialize Bust status

	// Loop until not stick
	for !willStick {
		// Simulate rolling the dice
		currentScore := Roll3Dice()

		// Add this roll to history
		team3.RollHistory = append(team3.RollHistory, currentScore)

		// Check if currentScore is higher than lastScore
		if currentScore > team3.LastScore {
			turnScore += currentScore
			team3.LastScore = currentScore
			willStick = team3.StickOrAgain(0, 0)

			if willStick {
				team3.DecideStick()
				break
			}
			team3.DecideRollAgain()
		} else {
			team3.Bust = true // Set bust status
			if team3.VerboseLevel > 4 {
				fmt.Printf("%s *BURSTED!* round: %v, current score: %v\n", team3.GetID(), rounds, currentScore)
			}
			turnScore = 0
			break
		}

		rounds++
	}

	// Add turn score to total score
	team3.Score += turnScore

	if team3.VerboseLevel > 4 {
		fmt.Printf("%s's turn score: %v, total score: %v\n", team3.GetID(), turnScore, team3.Score)
	}

	// After the turn is complete, calculate reward and train the model
	reward := 0.0
	if team3.Bust {
		// Get the score before busting
		lastScore := 0
		if len(team3.RollHistory) >= 2 {
			lastScore = team3.RollHistory[len(team3.RollHistory)-2]
		}
		// Penalize busting, but less if close to 20
		bustPenalty := -1.0 - float64(lastScore)/40.0 // Reduced penalty for higher scores
		reward = bustPenalty
	} else {
		// Reward for scores, with a focus on exceeding 20
		if team3.Score >= 20 {
			reward = 10.0 + float64(team3.Score-20)*0.5 // High reward for 20+ and extra for exceeding
		} else {
			reward = float64(team3.Score) / 20.0 // Proportional reward for scores below 20
		}
	}

	// Update the reward for the last training data entry
	if len(team3.TrainingHistory) > 0 {
		lastIdx := len(team3.TrainingHistory) - 1
		team3.TrainingHistory[lastIdx].Reward = reward
	}

	// Train the model with the latest data
	team3.trainModelStickRoll()
}

// Train the neural network
func (team3 *Team3Agent) trainModelStickRoll() {
	if len(team3.TrainingHistory) == 0 {
		return
	}

	// Use mini-batch of recent experiences (last 5 turns or all if less)
	batchSize := 5
	startIdx := math.Max(0, float64(len(team3.TrainingHistory)-batchSize))
	batch := team3.TrainingHistory[int(startIdx):]

	for _, example := range batch {
		// Normalize inputs
		normalizedScore := float64(example.CurrentScore) / 100.0
		normalizedRoll := float64(example.PreviousRoll) / 6.0

		input := mat.NewDense(1, 2, []float64{normalizedScore, normalizedRoll})

		// Forward pass with bias
		hidden := mat.NewDense(1, 36, nil)
		hidden.Mul(input, team3.inputWeights)
		hiddenActivated := mat.NewDense(1, 36, nil)
		for i := 0; i < 36; i++ {
			hiddenActivated.Set(0, i, sigmoid(hidden.At(0, i)+team3.hiddenBias.At(0, i)))
		}

		output := mat.NewDense(1, 1, nil)
		output.Mul(hiddenActivated, team3.outputWeights)
		prediction := sigmoid(output.At(0, 0) + team3.outputBias.At(0, 0))

		// Calculate target (normalize reward to [0,1] range)
		target := sigmoid(example.Reward / 20.0)
		error := target - prediction

		// Backpropagation
		delta := error * prediction * (1 - prediction)

		// Update weights and biases
		for i := 0; i < 36; i++ {
			// Update input weights
			for j := 0; j < 2; j++ {
				inputGradient := delta * hiddenActivated.At(0, i) * input.At(0, j)
				currentWeight := team3.inputWeights.At(j, i)
				team3.inputWeights.Set(j, i, currentWeight+team3.learningRate*inputGradient)
			}

			// Update hidden bias
			hiddenBiasGradient := delta * hiddenActivated.At(0, i)
			currentHiddenBias := team3.hiddenBias.At(0, i)
			team3.hiddenBias.Set(0, i, currentHiddenBias+team3.learningRate*hiddenBiasGradient)

			// Update output weights
			outputGradient := delta * hiddenActivated.At(0, i)
			currentWeight := team3.outputWeights.At(i, 0)
			team3.outputWeights.Set(i, 0, currentWeight+team3.learningRate*outputGradient)
		}

		// Update output bias
		outputBiasGradient := delta
		currentOutputBias := team3.outputBias.At(0, 0)
		team3.outputBias.Set(0, 0, currentOutputBias+team3.learningRate*outputBiasGradient)
	}

	// Save weights after training
	if err := saveWeights("neural_weights_stick_roll.json", team3); err != nil {
		fmt.Printf("Error saving weights: %v\n", err)
	}
}

// !!! NOTE: name and signature of functions below are subject to change by the infra team !!!

// Contribution Strategy
func (team3 *Team3Agent) DecideContribution() int {
	// Use the GetActualContribution method to determine the contribution
	actualContribution := team3.GetActualContribution(team3)
	return actualContribution
}

// Withdrawal Strategy
func (team3 *Team3Agent) DecideWithdrawal() int {
	// TODO: implement contribution strategy
	return 10
}

// Audit Strategy
func (team3 *Team3Agent) DecideAudit() bool {
	// TODO: implement audit strategy
	return true
}

// Punishment Strategy
func (team3 *Team3Agent) DecidePunishment() int {
	// TODO: implement punishment strategy
	return 1
}

// ----------------------- State Helpers -----------------------
// TODO: add helper functions for managing / using internal states

// ----------------------- Memory Management -----------------------

// Update agent memory with lies about contributions - only when auditing
func (team3 *Team3Agent) UpdateContributionLies(agentID uuid.UUID) {
	if !team3.HasTeam() || !team3.DecideAudit() {
		return
	}

	log.Printf("DEBUG [AUDIT START]: Agent %s is auditing Agent %s\n",
		team3.GetID(), agentID)

	expectedContribution := team3.Server.GetTeam(agentID).TeamAoA.GetExpectedContribution(agentID, team3.GetTrueScore())
	actualContribution := team3.Server.AccessAgentByID(agentID).GetActualContribution(team3)

	// Record lie only when actual is less than expected (agent contributed less than they should)
	if actualContribution < expectedContribution {
		lieAmount := expectedContribution - actualContribution
		team3.contributionLies[agentID] = lieAmount
		team3.numberOfLies[agentID]++

		log.Printf("DEBUG [AUDIT]: Agent %s LIED on contribution! Expected: %d, Actual: %d, Lie Amount: %d\n",
			agentID, expectedContribution, actualContribution, lieAmount)
	} else {
		// Reward honest contribution in memory
		currentScore := team3.GetAgentMemoryScore(agentID)
		newScore := currentScore + 5
		if newScore > 100 {
			newScore = 100 // Cap at 100
		}

		// Update memory score (you might need to add a method to set memory score)
		team3.SetAgentMemoryScore(agentID, newScore)

		log.Printf("DEBUG [AUDIT]: Agent %s honest on contribution. Memory score increased by 5 to %d\n",
			agentID, newScore)
	}
}

// Add this helper method to set memory scores
func (team3 *Team3Agent) SetAgentMemoryScore(agentID uuid.UUID, score int) {
	// Implement based on your memory system
	// This might involve updating multiple fields that affect the memory score

	// For example, you might want to reduce the number of recorded lies
	if score > team3.GetAgentMemoryScore(agentID) {
		team3.contributionLies[agentID] = 0
		team3.withdrawalLies[agentID] = 0
		team3.numberOfLies[agentID] = 0
	}
}

// Update agent memory with lies about withdrawals - only when auditing
func (team3 *Team3Agent) UpdateWithdrawalLies(agentID uuid.UUID) {
	if !team3.HasTeam() || !team3.DecideAudit() {
		return // Only proceed if we have a team and decide to audit
	}

	log.Printf("DEBUG [AUDIT START]: Agent %s is auditing Agent %s\n",
		team3.GetID(), agentID)

	commonPool := team3.Server.GetTeam(agentID).GetCommonPool()
	expectedWithdrawal := team3.Server.GetTeam(agentID).TeamAoA.GetExpectedWithdrawal(agentID, team3.GetTrueScore(), commonPool)
	actualWithdrawal := team3.Server.AccessAgentByID(agentID).GetActualWithdrawal(team3)

	// Record lie only when actual is more than expected (agent withdrew more than allowed)
	if actualWithdrawal > expectedWithdrawal {
		lieAmount := actualWithdrawal - expectedWithdrawal
		team3.withdrawalLies[agentID] = lieAmount
		team3.numberOfLies[agentID]++

		log.Printf("DEBUG [AUDIT]: Agent %s LIED on withdrawal! Expected: %d, Actual: %d, Lie Amount: %d\n",
			agentID, expectedWithdrawal, actualWithdrawal, lieAmount)
	} else {
		log.Printf("DEBUG [AUDIT]: Agent %s honest on withdrawal. Expected: %d, Actual: %d\n",
			agentID, expectedWithdrawal, actualWithdrawal)
	}
}

// DEBUGGING bellow
// Add this debug function to print lies
func (team3 *Team3Agent) PrintLies() {
	if team3.HasTeam() {
		log.Printf("=== Agent %s Lie Detection Report ===\n", team3.GetID())

		for agentID := range team3.contributionLies {
			totalLies := team3.GetTotalLies(agentID)
			numberOfLies := team3.numberOfLies[agentID]
			log.Printf("Agent %s TOTAL lie amount: %d, Number of Lies: %d\n", agentID, totalLies, numberOfLies)
		}

		log.Printf("=====================================\n")
	}
}

// Add this to HandleContributionMessage
func (team3 *Team3Agent) HandleContributionMessage(msg *common.ContributionMessage) {
	team3.UpdateContributionLies(msg.GetSender())
	team3.PrintLies()         // Print after updating contribution lies
	team3.PrintMemoryReport() // Add memory report after each contribution
}

// Add this to HandleWithdrawalMessage
func (team3 *Team3Agent) HandleWithdrawalMessage(msg *common.WithdrawalMessage) {
	team3.UpdateWithdrawalLies(msg.GetSender())
	team3.PrintLies()         // Print after updating withdrawal lies
	team3.PrintMemoryReport() // Add memory report after each withdrawal
}

// Add this to HandleTeamFormationResponse
func (team3 *Team3Agent) HandleTeamFormationResponse(senderID uuid.UUID, accepted bool) {
	team3.invitationResponses[senderID] = accepted

	// If they accepted and we're now in the same team, update our records
	if accepted && team3.GetTeamID() == team3.Server.AccessAgentByID(senderID).GetTeamID() {
		team3.invitationResponses[senderID] = true
		log.Printf("DEBUG: Confirmed team formation - Agent %s and Agent %s are now teammates\n",
			team3.GetID(), senderID)
	}

	log.Printf("DEBUG: Invitation tracking state for Agent %s:", team3.GetID())
	log.Printf("- invitationsSent: %v", team3.invitationsSent)
	log.Printf("- invitationResponses: %v", team3.invitationResponses)

	team3.PrintLikeabilityStatus()
	team3.PrintMemoryReport()
}

//DEBUGGING above

func (team3 *Team3Agent) GetTotalLies(agentID uuid.UUID) int {
	contributionLie := team3.contributionLies[agentID]
	withdrawalLie := team3.withdrawalLies[agentID]
	return contributionLie + withdrawalLie
}

// ----------------------- Team Formation, checks if an agent likes you or not -----------------------

// Track when you send invitations
func (team3 *Team3Agent) SendTeamFormingInvitation(agentIDs []uuid.UUID) {
	for _, agentID := range agentIDs {
		log.Printf("Sending team formation invitation from %s to %s\n", team3.GetID(), agentID)
		// Implement the logic to send an invitation
		// This might involve updating a map or sending a message
		team3.invitationsSent[agentID] = true
	}
}

// Check if an agent likes you
func (team3 *Team3Agent) DoesAgentLikeMe(agentID uuid.UUID) bool {
	if response, exists := team3.invitationResponses[agentID]; exists {
		return response
	}
	return false // If we haven't interacted, assume neutral/false
}

// Print the current state of who likes/dislikes you
func (team3 *Team3Agent) PrintLikeabilityStatus() {
	log.Printf("=== Agent %s Likeability Report ===\n", team3.GetID())
	for agentID, response := range team3.invitationResponses {
		if response {
			log.Printf("Agent %s LIKES us\n", agentID)
		} else {
			log.Printf("Agent %s DISLIKES us\n", agentID)
		}
	}
	log.Printf("=====================================\n")
}

//------overall Memory Score ----------------------------------------------------------------

// Simplified GetAgentMemoryScore to deduct points for rejection without checking team status
func (team3 *Team3Agent) GetAgentMemoryScore(agentID uuid.UUID) int {
	// Start with base score of 100
	score := 100
	//log.Printf("DEBUG: Initial score for Agent %s: %d\n", agentID, score)

	// Calculate lie impact
	totalLieAmount := team3.contributionLies[agentID] + team3.withdrawalLies[agentID]
	numberOfLies := team3.numberOfLies[agentID]

	// Calculate lie penalty: multiply total amount by number of occurrences
	liePenalty := totalLieAmount * numberOfLies
	if liePenalty > 50 {
		liePenalty = 50
	}
	score -= liePenalty
	//log.Printf("DEBUG: Score after lie penalty for Agent %s: %d (penalty: %d)\n", agentID, score, liePenalty)

	// Check invitation response
	if response, exists := team3.invitationResponses[agentID]; exists {
		if !response {
			score -= 50
			log.Printf("DEBUG: Deducting 50 points from Agent %s's score for rejection. New score: %d\n", agentID, score)
		}
	} else {
		//log.Printf("DEBUG: No invitation response recorded for Agent %s\n", agentID)
	}

	// Ensure score stays within 0-100 range
	if score < 0 {
		score = 0
	}
	//log.Printf("DEBUG: Final memory score for Agent %s: %d\n", agentID, score)

	return score
}

// Print comprehensive memory report
func (team3 *Team3Agent) PrintMemoryReport() {
	log.Printf("=== Agent %s Memory Report ===\n", team3.GetID())

	seenAgents := make(map[uuid.UUID]bool)

	// Add agents from all tracking maps
	for agentID := range team3.contributionLies {
		seenAgents[agentID] = true
	}
	for agentID := range team3.invitationsSent {
		seenAgents[agentID] = true
	}
	for agentID := range team3.invitationResponses {
		seenAgents[agentID] = true
	}

	// Print report for each agent
	for agentID := range seenAgents {
		// Check if the agent exists in the server
		agent := team3.Server.AccessAgentByID(agentID)
		if agent == nil {
			log.Printf("Agent %s is nil, skipping...\n", agentID)
			continue
		}

		score := team3.GetAgentMemoryScore(agentID)
		totalLies := team3.GetTotalLies(agentID)
		numberOfLies := team3.numberOfLies[agentID]

		log.Printf("Agent %s:\n", agentID)
		log.Printf("  - Memory Score: %d/100\n", score)
		log.Printf("  - Total Lie Amount: %d\n", totalLies)
		log.Printf("  - Number of Lies: %d\n", numberOfLies)

		// Check actual team status
		sameTeam := team3.HasTeam() &&
			team3.GetTeamID() == agent.GetTeamID()

		log.Printf("  - Team Formation Status:")
		if sameTeam {
			log.Printf("    * Currently teammates")
		} else {
			if response, exists := team3.invitationResponses[agentID]; exists {
				if response {
					log.Printf("    * Accepted our invitation")
				} else {
					log.Printf("    * Rejected our invitation")
				}
			} else if team3.invitationsSent[agentID] {
				log.Printf("    * Invitation sent, awaiting response")
			} else {
				log.Printf("    * No invitation history")
			}
		}
	}

	log.Printf("=====================================\n")
}

// // HandleTeamFormationMessage handles receiving team formation invitations
// func (team3 *Team3Agent) HandleTeamFormationMessage(msg *common.TeamFormationMessage) {
// 	senderID := msg.GetSender()

// 	// Check if we've interacted with this agent before
// 	if _, exists := team3.contributionLies[senderID]; exists {
// 		// We know this agent - check their memory score
// 		score := team3.GetAgentMemoryScore(senderID)
// 		shouldAccept := score > 50

// 		log.Printf("Agent %v received team invitation from known agent %v with memory score %d. Accepting: %v",
// 			team3.GetID(), senderID, score, shouldAccept)

// 		// Record response
// 		team3.invitationResponses[senderID] = shouldAccept

// 	} else {
// 		// New agent - accept invitation
// 		log.Printf("Agent %v received team invitation from unknown agent %v. Accepting by default.",
// 			team3.GetID(), senderID)

// 		team3.invitationResponses[senderID] = true
// 	}

// 	// Update tracking
// 	team3.invitationsSent[senderID] = true

// 	// Print debug information
// 	team3.PrintLikeabilityStatus()
// 	team3.PrintMemoryReport()
// }

// Add these new types
type NeuralNetwork struct {
	inputLayer  *mat.Dense
	hiddenLayer *mat.Dense
	outputLayer *mat.Dense
	weights1    *mat.Dense
	weights2    *mat.Dense
}

type CheatRecord struct {
	inputs     []float64
	outcome    bool
	wasAudited bool
	score      int
	commonPool int
}

// Add these new methods
func NewNeuralNetwork(inputSize, hiddenSize int) *NeuralNetwork {
	// Initialize weights with random values
	w1Data := make([]float64, inputSize*hiddenSize)
	w2Data := make([]float64, hiddenSize)

	for i := range w1Data {
		w1Data[i] = rand.Float64()*2 - 1
	}
	for i := range w2Data {
		w2Data[i] = rand.Float64()*2 - 1
	}

	return &NeuralNetwork{
		inputLayer:  mat.NewDense(1, inputSize, nil),
		hiddenLayer: mat.NewDense(1, hiddenSize, nil),
		outputLayer: mat.NewDense(1, 1, nil),
		weights1:    mat.NewDense(inputSize, hiddenSize, w1Data),
		weights2:    mat.NewDense(hiddenSize, 1, w2Data),
	}
}

// Add neural network methods
func (nn *NeuralNetwork) Forward(inputs []float64) float64 {
	nn.inputLayer.SetRow(0, inputs)

	nn.hiddenLayer.Mul(nn.inputLayer, nn.weights1)
	applyFunction(nn.hiddenLayer, sigmoid)

	nn.outputLayer.Mul(nn.hiddenLayer, nn.weights2)
	applyFunction(nn.outputLayer, sigmoid)

	return nn.outputLayer.At(0, 0)
}

func (team3 *Team3Agent) prepareCheatInputs() []float64 {
	inputs := make([]float64, 4)

	// Normalize inputs
	score := float64(team3.Score)
	giniIndex := calculateGiniIndex(team3) // Assume this function calculates the Gini index
	commonPool := float64(team3.Server.GetTeamCommonPool(team3.GetTeamID()))
	teamSize := float64(len(team3.Server.GetTeam(team3.GetID()).Agents))

	inputs[0] = score / 100.0              // Normalized score
	inputs[1] = giniIndex                  // Gini index
	inputs[2] = commonPool / (score * 2.0) // Pool relative to score
	inputs[3] = teamSize / 10.0            // Normalized team size

	return inputs
}

// Modify GetActualContribution to use neural network for cheating decisions
func (team3 *Team3Agent) GetActualContribution(instance common.IExtendedAgent) int {
	fmt.Println("Entering GetActualContribution")

	if !team3.HasTeam() {
		fmt.Println("No team, returning 0")
		return 0
	}

	expectedContribution := team3.Server.GetTeam(team3.GetID()).TeamAoA.GetExpectedContribution(team3.GetID(), team3.GetTrueScore())

	// Get cheat probability from neural network
	cheatInputs := team3.prepareCheatInputs()
	cheatProbability := team3.cheatNN.Forward(cheatInputs)
	team3.lastCheatProbability = cheatProbability

	// Record this decision
	team3.cheatHistory = append(team3.cheatHistory, CheatRecord{
		inputs:     cheatInputs,
		outcome:    false, // Will be updated after audit
		wasAudited: false,
		score:      team3.Score,
		commonPool: team3.Server.GetTeamCommonPool(team3.GetTeamID()),
	})

	// Decide whether to cheat based on probability
	if cheatProbability > 0.5 {
		// Cheat by contributing less
		actualContribution := int(float64(expectedContribution) * (0.5 + (rand.Float64() * 0.3)))
		if actualContribution > team3.Score {
			actualContribution = team3.Score
		}
		fmt.Printf("Agent %s decided to CHEAT. Actual Contribution: %d\n", team3.GetID(), actualContribution)
		return actualContribution
	}

	// Honest contribution
	if team3.Score < expectedContribution {
		fmt.Printf("Agent %s decided to be HONEST. Actual Contribution: %d\n", team3.GetID(), team3.Score)
		return team3.Score
	}
	fmt.Printf("Agent %s decided to be HONEST. Actual Contribution: %d\n", team3.GetID(), expectedContribution)
	return expectedContribution
}

// Add this to handle audit results
func (team3 *Team3Agent) SetAgentContributionAuditResult(agentID uuid.UUID, result bool) {
	if agentID == team3.GetID() {
		team3.wasAudited = true
		team3.wasCaughtCheating = !result
		team3.totalAudits++

		if !team3.wasCaughtCheating && team3.lastCheatProbability > 0.5 {
			team3.successfulCheats++
		}
		team3.cheatSuccessRate = float64(team3.successfulCheats) / float64(team3.totalAudits)

		fmt.Println("---------------------")
		fmt.Printf("**AUDIT RESULTS**\n")
		fmt.Printf("***Agent ID: %v\n", team3.GetID())
		fmt.Printf("***Last Cheat Probability: %.2f\n", team3.lastCheatProbability)
		fmt.Printf("***Was Caught: %v\n", team3.wasCaughtCheating)
		fmt.Printf("***Total Successful Cheats: %d\n", team3.successfulCheats)
		fmt.Printf("***Total Audits: %d\n", team3.totalAudits)
		fmt.Printf("***Overall Success Rate: %.2f%%\n", team3.cheatSuccessRate*100)

		if len(team3.cheatHistory) > 0 {
			lastIdx := len(team3.cheatHistory) - 1
			team3.cheatHistory[lastIdx].wasAudited = true
			team3.cheatHistory[lastIdx].outcome = result
		}
	}
}

// Add these helper functions
func applyFunction(m *mat.Dense, fn func(float64) float64) {
	rows, cols := m.Dims()
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			m.Set(i, j, fn(m.At(i, j)))
		}
	}
}

// Add these new types
type CheatNeuralNetwork struct {
	inputLayer  *mat.Dense
	hiddenLayer *mat.Dense
	outputLayer *mat.Dense
	weights1    *mat.Dense
	weights2    *mat.Dense
}

// Add these new methods
func NewCheatNeuralNetwork(inputSize, hiddenSize int) *CheatNeuralNetwork {
	// Initialize weights with random values
	w1Data := make([]float64, inputSize*hiddenSize)
	w2Data := make([]float64, hiddenSize)

	for i := range w1Data {
		w1Data[i] = rand.Float64()*2 - 1
	}
	for i := range w2Data {
		w2Data[i] = rand.Float64()*2 - 1
	}

	return &CheatNeuralNetwork{
		inputLayer:  mat.NewDense(1, inputSize, nil),
		hiddenLayer: mat.NewDense(1, hiddenSize, nil),
		outputLayer: mat.NewDense(1, 1, nil),
		weights1:    mat.NewDense(inputSize, hiddenSize, w1Data),
		weights2:    mat.NewDense(hiddenSize, 1, w2Data),
	}
}

// Add neural network methods
func (nn *CheatNeuralNetwork) Forward(inputs []float64) float64 {
	nn.inputLayer.SetRow(0, inputs)

	nn.hiddenLayer.Mul(nn.inputLayer, nn.weights1)
	applyFunction(nn.hiddenLayer, sigmoid)

	nn.outputLayer.Mul(nn.hiddenLayer, nn.weights2)
	applyFunction(nn.outputLayer, sigmoid)

	return nn.outputLayer.At(0, 0)
}

func calculateGiniIndex(team3 *Team3Agent) float64 {
	team := team3.Server.GetTeam(team3.GetID())
	if team == nil || len(team.Agents) < 2 {
		return 0.0
	}

	scores := make([]float64, len(team.Agents))
	for i, agentID := range team.Agents {
		scores[i] = float64(team3.Server.AccessAgentByID(agentID).GetTrueScore())
	}

	mean := 0.0
	for _, score := range scores {
		mean += score
	}
	mean /= float64(len(scores))

	sumDiffs := 0.0
	for i := range scores {
		for j := range scores {
			sumDiffs += math.Abs(scores[i] - scores[j])
		}
	}

	return sumDiffs / (2 * float64(len(scores)) * mean)
}
