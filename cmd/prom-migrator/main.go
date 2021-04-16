// This file and its contents are licensed under the Apache License 2.0.
// Please see the included NOTICE for copyright information and
// LICENSE for a copy of the license.

package main

import (
	"context"
	"fmt"
	"github.com/inhies/go-bytesize"
	"github.com/timescale/promscale/pkg/log"
	plan "github.com/timescale/promscale/pkg/migration-tool/planner"
	"github.com/timescale/promscale/pkg/migration-tool/reader"
	"github.com/timescale/promscale/pkg/migration-tool/utils"
	"github.com/timescale/promscale/pkg/migration-tool/writer"
	"github.com/timescale/promscale/pkg/version"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	migrationJobName     = "prom-migrator"
	progressMetricName   = "prom_migrator_progress"
	validMetricNameRegex = `^[a-zA-Z_:][a-zA-Z0-9_:]*$`
	defaultTimeout       = time.Minute * 5
)

type config struct {
	name               string
	mint               int64
	mintSec            int64
	maxt               int64
	maxtSec            int64
	maxSlabSizeBytes   int64
	maxSlabSize        string
	concurrentPulls    int
	concurrentPush     int
	readURL            string
	writeURL           string
	progressMetricName string
	progressMetricURL  string
	progressEnabled    bool

	// Auth.
	readerAuth utils.Auth
	writerAuth utils.Auth

	// Timeouts.
	readerTimeout time.Duration
	writerTimeout time.Duration
}

func main() {
	conf := new(config)
	args := os.Args[1:]
	if shouldProceed := parseArgs(args); !shouldProceed {
		os.Exit(0)
	}

	parseFlags(conf, os.Args[1:])

	if err := log.Init(log.Config{Format: "logfmt", Level: "debug"}); err != nil {
		fmt.Println("Version: ", version.PromMigrator)
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	log.Info("Version", version.PromMigrator)
	if err := validateConf(conf); err != nil {
		log.Error("msg", "could not parse flags", "error", err)
		os.Exit(1)
	}
	log.Info("msg", fmt.Sprintf("%v+", conf))
	if err := utils.SetAuthStore(utils.Read, conf.readerAuth.ToHTTPClientConfig()); err != nil {
		log.Error("msg", "could not set read-auth in authStore", "error", err)
		os.Exit(1)
	}
	if err := utils.SetAuthStore(utils.Write, conf.writerAuth.ToHTTPClientConfig()); err != nil {
		log.Error("msg", "could not set write-auth in authStore", "error", err)
		os.Exit(1)
	}

	planConfig := &plan.Config{
		Mint:               conf.mint,
		Maxt:               conf.maxt,
		JobName:            conf.name,
		SlabSizeLimitBytes: conf.maxSlabSizeBytes,
		NumStores:          conf.concurrentPulls,
		ProgressEnabled:    conf.progressEnabled,
		ProgressMetricName: conf.progressMetricName,
		ProgressMetricURL:  conf.progressMetricURL,
	}
	planner, proceed, err := plan.Init(planConfig)
	if err != nil {
		log.Error("msg", "could not create plan", "error", err)
		os.Exit(2)
	}
	if !proceed {
		os.Exit(0)
	}

	var (
		readErrChan  = make(chan error)
		writeErrChan = make(chan error)
		sigSlabRead  = make(chan *plan.Slab)
	)
	cont, cancelFunc := context.WithCancel(context.Background())
	read, err := reader.New(cont, conf.readURL, planner, conf.concurrentPulls, sigSlabRead)
	if err != nil {
		log.Error("msg", "could not create reader", "error", err)
		os.Exit(2)
	}
	write, err := writer.New(cont, conf.writeURL, conf.progressMetricName, conf.name, conf.concurrentPush, conf.progressEnabled, sigSlabRead)
	if err != nil {
		log.Error("msg", "could not create writer", "error", err)
		os.Exit(2)
	}

	read.Run(readErrChan)
	write.Run(writeErrChan)

loop:
	for {
		select {
		case err = <-readErrChan:
			if err != nil {
				cancelFunc()
				log.Error("msg", fmt.Errorf("running reader: %w", err).Error())
				os.Exit(2)
			}
		case err, ok := <-writeErrChan:
			cancelFunc() // As in any ideal case, the reader will always exit normally first.
			if ok {
				log.Error("msg", fmt.Errorf("running writer: %w", err).Error())
				os.Exit(2)
			}
			break loop
		}
	}

	log.Info("msg", "migration successfully carried out")
	log.Info("msg", "exiting!")
}

func parseArgs(args []string) (shouldProceed bool) {
	shouldProceed = true // Some flags like 'version' are just to get information and not proceed the actual execution. We should stop in such cases.
	for _, flag := range args {
		flag = flag[1:]
		switch flag {
		case "version":
			shouldProceed = false
			fmt.Println(version.PromMigrator)
		}
	}
	return
}

func convertSecFlagToMs(conf *config) {
	// remote-storages tend to respond to time in milliseconds. So, we convert the received values in seconds to milliseconds.
	conf.mint = conf.mintSec * 1000
	conf.maxt = conf.maxtSec * 1000
}

func validateConf(conf *config) error {
	switch {
	case conf.mint == 0:
		return fmt.Errorf("mint should be provided for the migration to begin")
	case conf.mint < 0:
		return fmt.Errorf("invalid mint: %d", conf.mint)
	case conf.maxt < 0:
		return fmt.Errorf("invalid maxt: %d", conf.maxt)
	case conf.mint > conf.maxt:
		return fmt.Errorf("invalid input: minimum timestamp value (mint) cannot be greater than the maximum timestamp value (maxt)")
	case conf.progressMetricName != progressMetricName:
		if !regexp.MustCompile(validMetricNameRegex).MatchString(conf.progressMetricName) {
			return fmt.Errorf("invalid metric-name regex match: prom metric must match %s: recieved: %s", validMetricNameRegex, conf.progressMetricName)
		}
	case strings.TrimSpace(conf.readURL) == "" && strings.TrimSpace(conf.writeURL) == "":
		return fmt.Errorf("remote read storage url and remote write storage url must be specified. Without these, data migration cannot begin")
	case strings.TrimSpace(conf.readURL) == "":
		return fmt.Errorf("remote read storage url needs to be specified. Without read storage url, data migration cannot begin")
	case strings.TrimSpace(conf.writeURL) == "":
		return fmt.Errorf("remote write storage url needs to be specified. Without write storage url, data migration cannot begin")
	case conf.progressEnabled && strings.TrimSpace(conf.progressMetricURL) == "":
		return fmt.Errorf("invalid input: read url for remote-write storage should be provided when progress metric is enabled. To disable progress metric, use -progress-enabled=false")
	}
	httpConfig := conf.readerAuth.ToHTTPClientConfig()
	if err := httpConfig.Validate(); err != nil {
		return fmt.Errorf("reader auth validation: %w", err)
	}
	httpConfig = conf.writerAuth.ToHTTPClientConfig()
	if err := httpConfig.Validate(); err != nil {
		return fmt.Errorf("writer auth validation: %w", err)
	}

	maxSlabSizeBytes, err := bytesize.Parse(conf.maxSlabSize)
	if err != nil {
		return fmt.Errorf("parsing byte-size: %w", err)
	}
	conf.maxSlabSizeBytes = int64(maxSlabSizeBytes)
	return nil
}
