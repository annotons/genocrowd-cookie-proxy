package main

import (
	"time"

	log "github.com/Sirupsen/logrus"
	_ "github.com/lib/pq"
	"github.com/quipo/statsd"
)

var (
	Metrics        *statsd.StatsdClient
	statsdInfluxdb bool
)

func configureMetrics(statsdAddress, statsdPrefix string) {
	if len(statsdAddress) > 0 {
		Metrics = statsd.NewStatsdClient(statsdAddress, statsdPrefix)
		err := Metrics.CreateSocket()
		if err != nil {
			log.Fatal("Could not configure StatsD connection")
		}
		log.Printf("Loaded StatsD connection: %#v", Metrics)
	}
}

func metricIncr(val string) {
	if Metrics != nil {
		var err error
		if statsdInfluxdb {
			err = Metrics.Incr(",key="+val, 1)
		} else {
			err = Metrics.Incr(val, 1)
		}
		if err != nil {
			log.Error(err)
		}
	}
}

func metricTime(val string, elapsed time.Duration) {
	if Metrics != nil {
		var err error
		if statsdInfluxdb {
			err = Metrics.PrecisionTiming(",key=query_timing", elapsed)
		} else {
			err = Metrics.PrecisionTiming("query_timing", elapsed)
		}

		if err != nil {
			log.Error(err)
		}
	}
}
