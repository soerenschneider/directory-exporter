# directory-exporter

[![Go Report Card](https://goreportcard.com/badge/github.com/soerenschneider/directory-exporter)](https://goreportcard.com/report/github.com/soerenschneider/directory-exporter)
![test-workflow](https://github.com/soerenschneider/directory-exporter/actions/workflows/test.yaml/badge.svg)
![release-workflow](https://github.com/soerenschneider/directory-exporter/actions/workflows/release-container.yaml/badge.svg)
![golangci-lint-workflow](https://github.com/soerenschneider/directory-exporter/actions/workflows/golangci-lint.yaml/badge.svg)

A Prometheus exporter that watches directories on the local filesystem

## Features

🔍 Watches multiple dirs and  their files for changes<br/>
🎯 Filters can be supplied that both in- or exclude files / subdirectories<br/>
🔭 Directory information is exposed as Prometheus metrics<br/>

## Use Cases

Get alerted<br/>
⚠️ when a directory contains files / subdirectories that should not be there<br/>
👻 when a directory does not contain files / subdirectories that should be there<br/>
💥 when the size of an object does not lie within a given threshold<br/>

## Installation

### Docker / Podman
````shell
$ git clone https://github.com/soerenschneider/directory-exporter.git
$ cd directory-exporter
$ docker run -v $(pwd)/contrib:/config ghcr.io/soerenschneider/directory-exporter:main -config /config/directory-exporter.json
````

### Binaries
Head over to the [prebuilt binaries](https://github.com/soerenschneider/directory-exporter/releases) and download the correct binary for your system.

### From Source
As a prerequisite, you need to have [Golang SDK](https://go.dev/dl/) installed. After that, you can install directory-exporter from source by invoking:
```text
$ go install github.com/soerenschneider/directory-exporter@latest
```

## Configuration
A minimal example can be found [here](contrib/directory-exporter.json). 

To read about the configuration options, head over to the [configuration section](docs/configuration.md)

## Exposed Metrics

All metrics are prefixed with `directory_exporter`

| Name                              | Type     | Labels | Help                                                               |
|-----------------------------------|----------|--------|--------------------------------------------------------------------|
| file_count_total                  | GaugeVec | dir    | The total number of files found recursively under given directory  |
| file_size_bytes                   | GaugeVec | dir    | The size of all files that have been included or not been excluded |
| dir_size_bytes                    | Counters | dir    | The size of all files in the directory, even excluded files        |
| excluded_files_total              | GaugeVec | dir    | The total number of excluded files under given directory           |
| errors_total                      | GaugeVec | dir    | Errors while trying to access a directory                          |
| files_next_scan_timestamp_seconds | GaugeVec | dir    | Timestamp when next scan for given dir is started                  |
| files_scan_process_seconds        | GaugeVec | dir    | Seconds taken to scan given directory                              |
| heartbeat_seconds                 | Gauge    | -      | Continuous heartbeat of the exporter                               |

## CHANGELOG
The changelog can be found [here](CHANGELOG.md)
