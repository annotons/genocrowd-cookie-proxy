package main

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	log "github.com/Sirupsen/logrus"
	_ "github.com/lib/pq"

	"github.com/patrickmn/go-cache"
	"github.com/urfave/cli"
	"golang.org/x/crypto/blowfish"
)

var (
	version   string
	builddate string
	logger    *log.Logger
)

var hexReg, _ = regexp.Compile("[^a-fA-F0-9]+")

func main2(genocrowdSecret, listenAddr, connect, header, statsdAddress, statsdPrefix string, watchdogEnable bool, watchdogInterval, watchdogExpect int, watchdogCookie string) {
	// Setup metrics stuff
	configure_metrics(statsdAddress, statsdPrefix)

	bf, err := blowfish.NewCipher([]byte(genocrowdSecret))
	if err != nil {
		log.Fatal(err)
	}

	var requestHandler http.Handler = &ProxyHandler{
		Transport: &http.Transport{
			DisableKeepAlives:  false,
			DisableCompression: false,
		},
		// Frontend
		AddForwarded: true,
		// Backend
		BackendScheme:  "http",
		BackendAddress: connect,
		GalaxyCipher:   bf,
		Cache:          cache.New(1*time.Hour, 5*time.Minute),
		EmailCache:     cache.New(1*time.Hour, 5*time.Minute),
		Header:         header,
	}

	if logger != nil {
		requestHandler = NewRequestLogger(requestHandler, *logger)
	}

	mux := http.NewServeMux()
	mux.Handle("/", requestHandler)
	srv := &http.Server{Handler: mux, Addr: listenAddr}

	if watchdogEnable {
		launchWatchdog(watchdogInterval, listenAddr, watchdogExpect, watchdogCookie)
	}

	log.Printf("Listening on %s", listenAddr)
	if err := srv.ListenAndServe(); err != nil {
		log.Printf("Starting proxy failed: %v", err)
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "genocrowd-cookie-proxy"
	app.Usage = "Proxy requests, transparently determining genocrowd user based on session cookie and adding a REMOTE_USER header"
	app.Version = fmt.Sprintf("%s (%s)", version, builddate)

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "genocrowdSecret",
			Value:  "USING THE DEFAULT IS NOT SECURE!",
			Usage:  "Genocrowd Secret",
			EnvVar: "GENOCROWD_SECRET",
		},
		cli.StringFlag{
			Name:   "listenAddr",
			Value:  "0.0.0.0:5000",
			Usage:  "Address to listen on",
			EnvVar: "GXC_LISTEN_ADDR",
		},
		cli.StringFlag{
			Name:   "connect",
			Value:  "localhost:8000",
			Usage:  "Backend URL.",
			EnvVar: "GXC_BACKEND_URL",
		},
		cli.StringFlag{
			Name:   "logLevel",
			Value:  "INFO",
			Usage:  "Log level, choose from (DEBUG, INFO, WARN, ERROR)",
			EnvVar: "GXC_LOGLEVEL",
		},
		cli.StringFlag{
			Name:   "header",
			Value:  "REMOTE_USER",
			Usage:  "Customize the HTTP Header (for those picky applications)",
			EnvVar: "GXC_HEADER",
		},
		cli.StringFlag{
			Name:   "statsdAddress",
			Value:  "",
			Usage:  "Set this if you wish to send data to statsd somewhere",
			EnvVar: "GXC_STATSD",
		},
		cli.StringFlag{
			Name:   "statsdPrefix",
			Value:  "gxc.",
			Usage:  "statsd statistics prefix.",
			EnvVar: "GXC_statsdPrefix",
		},
		cli.BoolFlag{
			Name:   "statsd_influxdb",
			Usage:  "Format statsd output to be compatible with influxdb/telegraf",
			EnvVar: "GXC_STATSD_INFLUXDB",
		},
		cli.BoolFlag{
			Name:   "watchdogEnable",
			Usage:  "Enable the SystemD watchdog integration",
			EnvVar: "GXC_WATCHDOG",
		},
		cli.IntFlag{
			Name:   "watchdogInterval",
			Value:  30,
			Usage:  "Watchdog check interval (seconds)",
			EnvVar: "GXC_watchdogInterval",
		},
		cli.IntFlag{
			Name:   "watchdogExpect",
			Value:  200,
			Usage:  "Error code to expect for a successful request",
			EnvVar: "GXC_watchdogExpect",
		},
		cli.StringFlag{
			Name:   "watchdogCookie",
			Value:  "",
			Usage:  "Genocrowd cookie to provide to watchdog request",
			EnvVar: "GXC_watchdogCookie",
		},
	}

	app.Action = func(c *cli.Context) {

		// Output to stdout instead of the default stderr, could also be a file.
		log.SetOutput(os.Stdout)
		// Only log the warning severity or above.
		if c.String("logLevel") == "DEBUG" {
			log.SetLevel(log.DebugLevel)
		} else if c.String("logLevel") == "INFO" {
			log.SetLevel(log.InfoLevel)
		} else if c.String("logLevel") == "WARN" {
			log.SetLevel(log.WarnLevel)
		} else if c.String("logLevel") == "ERROR" {
			log.SetLevel(log.ErrorLevel)
		} else {
			panic("Unknown log level")
		}

		statsd_influxdb = c.Bool("statsd_influxdb")

		main2(
			c.String("genocrowdSecret"),
			c.String("listenAddr"),
			c.String("connect"),
			c.String("header"),
			c.String("statsdAddress"),
			c.String("statsdPrefix"),
			c.Bool("watchdogEnable"),
			c.Int("watchdogInterval"),
			c.Int("watchdogExpect"),
			c.String("watchdogCookie"),
		)
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
