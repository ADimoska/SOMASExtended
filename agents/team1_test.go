package agents

/* Code to test the helper functions used in Team1AoA_ExtendedAgent */

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"time"

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

func TestGenerateCandidates(t *testing.T) {

	serv := &envServer.EnvironmentServer{
		// note: the zero turn is used for team forming
		BaseServer: baseServer.CreateBaseServer[common.IExtendedAgent](2, 3, 1000*time.Millisecond, 10),
		Teams:      make(map[uuid.UUID]*common.Team),
	}
	serv.SetGameRunner(serv)

	testAgent := GetBaseAgents(serv, agentConfig)
	serv.AddAgent(testAgent)

	// Random data for possible rankings
	data := [][5]int{
		{12, 18, 25, 29, 40},
		{1, 3, 5, 7, 9},
		{10, 20, 30, 40, 50},
		{11, 23, 31, 41, 50},
	}

	// Extract the non-exported function via reflection
	value := reflect.ValueOf(testAgent).MethodByName("TestableCandidateGenFunc")
	testData := []reflect.Value{reflect.ValueOf(data)} // don't ask
	testableCandidateGenFunc := value.Call(testData)[0]

	expected_lower := [5]int{6, 11, 15, 18, 25}
	expected_average := [5]int{11, 19, 28, 35, 45}
	expected_upper := [5]int{12, 22, 31, 41, 50}

	candidates := testableCandidateGenFunc.Call(nil)[0]

	assert.Equal(t, expected_lower, candidates.Index(0).Interface().([5]int))
	assert.Equal(t, expected_average, candidates.Index(1).Interface().([5]int))
	assert.Equal(t, expected_upper, candidates.Index(2).Interface().([5]int))
}
