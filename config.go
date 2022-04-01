package main

import (
	"io/ioutil"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Configuration File
type (
	Config struct {
		Settings *settings  `yaml:"global"`
		Monitors []*monitor `yaml:"monitors"`
	}
	settings struct {
		PushGW        string `yaml:"push_gw"`
		CheckInterval int    `yaml:"check_interval"`
		LogLevel      string `yaml:"log_level"`
		SocketLXD     string `yaml:"socket_lxd"`
		SocketDocker  string `yaml:"socket_docker"`
	}
)

// Read configuration from file
// File can be specified via command-line flag
// Default is config.yaml
func (c *Config) getConfig() {
	// Read the file
	yamlConf, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration from %s", configFile)
	}
	// Parse the file
	err = yaml.Unmarshal(yamlConf, c)
	if err != nil {
		log.Fatalf("Unable to load Config: %+v", err)
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
}

// Overwrites yaml settings if specified on command-line
func (c *Config) mergeSettings() {
	// Log Level
	if logLevel != "" {
		c.Settings.LogLevel = logLevel
	} else if c.Settings.LogLevel == "" {
		c.Settings.LogLevel = defaultLogLevel
	}
	// Socket locations
	if socketLXD != "" {
		c.Settings.SocketLXD = socketLXD
	}
	if socketDocker != "" {
		c.Settings.SocketDocker = socketDocker
	}
	// Prometheus Pushgateway URI
	if pushgw != "" {
		c.Settings.PushGW = pushgw
	}
	// Interval at which to refresh metrics from pushgateway
	if checkInterval != 0 {
		c.Settings.CheckInterval = checkInterval
	} else if c.Settings.CheckInterval == 0 {
		c.Settings.CheckInterval = defaultCheckInterval
	}
}

// Check for log level in config, use provided default if not found
func (c *Config) getLogLevel() logrus.Level {
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