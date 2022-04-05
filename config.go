package main

import (
	"io/ioutil"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// configuration File
type (
	config struct {
		Settings *settings  `yaml:"global"`
		Monitors []*monitor `yaml:"monitors"`
	}
	settings struct {
		PushGW        string `yaml:"push_gw"`
		CheckInterval string `yaml:"check_interval"`
		LogLevel      string `yaml:"log_level"`
		SocketLXD     string `yaml:"socket_lxd"`
		SocketDocker  string `yaml:"socket_docker"`
		Addr          string `yaml:"port"`
	}
)

// Read configuration from file
// File can be specified via command-line flag
// Default is config.yaml
func (c *config) getConfig() {
	// Read the file
	yamlConf, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration from %s", configFile)
	}

	// Parse the file
	err = yaml.Unmarshal(yamlConf, c)
	if err != nil {
		log.Fatalf("Unable to load config: %+v", err)
	}

	// Sanity checks
	for _, m := range c.Monitors {
		var err error
		// Check to make sure MaxAge can be converted to time.Duration
		// Convert to string and assign to MaxAgeSecs if it can be
		var duration time.Duration
		duration, err = time.ParseDuration(m.MaxAge)
		if err != nil {
			log.WithFields(logrus.Fields{
				"monitor": m.Name,
				"maxAge":  m.MaxAge,
				"error":   err,
			}).Error("Unable to parse monitor maxAge")
		} else {
			m.MaxAgeSecs = int(duration.Seconds())
		}
	}

	// Overwrite settings from flags
	c.mergeSettings()

	// Assign Handlers to monitors
	for _, m := range c.Monitors {
		c.setHandler(m)
		log.WithFields(logrus.Fields{
			"monitor": m.Name,
			"handler": m.handler,
		}).Debug("Monitor handler assigned")
	}
}

// Find the correct handler for the monitor and set it
func (c *config) setHandler(m *monitor) {
	if m.Type == "lxd" {
		m.handler = lxd
	} else if m.Type == "docker" {
		m.handler = docker
	} else {
		log.WithFields(logrus.Fields{
			"monitor": m.Name,
			"handler": m.Type,
		}).Error("Monitor handler not found")
		m.handler = nil
	}
}

// Overwrites yaml settings if specified on command-line
func (c *config) mergeSettings() {
	// Log Level
	c.Settings.LogLevel = getConfStr(logLevel, c.Settings.LogLevel, defaultLogLevel)
	// Socket locations
	c.Settings.SocketLXD = getConfStr(socketLXD, c.Settings.SocketLXD, defaultSocketLXD)
	c.Settings.SocketDocker = getConfStr(socketDocker, c.Settings.SocketDocker, defaultSocketDocker)
	// Prometheus Pushgateway URI
	c.Settings.PushGW = getConfStr(pushgw, c.Settings.PushGW, defaultPushGW)
	// Interval at which to refresh metrics from pushgateway
	c.Settings.CheckInterval = getConfStr(checkInterval, c.Settings.CheckInterval, defaultCheckInterval)
	// Default listen addr
	c.Settings.Addr = getConfStr(addr, c.Settings.Addr, defaultAddr)
}

// Return highest priority non-empty string
func getConfStr(p1, p2, p3 string) string {
	if p1 != "" {
		return p1
	} else if p2 != "" {
		return p2
	} else {
		return p3
	}
}

// Check for log level in config, use provided default if not found
func (c *config) getLogLevel() logrus.Level {
	var level logrus.Level
	switch c.Settings.LogLevel {
	case "error":
		level = logrus.ErrorLevel
	case "warn":
		level = logrus.WarnLevel
	case "info":
		level = logrus.InfoLevel
	case "debug":
		level = logrus.DebugLevel
	case "trace":
		level = logrus.TraceLevel
	default:
		level = logrus.WarnLevel
	}
	return level
}

// Check to see if specified handler is needed
func (c *config) hasHandler(name string) bool {
	var present bool = false
	for _, m := range c.Monitors {
		if m.Type == name {
			present = true
		}
	}
	return present
}
