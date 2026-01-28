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
	"github.com/kyma-project/infrastructure-manager/internal/rtbootstrapper"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type UpdateRsc func(context.Context) error

// ConfigWatcher reconciles a Secret object
type ConfigWatcher struct {
	Scheme *runtime.Scheme
	client.Client
	rtbootstrapper.Config
	*rtbootstrapper.Configurator
}

var ErrConfigUpdateFailed = fmt.Errorf("configuration update failed")

// +kubebuilder:rbac:groups=kyma-project.io,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kyma-project.io,resources=secrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kyma-project.io,resources=secrets/finalizers,verbs=update

func (r *ConfigWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	timeReconciliationStarted := time.Now()

	logger := logf.FromContext(ctx)
	defer func() {
		reconciliationDuration := time.Since(timeReconciliationStarted)
		logger.Info("reconciliation finished",
			"milliseconds", reconciliationDuration.Milliseconds())
	}()

	logger.Info("reconciliation started")

	var runtimes imv1.RuntimeList
	err := r.List(ctx, &runtimes, &client.ListOptions{
		Namespace: r.DeploymentNamespacedName,
	})

	if err != nil {
		logger.Error(err, "failed to list runtimes")
		return ctrl.Result{}, ErrConfigUpdateFailed
	}

	success := true
	for _, item := range runtimes.Items {
		if err := r.Configure(ctx, item); err != nil {
			logger.Error(err, "runtime configuration failed",
				"name", item.Name,
				"namespace", item.Namespace)
			success = false
		}

		logger.Info("runtime configured",
			"name", item.Name,
			"namespace", item.Namespace)
	}

	if !success {
		return ctrl.Result{}, ErrConfigUpdateFailed
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigWatcher) SetupWithManager(mgr ctrl.Manager) error {

	clusterTrustBundleID := types.NamespacedName{
		Name: r.ClusterTrustBundleName,
	}

	rtBootstrapperConfigurationID := types.NamespacedName{
		Name:      r.ConfigName,
		Namespace: r.DeploymentNamespacedName,
	}

	imagePullSecretID := types.NamespacedName{
		Name:      r.PullSecretName,
		Namespace: r.DeploymentNamespacedName,
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named("config").
		Watches(&certificatesv1beta1.ClusterTrustBundle{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(createResourcePredicate{clusterTrustBundleID})).
		Watches(&corev1.Secret{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(createResourcePredicate{imagePullSecretID})).
		Watches(&corev1.ConfigMap{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(createResourcePredicate{rtBootstrapperConfigurationID})).
		Complete(r)
}
