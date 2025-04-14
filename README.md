# paw

paw (Performance Analysis Workbench) is a very simple tool for performance testing, analysis, and continuous iteration over results.

The tool is tailored for such workflow:
1. Run performance tests and collect data (for example, collect on-CPU, off-CPU flame graphs)
2. Analyze collected data to find spots for potential optimizations
3. Implement optimizations
4. Re-run performance tests, recollect data, and compare the old runs with the new ones.
5. Repeat :)

## Usage Example

By default, config is not required, `clickhouse` driver is used by default, and all drivers and collectors will have default settings.

Config `config.yaml` example:
```
profiles:
  - name: clickhouse
    driver: clickhouse
    settings:
      host: 127.0.0.1
      port: 9000
  - name: clickhouse_scatter_aggregation
    driver: clickhouse
    settings:
      host: 127.0.0.1
      port: 9000
      force_scatter_aggregation: 1
  - name: clickhouse_aggregation_scatter_copy_chunks
    driver: clickhouse
    settings:
      host: 127.0.0.1
      port: 9000
      aggregation_scatter_based_on_statistics: 1
      aggregation_default_scatter_algorithm: copy_chunks
  - name: clickhouse_aggregation_scatter_indexes_info
    driver: clickhouse
    settings:
      host: 127.0.0.1
      port: 9000
      aggregation_scatter_based_on_statistics: 1
      aggregation_default_scatter_algorithm: indexes_info
collector_profiles:
  - name: cpu_flamegraph
    collector: cpu_flamegraph
    settings:
      build_seconds: 5
  - name: off_cpu_flamegraph
    collector: off_cpu_flamegraph
    settings:
      build_seconds: 5
settings:
  query_measure_runs: 5
```

Test `clickbench_simple.yaml` example:
```
name: ClickBenchSimple
collectors:
  - cpu_flamegraph
  - off_cpu_flamegraph
queries:
  - SELECT COUNT(*) FROM hits WHERE URL LIKE '%google%';
```

Run `clickbench_simple.yaml` test using config from `config.yaml` and output to `clickbench_simple_result` folder:
```
./paw record clickbench_simple.yaml -c config.yaml -o clickbench_simple_result
```

After this, you can view results in `clickbench_simple_result` folder using web UI:
```
./paw view clickbench_simple_result
```

After this, you found some places for optimizations, implemented them and want to re-run test to see if there is performance improvement:
```
./paw record clickbench_simple.yaml -c config.yaml -o clickbench_simple_result_updated
```

After this, you can view diff between old and new results, and compare collected flame graphs and other statistics:
```
./paw view clickbench_simple_result clickbench_simple_result_updated
```

## Example commands

Record using test file `clickbench.yaml` config file `config/config.yaml` and output to `paw_test_result` folder:
```
sudo ./paw record clickbench.yaml -c config.yaml -o paw_test_result
```

Same as above, but run in `debug` mode to have more debug information and trace all collectors commands:
```
sudo ./paw record clickbench.yaml -c config.yaml -o paw_test_result --debug
```

View results from `paw_test_result` folder using web UI:
```
sudo ./paw view paw_test_result
```

View diff between results from `paw_test_result_lhs` folder and `paw_test_result_rhs` folder using web UI:
```
sudo ./paw view paw_test_result_lhs paw_test_result_rhs
```

## Motivation

I wanted a tool to help me find places for performance optimizations in ClickHouse.

Main points why I decided to implement my own solution:
1. Tailored to my workflow. My usual workflow is to record the performance of queries (build CPU/off-CPU flame graphs), then analyze them to find potential places for improvements, make performance improvements, record once again, and check that there are no regressions and are only improvements. Then repeat.
2. The tool needs to be a single binary, quick, and easy to use. I need to use it during development without any pain. Potentially after every ClickHouse rebuild. I need to avoid complex configurations, setup of multiple services that must communicate with each other, and other stuff like that.
3. It will be possible to implement a complex view for `diff` command that can compare results from multiple test runs. This is very important for quick iteration during development.
4. Run tests with different settings. For example, I want to run a performance test suite with 10 different ClickHouse profiles, and each profile will have a different combination of settings.
5. The tool needs to have enough flexibility so I will be able to use it for different databases/engines, add new drivers, and add new collectors. I usually optimize ClickHouse, so the tool can be the first battle-tested to optimize ClickHouse's performance. ClickHouse is already heavily optimized, so if the tool will be useful to find places for optimizations, then it can be adapted for other databases/engines.

## Contacts

If you have any questions or suggestions, you can contact me at kitaetoya@gmail.com.
