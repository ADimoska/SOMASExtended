package agents

/* Code to test the helper functions used in Team1AoA_ExtendedAgent */

import (
	"reflect"
	"testing"
	"time"
	// "github.com/stretchr/testify/assert"
	"github.com/google/uuid"

	common "github.com/ADimoska/SOMASExtended/common"
	envServer "github.com/ADimoska/SOMASExtended/server"
	baseServer "github.com/MattSScott/basePlatformSOMAS/v2/pkg/server"
)

// Agent test configuration
var agentConfig = AgentConfig{
	InitScore:    0,
	VerboseLevel: 10,
}

// Tests the base implementation of gathering opinions on rank boundaries
func TestGatherRankBoundaryOpinions(t *testing.T) {

	serv := &envServer.EnvironmentServer{
		// note: the zero turn is used for team forming
		BaseServer: baseServer.CreateBaseServer[common.IExtendedAgent](2, 3, 1000*time.Millisecond, 10),
		Teams:      make(map[uuid.UUID]*common.Team),
	}
	serv.SetGameRunner(serv)

	agentPopulation := []common.IExtendedAgent{}
	// Add 4 agents with a low score of 20
	for range 4 {
		agentPopulation = append(agentPopulation, GetBaseAgents(serv, agentConfig))
	}

	// Add all agents to server
	agentIDs := make([]uuid.UUID, 0)
	for _, agent := range agentPopulation {
		serv.AddAgent(agent)
		agentIDs = append(agentIDs, agent.GetID())
	}
	serv.CreateAndInitTeamWithAgents(agentIDs)

	// Extract the non-exported function via reflection
	testAgent := agentPopulation[0]
	value := reflect.ValueOf(testAgent).MethodByName("TestableGatherFunc")
	testableGatherFunc := value.Call(nil)[0]

	// Finally call the gather function
	testableGatherFunc.Call(nil)
}
