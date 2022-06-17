/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	operatorv1alpha1 "github.com/lkh1434/ca-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// CAIstioReconciler reconciles a CAIstio object
type CAIstioReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=operator.ca.com,resources=caistios,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.ca.com,resources=caistios/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.ca.com,resources=caistios/finalizers,verbs=update
//+kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.istio.io,resources=serviceentries,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CAIstio object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *CAIstioReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrlog.FromContext(ctx)
	goobers.Inc()
	gooberFailures.Inc()
	// lookup the CAIstio Resources for this reconsile request
	caistio := &operatorv1alpha1.CAIstio{}
	err := r.Get(ctx, req.NamespacedName, caistio)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Deleted Resource Name: " + caistio.Name)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get CAIstio resources")
		return ctrl.Result{}, err
	}

	log.Info("Created or Modidied Resource Name: " + caistio.Name)

	//update cuurent destination
	curDestination, err := r.setInitialStauts(ctx, caistio)
	if err != nil {
		log.Error(err, "Failed to set Initial Status")
	}
	if curDestination != "" {
		log.Info("Current Destionation : " + curDestination)
	}

	//  Create Istio ServiceEntry(if not exists)
	err = r.checkServiceEntry(ctx, caistio)
	if err != nil {
		log.Error(err, "Failed to check Istio Service Entry")
	}

	return ctrl.Result{}, err
}

func ignoreStatusUpdatePredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
		// DeleteFunc: func(e event.DeleteEvent) bool {
		// 	// Evaluates to false if the object has been confirmed deleted.
		// 	return !e.DeleteStateUnknown
		// },
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *CAIstioReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.CAIstio{}).
		// Owns(&istionetworkingv1beta1.VirtualService{}).
		WithEventFilter(ignoreStatusUpdatePredicate()).
		Complete(r)
}
