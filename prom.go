package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Prepare Prometheus
	prom    = prometheus.NewRegistry()
	metrics = promauto.With(prom)
	// Gauges
	monitorsQty = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "monitors_configured",
		Help: "Number of monitors configured for pushgw_bouncer",
	}, []string{"pushgateway"})
	// Counters
	monitorUpdates = metrics.NewCounterVec(prometheus.CounterOpts{
		Name: "monitor_updates",
		Help: "Number of times the metrics from the pushgateway were retrieved",
	}, []string{"pushgateway", "monitor", "result"})
	monitorChecks = metrics.NewCounterVec(prometheus.CounterOpts{
		Name: "monitor_checks",
		Help: "Number of times a service monitor check was performed",
	}, []string{"pushgateway", "monitor", "result"})
	monitorBounces = metrics.NewCounterVec(prometheus.CounterOpts{
		Name: "monitor_bounces",
		Help: "Number of times a monitor was bounced",
	}, []string{"pushgateway", "monitor", "result"})
)

func promInit() {
	// Set the pushgateway label
	monitorUpdates, _ = monitorUpdates.
		CurryWith(prometheus.Labels{"pushgateway": conf.Settings.PushGW})
	monitorChecks, _ = monitorChecks.
		CurryWith(prometheus.Labels{"pushgateway": conf.Settings.PushGW})
	monitorBounces, _ = monitorBounces.
		CurryWith(prometheus.Labels{"pushgateway": conf.Settings.PushGW})
	monitorsQty, _ = monitorsQty.
		CurryWith(prometheus.Labels{"pushgateway": conf.Settings.PushGW})
	// Update monitor qty
	monitorsQty.WithLabelValues().Set(float64(len(conf.Monitors)))
	// Serve endpoint
	http.Handle("/metrics", promhttp.HandlerFor(prom, promhttp.HandlerOpts{}))
	http.ListenAndServe(conf.Settings.Addr, nil)
}
