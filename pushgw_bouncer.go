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
	defaultAddr          string = ":9090"
	defaultSocketLXD     string = "/var/snap/lxd/common/lxd/unix.socket"
	defaultSocketDocker  string = "/var/run/docker.sock"
	defaultPushGW        string = "http://localhost:9091"
)

// Command-line Flags override YAML config
var (
	checkInterval string
	pushgw        string
	socketLXD     string
	socketDocker  string
	logLevel      string
	configFile    string = "config.yml"
	addr          string
)

// Global objects
var (
	conf   *config
	log    *logrus.Logger
	pushGW *pushgwAPI
	lxdH   *handlers.LXDConn
	dkrH   *handlers.DockerConn
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
	flag.StringVar(&addr, "addr", addr, "Listen address for Prometheus metrics")
	flag.Parse()

	// Setup
	log = logrus.New()
	conf = new(config)
	conf.getConfig()

	// Logging
	log.WithField("level", log.GetLevel()).
		Debug("Logging Initialized")

	// Hello World
	log.Info("Starting PushGW Bouncer")

	// Prepare pushGW API, LXD Client, and Docker Client
	pushGW = &pushgwAPI{log: log}

	// Connect to LXD socketLXD
	if conf.hasHandler("lxd") {
		lxdH = &handlers.LXDConn{
			Socket: conf.Settings.SocketLXD,
			Log:    log,
		}
		// lxdH = new(handlers.LXDConn)
		// lxdH.Socket = conf.Settings.SocketLXD
		// lxdH.Log = log
		if err := lxdH.Connect(); err != nil {
			log.WithFields(logrus.Fields{"socket": conf.Settings.SocketLXD, "error": err}).
				Fatal("Unable to connect to LXD socket")
		}
		log.WithField("LXD Socket", conf.Settings.SocketLXD).Info("LXD Connected")
	}

	// Connect to Docker
	if conf.hasHandler("docker") {
		dkrH = &handlers.DockerConn{
			Socket: conf.Settings.SocketDocker,
			Log:    log,
		}
		if err := dkrH.Connect(); err != nil {
			log.WithFields(logrus.Fields{"socket": conf.Settings.SocketDocker, "error": err}).
				Fatal("Unable to connect to Docker socket")
		}
		log.WithField("Docker Socket", conf.Settings.SocketDocker).Info("Docker Connected")
	}

	// Add handlers to monitors
	conf.setHandlers()

	// Serve Prometheus Metrics
	log.WithFields(logrus.Fields{
		"pushgateway": conf.Settings.PushGW,
		"listenAddr":  conf.Settings.Addr,
	}).Info("Launching prometheus metrics goroutine")
	go promInit()
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
			if err := monitor.setLastUpdate(pushGW); err != nil {
				// Something went wround, log the error and attempt to bounce
				log.WithFields(logrus.Fields{
					"monitor": monitor.Name,
					"error":   err,
				}).Warn("Failed to find monitor metrics in pushgateway, attempting bounce")
				if err = monitor.bounce(); err != nil {
					// Couldn't bounce (may be ok if recently attempted)
					if monitor.canBounce() {
						monitorBounces.WithLabelValues(monitor.Name, "failed").Inc()
					} else {
						monitorBounces.WithLabelValues(monitor.Name, "ineligible").Inc()
					}
					log.WithFields(logrus.Fields{
						"monitor": monitor.Name,
						"error":   err,
					}).Warn("Unable to bounce monitor")
				} else {
					// We bounced it
					monitorBounces.WithLabelValues(monitor.Name, "ok").Inc()
					log.WithField("monitor", monitor.Name).Warn("Monitor bounced")
				}
				// Done, next
				continue
			}

			// We got data, log it
			log.WithFields(logrus.Fields{
				"monitor":        monitor.Name,
				"lastUpdate":     monitor.lastUpdateTime,
				"lastUpdateSecs": monitor.lastUpdateSecs,
			}).Trace("Retrieved monitor update info")

			// Check to see if it is considered live
			if !monitor.isLively() {
				log.WithFields(logrus.Fields{
					"monitor":        monitor.Name,
					"lastUpdate":     monitor.lastUpdateString,
					"lastUpdateSecs": monitor.lastUpdateSecs,
					"maxAge":         monitor.MaxAge,
				}).Warn("Monitor is not lively!")

				// Attempt to bounce, log result
				if err := monitor.bounce(); err != nil {

					// FAILED, Update counter
					monitorBounces.WithLabelValues(monitor.Name, "failed").Inc()
					log.WithFields(logrus.Fields{
						"monitor": monitor.Name,
						"error":   err,
					}).Error("Failed to bounce monitor")
				} else {

					// SUCCESS, Update counter
					monitorBounces.WithLabelValues(monitor.Name, "ok").Inc()
					log.WithFields(logrus.Fields{
						"monitor":    monitor.Name,
						"container":  monitor.ContainerName,
						"lastUpdate": monitor.lastUpdateString,
					}).Warn("Bounced stuck monitor")
				}

				// Monitor is fine, log and continue
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
