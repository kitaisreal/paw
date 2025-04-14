package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kitaisreal/paw/internal/collector"
	"github.com/kitaisreal/paw/internal/config"
	"github.com/kitaisreal/paw/internal/driver"
	"github.com/kitaisreal/paw/internal/logger"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

func Record(_ *cobra.Command, args []string) {
	if len(args) == 0 {
		logger.Log.Error("No test files specified")
		os.Exit(1)
	}

	ctx := context.Background()
	testFilePath := args[0]

	configuration := config.CreateDefaultConfig()
	usingConfigMessage := "using default config"

	if configPath != "" {
		parsedConfiguration, err := config.ParseConfigFileYaml(configPath)
		if err != nil {
			logger.Log.Errorf("Failed to parse config file: %v", err)
			os.Exit(1)
		}

		configuration = parsedConfiguration
		usingConfigMessage = fmt.Sprintf("using config file: %s", configPath)
	}

	configurationSettings := configuration.Settings
	test, err := config.ParseTestFileYaml(testFilePath)
	if err != nil {
		logger.Log.Errorf("Failed to parse test file %s: %v", testFilePath, err)
		os.Exit(1)
	}

	logger.Log.Infof("Recording performance for test file: %v %s, profile: %s, measure runs: %v",
		testFilePath,
		usingConfigMessage,
		profile,
		configurationSettings.QueryMeasureRuns)

	if outputPath == "" {
		outputPath = test.Name
	}

	createOutputPath(outputPath)
	copyConfigurationFiles(configPath, testFilePath, outputPath)

	driver := buildDriver(configuration, profile)
	collectors := buildCollectors(configuration, test)
	for _, collector := range collectors {
		defer collector.cleanup()
	}

	logger.Log.Debugf("Recording started")

	const fixedDescriptionWidth = 80

	description := "Starting..."
	description += strings.Repeat(" ", fixedDescriptionWidth-len(description))

	var progressBar *progressbar.ProgressBar
	if debug {
		progressBar = progressbar.DefaultSilent(int64(len(test.Queries)))
	} else {
		progressBar = progressbar.Default(int64(len(test.Queries)))
	}

	progressBar.Describe(description)
	_ = progressBar.RenderBlank() //nolint:errcheck

	for index, query := range test.Queries {
		if queryIndex >= 0 && queryIndex != index {
			continue
		}

		description = fmt.Sprintf("Running query %d: %s", index, query)
		if len(description) < fixedDescriptionWidth {
			description += strings.Repeat(" ", fixedDescriptionWidth-len(description))
		} else if len(description) > fixedDescriptionWidth {
			description = description[:fixedDescriptionWidth-3] + "..."
		}

		progressBar.Describe(description)
		_ = progressBar.RenderBlank() //nolint:errcheck

		queryDirName := fmt.Sprintf("%s/query_%d", outputPath, index)

		removeDirectoryOrExit(queryDirName)
		createDirectoryOrExit(queryDirName)

		queryRecord := recordQuery(ctx,
			driver,
			collectors,
			configurationSettings.QueryMeasureRuns,
			index,
			query,
			queryDirName,
		)

		fileName := fmt.Sprintf("%s/query_record.json", queryDirName)
		err := serializeQueryRecord(fileName, queryRecord)
		if err != nil {
			logger.Log.Errorf("Failed to save %v query '%v' record result to %s: %v", index, query, fileName, err)
			os.Exit(1)
		}

		logger.Log.Debugf("Saved %v query '%v' record result to %s", index, query, fileName)
		_ = progressBar.Add(1) //nolint:errcheck
	}

	logger.Log.Debugf("Recording completed")
}

func createOutputPath(outputPath string) {
	if _, err := os.Stat(outputPath); err == nil {
		fmt.Printf("Output folder %s already exists. Type 'delete' to remove: ", outputPath)

		var response string
		_, err := fmt.Scanln(&response)
		if err != nil {
			logger.Log.Errorf("Failed to read response: %v", err)
			os.Exit(1)
		}

		if response != "delete" {
			logger.Log.Error("Operation cancelled")
			os.Exit(1)
		}

		if err := os.RemoveAll(outputPath); err != nil {
			logger.Log.Errorf("Failed to remove directory %s: %v", outputPath, err)
			os.Exit(1)
		}
	}

	createDirectoryOrExit(outputPath)
}

func copyConfigurationFiles(configPath string, testFilePath string, outputPath string) {
	if configPath != "" {
		configDestPath := filepath.Join(outputPath, "config.yaml")
		err := copyFile(configPath, configDestPath)
		if err != nil {
			logger.Log.Errorf("Failed to copy config file to %s: %v", configDestPath, err)
			os.Exit(1)
		}
	}

	testDestPath := filepath.Join(outputPath, "test_file.yaml")
	err := copyFile(testFilePath, testDestPath)
	if err != nil {
		logger.Log.Errorf("Failed to copy test file to %s: %v", testDestPath, err)
		os.Exit(1)
	}
}

func recordQuery(ctx context.Context,
	driver driver.Driver,
	collectors []CollectorWithName,
	measureRuns uint64,
	queryNumber int,
	query string,
	outputPath string,
) QueryRecord {
	queryRecord := QueryRecord{
		QueryNumber: queryNumber,
		Query:       query,
	}

	logger.Log.Debugf("Running %v query '%v' measure runs %v", queryNumber, query, measureRuns)

	for run := uint64(0); run < measureRuns; run++ {
		executionTime, err := driver.Run(ctx, query)
		if err != nil {
			logger.Log.Errorf("Failed to run %v query '%v': %v", queryNumber, query, err)
			os.Exit(1)
		}

		queryRecord.ExecutionTimes = append(queryRecord.ExecutionTimes, executionTime)
	}

	logger.Log.Debugf("Finished running %v query '%v' measure runs %v", queryNumber, query, measureRuns)

	for _, collectorWithName := range collectors {
		collector, collectorName := collectorWithName.collector, collectorWithName.name

		collectorDirName := fmt.Sprintf("%s/%s", outputPath, collectorName)
		err := os.MkdirAll(collectorDirName, 0755)
		if err != nil {
			logger.Log.Errorf("Failed to create directory %s for collector %s: %v", collectorDirName, collectorName, err)
			os.Exit(1)
		}

		logger.Log.Debugf("Collecting using %s collector for %v query '%v' saving to %s",
			collectorName,
			queryNumber,
			query,
			collectorDirName,
		)
		collectorResult, err := collector.Collect(ctx, driver, query, collectorDirName)
		if err != nil {
			logger.Log.Errorf("Failed to collect %s: %v", collectorName, err)
			os.Exit(1)
		}

		logger.Log.Debugf("Collected using %s collector for %v query '%v' finished",
			collectorName,
			queryNumber,
			query,
		)
		queryRecord.CollectorResults = append(queryRecord.CollectorResults, collectorResult)
	}

	return queryRecord
}

func buildDriver(configuration config.Config, profile string) driver.Driver {
	profiles := buildProfiles(configuration)
	driverProfile, ok := profiles[profile]
	if !ok {
		logger.Log.Errorf("Profile %s not found", profile)
		os.Exit(1)
	}

	driver, err := driver.CreateDriver(driverProfile.Driver, driverProfile.Settings)
	if err != nil {
		logger.Log.Errorf("Failed to create driver: %v", err)
		os.Exit(1)
	}

	return driver
}

type CollectorWithName struct {
	collector collector.Collector
	cleanup   collector.CleanupFunc
	name      string
}

func buildCollectors(configuration config.Config, test config.Test) []CollectorWithName {
	collectors := []CollectorWithName{}

	collectorProfiles := buildCollectorProfiles(configuration)

	for _, collectorName := range test.Collectors {
		collectorProfile, ok := collectorProfiles[collectorName]
		if !ok {
			logger.Log.Errorf("Collector profile %s not found", collectorName)
			os.Exit(1)
		}

		collector, cleanup, err := collector.CreateCollector(collectorProfile.Collector, collectorProfile.Settings)
		if err != nil {
			logger.Log.Errorf("Failed to create collector %s: %v", collectorName, err)
			os.Exit(1)
		}

		collectors = append(collectors, CollectorWithName{
			collector: collector,
			cleanup:   cleanup,
			name:      collectorName,
		})
	}

	return collectors
}

func buildProfiles(configuration config.Config) map[string]config.Profile {
	profiles := map[string]config.Profile{}

	for name := range driver.Drivers {
		profiles[name] = config.Profile{
			Name:     name,
			Driver:   name,
			Settings: driver.Settings{},
		}
	}

	for _, profile := range configuration.Profiles {
		profiles[profile.Name] = profile
	}

	return profiles
}

func buildCollectorProfiles(configuration config.Config) map[string]config.CollectorProfile {
	profiles := map[string]config.CollectorProfile{}

	for name := range collector.Collectors {
		profiles[name] = config.CollectorProfile{
			Name:      name,
			Collector: name,
			Settings:  collector.Settings{},
		}
	}

	for _, profile := range configuration.CollectorProfiles {
		profiles[profile.Name] = profile
	}

	return profiles
}
