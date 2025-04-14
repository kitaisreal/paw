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

type Settings = map[string]any

type Driver interface {
	Run(ctx context.Context, command string) (ExecutionTime, error)
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
