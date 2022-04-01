package main

import (
	"flag"
	"time"

	"rdmcguire/pushgw_bouncer/handlers"

	"github.com/sirupsen/logrus"
)

// Defaults if left unspecified
const (
	defaultCheckInterval string = "1m"
	defaultLogLevel      string = "info"
)

// Command-line Flags override YAML config
var (
	checkInterval string
	pushgw        string
	socketLXD     string
	socketDocker  string
	logLevel      string
	configFile    string = "config.yml"
)

// Global objects
var (
	conf   *config
	log    *logrus.Logger
	pushGW *pushgwAPI
	lxd    *handlers.LXDConn
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
	flag.StringVar(&checkInterval, "checkInterval", checkInterval, "Interval at which to perform liveliness check")
	flag.Parse()

	// Setup
	log = logrus.New()
	conf = new(config)
	conf.getConfig()

	// Set logging level
	log.SetLevel(conf.getLogLevel())
	log.WithField("level", log.GetLevel()).
		Debug("Logging Initialized")

	// Hello World
	log.WithField("pushGW", conf.Settings.PushGW).
		Info("Starting PushGW Bouncer")

	// Prepare pushGW API, LXD Client, and Docker Client
	pushGW = &pushgwAPI{log: log}
	lxd = &handlers.LXDConn{Socket: conf.Settings.SocketLXD}

	// Connect to LXD socketLXD
	if err := lxd.Connect(); err != nil {
		log.WithFields(logrus.Fields{"socket": socketLXD, "error": err}).
			Fatal("Unable to connect to LXD socket")
	}
	log.WithField("LXD Socket", conf.Settings.SocketLXD).Info("LXD Connected")

	// TODO Connect to Docker socket
}

func main() {
	// Tick away
	interval, err := time.ParseDuration(conf.Settings.CheckInterval)
	if err != nil {
		log.WithField("error", err).Fatal("Unable to parse check interval")
	} else {
		log.WithField("interval", interval).
			Info("Starting main loop")
	}
	ticker := time.NewTicker(interval)
	for range ticker.C {
		// First refresh metrics from pushgateway
		pushGW.getMetrics()
		log.Debugf("Updated metrics from pushgateway")
		// Next check each monitor for liveness
		for _, monitor := range conf.Monitors {
			// Update the monitor's last update from the pushGW metrics
			monitor.setLastUpdate(pushGW)
			log.WithFields(logrus.Fields{
				"monitor":        monitor.Name,
				"lastUpdate":     monitor.lastUpdateTime,
				"lastUpdateSecs": monitor.lastUpdateSecs,
			}).Trace("Retrieved monitor update info")
			// Check to see if it is considered live
			if !monitor.isLively() {
				log.WithField("monitor", monitor.Name).Warn("Monitor is not lively!")
				// Attempt to bounce, log result
				if err := monitor.bounce(); err != nil {
					log.WithFields(logrus.Fields{
						"monitor": monitor.Name,
						"error":   err,
					}).Error("Failed to bounce monitor")
				} else {
					log.WithFields(logrus.Fields{
						"monitor":    monitor.Name,
						"container":  monitor.ContainerName,
						"lastUpdate": monitor.lastUpdateString,
					}).Warn("Bounced stuck monitor")
				}
			} else {
				log.WithFields(logrus.Fields{
					"monitor":        monitor.Name,
					"lastUpdateSecs": monitor.lastUpdateSecs,
					"maxAge":         monitor.MaxAgeSecs,
				}).Debug("Monitor liveness ok")
			}
		}
	}
}
