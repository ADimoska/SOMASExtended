package common

import (
	"testing"

	// "github.com/google/uuid"
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
