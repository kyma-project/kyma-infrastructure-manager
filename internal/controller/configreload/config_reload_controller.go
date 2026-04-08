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

package configreload

import (
	"context"
	"fmt"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/reconciler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	ErrRuntimeNotificationFailed = fmt.Errorf("runtime notification failed")
	fieldManager                 = "config-watcher"
)

type RuntimePredicate func(configObject types.NamespacedName, runtime imv1.Runtime) bool

// ConfigReloadWatcher reconciles a Secret object
type ConfigReloadWatcher struct {
	KcpClient           client.Client
	Namespace           string
	ConfigMapPredicates []ObjectUpdatedPredicate
	SecretPredicates    []ObjectUpdatedPredicate
	RuntimePredicate    RuntimePredicate
}

// +kubebuilder:rbac:groups="",resources=secrets,verbs=watch;list,namespace=kcp-system
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=watch;list,namespace=kcp-system
// +kubebuilder:rbac:groups=certificates.k8s.io,resources=clustertrustbundles,verbs=watch;list,namespace=kcp-system
// +kubebuilder:rbac:groups=infrastructuremanager.kyma-project.io,resources=runtimes,verbs=list;patch,namespace=kcp-system

func (r *ConfigReloadWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	var runtimes imv1.RuntimeList
	err := r.KcpClient.List(ctx, &runtimes, &client.ListOptions{
		Namespace: r.Namespace,
	})
	if err != nil {
		logger.Error(err, "unable to list runtimes",
			"namespace", r.Namespace)
		return ctrl.Result{}, err
	}

	success := true

	logger.Info("Forcing configuration reloading on runtimes")

	for _, item := range runtimes.Items {
		if r.RuntimePredicate != nil && !r.RuntimePredicate(req.NamespacedName, item) {
			continue
		}

		if item.Annotations != nil && item.Annotations[reconciler.ForceReconcileAnnotation] == "true" {
			continue
		}

		newItem := item.DeepCopy()
		if newItem.Annotations == nil {
			newItem.Annotations = map[string]string{}
		}
		newItem.Annotations[reconciler.ForceReconcileAnnotation] = "true"
		newItem.ManagedFields = nil

		if err := r.KcpClient.Patch(ctx, newItem, client.Apply, &client.PatchOptions{
			FieldManager: fieldManager,
			Force:        ptr.To(true),
		}); err != nil {
			logger.Error(err, "unable to annotate runtime",
				"namespace", newItem.Namespace,
				"name", newItem.Name)

			success = false
		}
	}

	if !success {
		return ctrl.Result{}, ErrRuntimeNotificationFailed
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigReloadWatcher) SetupWithManager(mgr ctrl.Manager) error {
	controller := ctrl.NewControllerManagedBy(mgr).
		Named("config")

	for _, p := range r.ConfigMapPredicates {
		controller = controller.Watches(&corev1.ConfigMap{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(p))
	}

	for _, p := range r.SecretPredicates {
		controller = controller.Watches(&corev1.Secret{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(p))
	}

	return controller.Complete(r)
}
