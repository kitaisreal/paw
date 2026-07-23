package driver

import (
	"context"
	"fmt"
	"time"
)

type ExecutionTime struct {
	ClientDuration time.Duration `json:"client_duration"`
	ServerDuration time.Duration `json:"server_duration"`
}

// ProfileEvents holds engine-specific performance counters collected for a
// single query run.
type ProfileEvents = map[string]uint64

type Settings = map[string]any

type Driver interface {
	Run(ctx context.Context, query string) (ExecutionTime, error)

	// SupportsProfileEvents reports whether RunWithProfileEvents is implemented.
	// Callers must check this before calling RunWithProfileEvents.
	SupportsProfileEvents() bool

	// RunWithProfileEvents runs the query and returns its profile events.
	// It panics if SupportsProfileEvents returns false.
	RunWithProfileEvents(ctx context.Context, query string) (ExecutionTime, ProfileEvents, error)
}

type Creator = func(settings Settings) (Driver, error)

var Drivers = map[string]Creator{}

func RegisterDriver(name string, creator Creator) {
	Drivers[name] = creator
}

func CreateDriver(name string, settings Settings) (Driver, error) {
	creator, ok := Drivers[name]
	if !ok {
		return nil, fmt.Errorf("driver %s not found", name)
	}

	return creator(settings)
}
