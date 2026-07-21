package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ClickHouseDriver struct {
	Host     string
	Port     int
	client   *http.Client
	settings Settings
}

const (
	clickhouseDriverName            = "clickhouse"
	clickhouseDriverDefaultHost     = "127.0.0.1"
	clickhouseDriverHostSettingName = "host"
	clickhouseDriverDefaultPort     = 8123
	clickhouseDriverPortSettingName = "port"
)

func NewClickHouseDriver(host string, port int, settings Settings) (*ClickHouseDriver, error) {
	client := &http.Client{}
	c := &ClickHouseDriver{Host: host, Port: port, client: client, settings: settings}
	return c, nil
}

func (c *ClickHouseDriver) Run(ctx context.Context, command string) (ExecutionTime, error) {
	queryURL, err := url.Parse(fmt.Sprintf("http://%s:%d/", c.Host, c.Port))
	if err != nil {
		return ExecutionTime{}, fmt.Errorf("%s driver url parse error: %w", clickhouseDriverName, err)
	}

	params := queryURL.Query()
	for name, value := range c.settings {
		params.Set(name, fmt.Sprintf("%v", value))
	}
	queryURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, queryURL.String(), strings.NewReader(command))
	if err != nil {
		return ExecutionTime{}, fmt.Errorf("%s driver request create error: %w", clickhouseDriverName, err)
	}

	queryStartTime := time.Now()

	resp, err := c.client.Do(req)
	if err != nil {
		return ExecutionTime{}, fmt.Errorf("%s driver query error: %w", clickhouseDriverName, err)
	}
	defer resp.Body.Close()

	// Read full response body to ensure query completes
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ExecutionTime{}, fmt.Errorf("%s driver response read error: %w", clickhouseDriverName, err)
	}

	clientDuration := time.Since(queryStartTime)

	if resp.StatusCode != http.StatusOK {
		return ExecutionTime{}, fmt.Errorf("%s driver query error: HTTP %d: %s",
			clickhouseDriverName,
			resp.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	// Parse server duration from X-ClickHouse-Summary header
	var serverDuration time.Duration
	if summary := resp.Header.Get("X-ClickHouse-Summary"); summary != "" {
		var summaryData struct {
			ElapsedNs string `json:"elapsed_ns"`
		}
		if err := json.Unmarshal([]byte(summary), &summaryData); err == nil {
			if elapsedNs, err := strconv.ParseUint(summaryData.ElapsedNs, 10, 64); err == nil {
				serverDuration = time.Duration(elapsedNs) * time.Nanosecond
			}
		}
	}

	return ExecutionTime{ClientDuration: clientDuration, ServerDuration: serverDuration}, nil
}

func init() {
	RegisterDriver(clickhouseDriverName, func(settings Settings) (Driver, error) {
		host := clickhouseDriverDefaultHost
		port := clickhouseDriverDefaultPort

		if hostAny, ok := settings[clickhouseDriverHostSettingName]; ok {
			host, ok = hostAny.(string)
			if !ok {
				return nil, fmt.Errorf("%s driver profile setting '%s' is not string",
					clickhouseDriverName,
					clickhouseDriverHostSettingName,
				)
			}
		}
		if portAny, ok := settings[clickhouseDriverPortSettingName]; ok {
			port, ok = portAny.(int)
			if !ok {
				return nil, fmt.Errorf("%s driver profile setting '%s' is not int",
					clickhouseDriverName,
					clickhouseDriverPortSettingName,
				)
			}
		}

		driverSettings := Settings{}
		for name, value := range settings {
			if name != clickhouseDriverHostSettingName && name != clickhouseDriverPortSettingName {
				driverSettings[name] = value
			}
		}

		return NewClickHouseDriver(host, port, driverSettings)
	})
}
