package main

import (
	"flag"
	"time"

	"github.com/sirupsen/logrus"
)

// Defaults if left unspecified
const (
	defaultCheckInterval int    = 60
	defaultLogLevel      string = "info"
)

// Command-line Flags override YAML config
var (
	checkInterval int
	pushgw        string
	socketLXD     string
	socketDocker  string
	logLevel      string
	configFile    string = "config.yml"
)

// Global objects
var (
	conf   *Config
	log    *logrus.Logger
	pushGW *pushgwAPI
	lxdC   *lxdConn
)

// All flags are optional
// ALl flags except configFile can be overridden in the config file
func init() {
	// Process flags
	flag.StringVar(&configFile, "configFile", configFile, "File to read config from (default config.yml)")
	flag.StringVar(&pushgw, "pushgw", pushgw, "Update Gateway for Metrics")
	flag.StringVar(&socketLXD, "socketLXD", socketLXD, "Location of LXD Unix socket")
	flag.StringVar(&socketDocker, "socketDocker", socketDocker, "Location of Docker Unix socket")
	flag.StringVar(&logLevel, "logLevel", logLevel, "Log level (error|warn|info|debug|trace)")
	flag.IntVar(&checkInterval, "checkInterval", checkInterval, "Interval at which to perform liveliness check")
	flag.Parse()

	// Setup
	log = logrus.New()
	conf = new(Config)
	conf.getConfig()

	// Set logging level
	log.SetLevel(conf.getLogLevel())
	log.WithField("level", log.GetLevel()).
		Debug("Logging Initialized")

	// Hello World
	log.WithField("pushGW", pushgw).
		Info("Starting PushGW Bouncer")

	// Prepare pushGW API, LXD Client, and Docker Client
	pushGW = &pushgwAPI{log: log}
	lxdC = &lxdConn{socket: socketLXD}

	// Connect to LXD socketLXD
	if err := lxdC.connect(); err != nil {
		log.WithFields(logrus.Fields{"socket": socketLXD, "error": err}).
			Fatal("Unable to connect to LXD socket")
	}
	log.WithField("LXD Socket", socketLXD).Info("LXD Connected")

	// TODO Connect to Docker socket
}

func main() {
	// Tick away
	ticker := time.NewTicker(time.Duration(checkInterval) * time.Second)
	for range ticker.C {
		// First refresh metrics from pushgateway
		pushGW.getMetrics()
		// Next check each monitor for liveness
		for _, monitor := range conf.Monitors {
			monitor.setLastUpdate(pushGW)
			if !monitor.isLively() {
				log.WithField("monitor", monitor).Warn("Monitor is not lively!")
			}
		}
	}
}
