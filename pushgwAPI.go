package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	client *http.Client = &http.Client{Timeout: 10 * time.Second}
)

type pushgwAPI struct {
	Status string                   `json:"status"`
	Data   []map[string]interface{} `json:"data"`
	log    *logrus.Logger
}

// Retrieve metrics from pushgateway
func (m *pushgwAPI) getMetrics() error {
	var err error
	var r *http.Response
	// Perform HTTP GET
	r, err = client.Get(conf.Settings.PushGW + "/api/v1/metrics")
	if err != nil {
		m.log.WithFields(logrus.Fields{"err": err, "gw": conf.Settings.PushGW}).
			Error("Unable to retrieve current metrics from PushGateway")
		// Update counter
		monitorUpdates.WithLabelValues("all", "failed").Inc()
	} else {
		// Update counter
		monitorUpdates.WithLabelValues("all", "ok").Inc()
	}
	// Decode Response into pushgwAPI
	json.NewDecoder(r.Body).Decode(m)
	r.Body.Close()
	return err
}

// Retrieve time_stamp from push_time_seconds if instance is weewx
// Takes an instance of monitor to locate setting
func (m *pushgwAPI) getLastUpdate(monitor *monitor) (string, error) {
	var lastUpdate string
	var err error
	for key := range m.Data {
		labels := m.Data[key]["labels"].(map[string]interface{})
		label := labels[monitor.LabelName].(string)
		if label == monitor.LabelValue {
			lastUpdate = m.Data[key]["push_time_seconds"].(map[string]interface{})["time_stamp"].(string)
		} else {
			continue
		}
	}
	if lastUpdate == "" {
		// Update counter
		err = errors.New("Unable to retrieve last update from pushgateway")
		monitorUpdates.WithLabelValues(monitor.Name, "failed").Inc()
	} else {
		// Update counter
		monitorUpdates.WithLabelValues(monitor.Name, "ok").Inc()
	}
	return lastUpdate, err
}
