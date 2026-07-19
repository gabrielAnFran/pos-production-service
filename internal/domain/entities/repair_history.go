package entities

import "time"

// RepairHistory is an append-only audit row written on every status transition.
type RepairHistory struct {
	ID         string          `bson:"_id"`
	OSID       string          `bson:"os_id"`
	FromStatus ExecutionStatus `bson:"from_status"`
	ToStatus   ExecutionStatus `bson:"to_status"`
	Notes      string          `bson:"notes,omitempty"`
	At         time.Time       `bson:"at"`
}
