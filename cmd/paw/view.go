package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kitaisreal/paw/internal/config"
	"github.com/kitaisreal/paw/internal/driver"
	"github.com/kitaisreal/paw/internal/logger"
	"github.com/kitaisreal/paw/internal/stats"
	"github.com/spf13/cobra"
)

type ViewHTMLPages struct {
	IndexHTML                     string
	QueryNumberToQueryDetailsHTML map[int]string
}

func View(cmd *cobra.Command, args []string) {
	if len(args) == 1 {
		ViewSingle(cmd, args)
	} else if len(args) == 2 {
		ViewDiff(cmd, args)
	} else {
		logger.Log.Errorf("Invalid number of arguments: %d", len(args))
		os.Exit(1)
	}
}

func ViewSingle(_ *cobra.Command, args []string) {
	folder := convertPathToFolder(args[0])
	viewSingleHTMLPages := buildViewSingleHTMLPages(folder)

	logger.Log.Infof("Viewing performance test results from folder: %s using port: %d", folder, port)
	runViewServer(viewSingleHTMLPages, folder, "")
}

func ViewDiff(_ *cobra.Command, args []string) {
	lhsFolder := convertPathToFolder(args[0])
	rhsFolder := convertPathToFolder(args[1])
	viewHTMLPages := buildViewDiffHTMLPages(lhsFolder, rhsFolder)

	logger.Log.Infof("Viewing performance difference test results from folders lhs: %s and rhs: %s using port: %d",
		lhsFolder,
		rhsFolder,
		port,
	)
	runViewServer(viewHTMLPages, lhsFolder, rhsFolder)
}

func runViewServer(viewHTMLPages ViewHTMLPages, lhsFolder string, rhsFolder string) {
	staticHandler := http.FileServer(http.FS(staticFS))
	http.Handle("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		}

		staticHandler.ServeHTTP(w, r)
	}))

	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, viewHTMLPages.IndexHTML)
	})

	http.HandleFunc("/query/", func(w http.ResponseWriter, r *http.Request) {
		queryNumber, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/query/"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid query number", http.StatusBadRequest)
			return
		}

		queryDetailsPage, ok := viewHTMLPages.QueryNumberToQueryDetailsHTML[int(queryNumber)]
		if !ok {
			http.Error(w, "Query not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, queryDetailsPage)
	})

	http.HandleFunc("/file/", func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()

		folderStr := queryParams.Get("folder")
		if folderStr == "" {
			http.Error(w, "Missing folder parameter", http.StatusBadRequest)
			return
		}

		var folder string

		if folderStr == "lhs" {
			folder = lhsFolder
		} else if folderStr == "rhs" {
			folder = rhsFolder
		} else {
			http.Error(w, "Invalid folder parameter", http.StatusBadRequest)
			return
		}

		queryNumberStr := queryParams.Get("query")
		if queryNumberStr == "" {
			http.Error(w, "Missing query parameter", http.StatusBadRequest)
			return
		}

		queryNumber, err := strconv.ParseUint(queryNumberStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid query number", http.StatusBadRequest)
			return
		}

		collectorName := queryParams.Get("collector")
		if collectorName == "" {
			http.Error(w, "Missing collector parameter", http.StatusBadRequest)
			return
		}

		filename := queryParams.Get("file")
		if filename == "" {
			http.Error(w, "Missing file parameter", http.StatusBadRequest)
			return
		}

		filePath := filepath.Join(folder, fmt.Sprintf("query_%d", queryNumber), collectorName, filename)
		http.ServeFile(w, r, filePath)
	})

	logger.Log.Debugf("Starting HTTP server on port %d", port)
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}
	err := srv.ListenAndServe()
	if err != nil {
		logger.Log.Errorf("Failed to start HTTP server: %v", err)
		os.Exit(1)
	}
}

type ViewSingleData struct {
	FolderName string
	Records    []QueryRecordWithStats
}

func buildViewSingleHTMLPages(folder string) ViewHTMLPages {
	records, err := parseTestFolder(folder)
	if err != nil {
		logger.Log.Errorf("Failed to parse test folder %s: %v", folder, err)
		os.Exit(1)
	}

	data := ViewSingleData{
		FolderName: folder,
		Records:    records,
	}

	viewSingleHTMLBuffer := bytes.NewBuffer(nil)
	err = viewSingleTemplate.ExecuteTemplate(viewSingleHTMLBuffer, "base.html", data)
	if err != nil {
		logger.Log.Errorf("Failed to execute template: %v", err)
		os.Exit(1)
	}

	viewSingleHTML := viewSingleHTMLBuffer.String()

	var wg sync.WaitGroup
	var mu sync.Mutex
	queryNumberToHTMLPage := map[int]string{}

	for _, record := range records {
		wg.Add(1)

		go func(record QueryRecordWithStats) {
			defer wg.Done()

			queryDetailsPageHTMLBuffer := bytes.NewBuffer(nil)
			err := viewSingleQueryDetailsTemplate.ExecuteTemplate(queryDetailsPageHTMLBuffer, "base.html", record)
			if err != nil {
				logger.Log.Errorf("Failed to execute template: %v", err)
				os.Exit(1)
			}

			mu.Lock()
			defer mu.Unlock()
			queryNumberToHTMLPage[record.Record.QueryNumber] = queryDetailsPageHTMLBuffer.String()
		}(record)
	}

	wg.Wait()

	return ViewHTMLPages{
		IndexHTML:                     viewSingleHTML,
		QueryNumberToQueryDetailsHTML: queryNumberToHTMLPage,
	}
}

type ViewDiffData struct {
	LHSFolder        string
	RHSFolder        string
	QueryRecordPairs []QueryRecordPairWithStats
}

func buildViewDiffHTMLPages(lhsFolder string, rhsFolder string) ViewHTMLPages {
	lhsRecords, err := parseTestFolder(lhsFolder)
	if err != nil {
		logger.Log.Errorf("Failed to parse lhs test folder %s: %v", lhsFolder, err)
		os.Exit(1)
	}

	rhsRecords, err := parseTestFolder(rhsFolder)
	if err != nil {
		logger.Log.Errorf("Failed to parse rhs test folder %s: %v", rhsFolder, err)
		os.Exit(1)
	}

	queryRecordPairs := buildQueryRecordsDiff(lhsRecords, rhsRecords)
	viewData := ViewDiffData{
		LHSFolder:        lhsFolder,
		RHSFolder:        rhsFolder,
		QueryRecordPairs: queryRecordPairs,
	}

	viewDiffHTMLBuffer := bytes.NewBuffer(nil)
	err = viewDiffTemplate.ExecuteTemplate(viewDiffHTMLBuffer, "base.html", viewData)
	if err != nil {
		logger.Log.Errorf("Failed to execute template: %v", err)
		os.Exit(1)
	}

	viewDiffHTML := viewDiffHTMLBuffer.String()

	var wg sync.WaitGroup
	var mu sync.Mutex
	queryNumberToHTMLPage := map[int]string{}

	for _, queryRecordPair := range queryRecordPairs {
		wg.Add(1)

		go func(queryRecordPair QueryRecordPairWithStats) {
			defer wg.Done()

			queryDetailsPageHTMLBuffer := bytes.NewBuffer(nil)
			err := viewDiffQueryDetailsTemplate.ExecuteTemplate(queryDetailsPageHTMLBuffer, "base.html", queryRecordPair)
			if err != nil {
				logger.Log.Errorf("Failed to execute template: %v", err)
				os.Exit(1)
			}

			mu.Lock()
			defer mu.Unlock()
			queryNumberToHTMLPage[queryRecordPair.LHS.Record.QueryNumber] = queryDetailsPageHTMLBuffer.String()
		}(queryRecordPair)
	}

	wg.Wait()

	return ViewHTMLPages{
		IndexHTML:                     viewDiffHTML,
		QueryNumberToQueryDetailsHTML: queryNumberToHTMLPage,
	}
}

func buildQueryRecordsDiff(
	lhsRecords []QueryRecordWithStats,
	rhsRecords []QueryRecordWithStats,
) []QueryRecordPairWithStats {
	lhsMap := map[int]QueryRecordWithStats{}
	for _, lhsRecord := range lhsRecords {
		lhsMap[lhsRecord.Record.QueryNumber] = lhsRecord
	}

	queryPairs := []QueryRecordPairWithStats{}
	for _, rhsRecord := range rhsRecords {
		if lhsRecord, ok := lhsMap[rhsRecord.Record.QueryNumber]; ok {
			queryPairs = append(queryPairs, QueryRecordPairWithStats{LHS: lhsRecord, RHS: rhsRecord})
		}
	}

	sort.Slice(queryPairs, func(i, j int) bool {
		return queryPairs[i].LHS.Record.QueryNumber < queryPairs[j].LHS.Record.QueryNumber
	})

	return queryPairs
}

func parseTestFolder(folder string) ([]QueryRecordWithStats, error) {
	var records []QueryRecordWithStats

	const (
		queryFolderPrefix = "query_"
		queryRecordFile   = "query_record.json"
	)

	files, err := os.ReadDir(folder)
	if err != nil {
		return nil, fmt.Errorf("error reading directory %s: %w", folder, err)
	}

	for _, file := range files {
		if !file.IsDir() || !strings.HasPrefix(file.Name(), queryFolderPrefix) {
			continue
		}

		queryIndexStr := strings.TrimPrefix(file.Name(), queryFolderPrefix)
		_, err := strconv.ParseUint(queryIndexStr, 10, 64)
		if err != nil {
			continue
		}

		queryRecordPath := filepath.Join(folder, file.Name(), queryRecordFile)
		queryRecord, err := deserializeQueryRecord(queryRecordPath)
		if err != nil {
			return nil, fmt.Errorf("error reading query record file %s: %w", queryRecordPath, err)
		}

		records = append(records, QueryRecordWithStats{
			Record: queryRecord,
			Stats:  stats.GetStats(queryRecord.ExecutionTimes),
		})
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Record.QueryNumber < records[j].Record.QueryNumber
	})

	return records, nil
}

func convertPathToFolder(path string) string {
	if strings.HasSuffix(path, ".yaml") {
		test, err := config.ParseTestFileYaml(path)
		if err != nil {
			logger.Log.Errorf("Failed to parse test file %s: %v", path, err)
			os.Exit(1)
		}

		return test.Name
	}

	return path
}

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

var (
	viewSingleTemplate             *template.Template
	viewSingleQueryDetailsTemplate *template.Template
	viewDiffTemplate               *template.Template
	viewDiffQueryDetailsTemplate   *template.Template
)

func init() {
	var getRelativeDiff = func(lhs, rhs float64) float64 {
		if lhs == 0 {
			lhs = 1e-6
		}

		return (rhs - lhs) / lhs * 100
	}

	var getMedianRowClass = func(lhs, rhs float64) string {
		relativeDifference := getRelativeDiff(lhs, rhs)

		if math.Abs(relativeDifference) > 5 {
			if relativeDifference > 0 {
				return "significant-negative-diff"
			}

			return "significant-positive-diff"
		}

		return ""
	}

	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("invalid dict call")
			}

			dict := make(map[string]any, len(values)/2)

			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}

				dict[key] = values[i+1]
			}

			return dict, nil
		},
		"getServerDurationMilliseconds": func(executionTime driver.ExecutionTime) float64 {
			return float64(executionTime.ServerDuration) / 1e6
		},
		"getMinServerDurationMilliseconds": func(s stats.Stats) float64 {
			return s.GetMinServerDurationMilliseconds()
		},
		"getMaxServerDurationMilliseconds": func(s stats.Stats) float64 {
			return s.GetMaxServerDurationMilliseconds()
		},
		"getMeanServerDurationMilliseconds": func(s stats.Stats) float64 {
			return s.GetMeanServerDurationMilliseconds()
		},
		"getMedianServerDurationMilliseconds": func(s stats.Stats) float64 {
			return s.GetMedianServerDurationMilliseconds()
		},
		"getStdDevServerDurationMilliseconds": func(s stats.Stats) float64 {
			return s.GetStdDevServerDurationMilliseconds()
		},
		"getRelativeMedianServerDurationDiff": func(lhs stats.Stats, rhs stats.Stats) float64 {
			return getRelativeDiff(lhs.GetMedianServerDurationMilliseconds(), rhs.GetMedianServerDurationMilliseconds())
		},
		"getMedianServerDurationRowClass": func(lhs, rhs stats.Stats) string {
			return getMedianRowClass(lhs.GetMedianServerDurationMilliseconds(), rhs.GetMedianServerDurationMilliseconds())
		},
		"getClientDurationMilliseconds": func(executionTime driver.ExecutionTime) float64 {
			return float64(executionTime.ClientDuration) / 1e6
		},
		"getMinClientDurationMilliseconds": func(s stats.Stats) float64 {
			return s.GetMinClientDurationMilliseconds()
		},
		"getMaxClientDurationMilliseconds": func(s stats.Stats) float64 {
			return s.GetMaxClientDurationMilliseconds()
		},
		"getMeanClientDurationMilliseconds": func(s stats.Stats) float64 {
			return s.GetMeanClientDurationMilliseconds()
		},
		"getMedianClientDurationMilliseconds": func(s stats.Stats) float64 {
			return s.GetMedianClientDurationMilliseconds()
		},
		"getStdDevClientDurationMilliseconds": func(s stats.Stats) float64 {
			return s.GetStdDevClientDurationMilliseconds()
		},
		"getRelativeMedianClientDurationDiff": func(lhs stats.Stats, rhs stats.Stats) float64 {
			return getRelativeDiff(lhs.GetMedianClientDurationMilliseconds(), rhs.GetMedianClientDurationMilliseconds())
		},
		"getMedianClientDurationRowClass": func(lhs, rhs stats.Stats) string {
			return getMedianRowClass(lhs.GetMedianClientDurationMilliseconds(), rhs.GetMedianClientDurationMilliseconds())
		},
	}

	var buildTemplate = func(pageTemplate string) *template.Template {
		template, err := template.New("base.html").Funcs(funcMap).ParseFS(templateFS,
			"templates/base.html",
			pageTemplate,
			"templates/tables.html",
			"templates/collector_tables.html",
			"templates/iframes_scroll.html",
		)
		if err != nil {
			panic(fmt.Errorf("error parsing templates: %w", err))
		}

		return template
	}

	viewSingleTemplate = buildTemplate("templates/view_single.html")
	viewSingleQueryDetailsTemplate = buildTemplate("templates/view_single_query_details.html")
	viewDiffTemplate = buildTemplate("templates/view_diff.html")
	viewDiffQueryDetailsTemplate = buildTemplate("templates/view_diff_query_details.html")
}
