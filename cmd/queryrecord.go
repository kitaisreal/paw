package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kitaisreal/paw/internal/collector"
	"github.com/kitaisreal/paw/internal/driver"
	"github.com/kitaisreal/paw/internal/stats"
)

type QueryRecord struct {
	QueryNumber      int                    `json:"query_number"`
	Query            string                 `json:"query"`
	ExecutionTimes   []driver.ExecutionTime `json:"execution_times"`
	CollectorResults []collector.Result     `json:"collector_results"`
}

type QueryRecordWithStats struct {
	Record QueryRecord
	Stats  stats.Stats
}

type QueryRecordPair struct {
	LHS QueryRecord
	RHS QueryRecord
}

type QueryRecordPairWithStats struct {
	LHS QueryRecordWithStats
	RHS QueryRecordWithStats
}

func serializeQueryRecord(filePath string, record QueryRecord) error {
	jsonData, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, jsonData, 0644)
}

func deserializeQueryRecord(filePath string) (QueryRecord, error) {
	var record QueryRecord

	content, err := os.ReadFile(filePath)
	if err != nil {
		return record, fmt.Errorf("error reading query record file %s: %w", filePath, err)
	}

	err = json.Unmarshal(content, &record)
	if err != nil {
		return record, fmt.Errorf("error unmarshalling query record file %s: %w", filePath, err)
	}

	return record, nil
}
