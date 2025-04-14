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
	cpuFlameGraphCollectorName                              = "cpu_flamegraph"
	cpuFlameGraphCollectorFlameGraphDefaultBuildSeconds     = 5
	cpuFlameGraphCollectorFlameGraphBuildSecondsSettingName = "build_seconds"
	cpuFlameGraphCollectorOutputFile                        = "cpu_flamegraph.svg"
)

type CPUFlamegraphCollector struct {
	flameGraphBuildSeconds  int
	tempDir                 string
	stackCollapseScriptPath string
	flamegraphScriptPath    string
}

func CreateCPUFlamegraphCollector(flameGraphBuildSeconds int) (Collector, CleanupFunc, error) {
	tempDir, err := os.MkdirTemp("", cpuFlameGraphCollectorName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	logger.Log.Debugf("Collector %s created temp directory: %s", cpuFlameGraphCollectorName, tempDir)

	cleanup := func() {
		logger.Log.Debugf("Collector %s removing temp directory: %s", cpuFlameGraphCollectorName, tempDir)
		err := os.RemoveAll(tempDir)
		if err != nil {
			logger.Log.Errorf("Collector %s failed to remove temp directory: %v", cpuFlameGraphCollectorName, err)
		}
	}

	stackCollapseScriptPath := filepath.Join(tempDir, "stackcollapse-perf.pl")
	if err := os.WriteFile(stackCollapseScriptPath, flamegraph.StackCollapseScript, 0755); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to write stack collapse script: %w", err)
	}

	flamegraphScriptPath := filepath.Join(tempDir, "flamegraph.pl")
	if err := os.WriteFile(flamegraphScriptPath, flamegraph.FlameGraphScript, 0755); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to write flamegraph script: %w", err)
	}

	collector := &CPUFlamegraphCollector{
		flameGraphBuildSeconds:  flameGraphBuildSeconds,
		tempDir:                 tempDir,
		stackCollapseScriptPath: stackCollapseScriptPath,
		flamegraphScriptPath:    flamegraphScriptPath,
	}

	logger.Log.Debugf("Collector %s created with flamegraph build seconds: %d",
		cpuFlameGraphCollectorName,
		flameGraphBuildSeconds,
	)

	return collector, cleanup, nil
}

func (c *CPUFlamegraphCollector) Collect(
	ctx context.Context,
	drv driver.Driver,
	query string,
	outputFolder string,
) (Result, error) {
	collectorResult := Result{
		Name: cpuFlameGraphCollectorName,
		Files: []ResultFile{
			{Type: FileTypeFlamegraph, Name: cpuFlameGraphCollectorOutputFile},
		},
		ExecutionTimes: []driver.ExecutionTime{},
	}

	waitChan := make(chan error)

	go func() {
		pawDataFileName := filepath.Join(c.tempDir, "paw.perf.data")
		perfRecordCmd := exec.CommandContext(ctx,
			"perf",
			"record",
			"-F",
			"99",
			"-a",
			"-g",
			"-o", pawDataFileName,
			"--",
			"sleep",
			fmt.Sprint(c.flameGraphBuildSeconds),
		)
		logger.Log.Debugf("Collector %s perf record command: %v", cpuFlameGraphCollectorName, perfRecordCmd.String())

		if err := perfRecordCmd.Run(); err != nil {
			waitChan <- fmt.Errorf("collector %s perf record %v error: %w",
				cpuFlameGraphCollectorName,
				perfRecordCmd.String(),
				err,
			)
			return
		}

		pawFoldedDataFileName := filepath.Join(c.tempDir, "paw.out.perf-folded")
		foldPerfDataCmdArg := fmt.Sprintf(
			"perf script -i %s | %s > %s",
			pawDataFileName,
			c.stackCollapseScriptPath,
			pawFoldedDataFileName,
		)
		foldPerfDataCmd := exec.CommandContext(ctx,
			"sh",
			"-c",
			foldPerfDataCmdArg,
		)
		logger.Log.Debugf("Collector %s perf fold command: %v", cpuFlameGraphCollectorName, foldPerfDataCmd.String())

		if err := foldPerfDataCmd.Run(); err != nil {
			waitChan <- fmt.Errorf("collector %s perf script %v error: %w",
				cpuFlameGraphCollectorName,
				foldPerfDataCmd.String(),
				err)
			return
		}

		cpuFlamegraphOutputFilePath := filepath.Join(outputFolder, cpuFlameGraphCollectorOutputFile)
		flameGraphCmd := exec.CommandContext(ctx,
			"sh",
			"-c",
			fmt.Sprintf("%s %s > %s",
				c.flamegraphScriptPath,
				pawFoldedDataFileName,
				cpuFlamegraphOutputFilePath,
			),
		)
		logger.Log.Debugf("Collector %s flamegraph command: %v", cpuFlameGraphCollectorName, flameGraphCmd.String())

		if err := flameGraphCmd.Run(); err != nil {
			waitChan <- fmt.Errorf("collector %s flamegraph script %v error: %w",
				cpuFlameGraphCollectorName,
				flameGraphCmd.String(),
				err,
			)
			return
		}

		if err := os.Chmod(cpuFlamegraphOutputFilePath, 0664); err != nil {
			waitChan <- fmt.Errorf("collector %s failed to set permissions for flamegraph output file: %w",
				cpuFlameGraphCollectorName,
				err)
			return
		}

		waitChan <- nil
	}()

	var resultErr error

collectCPUFlamegraph:
	for {
		execTime, err := drv.Run(ctx, query)
		if err != nil {
			logger.Log.Errorf("Collector %s failed to run query '%v': %v",
				cpuFlameGraphCollectorName,
				query,
				err,
			)
			os.Exit(1)
		}

		collectorResult.ExecutionTimes = append(collectorResult.ExecutionTimes, execTime)

		select {
		case resultErr = <-waitChan:
			break collectCPUFlamegraph
		default:
		}
	}

	return collectorResult, resultErr
}

func init() {
	RegisterCollector(cpuFlameGraphCollectorName, func(settings Settings) (Collector, CleanupFunc, error) {
		flameGraphBuildSeconds := cpuFlameGraphCollectorFlameGraphDefaultBuildSeconds

		if flameGraphBuildSecondsAny, ok := settings[cpuFlameGraphCollectorFlameGraphBuildSecondsSettingName]; ok {
			flameGraphBuildSeconds, ok = flameGraphBuildSecondsAny.(int)
			if !ok {
				return nil, nil, fmt.Errorf("collector %s setting '%s' is not int",
					cpuFlameGraphCollectorName,
					cpuFlameGraphCollectorFlameGraphBuildSecondsSettingName,
				)
			}
		}

		return CreateCPUFlamegraphCollector(flameGraphBuildSeconds)
	})
}
