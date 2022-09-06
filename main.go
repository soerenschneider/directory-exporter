package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
	"time"
)

const (
	defaultPrometheusListen = ":2112"
	defaultTickerFrequency  = 30 * time.Second
	defaultNamespace        = "directory_exporter"
	defaultConfigFile       = "directory-exporter.json"
)

var (
	metricDirFileCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: defaultNamespace,
		Name:      "file_count_total",
		Help:      "The total number of files found recursively under given directory",
	}, []string{"dir"})

	metricDirExcludedFileCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: defaultNamespace,
		Name:      "excluded_files_total",
		Help:      "The total number of excluded files under given directory",
	}, []string{"dir"})

	metricDirErrors = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: defaultNamespace,
		Name:      "errors_total",
		Help:      "Errors while trying to access a directory",
	}, []string{"dir"})

	metricNextScan = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: defaultNamespace,
		Name:      "files_next_scan_timestamp_seconds",
		Help:      "Timestamp when next scan for given dir is started",
	}, []string{"dir"})

	metricScanTimeTaken = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: defaultNamespace,
		Name:      "files_scan_process_seconds",
		Help:      "Seconds taken to scan given directory",
	}, []string{"dir"})

	metricHeartbeat = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "heartbeat_seconds",
		Help: "Continuous heartbeat of the exporter",
	})

	dirScanFrequency      = time.Duration(5) * time.Minute
	prometheusListen      = defaultPrometheusListen
	configuredDirectories = make(map[string]*DirConfig)
	config                Config

	BuildVersion string
	CommitHash   string
)

type Config struct {
	Dirs []*DirConfig `json:"dirs"`
}

type DirConfig struct {
	Frequency         *time.Duration `json:"frequency,omitempty"`
	Dir               string         `json:"dir"`
	OnlyFiles         bool           `json:"only_files"`
	ExcludeFiles      []string       `json:"exclude_files"`
	IncludeFiles      []string       `json:"include_files"`
	NextScan          time.Time
	RegexExcludeFiles []regexp.Regexp
	RegexIncludeFiles []regexp.Regexp
}

func updateMetricsForDir(directory string) {
	dirConf := configuredDirectories[directory]

	// This is a workaround for symlinks. If we detected a change on a directory that has no configuration attached
	// to it, it means the directory we found updates for is actually symlink. We'll try to set the same config
	// of the defined dir to the symlinked directory we found updates for.
	if dirConf == nil {
		log.Warn().Msgf("No config found for directory '%s', trying to find symlink and attach config to this directory.", directory)
		pathUpdateDetected, _ := os.Stat(directory)
		for dir, dirConf := range configuredDirectories {
			pathConfigured, _ := os.Stat(dir)
			if os.SameFile(pathConfigured, pathUpdateDetected) {
				log.Info().Msgf("Found symlink from '%s' -> '%s', attaching config for '%s'", directory, dir, dir)
				configuredDirectories[directory] = dirConf
				break
			}
		}

		// If the code above did not work and we could not set an existing config, we'll try to set an implicit
		// default config. This should actually not happen but handling it makes the code resilient.
		dirConf = configuredDirectories[directory]
		if dirConf == nil {
			log.Warn().Msgf("Building default dir config for directory '%s'. This should not happen.", directory)
			dirConf = &DirConfig{
				Frequency: &dirScanFrequency,
				Dir:       directory,
			}
			configuredDirectories[directory] = dirConf
		}
	}

	scanTimeStart := time.Now()
	fileCnt, err := getFilesCount(directory, dirConf)
	scanTimeTotal := time.Now().Sub(scanTimeStart)
	if err != nil {
		metricDirErrors.WithLabelValues(dirConf.Dir).Inc()
		log.Error().Err(err).Msgf("Error getting count of files for '%s'", dirConf.Dir)
		fileCnt = -1
	}

	dirConf.NextScan = time.Now().Add(*dirConf.Frequency)

	metricScanTimeTaken.WithLabelValues(dirConf.Dir).Set(scanTimeTotal.Seconds())
	metricDirFileCount.WithLabelValues(dirConf.Dir).Set(float64(fileCnt))
	metricNextScan.WithLabelValues(dirConf.Dir).Set(float64(dirConf.NextScan.Unix()))
}

func getFilesCount(directory string, dirConfig *DirConfig) (int, error) {
	cnt := 0
	if len(dirConfig.RegexExcludeFiles) > 0 {
		metricDirExcludedFileCount.WithLabelValues(directory).Set(0)
	}

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error().Err(err).Msgf("Error iterating path '%s'", path)
			metricDirErrors.WithLabelValues(directory).Inc()
		}

		if dirConfig.OnlyFiles {
			fileInfo, err := os.Stat(path)
			if err != nil {
				log.Error().Err(err).Msgf("Can't get stat for path '%s'", path)
			}
			if fileInfo.IsDir() {
				return nil
			}
		}

		if len(dirConfig.ExcludeFiles) > 0 {
			for _, regex := range dirConfig.RegexExcludeFiles {
				if regex.MatchString(path) {
					metricDirExcludedFileCount.WithLabelValues(directory).Inc()
					return nil
				}
			}
		}

		if len(dirConfig.IncludeFiles) > 0 {
			for _, regex := range dirConfig.RegexIncludeFiles {
				if regex.MatchString(path) {
					cnt += 1
					return nil
				}
			}
			// don't proceed if included files regex have been defined
			return nil
		}

		cnt += 1
		return nil
	})

	return cnt, err
}

func scanDirectories() {
	for dir, dirConf := range configuredDirectories {
		if dirConf.NextScan.After(time.Now()) {
			nextScan := dirConf.NextScan.Sub(time.Now())
			log.Debug().Msgf("Ignoring dir '%s' for '%v'", dir, nextScan)
		} else {
			updateMetricsForDir(dir)
		}
	}
}

func watchDirs() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not build watcher")
	}
	defer watcher.Close()

	for dir, _ := range configuredDirectories {
		log.Info().Msgf("Watching dir '%s'", dir)
		err := watcher.Add(dir)
		if err != nil {
			log.Fatal().Err(err).Msgf("Can not watch dir '%s'", dir)
		}
	}

	ticker := time.NewTicker(defaultTickerFrequency)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	scanDirectories()

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			path := filepath.Dir(event.Name)
			log.Debug().Msgf("Update detected on path '%s' (%s)", path, event.Name)
			updateMetricsForDir(path)

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			metricDirErrors.WithLabelValues("UNSPECIFIED").Inc()
			log.Error().Msgf("Error while watching configuredDirectories: %v", err)

		case <-ticker.C:
			metricHeartbeat.SetToCurrentTime()
			scanDirectories()

		case <-quit:
			log.Info().Msg("Caught signal, stopping.")
			ticker.Stop()
			return
		}
	}
}

func parseFlags() {
	var confFile *string
	confFile = flag.String("config", defaultConfigFile, "path to the JSON config file")
	debug := flag.Bool("debug", false, "sets log level to debug")
	version := flag.Bool("version", false, "Print version and exit")
	prometheusListen = *flag.String("listen", defaultPrometheusListen, "Listener for prometheus metrics handler")
	flag.Parse()

	if *version {
		fmt.Printf("%s (commit: %s)", BuildVersion, CommitHash)
		os.Exit(0)
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	content, err := os.ReadFile(*confFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error when trying to open config file '%s'", *confFile)
	}

	err = json.Unmarshal(content, &config)
	if err != nil {
		log.Fatal().Err(err).Msg("Error unmarshalling config")
	}

	buildRegexes()
}

func buildRegexes() {
	for _, dirConf := range config.Dirs {
		if len(dirConf.ExcludeFiles) > 0 {
			dirConf.RegexExcludeFiles = make([]regexp.Regexp, 0, len(dirConf.ExcludeFiles))
			log.Info().Msgf("Building %d exclude regexes for dir '%s'", len(dirConf.ExcludeFiles), dirConf.Dir)
			for _, exclusion := range dirConf.ExcludeFiles {
				reg, err := regexp.Compile(exclusion)
				if err != nil {
					log.Fatal().Err(err).Msgf("Invalid 'exclude' pattern supplied")
				}
				dirConf.RegexExcludeFiles = append(dirConf.RegexExcludeFiles, *reg)
			}
		}

		if len(dirConf.IncludeFiles) > 0 {
			dirConf.RegexIncludeFiles = make([]regexp.Regexp, 0, len(dirConf.IncludeFiles))
			log.Info().Msgf("Building %d include regexes for dir '%s'", len(dirConf.IncludeFiles), dirConf.Dir)
			for _, inclusion := range dirConf.IncludeFiles {
				reg, err := regexp.Compile(inclusion)
				if err != nil {
					log.Fatal().Err(err).Msgf("Invalid 'include' pattern supplied")
				}
				dirConf.RegexIncludeFiles = append(dirConf.RegexIncludeFiles, *reg)
			}
		}
	}
}

func startPromHandler() {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe(prometheusListen, nil)
		if err != nil {
			log.Fatal().Err(err).Msgf("Could not start listener")
		}
	}()
}

func main() {
	parseFlags()
	log.Info().Msgf("directory-exporter %s, commit %s", BuildVersion, CommitHash)

	startPromHandler()

	nextScan := time.Now()
	for _, dir := range config.Dirs {
		if dir.Frequency == nil {
			dir.Frequency = &dirScanFrequency
		}
		dir.NextScan = nextScan

		//  fixes symlinks: get the destination of every configured dir and use its result instead of blindly using
		// configured configuredDirectories.
		watchedDir := dir.Dir
		dest, err := filepath.EvalSymlinks(dir.Dir)
		if err == nil && dest != dir.Dir {
			log.Warn().Msgf("%s is a symlink to %s", dir.Dir, dest)
			watchedDir = dest
		}
		copied := dir
		configuredDirectories[watchedDir] = copied
	}
	watchDirs()
}
