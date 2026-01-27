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

type Cfg struct {
	ClusterTrustBundle types.NamespacedName
	ImagePullSecret    types.NamespacedName
	client.Client
}

// ConfigWatcher reconciles a Secret object
type ConfigWatcher struct {
	Scheme *runtime.Scheme
	Kcp    Cfg
}

var ErrConfigUpdateFailed = fmt.Errorf("configuration update failed")

// +kubebuilder:rbac:groups=kyma-project.io,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kyma-project.io,resources=secrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kyma-project.io,resources=secrets/finalizers,verbs=update

func (r *ConfigWatcher) UpdateImagePullSecret(ctx context.Context) error {
	var kcpSecret corev1.Secret

	if err := r.Kcp.Get(ctx, r.Kcp.ImagePullSecret, &kcpSecret); err != nil {
		return err
	}

	panic("not implemented yet")
}

func (r *ConfigWatcher) UpdateClusterTrustBundle(ctx context.Context) error {

	// err := r.Kcp.Get(ctx, r.Kcp.ClusterTrustBundle, &kcpClusterTrustBundle)

	panic("not implemented yet")
}

func (r *ConfigWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	updateSuccess := true
	for _, step := range []struct {
		name    string
		execute UpdateRsc
	}{
		{
			name:    "update image-pull-secret data",
			execute: r.UpdateImagePullSecret,
		},
	} {
		if err := step.execute(ctx); err != nil {
			logger.Error(err, "configuration update failed",
				"stepName", step.name,
			)
			updateSuccess = false
		}
	}

	if !updateSuccess {
		logger.Error(ErrConfigUpdateFailed, "failed to update rt-bootstrapper configuration")
		return ctrl.Result{}, ErrConfigUpdateFailed
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
		Complete(r)
}
