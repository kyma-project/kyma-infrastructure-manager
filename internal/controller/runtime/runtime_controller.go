/*
Copyright 2023.

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

package runtime

import (
	"context"
	"sync/atomic"

	"github.com/go-logr/logr"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// RuntimeReconciler reconciles a Runtime object
// nolint:revive
type RuntimeReconciler struct {
	KcpClient client.Client
	Scheme    *runtime.Scheme
	SeedClient    client.Client
	Log           logr.Logger
	Cfg           fsm.RCCfg
	EventRecorder record.EventRecorder
	RequestID     atomic.Uint64
}

//+kubebuilder:rbac:groups=infrastructuremanager.kyma-project.io,resources=runtimes,verbs=get;list;watch;create;update;patch,namespace=kcp-system
//+kubebuilder:rbac:groups=infrastructuremanager.kyma-project.io,resources=runtimes/status,verbs=get;list;delete;create;update;patch,namespace=kcp-system
//+kubebuilder:rbac:groups=infrastructuremanager.kyma-project.io,resources=runtimes/finalizers,verbs=get;list;delete;create;update;patch,namespace=kcp-system

func (r *RuntimeReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	r.Log.V(log_level.TRACE).Info(request.String())

	var runtime imv1.Runtime
	if err := r.KcpClient.Get(ctx, request.NamespacedName, &runtime); err != nil {
		return ctrl.Result{
			Requeue: false,
		}, client.IgnoreNotFound(err)
	}

	runtimeID, ok := runtime.Labels["kyma-project.io/runtime-id"]
	if !ok {
		runtimeID = runtime.Name
	}

	log := r.Log.WithValues("runtimeID", runtimeID, "shootName", runtime.Spec.Shoot.Name, "requestID", r.RequestID.Add(1))
	log.Info("Reconciling Runtime", "Name", runtime.Name, "Namespace", runtime.Namespace)

	stateFSM := fsm.NewFsm(
		log,
		r.Cfg,
		fsm.K8s{
			KcpClient:     r.KcpClient,
			SeedClient:    r.SeedClient,
			EventRecorder: r.EventRecorder,
		})

	return stateFSM.Run(ctx, runtime)
}

func NewRuntimeReconciler(mgr ctrl.Manager, shootClient client.Client, logger logr.Logger, cfg fsm.RCCfg) *RuntimeReconciler {
	return &RuntimeReconciler{
		KcpClient:     mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		SeedClient:    shootClient,
		EventRecorder: mgr.GetEventRecorderFor("runtime-controller"),
		Log:           logger,
		Cfg:           cfg,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RuntimeReconciler) SetupWithManager(mgr ctrl.Manager, numberOfWorkers int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&imv1.Runtime{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: numberOfWorkers}).
		WithEventFilter(predicate.Or(
			predicate.GenerationChangedPredicate{},
			predicate.LabelChangedPredicate{},
			predicate.AnnotationChangedPredicate{},
		)).
		Complete(r)
}
