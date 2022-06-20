package runners

import (
	"context"
	"strconv"

	// "fmt"
	// "math"
	// "strconv"
	// "strings"
	"time"

	"github.com/go-logr/logr"

	// "github.com/topolvm/pvc-autoresizer/metrics"
	istionetworkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"

	// corev1 "k8s.io/api/core/v1"
	// storagev1 "k8s.io/api/storage/v1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	// ctrl "sigs.k8s.io/controller-runtime"
	operatorv1alpha1 "github.com/lkh1434/ca-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;update;patch

// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create

// NewVSModifier returns a new istioModifier struct
func NewVSModifier(c client.Client, log logr.Logger, interval time.Duration, recorder record.EventRecorder) manager.Runnable {

	return &istioModifier{
		client:   c,
		log:      log,
		interval: interval,
		recorder: recorder,
	}
}

type istioModifier struct {
	client   client.Client
	interval time.Duration
	log      logr.Logger
	recorder record.EventRecorder
}

// Start implements manager.Runnable
func (w *istioModifier) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)

	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// startTime := time.Now()
			err := w.reconcile(ctx)
			// metrics.ResizerLoopSecondsTotal.Add(time.Since(startTime).Seconds())
			if err != nil {
				w.log.Error(err, "failed to reconcile")
				return err
			}
		}
	}
}

func (w *istioModifier) getVirtualServiceList(ctx context.Context) (*istionetworkingv1beta1.VirtualServiceList, error) {
	var vsl istionetworkingv1beta1.VirtualServiceList
	// 모니터링 대상은 istio virtualservice 리소스에 라벨키 vs-modifier-enable이 추가되어 있고 값이 true인 virtual service 들
	err := w.client.List(ctx, &vsl, client.MatchingLabels{"vs-modifier-enable": "true"})
	if err != nil {
		return nil, err
	}

	return &vsl, nil
}

type caIstioInfo struct {
	name                string
	namespace           string
	chainid             string
	nodeservice         string
	nodeserviceentry    string
	destination         string
	responsefailedcount int
	latestblockheight   int
	heightfailedcount   int
}

func (w *istioModifier) getCAIstioInfoList(ctx context.Context, vsl *istionetworkingv1beta1.VirtualServiceList) (map[string]caIstioInfo, error) {
	var caistio operatorv1alpha1.CAIstioList
	err := w.client.List(ctx, &caistio)
	if err != nil {
		return nil, err
	}

	var chainNameArr []string
	for _, vs := range vsl.Items {
		chainNameArr = append(chainNameArr, vs.Namespace)
	}

	caIstioInfoList := make(map[string]caIstioInfo)
	for _, cname := range chainNameArr {
		for _, res := range caistio.Items {
			if cname == res.Spec.ChainID {
				caIstioInfoList[res.Spec.ChainID] = caIstioInfo{
					res.Name,
					res.Namespace,
					res.Spec.ChainID,
					res.Spec.NodeService,
					res.Spec.NodeServiceEntry,
					res.Status.Destination,
					res.Status.ResponseFailedCount,
					res.Status.LatestBlockHeight,
					res.Status.HeightFailedCount,
				}
			}
		}
	}

	if len(caIstioInfoList) == 0 {
		return nil, nil
	}

	return caIstioInfoList, nil
}

func (w *istioModifier) changeDestination(ctx context.Context, chainID string, destination string) error {
	var vsl istionetworkingv1beta1.VirtualServiceList
	//일단은 istio virtual service의 네임스페이스 이름과 체인아이디가 같다고 가정하여 변경 대상을 조회
	err := w.client.List(ctx, &vsl, client.MatchingLabels{"vs-modifier-enable": "true"}, client.InNamespace(chainID))
	if err != nil {
		w.log.Error(err, "Failed to get VirtualServiceList ", "changeDestination()")
		return err
	}
	// 조회된 virtual service들의 destination을 모두 변경
	for _, vs := range vsl.Items {
		w.log.Info("vs:" + vs.Name)
		httproute := vs.Spec.GetHttp()
		for i := 0; i < len(httproute); i++ {
			routes := httproute[i].GetRoute()
			for j := 0; j < len(routes); j++ {
				httproute[i].GetRoute()[j].Destination.Host = destination
			}
		}
		err = w.client.Update(ctx, vs)
		if err != nil {
			w.log.WithValues("ChianID", chainID).Error(err, "failed to change Destination to "+destination)
		}
	}

	return nil
}

func (w *istioModifier) updateCurrentDestinationStatus(ctx context.Context) error {
	var cil operatorv1alpha1.CAIstioList
	var vsl istionetworkingv1beta1.VirtualServiceList

	err := w.client.List(ctx, &cil)
	if err != nil {
		w.log.Error(err, "Failed to get CAIstio List")
		return err
	}

	for _, ci := range cil.Items {
		opts := []client.ListOption{
			client.InNamespace(ci.Spec.ChainID),
			client.MatchingLabels{"vs-modifier-enable": "true"},
		}
		err = w.client.List(ctx, &vsl, opts...)

		if err != nil {
			w.log.Error(err, "Failed to get VirtualServices for "+ci.Spec.ChainID)
		}

		if len(vsl.Items) == 0 {
			w.log.Info("Cannot fild VirtualService for " + ci.Spec.ChainID)
		}

		destination := vsl.Items[0].Spec.GetHttp()[0].GetRoute()[0].Destination.Host
		ci.Status.Destination = destination

		err = w.client.Status().Update(ctx, &ci)

		if err != nil {
			w.log.Error(err, "Failed to update status['destination'] of "+ci.Name)
		}
	}
	return nil
}

func (w *istioModifier) reconcile(ctx context.Context) error {
	w.updateCurrentDestinationStatus(ctx)
	vsl, err := w.getVirtualServiceList(ctx)

	if err != nil {
		w.log.Error(err, "getVirtualServiceList failed")
		return nil
	}

	// caistio 리소스로 등록 된(백업 노드가 있는) 체인의 virtual service만 필터링하여 상태 모니터링
	//   -> caistio 리소스의 chainID값을 namespace 값으로 가지고 있는 virtual service

	caIstioInfoList, err := w.getCAIstioInfoList(ctx, vsl)

	if err != nil {
		w.log.Error(err, "getCAIstioInfoList failed")
		return nil
	}

	if len(caIstioInfoList) == 0 {
		w.log.Info("no virtual service to monitor")
		return nil
	}

	// monitor each chain from promtheus or ......
	// test :  0 ~ 6 failed count increment
	heightFailedCount := 0
	for k, v := range caIstioInfoList {
		var caistio operatorv1alpha1.CAIstio
		heightFailedCount = 0
		w.client.Get(ctx, types.NamespacedName{Namespace: v.namespace, Name: v.name}, &caistio)
		if caistio.Status.HeightFailedCount == 5 {
			heightFailedCount = 0
		} else {
			heightFailedCount = v.heightfailedcount + 1
		}

		caistio.Status.HeightFailedCount = heightFailedCount
		err = w.client.Status().Update(ctx, &caistio)
		if err != nil {
			w.log.Error(err, k+" get 'failed count' fail")
		}

		w.log.Info(k + " failed count (" + strconv.Itoa(heightFailedCount) + ")")
	}

	// change destination
	threshold := 3
	for k, v := range caIstioInfoList {
		if heightFailedCount >= threshold {
			//patch destination to service entry
			w.changeDestination(ctx, v.chainid, v.nodeserviceentry)
			w.log.Info(k + " destination is external node")
		} else {
			//patch destination to service
			w.changeDestination(ctx, v.chainid, v.nodeservice)
			w.log.Info(k + " destination is internal node")
		}

	}

	return nil
}
