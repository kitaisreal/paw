package flamegraph

import (
	"embed"
	"fmt"
)

//go:embed flamegraph.pl stackcollapse-perf.pl
var flameGraphFiles embed.FS

var (
	FlameGraphScript    []byte
	StackCollapseScript []byte
)

func init() {
	var err error

	FlameGraphScript, err = flameGraphFiles.ReadFile("flamegraph.pl")
	if err != nil {
		panic(fmt.Errorf("failed to read flamegraph.pl: %w", err))
	}

	StackCollapseScript, err = flameGraphFiles.ReadFile("stackcollapse-perf.pl")
	if err != nil {
		panic(fmt.Errorf("failed to read stackcollapse-perf.pl: %w", err))
	}
}
