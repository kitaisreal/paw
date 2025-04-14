package driver

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

type ClickHouseDriver struct {
	Host string
	Port int
	Auth clickhouse.Auth
	conn clickhouse.Conn
}

const (
	clickhouseDriverName            = "clickhouse"
	clickhouseDriverDefaultHost     = "127.0.0.1"
	clickhouseDriverHostSettingName = "host"
	clickhouseDriverDefaultPort     = 9000
	clickhouseDriverPortSettingName = "port"
)

func NewClickHouseDriver(host string, port int, settings Settings) (*ClickHouseDriver, error) {
	auth := clickhouse.Auth{}
	options := &clickhouse.Options{
		Addr:     []string{fmt.Sprintf("%s:%d", host, port)},
		Auth:     auth,
		Protocol: clickhouse.HTTP,
		Settings: settings,
	}

	conn, err := clickhouse.Open(options)
	if err != nil {
		return nil, err
	}

	c := &ClickHouseDriver{Host: host, Port: port, Auth: auth, conn: conn}

	return c, nil
}

func (c *ClickHouseDriver) Run(ctx context.Context, command string) (ExecutionTime, error) {
	var serverDuration time.Duration
	clickhouseCtx := clickhouse.Context(ctx, clickhouse.WithProgress(func(progress *clickhouse.Progress) {
		serverDuration += progress.Elapsed
	}))

	queryStartTime := time.Now()

	rows, err := c.conn.Query(clickhouseCtx, command)
	if err != nil {
		return ExecutionTime{}, fmt.Errorf("%s driver query error: %w", clickhouseDriverName, err)
	}

	defer rows.Close()
	for rows.Next() {
	}

	if err := rows.Err(); err != nil {
		return ExecutionTime{}, fmt.Errorf("%s driver rows final error: %w", clickhouseDriverName, err)
	}

	clientDuration := time.Since(queryStartTime)
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
