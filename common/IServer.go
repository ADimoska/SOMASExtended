package common

import (
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
)

type IServer interface {
	agent.IExposedServerFunctions[IExtendedAgent]
	// Team management functions
	CreateTeam()
	AddAgentToTeam(agentID uuid.UUID, teamID uuid.UUID)
	GetAgentsInTeam(teamID uuid.UUID) []uuid.UUID
	CheckAgentAlreadyInTeam(agentID uuid.UUID) bool
	CreateAndInitTeamWithAgents(agentIDs []uuid.UUID) uuid.UUID
	UpdateAndGetAgentExposedInfo() []ExposedAgentInfo
	IsAgentDead(agentID uuid.UUID) bool
	GetAgentKilledScore(agentID uuid.UUID) int
	StartAgentTeamForming()

	GetTeam(agentID uuid.UUID) *Team
	GetTeamFromTeamID(teamID uuid.UUID) *Team

	// Debug functions
	LogAgentStatus()
	PrintOrphanPool()
}
