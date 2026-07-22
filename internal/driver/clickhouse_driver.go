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

// clickHouseResponse is the parsed result of a single HTTP request.
type clickHouseResponse struct {
	body           []byte
	queryID        string
	serverDuration time.Duration
	clientDuration time.Duration
}

// doRequest posts command to the ClickHouse HTTP interface with the driver
// settings applied as query parameters and parses the response.
func (c *ClickHouseDriver) doRequest(ctx context.Context, command string) (clickHouseResponse, error) {
	queryURL, err := url.Parse(fmt.Sprintf("http://%s:%d/", c.Host, c.Port))
	if err != nil {
		return clickHouseResponse{}, fmt.Errorf("%s driver url parse error: %w", clickhouseDriverName, err)
	}

	queryParams := make(url.Values, len(c.settings))
	for name, value := range c.settings {
		queryParams.Set(name, fmt.Sprintf("%v", value))
	}
	queryURL.RawQuery = queryParams.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, queryURL.String(), strings.NewReader(command))
	if err != nil {
		return clickHouseResponse{}, fmt.Errorf("%s driver request create error: %w", clickhouseDriverName, err)
	}

	queryStartTime := time.Now()

	resp, err := c.client.Do(req)
	if err != nil {
		return clickHouseResponse{}, fmt.Errorf("%s driver query error: %w", clickhouseDriverName, err)
	}
	defer resp.Body.Close()

	// Read full response body to ensure query completes
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return clickHouseResponse{}, fmt.Errorf("%s driver response read error: %w", clickhouseDriverName, err)
	}

	clientDuration := time.Since(queryStartTime)

	if resp.StatusCode != http.StatusOK {
		return clickHouseResponse{}, fmt.Errorf("%s driver query error: HTTP %d: %s",
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

	return clickHouseResponse{
		body:           body,
		queryID:        resp.Header.Get("X-ClickHouse-Query-Id"),
		serverDuration: serverDuration,
		clientDuration: clientDuration,
	}, nil
}

func (c *ClickHouseDriver) Run(ctx context.Context, query string) (ExecutionTime, error) {
	resp, err := c.doRequest(ctx, query)
	if err != nil {
		return ExecutionTime{}, err
	}

	return ExecutionTime{ClientDuration: resp.clientDuration, ServerDuration: resp.serverDuration}, nil
}

func (c *ClickHouseDriver) SupportsProfileEvents() bool {
	return true
}

func (c *ClickHouseDriver) RunWithProfileEvents(
	ctx context.Context,
	query string,
) (ExecutionTime, ProfileEvents, error) {
	// Run the benchmark query under the profile settings. ClickHouse assigns a
	// query id which is returned in the X-ClickHouse-Query-Id response header.
	resp, err := c.doRequest(ctx, query)
	if err != nil {
		return ExecutionTime{}, nil, err
	}

	executionTime := ExecutionTime{ClientDuration: resp.clientDuration, ServerDuration: resp.serverDuration}

	if resp.queryID == "" {
		return ExecutionTime{}, nil, fmt.Errorf(
			"%s driver did not return a query id, cannot collect profile events",
			clickhouseDriverName,
		)
	}

	// query_log is buffered, force a flush so the finished query is queryable.
	if _, err := c.doRequest(ctx, "SYSTEM FLUSH LOGS query_log"); err != nil {
		return ExecutionTime{}, nil, fmt.Errorf("%s driver flush query_log error: %w", clickhouseDriverName, err)
	}

	// output_format_json_quote_64bit_integers = 0 keeps ProfileEvents values as
	// numbers so they decode into uint64 instead of quoted strings.
	profileEventsQuery := fmt.Sprintf(
		"SELECT ProfileEvents FROM system.query_log "+
			"WHERE query_id = '%s' AND type = 'QueryFinish' AND is_initial_query = 1 "+
			"ORDER BY event_time DESC LIMIT 1 "+
			"SETTINGS output_format_json_quote_64bit_integers = 0 FORMAT JSONEachRow",
		resp.queryID,
	)

	profileEventsResp, err := c.doRequest(ctx, profileEventsQuery)
	if err != nil {
		return ExecutionTime{}, nil, fmt.Errorf("%s driver profile events query error: %w", clickhouseDriverName, err)
	}

	profileEventsBody := strings.TrimSpace(string(profileEventsResp.body))
	if profileEventsBody == "" {
		return ExecutionTime{}, nil, fmt.Errorf(
			"%s driver did not find a query_log entry for query id %s",
			clickhouseDriverName,
			resp.queryID,
		)
	}

	var row struct {
		ProfileEvents ProfileEvents `json:"ProfileEvents"`
	}
	if err := json.Unmarshal([]byte(profileEventsBody), &row); err != nil {
		return ExecutionTime{}, nil, fmt.Errorf("%s driver failed to parse profile events: %w", clickhouseDriverName, err)
	}

	return executionTime, row.ProfileEvents, nil
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
