package stats_test

import (
	"testing"
	"time"

	"github.com/kitaisreal/paw/internal/driver"
	"github.com/kitaisreal/paw/internal/stats"
	"github.com/stretchr/testify/require"
)

func TestGetStats(t *testing.T) {
	executionTimes := []driver.ExecutionTime{}

	for i := range 11 {
		executionTimes = append(executionTimes, driver.ExecutionTime{
			ServerDuration: time.Second * time.Duration(i+1),
			ClientDuration: time.Second * time.Duration(i+1),
		})
	}

	stats := stats.GetStats(executionTimes)

	require.Equal(t, stats.MinServerDuration, time.Second)
	require.Equal(t, stats.MaxServerDuration, time.Second*11)
	require.Equal(t, stats.MeanServerDuration, time.Second*6)
	require.Equal(t, stats.MedianServerDuration, time.Second*6)

	require.Equal(t, stats.MinClientDuration, time.Second)
	require.Equal(t, stats.MaxClientDuration, time.Second*11)
	require.Equal(t, stats.MeanClientDuration, time.Second*6)
	require.Equal(t, stats.MedianClientDuration, time.Second*6)
}

func TestGetEmptyStats(t *testing.T) {
	executionTimes := []driver.ExecutionTime{}

	stats := stats.GetStats(executionTimes)

	require.Equal(t, stats.MinServerDuration, time.Duration(0))
	require.Equal(t, stats.MaxServerDuration, time.Duration(0))
	require.Equal(t, stats.MeanServerDuration, time.Duration(0))
	require.Equal(t, stats.MedianServerDuration, time.Duration(0))

	require.Equal(t, stats.MinClientDuration, time.Duration(0))
	require.Equal(t, stats.MaxClientDuration, time.Duration(0))
	require.Equal(t, stats.MeanClientDuration, time.Duration(0))
	require.Equal(t, stats.MedianClientDuration, time.Duration(0))
}
