package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"time"

	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
	"github.com/sirupsen/logrus"
)

var (
	maxSecs   int    = 120
	checkSecs int    = 180
	pushgw    string = "http://retro:9091"
	socket    string = "/var/snap/lxd/common/lxd/unix.socket"
	container string = "weewx"
	client           = &http.Client{Timeout: 10 * time.Second}
	logLevel  string = "info"
	log       *logrus.Logger
	lxdClient lxd.InstanceServer
)

type pushgwAPI struct {
	Status           string                   `json:"status"`
	Data             []map[string]interface{} `json:"data"`
	lastUpdateString string
	lastUpdateTime   time.Time
	lastUpdateSecs   int
	log              *logrus.Logger
}

// Retrieve metrics from pushgateway
func (m *pushgwAPI) getMetrics() error {
	var err error
	var r *http.Response
	r, err = client.Get(pushgw + "/api/v1/metrics")
	if err != nil {
		m.log.WithFields(logrus.Fields{"err": err, "gw": pushgw}).
			Error("Unable to retrieve current metrics from PushGateway")
	}
	json.NewDecoder(r.Body).Decode(m)
	r.Body.Close()
	return err
}

// Retrieve time_stamp from push_time_seconds if instance is weewx
func (m *pushgwAPI) getWeewxLastUpdate() {
	for key := range m.Data {
		if m.Data[key]["labels"].(map[string]interface{})["instance"].(string) == "weewx" {
			m.lastUpdateString = m.Data[key]["push_time_seconds"].(map[string]interface{})["time_stamp"].(string)
		} else {
			continue
		}
	}
	if m.lastUpdateString != "" {
		m.calcLastUpdate()
	}
}

// Convert time_stamp field from string to time.Time
// Calculate seconds since last update
func (m *pushgwAPI) calcLastUpdate() {
	lastUpdate, err := time.Parse(time.RFC3339, m.lastUpdateString)
	if err != nil {
		m.log.WithFields(logrus.Fields{"err": err, "timestamp": m.lastUpdateString}).
			Error("Failed to parse timestamp")
		return
	}
	m.lastUpdateTime = lastUpdate
	m.lastUpdateSecs = int(time.Now().Sub(m.lastUpdateTime).Seconds())
}

// Fetch new metrics from pushgateway
// Determine if too much time has passed
func (m *pushgwAPI) check() bool {
	var status bool = true
	if err := m.getMetrics(); err != nil {
		log.WithField("error", err).Error("Unable to retrieve metrics from Pushgateway")
	} else {
		m.getWeewxLastUpdate()
		if m.lastUpdateString == "" || m.lastUpdateSecs > maxSecs {
			m.log.WithField("Seconds", m.lastUpdateSecs).Warn("Last update too long ago or never updated")
			status = false
		}
	}
	return status
}

// Perform a systemctl restart weewx using the LXD unix socket
func restartWeewx(c lxd.InstanceServer) {
	req := api.InstanceExecPost{
		Command: []string{
			"/bin/systemctl",
			"restart",
			"weewx",
		},
		Interactive: false,
	}
	logFields := logrus.Fields{
		"command":   req,
		"socket":    socket,
		"container": container,
	}
	// Send the command
	restart, err := c.ExecInstance(container, req, nil)
	if err != nil {
		log.WithFields(logFields).WithField("error", err).
			Error("Failed to send restart command to WeeWX")
	}
	// Wait for it to complete
	err = restart.Wait()
	if err != nil {
		log.WithFields(logFields).WithField("error", err).
			Error("Failed to restart WeeWX")
	}
}

func main() {
	// Process flags
	flag.StringVar(&pushgw, "pushgw", pushgw, "Update Gateway for Metrics")
	flag.StringVar(&socket, "socket", socket, "Location of LXD Unix Socket")
	flag.StringVar(&logLevel, "logLevel", logLevel, "Log level (error|warn|info|debug|trace)")
	flag.StringVar(&container, "container", container, "Name of WeeWX Container")
	flag.IntVar(&maxSecs, "maxSecs", maxSecs, "Maximum seconds since last update before we whack weewx")
	flag.IntVar(&checkSecs, "checkSecs", checkSecs, "Interval at which to perform liveliness check")
	flag.Parse()

	log := logrus.New()

	// Set logging level
	switch logLevel {
	case "error":
		log.Level = logrus.ErrorLevel
	case "warn":
		log.Level = logrus.WarnLevel
	case "info":
		log.Level = logrus.InfoLevel
	case "debug":
		log.Level = logrus.DebugLevel
	case "trace":
		log.Level = logrus.TraceLevel
	}

	// Announce Init
	log.WithField("pushGW", pushgw).
		Info("Starting WEEWX Bouncer")

	// Create pushgwAPI checker instance
	checker := pushgwAPI{log: log}

	// Connect to LXD socket
	lxdClient, err := lxd.ConnectLXDUnix(socket, nil)
	if err != nil {
		log.WithFields(logrus.Fields{"socket": socket, "error": err}).
			Fatal("Unable to connect to LXD Socket")
	}
	log.WithField("socket", socket).Info("LXD Connected")

	// Debug LXD Connection Info
	connInfo, _ := lxdClient.GetConnectionInfo()
	log.WithField("lxd", connInfo).Debug("LXD Connected")

	// Tick away
	ticker := time.NewTicker(time.Duration(checkSecs) * time.Second)
	for range ticker.C {
		if !checker.check() {
			log.Warnf("Bouncing WeeWX")
			restartWeewx(lxdClient)
		} else {
			log.WithField("LastUpdateSecs", checker.lastUpdateSecs).Info("WeeWX Last Update Timely")
		}
	}
}
