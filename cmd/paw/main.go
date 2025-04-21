package main

import (
	"os"

	"github.com/kitaisreal/paw/internal/logger"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "paw",
		Short: "Performance testing tool",
		Long:  "A tool for performance testing",
	}

	recordCmd = &cobra.Command{
		Use:              "record [test_file]",
		Short:            "Record performance for test",
		Long:             "Record performance for test",
		PersistentPreRun: prerunEnableDebugLogger,
		Run:              Record,
	}

	viewCmd = &cobra.Command{
		Use:              "view [folder]",
		Short:            "View performance test results from a specified folder or difference between two folders",
		Long:             "View performance test results from a specified folder or difference between two folders",
		PersistentPreRun: prerunEnableDebugLogger,
		Run:              View,
	}
)

var (
	configPath string
	profile    string
	outputPath string
	queryIndex int
	port       int
	debug      bool
)

func prerunEnableDebugLogger(_ *cobra.Command, _ []string) {
	if debug {
		logger.UseDebugLogger()
	}
}

func init() {
	rootCmd.AddCommand(recordCmd)
	recordCmd.Flags().StringVarP(&configPath,
		"config",
		"c",
		"",
		"config file for recording",
	)
	recordCmd.Flags().StringVarP(&profile, "profile", "p", "clickhouse", "profile for recording (default is clickhouse)")
	recordCmd.Flags().IntVarP(&queryIndex, "query", "q", -1, "query index for recording (default is all queries)")
	recordCmd.Flags().StringVarP(&outputPath, "output", "o", "", "output path for recording (default is test name)")
	recordCmd.Flags().BoolVarP(&debug, "debug", "", false, "enable debug mode")
	recordCmd.Args = cobra.ExactArgs(1)

	rootCmd.AddCommand(viewCmd)
	viewCmd.Flags().IntVarP(&port, "port", "p", 2323, "optional port for viewing (default is 2323)")
	viewCmd.Flags().BoolVarP(&debug, "debug", "", false, "enable debug mode")
	viewCmd.Args = cobra.MaximumNArgs(2)
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		logger.Log.Errorf("Error executing root command: %v", err)
		os.Exit(1)
	}
}
