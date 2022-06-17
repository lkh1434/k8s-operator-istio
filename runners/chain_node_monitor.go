package runners

import (
	"context"
	"net/http"
	"time"

	operatorv1alpha1 "github.com/lkh1434/ca-operator/api/v1alpha1"
	"github.com/lkh1434/ca-operator/metrics"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func NewChainNodeMonitor(c client.Client, log logr.Logger, interval time.Duration) manager.Runnable {
	return &chainNodeMonitor{
		client:   c,
		log:      log,
		interval: interval,
	}
}

type chainNodeMonitor struct {
	client   client.Client
	interval time.Duration
	log      logr.Logger
}

// Start implements manager.Runnable
func (w *chainNodeMonitor) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)

	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// startTime := time.Now()
			err := w.reconcile(ctx)
			if err != nil {
				w.log.Error(err, "failed to reconcile")
				return err
			}
		}
	}
}

func (w *chainNodeMonitor) reconcile(ctx context.Context) error {
	// timeout := time.Duration(10) * time.Second
	// waitForHTTPSuccessStatusCode(timeout)

	// get chain monitoring urls from caistio custorm resource
	cil := &operatorv1alpha1.CAIstioList{}

	err := w.client.List(ctx, cil)

	if err != nil {
		w.log.Error(err, "Failed to get CAIstio Resources")
		return err
	}
	// for _, ci := range cil.Items{

	// }

	return nil
}

func waitForHTTPSuccessStatusCode(timeout time.Duration, url string) error {

	var resp *http.Response
	err := wait.Poll(time.Second, timeout, func() (bool, error) {
		timer := prometheus.NewTimer(metrics.ChainNodeResponseTime.Add(url, "aa", 300))
		var err error
		resp, err = http.Get(url)
		if err == nil && resp.StatusCode == 200 {
			timer.ObserveDuration()
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		return err
	}
	return nil

}
