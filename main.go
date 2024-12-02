package main

import (
	"fmt"
	"github.com/google/uuid"
	"time"

	baseServer "github.com/MattSScott/basePlatformSOMAS/v2/pkg/server"

	agents "github.com/ADimoska/SOMASExtended/agents"
	common "github.com/ADimoska/SOMASExtended/common"
	envServer "github.com/ADimoska/SOMASExtended/server"
)

func main() {
	fmt.Println("main function started.")

	// agent configuration
	agentConfig := agents.AgentConfig{
		InitScore:    0,
		VerboseLevel: 10,
	}

	serv := &envServer.EnvironmentServer{
		// note: the zero turn is used for team forming
		BaseServer: baseServer.CreateBaseServer[common.IExtendedAgent](2, 3, 1000*time.Millisecond, 10),
		Teams:      make(map[uuid.UUID]*common.Team),
	}
	serv.SetGameRunner(serv)

	const numAgents int = 2

	agentPopulation := []common.IExtendedAgent{}
	for i := 0; i < numAgents; i++ {
		agentPopulation = append(agentPopulation, agents.Team4_CreateAgent(serv, agentConfig))
		agentPopulation = append(agentPopulation, agents.GetBaseAgents(serv, agentConfig))
		// Add other teams' agents here
	}

	for _, agent := range agentPopulation {
		serv.AddAgent(agent)
	}

	//serv.ReportMessagingDiagnostics()
	serv.Start()

	// custom function to see agent result
	serv.LogAgentStatus()
	serv.LogTeamStatus()
}
