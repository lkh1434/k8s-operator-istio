package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	runtimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Metrics subsystem and all of the keys used by the chain node monitor.
const (
	ChainNodeResponseTimeKey = "chain_node_response_time"
)

func init() {
	registerResizerMetrics()
}

type chainNodeResponseTimeAdapter struct {
	metric prometheus.HistogramVec
}

func (a *chainNodeResponseTimeAdapter) Add(url string, chainID string, status int) prometheus.Observer {
	return a.metric.WithLabelValues(url, chainID, fmt.Sprintf("%d", status))
}

var (
	chainNodeResponseTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: MetricsNamespace,
		Name:      ChainNodeResponseTimeKey,
		Help:      "Chain Node HTTP Response Time",
		Buckets:   prometheus.DefBuckets,
	}, []string{"url", "chainid", "status"})

	ChainNodeResponseTime *chainNodeResponseTimeAdapter = &chainNodeResponseTimeAdapter{metric: *chainNodeResponseTime}
)

func registerResizerMetrics() {
	runtimemetrics.Registry.MustRegister(chainNodeResponseTime)
}

// func NewHandlerWithHistogram(handler http.Handler, histogram *chainNodeResponseTimeAdapter, chainID string) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
// 		start := time.Now()
// 		status := http.StatusOK

// 		defer func() {
// 			histogram.Add(chainID, status, time.Since(start).Seconds())
// 			// histogram.WithLabelValues(chainID, fmt.Sprintf("%d", status)).Observe(time.Since(start).Seconds())
// 		}()

// 		if req.Method == http.MethodGet {
// 			handler.ServeHTTP(w, req)
// 			return
// 		}
// 		status = http.StatusBadRequest

// 		w.WriteHeader(status)
// 	})
// }
