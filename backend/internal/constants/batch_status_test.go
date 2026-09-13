package constants

import "testing"

func TestBatchTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    BatchStatus
		to      BatchStatus
		allowed bool
	}{
		{"draft starts", BatchStatusDraft, BatchStatusRunning, true},
		{"running holds", BatchStatusRunning, BatchStatusHold, true},
		{"hold resumes", BatchStatusHold, BatchStatusRunning, true},
		{"rework releases", BatchStatusRework, BatchStatusReleased, true},
		{"draft cannot release", BatchStatusDraft, BatchStatusReleased, false},
		{"released is terminal", BatchStatusReleased, BatchStatusRunning, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.allowed {
				t.Fatalf("transition %s -> %s = %v, want %v", tt.from, tt.to, got, tt.allowed)
			}
		})
	}
}

func TestRolePermissions(t *testing.T) {
	if !RoleAdmin.Can("release:write") {
		t.Fatal("admin must be able to release")
	}
	if !RoleInspector.Can("inspection:write") {
		t.Fatal("inspector must write inspections")
	}
	if RoleInspector.Can("release:write") {
		t.Fatal("inspector must not release")
	}
	if RoleViewer.Can("batch:write") {
		t.Fatal("viewer must be read-only")
	}
}
