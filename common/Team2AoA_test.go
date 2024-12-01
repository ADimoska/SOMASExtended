package common

import (
	"testing"

	"github.com/google/uuid"
)

func TestAuditQueue_AddToQueue(t *testing.T) {
	queue := NewAuditQueue(3)

	queue.AddToQueue(true)
	queue.AddToQueue(false)
	queue.AddToQueue(true)

	if queue.GetLength() != 3 {
		t.Errorf("Expected 3, got %v", queue.GetLength())
	}

	if queue.GetWarnings() != 2 {
		t.Errorf("Expected 2, got %v", queue.GetWarnings())
	}

	queue.AddToQueue(false)

	if queue.GetWarnings() != 1 {
		t.Errorf("Expected 2, got %v", queue.GetWarnings())
	}

	queue.SetLength(4)

	queue.AddToQueue(true)

	if queue.GetWarnings() != 2 {
		t.Errorf("Expected 2, got %v", queue.GetWarnings())
	}
}

func TestAuditQueue_Reset(t *testing.T) {
	queue := NewAuditQueue(3)
	queue.AddToQueue(true)
	queue.AddToQueue(false)

	queue.Reset()

	if queue.GetWarnings() != 0 {
		t.Errorf("Expected 0 warnings after reset, got %d", queue.GetWarnings())
	}
}

func TestAuditQueue_GetLastRound(t *testing.T) {
	queue := NewAuditQueue(3)
	queue.AddToQueue(false)
	queue.AddToQueue(true)

	if !queue.GetLastRound() {
		t.Errorf("Expected last round to be true")
	}
}

func setUp () (uuid.UUID, uuid.UUID, uuid.UUID, *Team2AoA) {
	alice := uuid.New()
	bob := uuid.New()
	charlie := uuid.New()

	teamId := uuid.New()
	curTeam := NewTeam(teamId)

	curTeam.Agents = append(curTeam.Agents, alice)
	curTeam.Agents = append(curTeam.Agents, bob)
	curTeam.Agents = append(curTeam.Agents, charlie)

	curTeam.TeamAoA = CreateTeam2AoA(curTeam, alice)

	return alice, bob, charlie, curTeam.TeamAoA.(*Team2AoA)
}

func TestTeam2AoA_withdrawalOrder(t *testing.T) {
	alice, bob, charlie, aoa := setUp()

	if aoa.GetWithdrawalOrder([]uuid.UUID{alice, bob, charlie})[0] != alice {
		t.Errorf("Expected Alice to be first in withdrawal order")
	}
}

func TestTeam2Aoa_offences(t *testing.T) {
	alice, _, _, aoa := setUp()

	aoa.SetContributionAuditResult(alice, 100, 100, 100)
	// This is a warning
	aoa.SetWithdrawalAuditResult(alice, 100, 100, 100, 100)

	// Formalized as an offence
	if aoa.GetContributionAuditResult(alice) {
		t.Errorf("Expected false, got true")
	}

	// Warning has been converted to an offence
	if aoa.AuditMap[alice].GetWarnings() != 0 {
		t.Errorf("Expected 0, got %v", aoa.AuditMap[alice].GetWarnings())
	}

	// Offence has been incremented
	if aoa.OffenceMap[alice] != 1 {
		t.Errorf("Expected 1, got %v", aoa.OffenceMap[alice])
	}

	// This is a warning
	aoa.SetContributionAuditResult(alice, 100, 20, 100)
	// Legitimate behaviour -> Should lead to only one warning this round
	aoa.SetWithdrawalAuditResult(alice, 100, 25, 25, 100)

	if aoa.AuditMap[alice].GetWarnings() != 1 {
		t.Errorf("Expected 1, got %v", aoa.AuditMap[alice].GetWarnings())
	}
}
