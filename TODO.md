Major features:

1. Support client-server setup
2. Improve collectors post processing data performance. Currently each collector needs to collect and process collected data in `Collect` method. It is possible to only collect data in `Collect` method, and schedule post processing job in separate goroutines. At the end of `record` wait for all such jobs to finish.

Minor features:

1. Use analog of std::nth_element to find median in stats instead of sorting
2. Add cleanup for drivers, similar to collectors
3. Refactor collector profile name and collector name
4. Refactor collector settings
5. Collectors save only sample of execution times

View:

1. Add more statistics
2. Add p-value test to check for significant difference between two tests

Record:

1. Add more collectors (mpstat)
2. Allow to specify min number of query runs for collectors together with build time
3. CPU flamegraph collector per cpu perf record
4. Queries parameterization

CI:

1. Enable go-sec

