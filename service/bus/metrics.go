package bus

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"

	"github.com/weaveworks/flux/service"
)

// BusMetrics has metrics for messages buses.
type Metrics struct {
	KickCount metrics.Counter
}

var (
	MetricsImpl = Metrics{
		KickCount: prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "flux",
			Subsystem: "bus",
			Name:      "kick_total",
			Help:      "Count of bus subscriptions kicked off by a newer subscription.",
		}, []string{}),
	}
)

func (m Metrics) IncrKicks(inst service.InstanceID) {
	m.KickCount.Add(1)
}
