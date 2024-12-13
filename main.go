package main

import (
	"flag"
	"io"
	"log"
	"os"
	"time"

	"github.com/google/uuid"

	baseServer "github.com/MattSScott/basePlatformSOMAS/v2/pkg/server"

	agents "github.com/ADimoska/SOMASExtended/agents"
	common "github.com/ADimoska/SOMASExtended/common"
	gameRecorder "github.com/ADimoska/SOMASExtended/gameRecorder"
	envServer "github.com/ADimoska/SOMASExtended/server"
)

func main() {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}

	// Create log file with timestamp in name
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFile, err := os.OpenFile("logs/log_"+timestamp+".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Create a MultiWriter to write to both the log file and stdout
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Set log output to multiWriter
	log.SetOutput(multiWriter)

	// Remove date and time prefix from log entries
	log.SetFlags(0)

	log.Println("main function started.")

	// agent configuration
	agentConfig := agents.AgentConfig{
		InitScore:    0,
		VerboseLevel: 10,
	}

	argExposeThresholds := flag.Bool("exposeThresholds", false, "Expose the thresholds to the agents")
	flag.Parse()

	serv := &envServer.EnvironmentServer{
		// note: the zero turn is used for team forming
		BaseServer: baseServer.CreateBaseServer[common.IExtendedAgent](
			3,                   //  iterations
			120,                 //  turns per iteration
			50*time.Millisecond, //  max duration
			10),                 //  message bandwidth
		Teams: make(map[uuid.UUID]*common.Team),
	}
	serv.Init(
		3,                    // turns to apply threshold once
		*argExposeThresholds, // expose thresholds
	)
	serv.SetGameRunner(serv)

	const numAgents int = 10

	agentPopulation := []common.IExtendedAgent{}
	for i := 0; i < numAgents; i++ {
		agentPopulation = append(agentPopulation, agents.Team4_CreateAgent(serv, agentConfig))
		agentPopulation = append(agentPopulation, agents.Team2_CreateAgent(serv, agentConfig))
		// agentPopulation = append(agentPopulation, agents.GetBaseAgents(serv, agentConfig))
		// Add other teams' agents here
	}

	for i := 0; i < numAgents-4; i++ {
		// Add mostly honest agents
		agentPopulation = append(agentPopulation, agents.Create_Team1Agent(serv, agentConfig, agents.Rational))
	}

	// Add a short term and long term cheater agent from team 1
	for i := 0; i < 2; i++ {
		team1ShortTermCheater := agents.Create_Team1Agent(serv, agentConfig, agents.CheatShortTerm)
		log.Printf("Team1 %v is of type CheatShortTerm", team1ShortTermCheater.GetID())
		agentPopulation = append(agentPopulation, team1ShortTermCheater)

		team1LongTermCheater := agents.Create_Team1Agent(serv, agentConfig, agents.CheatLongTerm)
		log.Printf("Team1 %v is of type CheatLongTerm", team1LongTermCheater.GetID())
		agentPopulation = append(agentPopulation, team1LongTermCheater)
	}

	for i, agent := range agentPopulation {
		agent.SetName(i)
		serv.AddAgent(agent)
	}

	//serv.ReportMessagingDiagnostics()
	serv.Start()

	// custom function to see agent result
	serv.LogAgentStatus()
	serv.LogTeamStatus()

	// record data
	// serv.DataRecorder.GamePlaybackSummary()
	gameRecorder.ExportToCSV(serv.DataRecorder, "visualization_output/csv_data")
}
