package observer

import (
	"strconv"

	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"

	// _ are probes imported to trigger init func
	_ "github.com/letsencrypt/boulder/observer/probes/dns"
	_ "github.com/letsencrypt/boulder/observer/probes/http"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	statMonitors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "obs_monitors",
			Help: "count of configured monitors",
		},
		[]string{"name", "type", "valid"},
	)
	statObservations = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "obs_observations",
			Help:    "time taken for a monitor to perform a request/query",
			Buckets: metrics.InternetFacingBuckets,
		},
		[]string{"name", "type", "result"},
	)
)

// Observer contains the parsed, normalized, and validated configuration
// describing a collection of monitors and the metrics to be collected
type Observer struct {
	Logger   blog.Logger
	Metric   prometheus.Registerer
	Monitors []*monitor
}

// Start registers global metrics and spins off a goroutine for each of
// the configured monitors
func (o Observer) Start() {

	// start each monitor
	for _, mon := range o.Monitors {
		if mon.valid {
			go mon.start()
		}

	}
	// run forever
	select {}
}

// New attempts to populate and return an `Observer` object with the
// contents of an `ObsConf`. If the `ObsConf` cannot be validated, an
// error appropriate for end-user consumption is returned
func New(c ObsConf, l blog.Logger, p prometheus.Registerer) (*Observer, error) {
	// validate the `ObsConf`
	err := c.validate(l)
	if err != nil {
		return nil, err
	}

	// register metrics
	p.MustRegister(statObservations)
	p.MustRegister(statMonitors)

	// Create a `monitor` for each `MonConf`
	var monitors []*monitor
	for _, m := range c.MonConfs {
		if !m.Valid {
			statMonitors.WithLabelValues(
				"", m.Kind, strconv.FormatBool(m.Valid)).Inc()
		} else {
			monitors = append(
				monitors, &monitor{m.Valid, m.Period.Duration, m.getProber(), l, p})
		}
	}
	return &Observer{l, p, monitors}, nil
}
