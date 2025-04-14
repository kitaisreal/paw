package stats

import (
	"math"
	"time"

	"slices"

	"github.com/kitaisreal/paw/internal/driver"
)

type Stats struct {
	MinServerDuration        time.Duration `json:"min_server_duration"`
	MaxServerDuration        time.Duration `json:"max_server_duration"`
	MeanServerDuration       time.Duration `json:"mean_server_duration"`
	MedianServerDuration     time.Duration `json:"median_server_duration"`
	DispersionServerDuration time.Duration `json:"dispersion_server_duration"`
	StdDevServerDuration     time.Duration `json:"std_dev_server_duration"`

	MinClientDuration        time.Duration `json:"min_client_duration"`
	MaxClientDuration        time.Duration `json:"max_client_duration"`
	MeanClientDuration       time.Duration `json:"mean_client_duration"`
	MedianClientDuration     time.Duration `json:"median_client_duration"`
	DispersionClientDuration time.Duration `json:"dispersion_client_duration"`
	StdDevClientDuration     time.Duration `json:"std_dev_client_duration"`
}

func (s *Stats) GetMinServerDurationMilliseconds() float64 {
	return float64(s.MinServerDuration) / 1e6
}

func (s *Stats) GetMaxServerDurationMilliseconds() float64 {
	return float64(s.MaxServerDuration) / 1e6
}

func (s *Stats) GetMeanServerDurationMilliseconds() float64 {
	return float64(s.MeanServerDuration) / 1e6
}

func (s *Stats) GetMedianServerDurationMilliseconds() float64 {
	return float64(s.MedianServerDuration) / 1e6
}

func (s *Stats) GetStdDevServerDurationMilliseconds() float64 {
	return float64(s.StdDevServerDuration) / 1e6
}

func (s *Stats) GetMinClientDurationMilliseconds() float64 {
	return float64(s.MinClientDuration) / 1e6
}

func (s *Stats) GetMaxClientDurationMilliseconds() float64 {
	return float64(s.MaxClientDuration) / 1e6
}

func (s *Stats) GetMeanClientDurationMilliseconds() float64 {
	return float64(s.MeanClientDuration) / 1e6
}

func (s *Stats) GetMedianClientDurationMilliseconds() float64 {
	return float64(s.MedianClientDuration) / 1e6
}

func (s *Stats) GetStdDevClientDurationMilliseconds() float64 {
	return float64(s.StdDevClientDuration) / 1e6
}

func GetStats(times []driver.ExecutionTime) Stats {
	result := Stats{}

	if len(times) == 0 {
		return result
	}

	result.MinServerDuration = times[0].ServerDuration
	result.MaxServerDuration = times[0].ServerDuration
	result.MeanServerDuration = times[0].ServerDuration

	result.MinClientDuration = times[0].ClientDuration
	result.MaxClientDuration = times[0].ClientDuration
	result.MeanClientDuration = times[0].ClientDuration

	for _, t := range times[1:] {
		if t.ServerDuration < result.MinServerDuration {
			result.MinServerDuration = t.ServerDuration
		}
		if t.ServerDuration > result.MaxServerDuration {
			result.MaxServerDuration = t.ServerDuration
		}

		if t.ClientDuration < result.MinClientDuration {
			result.MinClientDuration = t.ClientDuration
		}
		if t.ClientDuration > result.MaxClientDuration {
			result.MaxClientDuration = t.ClientDuration
		}

		result.MeanServerDuration += t.ServerDuration
		result.MeanClientDuration += t.ClientDuration
	}

	result.MeanServerDuration /= time.Duration(len(times))
	result.MeanClientDuration /= time.Duration(len(times))

	for _, t := range times {
		serverDurationDiff := t.ServerDuration - result.MeanServerDuration
		result.DispersionServerDuration += serverDurationDiff * serverDurationDiff

		clientDurationDiff := t.ClientDuration - result.MeanClientDuration
		result.DispersionClientDuration += clientDurationDiff * clientDurationDiff
	}

	result.DispersionServerDuration /= time.Duration(len(times))
	result.DispersionClientDuration /= time.Duration(len(times))

	result.StdDevServerDuration = time.Duration(math.Sqrt(float64(result.DispersionServerDuration)))
	result.StdDevClientDuration = time.Duration(math.Sqrt(float64(result.DispersionClientDuration)))

	result.MedianServerDuration = getMedianDuration(times, func(t driver.ExecutionTime) time.Duration {
		return t.ServerDuration
	})
	result.MedianClientDuration = getMedianDuration(times, func(t driver.ExecutionTime) time.Duration {
		return t.ClientDuration
	})

	return result
}

func getMedianDuration(
	times []driver.ExecutionTime,
	getDuration func(executionTime driver.ExecutionTime) time.Duration,
) time.Duration {
	if len(times) == 0 {
		return 0
	}

	durations := make([]time.Duration, len(times))
	for i, t := range times {
		durations[i] = getDuration(t)
	}
	slices.Sort(durations)

	mid := len(durations) / 2
	if len(durations)%2 == 0 {
		return durations[mid-1] + (durations[mid]-durations[mid-1])/2
	}

	return durations[mid]
}
