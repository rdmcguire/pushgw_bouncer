package main

import (
	"encoding/json"
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
// Takes an instance of monitor to locate setting
func (m *pushgwAPI) getLastUpdate(monitor *monitor) string {
	var lastUpdate string
	for key := range m.Data {
		labels := m.Data[key]["labels"].(map[string]interface{})
		label := labels[monitor.LabelName].(string)
		if label == monitor.LabelValue {
			lastUpdate = m.Data[key]["push_time_seconds"].(map[string]interface{})["time_stamp"].(string)
		} else {
			continue
		}
	}
	return lastUpdate
}
