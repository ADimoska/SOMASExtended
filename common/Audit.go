package common

import "github.com/google/uuid"

type AuditRecord struct {
	auditMap map[uuid.UUID][]int
	duration int
	cost     int
	// reliability float64
}

func NewAuditRecord(duration int) *AuditRecord {
	cost := calculateCost(duration)

	return &AuditRecord{
		auditMap: make(map[uuid.UUID][]int),
		duration: duration,
		cost:     cost,
	}
}

// Getters
func (a *AuditRecord) GetAuditMap() map[uuid.UUID][]int {
	return a.auditMap
}

func (a *AuditRecord) GetAuditDuration() int {
	return a.duration
}

func (a *AuditRecord) GetAuditCost() int {
	return a.cost
}

// Setters
func (a *AuditRecord) SetAuditDuration(duration int) {
	a.duration, a.cost = duration, calculateCost(duration)
}

// Implement a more sophisticated cost calculation if needed, could compound with reliability if implemented
func calculateCost(duration int) int {
	return duration
}

// Get the number of infractions in the last n rounds, given by the quality of the audit
func (a *AuditRecord) GetAllInfractions(agentId uuid.UUID) int {
	infractions := 0
	records := a.auditMap[agentId]

	history := min(a.duration, len(records))

	for _, infraction := range records[len(records)-history:] {
		infractions += infraction
	}

	return infractions
}

/**
* Clear all infractions for a given agent
* This may/may not be called in case the audit system is converted into a probability-based hybrid.
* In such a case, the infractions may need to be kept in case there is an unsuccessful audit.
 */
func (a *AuditRecord) ClearAllInfractions(agentId uuid.UUID) {
	a.auditMap[agentId] = []int{}
}

// After an agent's contribution, add a new record to the audit map - infraction could be 1 or 0 instead of bool
func (a *AuditRecord) AddRecord(agentId uuid.UUID, infraction int) {
	if _, ok := a.auditMap[agentId]; !ok {
		a.auditMap[agentId] = []int{}
	}

	a.auditMap[agentId] = append(a.auditMap[agentId], infraction)
}

// In case this is needed by individual AoAs
func (a *AuditRecord) GetLastRecord(agentId uuid.UUID) int {
	if _, ok := a.auditMap[agentId]; !ok {
		return 0
	}

	records := a.auditMap[agentId]
	return records[len(records)-1]
}

// After the agent's withdrawal, which is after the contribution, update the last record instead of adding a new one
func (a *AuditRecord) IncrementLastRecord(agentId uuid.UUID) {
	if _, ok := a.auditMap[agentId]; !ok {
		a.auditMap[agentId] = []int{}
	}

	records := a.auditMap[agentId]
	if len(records) == 0 {
		return
	}

	records[len(records)-1]++
}
