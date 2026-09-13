package constants

import "fmt"

type BatchStatus string

const (
	BatchStatusDraft    BatchStatus = "draft"
	BatchStatusRunning  BatchStatus = "running"
	BatchStatusHold     BatchStatus = "hold"
	BatchStatusRework   BatchStatus = "rework"
	BatchStatusReleased BatchStatus = "released"
)

var validBatchStatuses = map[BatchStatus]struct{}{
	BatchStatusDraft: {}, BatchStatusRunning: {}, BatchStatusHold: {},
	BatchStatusRework: {}, BatchStatusReleased: {},
}

func (s BatchStatus) Valid() bool {
	_, ok := validBatchStatuses[s]
	return ok
}

func (s BatchStatus) CanTransitionTo(next BatchStatus) bool {
	allowed := map[BatchStatus]map[BatchStatus]bool{
		BatchStatusDraft:    {BatchStatusRunning: true, BatchStatusHold: true},
		BatchStatusRunning:  {BatchStatusHold: true, BatchStatusRework: true, BatchStatusReleased: true},
		BatchStatusHold:     {BatchStatusRunning: true, BatchStatusRework: true},
		BatchStatusRework:   {BatchStatusRunning: true, BatchStatusHold: true, BatchStatusReleased: true},
		BatchStatusReleased: {},
	}
	return allowed[s][next]
}

func ValidateBatchTransition(current, next BatchStatus) error {
	if !next.Valid() {
		return fmt.Errorf("invalid batch status: %s", next)
	}
	if current == next {
		return nil
	}
	if !current.CanTransitionTo(next) {
		return fmt.Errorf("batch cannot transition from %s to %s", current, next)
	}
	return nil
}
