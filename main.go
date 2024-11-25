package main

import (
	"fmt"
	"time"

	"SOMAS_Extended/agents"
	envServer "SOMAS_Extended/server"
)

func main() {
	fmt.Println("main function started.")

	// agent configurations
	agentConfig := agents.AgentConfig{
		InitScore:    0,
		VerboseLevel: 10,
	}

	// parameters: agent num PER TEAM, iterations, turns, max duration, max thread
	// note: the zero turn is used for team forming
	serv := envServer.MakeEnvServer(2, 2, 3, 1000*time.Millisecond, 10, agentConfig)

	//serv.ReportMessagingDiagnostics()
	serv.Start()

	// custom function to see agent result
	serv.LogAgentStatus()
	serv.LogTeamStatus()
}
