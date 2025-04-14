package config

import (
	"os"

	"github.com/kitaisreal/paw/internal/collector"
	"github.com/kitaisreal/paw/internal/driver"
	"gopkg.in/yaml.v3"
)

type Profile struct {
	Name     string          `yaml:"name"`
	Driver   string          `yaml:"driver"`
	Settings driver.Settings `yaml:"settings"`
}

type CollectorProfile struct {
	Name      string             `yaml:"name"`
	Collector string             `yaml:"collector"`
	Settings  collector.Settings `yaml:"settings"`
}

type Settings struct {
	QueryMeasureRuns uint64 `yaml:"query_measure_runs"`
}

type Config struct {
	Profiles          []Profile          `yaml:"profiles"`
	CollectorProfiles []CollectorProfile `yaml:"collector_profiles"`
	Settings          Settings           `yaml:"settings"`
}

type Test struct {
	Name       string   `yaml:"name"`
	Collectors []string `yaml:"collectors"`
	Queries    []string `yaml:"queries"`
}

func CreateDefaultConfig() Config {
	return Config{
		Profiles:          []Profile{},
		CollectorProfiles: []CollectorProfile{},
		Settings:          Settings{QueryMeasureRuns: 5},
	}
}

func ParseConfigFileYaml(path string) (Config, error) {
	return parseYamlFile[Config](path)
}

func ParseTestFileYaml(path string) (Test, error) {
	return parseYamlFile[Test](path)
}

func parseYamlFile[T any](path string) (T, error) {
	var result T

	data, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}

	if err := yaml.Unmarshal(data, &result); err != nil {
		return result, err
	}

	return result, nil
}
