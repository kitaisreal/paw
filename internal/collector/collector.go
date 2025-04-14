package collector

import (
	"context"
	"fmt"

	"github.com/kitaisreal/paw/internal/driver"
)

type FileType string

const (
	FileTypeFlamegraph FileType = "flamegraph"
)

type ResultFile struct {
	Type FileType `json:"type"`
	Name string   `json:"name"`
}

type Result struct {
	Name           string                 `json:"name"`
	Files          []ResultFile           `json:"files"`
	ExecutionTimes []driver.ExecutionTime `json:"execution_times"`
}

type Settings = map[string]any

type Collector interface {
	Collect(ctx context.Context, driver driver.Driver, query string, outputFolder string) (Result, error)
}

type CleanupFunc = func()

type Creator = func(settings Settings) (Collector, CleanupFunc, error)

var Collectors = map[string]Creator{}

func RegisterCollector(name string, creator Creator) {
	Collectors[name] = creator
}

func CreateCollector(name string, settings Settings) (Collector, CleanupFunc, error) {
	creator, ok := Collectors[name]
	if !ok {
		return nil, nil, fmt.Errorf("collector %s not found", name)
	}

	return creator(settings)
}
