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

package config

import (
	"context"
	"fmt"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	AnnotationConfigurationChanged = "operator.kyma-project.io/configuration-updated"
	ErrRuntimeNotificationFailed   = fmt.Errorf("runtime notification failed")
	fieldManager                   = "config-watcher"
)

type UpdateRsc func(context.Context) error

// TODO talk about potential alerting if the configuration is invalid

type Cfg struct {
	Namespace          string
	ClusterTrustBundle types.NamespacedName
	ImagePullSecret    types.NamespacedName
	RtBootstrapperCfg  types.NamespacedName
	client.Client
}

// ConfigWatcher reconciles a Secret object
type ConfigWatcher struct {
	Scheme *runtime.Scheme
	Kcp    Cfg
}

// +kubebuilder:rbac:groups=kyma-project.io,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kyma-project.io,resources=secrets/status,verbs=get;update;patch

// +kubebuilder:rbac:groups=kyma-project.io,resources=secrets/finalizers,verbs=update

func (r *ConfigWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	var runtimes imv1.RuntimeList
	err := r.Kcp.List(ctx, &runtimes, &client.ListOptions{
		Namespace: r.Kcp.Namespace,
	})
	if err != nil {
		logger.Error(err, "unable to list runtimes",
			"namespace", r.Kcp.Namespace)
		return ctrl.Result{}, err
	}

	now := time.Now()
	success := true
	for _, item := range runtimes.Items {
		var rt imv1.Runtime

		rt.Name = item.Name
		rt.Namespace = item.Namespace
		rt.Annotations = item.Annotations
		rt.Annotations[AnnotationConfigurationChanged] = fmt.Sprintf("%d", now.UnixMicro())

		if err := r.Kcp.Patch(ctx, &item, client.Apply, &client.PatchOptions{
			FieldManager: fieldManager,
			Force:        ptr.To(true),
		}); err != nil {
			logger.Error(err, "unable to annotate runtime",
				"namespace", item.Namespace,
				"name", item.Name)

			success = false
		}
	}

	if !success {
		return ctrl.Result{}, ErrRuntimeNotificationFailed
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigWatcher) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		Named("config").
		Watches(&certificatesv1beta1.ClusterTrustBundle{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(createResourcePredicate{
				r.Kcp.ClusterTrustBundle})).
		Watches(&corev1.Secret{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(createResourcePredicate{
				r.Kcp.ImagePullSecret})).
		Watches(&corev1.ConfigMap{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(createResourcePredicate{
				r.Kcp.RtBootstrapperCfg})).
		Complete(r)
}
