package controllers

import (
	"context"

	operatorv1alpha1 "github.com/lkh1434/ca-operator/api/v1alpha1"
	"istio.io/api/networking/v1beta1"
	istionetworkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlog "sigs.k8s.io/controller-runtime/pkg/log"
)

func (w *CAIstioReconciler) setInitialStauts(ctx context.Context, caistio *operatorv1alpha1.CAIstio) (string, error) {
	log := ctrlog.FromContext(ctx)

	var vsl istionetworkingv1beta1.VirtualServiceList

	opts := []client.ListOption{
		client.InNamespace(caistio.Spec.ChainID),
		client.MatchingLabels{"vs-modifier-enable": "true"},
	}
	err := w.List(ctx, &vsl, opts...)
	if err != nil {
		log.Error(err, "Failed to get VirtualServices for "+caistio.Spec.ChainID)
		return "", err
	}

	if len(vsl.Items) == 0 {
		log.Info("Cannot fild VirtualService for " + caistio.Spec.ChainID)
		return "", nil
	}

	destination := vsl.Items[0].Spec.GetHttp()[0].GetRoute()[0].Destination.Host
	caistio.Status.Destination = destination

	// Reconciler가 caistio의 스테이터스를 초기화 시킬줄 알고 만들었는데 초기화 안시켜서 필요 없는데
	// 나중에 label 이나 annotation 키가 있는지 체크할 때 필요할 수도 있어서 일단 내비둠.
	// 스테이터스 정보를 json으로 받아서 맵에 넣어야 키 존재여부 체크가 가능
	// var statusMap map[string]interface{}
	// data, _ := json.Marshal(caistio.Status)
	// json.Unmarshal(data, &statusMap)
	// caistio.Status.ResponseFailedCount = 10
	// if _, ok := statusMap["responsefailedcount"]; !ok {
	// 	log.Info("다?" + "responsefailedcount")
	// 	caistio.Status.ResponseFailedCount = 0
	// }
	// if _, ok := statusMap["latestblockheight"]; !ok {
	// 	log.Info("다??" + "latestblockheight")
	// 	caistio.Status.LatestBlockHeight = 10
	// }
	// if _, ok := statusMap["heightfailedcount"]; !ok {
	// 	log.Info("다???" + "heightfailedcount")
	// 	caistio.Status.HeightFailedCount = 0
	// }

	err = w.Status().Update(ctx, caistio)

	if err != nil {
		log.Error(err, "Failed to update status['destination'] of "+caistio.Name)
		return "", err
	}

	return destination, nil
}

func (w *CAIstioReconciler) checkServiceEntry(ctx context.Context, caistio *operatorv1alpha1.CAIstio) error {
	log := ctrlog.FromContext(ctx)
	seDestination := caistio.Spec.NodeServiceEntry

	var sel istionetworkingv1beta1.ServiceEntryList

	// service entry 하나 당 endpoint는 하나여야 하고 같은 chain id와 네임스페이스가 같아야 함
	// label로 operator가 관리하는 서비스 엔트리로 지정할 순 있지만 label이나 annotaion은 권한 있는 사용자가 임의 변경 가능하기 때문에 대안을 확인할 필요 있음
	err := w.List(ctx, &sel, client.InNamespace(caistio.Spec.ChainID), client.MatchingLabels{"serviceentry/managed-by": "chainapsis"})
	if err != nil {
		log.Error(err, "Failed to get ServiceEntry")
		return err
	}

	createTF := false
	seName := caistio.Spec.ChainID + "-external-node"
	if len(sel.Items) > 0 {
		for _, se := range sel.Items {
			// 커스텀 리소스(caistio)에서 service entry 가 변경되었을대
			if (se.Name == seName) && (se.Spec.Endpoints[0].Address != seDestination) {
				se.Spec.Endpoints[0].Address = seDestination
				err = w.Update(ctx, se)
				if err != nil {
					log.Error(err, "Failed to update ServiceEntry")
				}
				log.Info("Update ServiceEntry")
				createTF = false
				break
			}
		}
	} else {
		createTF = true
	}

	if createTF {
		err = w.creatServiceEntry(ctx, caistio.Spec.ChainID, seDestination)
		if err != nil {
			log.Error(err, "Failed to create Service Entry for "+caistio.Spec.ChainID)
			return err
		}
	}
	return nil
}

func (w *CAIstioReconciler) creatServiceEntry(ctx context.Context, chainID string, seDestination string) error {
	log := ctrlog.FromContext(ctx)
	seName := chainID + "-external-node"

	labels := make(map[string]string)
	labels["serviceentry/managed-by"] = "chainapsis"

	var ports []*v1beta1.Port
	ports = append(ports, &v1beta1.Port{
		Number:   80,
		Protocol: "HTTP",
		Name:     "http",
	})
	// ports = append(ports,&v1beta1.Port{
	// 	Number:               443,
	// 	Protocol:             "HTTPS",
	// 	Name:                 "https",
	// })
	var endpoints []*v1beta1.WorkloadEntry

	endpointPortsMap := make(map[string]uint32)
	endpointPortsMap["http"] = 80

	endpoints = append(endpoints, &v1beta1.WorkloadEntry{
		Address: seDestination,
		Ports:   endpointPortsMap,
	})

	se := &istionetworkingv1beta1.ServiceEntry{
		ObjectMeta: v1.ObjectMeta{
			Name:      seName,
			Namespace: chainID,
			Labels:    labels,
		},
		Spec: v1beta1.ServiceEntry{
			Hosts:      []string{seName + ".com"},
			Ports:      ports,
			Location:   v1beta1.ServiceEntry_MESH_EXTERNAL,
			Resolution: v1beta1.ServiceEntry_DNS,
			Endpoints:  endpoints,
		},
	}

	existTF, err := w.checkNamespace(ctx, chainID)
	if err != nil {
		return nil
	}

	if existTF {
		err = w.Create(ctx, se)
		if err != nil {
			return err
		}
		log.Info("service entry careated")
	}

	return nil
}

func (w *CAIstioReconciler) checkNamespace(ctx context.Context, namespace string) (bool, error) {
	log := ctrlog.FromContext(ctx)
	existTF := true

	var nsl corev1.NamespaceList
	err := w.List(ctx, &nsl)
	if err != nil {
		log.Error(err, "Failed to check the namespace "+namespace)
		return false, err
	}
	for _, ns := range nsl.Items {
		if ns.Name == namespace {
			existTF = true
			break
		} else {
			existTF = false
		}
	}

	return existTF, nil
}
