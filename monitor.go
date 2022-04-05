package main

import (
	"rdmcguire/pushgw_bouncer/handlers"
	"time"
)

type monitor struct {
	Name             string   `yaml:"name"`
	Type             string   `yaml:"type"`
	ContainerName    string   `yaml:"container_name"`
	LabelName        string   `yaml:"label_name"`
	LabelValue       string   `yaml:"label_value"`
	MaxAge           string   `yaml:"max_age"`
	MaxAgeSecs       int      // Set by time.ParseDuration in config
	RestartType      string   `yaml:"restart_type"`
	RestartCommand   []string `yaml:"restart_command"`
	lastBounced      time.Time
	lastUpdateString string
	lastUpdateTime   time.Time
	lastUpdateSecs   int
	handler          handlers.Handler
}

// Fetches the last update string from the pushgateway metrics,
// then parses it and calculates how much time has elapsed
func (m *monitor) setLastUpdate(p *pushgwAPI) error {
	var err error
	// Get string from pushgw
	m.lastUpdateString, err = p.getLastUpdate(m)
	if err != nil {
		return err
	}
	// Convert to time.Time (parse)
	m.lastUpdateTime, err = time.Parse(time.RFC3339, m.lastUpdateString)
	if err != nil {
		return err
	}
	// Calculate offset in seconds
	m.lastUpdateSecs = int(time.Now().Sub(m.lastUpdateTime).Seconds())
	return err
}

// Check if the most recent update is within MaxAge
func (m *monitor) isLively() bool {
	var lively bool = true
	if m.lastUpdateSecs > m.MaxAgeSecs {
		// Update counter
		monitorChecks.WithLabelValues(m.Name, "unhealthy").Inc()
		lively = false
	} else {
		// Update counter
		monitorChecks.WithLabelValues(m.Name, "healthy").Inc()
	}
	return lively
}

// Uses the assigned handler to perform a restart
func (m *monitor) bounce() error {
	var err error
	if m.RestartType == "command" {
		err = m.handler.RunCommand(m.ContainerName, m.RestartCommand)
	} else if m.RestartType == "container" {
		err = m.handler.RestartContainer(m.ContainerName)
	}
	// Record the last bounced time if successful
	if err == nil {
		m.lastBounced = time.Now()
	}
	return err
}
