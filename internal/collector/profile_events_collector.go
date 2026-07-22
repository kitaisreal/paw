package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kitaisreal/paw/internal/driver"
	"github.com/kitaisreal/paw/internal/logger"
)

const (
	profileEventsCollectorName       = "profile_events"
	profileEventsCollectorOutputFile = "profile_events.json"
)

type ProfileEventsCollector struct{}

func CreateProfileEventsCollector() (Collector, CleanupFunc, error) {
	return &ProfileEventsCollector{}, func() {}, nil
}

func (c *ProfileEventsCollector) Collect(
	ctx context.Context,
	drv driver.Driver,
	query string,
	outputFolder string,
) (Result, error) {
	if !drv.SupportsProfileEvents() {
		return Result{}, fmt.Errorf(
			"collector %s requires a driver that supports profile events",
			profileEventsCollectorName,
		)
	}

	executionTime, profileEvents, err := drv.RunWithProfileEvents(ctx, query)
	if err != nil {
		return Result{}, fmt.Errorf(
			"collector %s failed to run query '%v': %w",
			profileEventsCollectorName,
			query,
			err,
		)
	}

	profileEventsJSON, err := json.MarshalIndent(profileEvents, "", "  ")
	if err != nil {
		return Result{}, fmt.Errorf(
			"collector %s failed to marshal profile events: %w",
			profileEventsCollectorName,
			err,
		)
	}

	outputFilePath := filepath.Join(outputFolder, profileEventsCollectorOutputFile)
	if err := os.WriteFile(outputFilePath, profileEventsJSON, 0644); err != nil {
		return Result{}, fmt.Errorf(
			"collector %s failed to write profile events file %s: %w",
			profileEventsCollectorName,
			outputFilePath,
			err,
		)
	}

	logger.Log.Debugf("Collector %s collected %d profile events for query '%v'",
		profileEventsCollectorName,
		len(profileEvents),
		query,
	)

	return Result{
		Name: profileEventsCollectorName,
		Files: []ResultFile{
			{Type: FileTypeProfileEvents, Name: profileEventsCollectorOutputFile},
		},
		ExecutionTimes: []driver.ExecutionTime{executionTime},
		ProfileEvents:  profileEvents,
	}, nil
}

func init() {
	RegisterCollector(profileEventsCollectorName, func(_ Settings) (Collector, CleanupFunc, error) {
		return CreateProfileEventsCollector()
	})
}
