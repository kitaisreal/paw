package collector

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kitaisreal/paw/internal/collector/flamegraph"
	"github.com/kitaisreal/paw/internal/driver"
	"github.com/kitaisreal/paw/internal/logger"
)

const (
	offCPUFlameGraphCollectorName                              = "off_cpu_flamegraph"
	offCPUFlameGraphCollectorFlameGraphDefaultBuildSeconds     = 5
	offCPUFlameGraphCollectorFlameGraphBuildSecondsSettingName = "build_seconds"
	offCPUFlameGraphCollectorOutputFile                        = "off_cpu_flamegraph.svg"
)

type OffCPUFlamegraphCollector struct {
	flameGraphBuildSeconds int
	tempDir                string
	flamegraphScriptPath   string
}

func CreateOffCPUFlamegraphCollector(flameGraphBuildSeconds int) (Collector, CleanupFunc, error) {
	tempDir, err := os.MkdirTemp("", offCPUFlameGraphCollectorName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	logger.Log.Debugf("Collector %s created temp directory: %s", offCPUFlameGraphCollectorName, tempDir)

	cleanup := func() {
		logger.Log.Debugf("Collector %s removing temp directory: %s", offCPUFlameGraphCollectorName, tempDir)
		err := os.RemoveAll(tempDir)
		if err != nil {
			logger.Log.Errorf("Collector %s failed to remove temp directory: %v", offCPUFlameGraphCollectorName, err)
		}
	}

	flamegraphScriptPath := filepath.Join(tempDir, "flamegraph.pl")
	if err := os.WriteFile(flamegraphScriptPath, flamegraph.FlameGraphScript, 0755); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to write flamegraph script: %w", err)
	}

	collector := &OffCPUFlamegraphCollector{
		flameGraphBuildSeconds: flameGraphBuildSeconds,
		tempDir:                tempDir,
		flamegraphScriptPath:   flamegraphScriptPath,
	}

	logger.Log.Debugf("Collector %s created with flamegraph build seconds: %d",
		offCPUFlameGraphCollectorName,
		flameGraphBuildSeconds,
	)

	return collector, cleanup, nil
}

func (c *OffCPUFlamegraphCollector) Collect(
	ctx context.Context,
	drv driver.Driver,
	query string,
	outputFolder string,
) (Result, error) {
	collectorResult := Result{
		Name: offCPUFlameGraphCollectorName,
		Files: []ResultFile{
			{Type: FileTypeFlamegraph, Name: offCPUFlameGraphCollectorOutputFile},
		},
		ExecutionTimes: []driver.ExecutionTime{},
	}

	waitChan := make(chan error)

	go func() {
		stacksFileName := filepath.Join(c.tempDir, "paw.out.stacks")
		offcputimeCmd := exec.CommandContext(ctx,
			"sh",
			"-c",
			fmt.Sprintf("offcputime-bpfcc -df %d > %s",
				c.flameGraphBuildSeconds,
				stacksFileName,
			),
		)
		logger.Log.Debugf("Collector %s offcputime command: %v", offCPUFlameGraphCollectorName, offcputimeCmd.String())

		if err := offcputimeCmd.Run(); err != nil {
			waitChan <- fmt.Errorf("collector %s offcputime %v error: %w",
				offCPUFlameGraphCollectorName,
				offcputimeCmd.String(),
				err,
			)
			return
		}

		offCPUFlamegraphOutputFilePath := filepath.Join(outputFolder, offCPUFlameGraphCollectorOutputFile)
		flameGraphCmd := exec.CommandContext(ctx,
			"sh",
			"-c",
			fmt.Sprintf("%s --color=io --title=\"Off-CPU Time Flame Graph\" --countname=us %s > %s",
				c.flamegraphScriptPath,
				stacksFileName,
				offCPUFlamegraphOutputFilePath,
			),
		)
		logger.Log.Debugf("Collector %s flamegraph command: %v", offCPUFlameGraphCollectorName, flameGraphCmd.String())

		if err := flameGraphCmd.Run(); err != nil {
			waitChan <- fmt.Errorf("collector %s flamegraph script %v error: %w",
				offCPUFlameGraphCollectorName,
				flameGraphCmd.String(),
				err,
			)
			return
		}

		if err := os.Chmod(offCPUFlamegraphOutputFilePath, 0664); err != nil {
			waitChan <- fmt.Errorf("collector %s failed to set permissions for flamegraph output file: %w",
				offCPUFlameGraphCollectorName,
				err,
			)
			return
		}

		waitChan <- nil
	}()

	var resultErr error

collectOffCPUFlamegraph:
	for {
		execTime, err := drv.Run(ctx, query)
		if err != nil {
			logger.Log.Errorf("Collector %s failed to run query '%v': %v",
				offCPUFlameGraphCollectorName,
				query,
				err,
			)
			os.Exit(1)
		}

		collectorResult.ExecutionTimes = append(collectorResult.ExecutionTimes, execTime)

		select {
		case resultErr = <-waitChan:
			break collectOffCPUFlamegraph
		default:
		}
	}

	return collectorResult, resultErr
}

func init() {
	RegisterCollector(offCPUFlameGraphCollectorName, func(settings Settings) (Collector, CleanupFunc, error) {
		flameGraphBuildSeconds := offCPUFlameGraphCollectorFlameGraphDefaultBuildSeconds

		if flameGraphBuildSecondsAny, ok := settings[offCPUFlameGraphCollectorFlameGraphBuildSecondsSettingName]; ok {
			flameGraphBuildSeconds, ok = flameGraphBuildSecondsAny.(int)
			if !ok {
				return nil, nil, fmt.Errorf("collector %s setting '%s' is not int",
					offCPUFlameGraphCollectorName,
					offCPUFlameGraphCollectorFlameGraphBuildSecondsSettingName,
				)
			}
		}

		return CreateOffCPUFlamegraphCollector(flameGraphBuildSeconds)
	})
}
