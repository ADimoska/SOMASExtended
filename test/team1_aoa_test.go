package main

/*
* Code to test the AoA functionality for Team 1
 */

import (
	"testing" // built-in go testing package
	// "github.com/stretchr/testify/assert" // assert package, easier to
)

func TestGatherRankBoundaryOpinions(t *testing.T) {
	serv, agentIDs := CreateTestServer()
	serv.CreateAndInitTeamWithAgents(agentIDs)

	// TODO: Black box testing once all functions done

	// Test all agents as a potential 'chair'. This might seem superfluous but
	// will make more sense as teams start adding their own strategies etc.
}
