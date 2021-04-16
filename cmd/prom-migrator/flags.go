package main

import (
	"flag"
	"time"
)

func parseFlags(conf *config, args []string) {
	// TODO: Auth/password via password_file and bearer_token via bearer_token_file.
	parseGeneralFlags(conf)
	parseReaderFlags(conf)
	parseWriterFlags(conf)
	_ = flag.CommandLine.Parse(args)
	convertSecFlagToMs(conf)
}

func parseGeneralFlags(conf *config) {
	flag.StringVar(&conf.name, "migration-name", migrationJobName, "Name for the current migration that is to be carried out. "+
		"It corresponds to the value of the label 'job' set inside the progress-metric-name.")
	flag.Int64Var(&conf.mintSec, "mint", 0, "Minimum timestamp (in seconds) for carrying out data migration. (inclusive)")
	flag.Int64Var(&conf.maxtSec, "maxt", time.Now().Unix(), "Maximum timestamp (in seconds) for carrying out data migration (exclusive). "+
		"Setting this value less than zero will indicate all data from mint upto now. ")
	flag.StringVar(&conf.maxSlabSize, "max-read-size", "500MB", "(units: B, KB, MB, GB, TB, PB) the maximum size of data that should be read at a single time. "+
		"More the read size, faster will be the migration but higher will be the memory usage. Example: 250MB.")
	flag.IntVar(&conf.concurrentPush, "concurrent-push", 1, "Concurrent push enables pushing of slabs concurrently. "+
		"Each slab is divided into 'concurrent-push' (value) parts and then pushed to the remote-write storage concurrently. This may lead to higher throughput on "+
		"the remote-write storage provided it is capable of handling the load. Note: Larger shards count will lead to significant memory usage.")
	flag.IntVar(&conf.concurrentPulls, "concurrent-pulls", 1, "Concurrent pulls enables fetching of data concurrently. "+
		"Each fetch query is divided into 'concurrent-pulls' (value) parts and then fetched concurrently. "+
		"This may enable higher throughput by pulling data faster from remote-read storage. "+
		"Note: Setting concurrent-pulls > 1 will show progress of concurrent fetching of data in the progress-bar and disable real-time transfer rate. "+
		"However, setting this value too high may cause TLS handshake error on the read storage side or may lead to starvation of fetch requests, depending on your internet bandwidth.")
	flag.StringVar(&conf.progressMetricName, "progress-metric-name", progressMetricName, "Prometheus metric name for tracking the last maximum timestamp pushed to the remote-write storage. "+
		"This is used to resume the migration process after a failure.")
	flag.StringVar(&conf.progressMetricURL, "progress-metric-url", "", "URL of the remote storage that contains the progress-metric. "+
		"Note: This url is used to fetch the last pushed timestamp. If you want the migration to resume from where it left, in case of a crash, "+
		"set this to the remote write storage that the migrator is writing along with the progress-enabled.")
	flag.BoolVar(&conf.progressEnabled, "progress-enabled", true, "This flag tells the migrator, whether or not to use the progress mechanism. It is helpful if you want to "+
		"carry out migration with the same time-range. If this is enabled, the migrator will resume the migration from the last time, where it was stopped/interrupted. "+
		"If you do not want any extra metric(s) while migration, you can set this to false. But, setting this to false will disble progress-metric and hence, the ability to resume migration.")
}

func parseReaderFlags(conf *config) {
	flag.StringVar(&conf.readURL, "read-url", "", "URL address for the storage where the data is to be read from.")

	// Authentication.
	flag.StringVar(&conf.readerAuth.Username, "read-auth-username", "", "Auth username for remote-read storage.")
	flag.StringVar(&conf.readerAuth.Password, "read-auth-password", "", "Auth password for remote-read storage.")
	flag.StringVar(&conf.readerAuth.BearerToken, "read-auth-bearer-token", "", "Bearer token for remote-read storage. "+
		"This should be mutually exclusive with username and password.")

	flag.DurationVar(&conf.readerTimeout, "reader-timeout", defaultTimeout, "Timeout for fetching data from 'read-url'")
}

func parseWriterFlags(conf *config) {
	flag.StringVar(&conf.writeURL, "write-url", "", "URL address for the storage where the data migration is to be written.")

	// Authentication.
	flag.StringVar(&conf.writerAuth.Username, "write-auth-username", "", "Auth username for remote-write storage.")
	flag.StringVar(&conf.writerAuth.Password, "write-auth-password", "", "Auth password for remote-write storage.")
	flag.StringVar(&conf.writerAuth.BearerToken, "write-auth-bearer-token", "", "Bearer token for remote-write storage. "+
		"This should be mutually exclusive with username and password.")

	flag.DurationVar(&conf.writerTimeout, "writer-timeout", defaultTimeout, "Timeout for fetching data from 'write-url'")

}
