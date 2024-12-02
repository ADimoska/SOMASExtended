package main

import (
	"github.com/ADimoska/SOMASExtended/common"
	"github.com/google/uuid"
	"testing"
)

// Test if the audit duration gets dynamically updated, and all the records are kept up to date
func TestAuditDuration(t *testing.T) {
	ar := common.NewAuditRecord(3)
	agentId := uuid.New()

	// Add four records, only the three most recent should be considered
	ar.AddRecord(agentId, 1)
	ar.AddRecord(agentId, 0)
	ar.AddRecord(agentId, 1)
	ar.AddRecord(agentId, 1)

	infractions := ar.GetAllInfractions(agentId)

	if ar.GetAllInfractions(agentId) != 2 {
		t.Errorf("expected %d infractions, got %d", 2, infractions)
	}

	// Increase duration before getting infractions
	ar.SetAuditDuration(4)
	infractions = ar.GetAllInfractions(agentId)

	if infractions != 3 {
		t.Errorf("expected %d infractions, got %d", 3, infractions)
	}
}

// Test that once cleared, the past infractions are not double counted in future checks
func TestClearAllInfractions(t *testing.T) {
	ar := common.NewAuditRecord(5)
	agentId := uuid.New()

	ar.AddRecord(agentId, 1)
	ar.AddRecord(agentId, 1)

	ar.ClearAllInfractions(agentId)

	ar.AddRecord(agentId, 0)

	infractions := ar.GetAllInfractions(agentId)
	if infractions != 0 {
		t.Errorf("expected 0 infractions, got %d", infractions)
	}
}

func TestIncrementLastRecord(t *testing.T) {
	ar := common.NewAuditRecord(5)
	agentId := uuid.New()

	ar.AddRecord(agentId, 1)
	ar.AddRecord(agentId, 0)

	ar.IncrementLastRecord(agentId)

	lastInfraction := ar.GetLastRecord(agentId)
	if lastInfraction != 1 {
		t.Errorf("expected 1, got %d", lastInfraction)
	}

	infractions := ar.GetAllInfractions(agentId)
	if infractions != 2 {
		t.Errorf("expected 2 infractions, got %d", infractions)
	}
}
