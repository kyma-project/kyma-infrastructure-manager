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

package controller

import (
	"context"
	"fmt"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/go-logr/logr"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	gardener_shoot "github.com/kyma-project/infrastructure-manager/internal/gardener/shoot"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	//	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type ShootClient interface {
	Create(ctx context.Context, shoot *gardener.Shoot, opts v1.CreateOptions) (*gardener.Shoot, error)
	Update(ctx context.Context, shoot *gardener.Shoot, opts v1.UpdateOptions) (*gardener.Shoot, error)
	UpdateStatus(ctx context.Context, shoot *gardener.Shoot, opts v1.UpdateOptions) (*gardener.Shoot, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*gardener.Shoot, error)
	List(ctx context.Context, opts v1.ListOptions) (*gardener.ShootList, error)
}

// RuntimeReconciler reconciles a Runtime object
type RuntimeReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	ShootClient ShootClient
	Log         logr.Logger
	requeueTime time.Duration
}

//+kubebuilder:rbac:groups=infrastructuremanager.kyma-project.io,resources=runtimes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructuremanager.kyma-project.io,resources=runtimes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructuremanager.kyma-project.io,resources=runtimes/finalizers,verbs=update

/*
const (
	// LastOperationTypeCreate indicates a 'create' operation.
	LastOperationTypeCreate LastOperationType = "Create"
	// LastOperationTypeReconcile indicates a 'reconcile' operation.
	LastOperationTypeReconcile LastOperationType = "Reconcile"
	// LastOperationTypeDelete indicates a 'delete' operation.
	LastOperationTypeDelete LastOperationType = "Delete"
	// LastOperationTypeMigrate indicates a 'migrate' operation.
	LastOperationTypeMigrate LastOperationType = "Migrate"
	// LastOperationTypeRestore indicates a 'restore' operation.
	LastOperationTypeRestore LastOperationType = "Restore"
)

*/

func (r *RuntimeReconciler) processShoot(ctx context.Context, shoot *gardener.Shoot, rt imv1.Runtime) (ctrl.Result, error) {
	// update parameters of the shoot if necessary

	if shoot.Spec.DNS == nil || shoot.Spec.DNS.Domain == nil {
		msg := fmt.Sprintf("DNS Domain is not set yet for shoot: %s, scheduling for retry", shoot.Name)
		r.Log.Info(msg)
		return ctrl.Result{RequeueAfter: r.requeueTime}, nil
	}

	lastOperation := shoot.Status.LastOperation

	if lastOperation == nil {
		msg := fmt.Sprintf("Last operation is nil for shoot: %s, scheduling for retry", shoot.Name)
		r.Log.Info(msg)
		return ctrl.Result{RequeueAfter: r.requeueTime}, nil
	}

	if shoot.Status.LastOperation.State == gardener.LastOperationStateSucceeded || shoot.Status.LastOperation.State == gardener.LastOperationStateProcessing {
		msg := fmt.Sprintf("Shoot %s is in %s state, scheduling for retry", shoot.Name, shoot.Status.LastOperation.State)
		r.Log.Info(msg)
		return ctrl.Result{RequeueAfter: r.requeueTime}, nil
	}

	// Error handling
	if lastOperation.State == gardener.LastOperationStateFailed {

		/*var reason apperrors.ErrReason

		if len(shoot.Status.LastErrors) > 0 {
			reason = util.GardenerErrCodesToErrReason(shoot.Status.LastErrors...)
		}

		if gardencorev1beta1helper.HasErrorCode(shoot.Status.LastErrors, v1beta1.ErrorInfraRateLimitsExceeded) {
			return operations.StageResult{}, apperrors.External("error during cluster provisioning: rate limits exceeded").SetComponent(apperrors.ErrGardener).SetReason(reason)
		}*/

		if lastOperation.Type == gardener.LastOperationTypeReconcile {
			msg := fmt.Sprintf("Shoot %s reconcilation error, scheduling for retry", shoot.Name)
			r.Log.Info(msg)
			return ctrl.Result{RequeueAfter: r.requeueTime}, nil
		}
	}

	if lastOperation.State == gardener.LastOperationStateSucceeded {
		// shoot is ready - end processing
		msg := fmt.Sprintf("Shoot %s is in %s state, shoot is ready", shoot.Name, shoot.Status.LastOperation.State)
		r.Log.Info(msg)
		return ctrl.Result{}, nil
	}

	msg := fmt.Sprintf("Shoot %s is in %s state, scheduling for reconcile", shoot.Name, shoot.Status.LastOperation.State)
	r.Log.Info(msg)
	return ctrl.Result{RequeueAfter: r.requeueTime}, nil

	r.Log.Info("Processing shoot", "Name", shoot.Name, "Namespace", shoot.Namespace)
	return ctrl.Result{}, nil
}

func (r *RuntimeReconciler) createShoot(ctx context.Context, runtime imv1.Runtime) (ctrl.Result, error) {

	converterConfig := fixConverterConfig()
	converter := gardener_shoot.NewConverter(converterConfig)
	shoot, err := converter.ToShoot(runtime)

	if err != nil {
		r.Log.Error(err, "unable to convert Runtime CR to a shoot object")
		return ctrl.Result{}, err
	}

	r.Log.Info("Shoot mapped", "Name", shoot.Name, "Namespace", shoot.Namespace, "Shoot", shoot)

	createdShoot, provisioningErr := r.ShootClient.Create(ctx, &shoot, v1.CreateOptions{})

	if provisioningErr != nil {
		r.Log.Error(provisioningErr, "unable to create Shoot")
		return ctrl.Result{}, provisioningErr
	}
	r.Log.Info("Shoot created successfully", "Name", createdShoot.Name, "Namespace", createdShoot.Namespace)

	return ctrl.Result{}, nil
}

func (r *RuntimeReconciler) deleteShoot(ctx context.Context, name types.NamespacedName) (ctrl.Result, error) {
	r.Log.Info("Deleting Shoot", "Name", name.Name, "from namespace", name.Namespace)
	// Delete the runtime
	deprovisioningErr := r.ShootClient.Delete(ctx, name.Name, v1.DeleteOptions{})

	if deprovisioningErr != nil {
		r.Log.Error(deprovisioningErr, "unable to delete Shoot")
		return ctrl.Result{}, deprovisioningErr
	}
	r.Log.Info("Shoot deleted successfully", "Name", name.Name, "Namespace", name.Namespace)

	return ctrl.Result{}, nil
}

func (r *RuntimeReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	r.Log.Info(request.String())

	var runtime imv1.Runtime

	err := r.Get(ctx, request.NamespacedName, &runtime)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return r.deleteShoot(ctx, request.NamespacedName)
		}
		r.Log.Error(err, "Error while fetching Runtime")
		return ctrl.Result{}, err
	}

	r.Log.Info("Reconciling Runtime", "Name", runtime.Name, "Namespace", runtime.Namespace)

	shoot, error := r.ShootClient.Get(ctx, runtime.Spec.Shoot.Name, v1.GetOptions{})

	if error != nil {
		if k8serrors.IsNotFound(err) {
			return r.createShoot(ctx, runtime)
		}
		r.Log.Error(err, "Error while checking if shoot already exists")
		return ctrl.Result{}, err
	}

	return r.processShoot(ctx, shoot, runtime)
}

func fixConverterConfig() gardener_shoot.ConverterConfig {
	return gardener_shoot.ConverterConfig{
		Kubernetes: gardener_shoot.KubernetesConfig{
			DefaultVersion: "1.29",
		},

		DNS: gardener_shoot.DNSConfig{
			SecretName:   "xxx-secret-dev",
			DomainPrefix: "runtimeprov.dev.kyma.ondemand.com",
			ProviderType: "aws-route53",
		},
		Provider: gardener_shoot.ProviderConfig{
			AWS: gardener_shoot.AWSConfig{
				EnableIMDSv2: true,
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RuntimeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&imv1.Runtime{}).
		WithEventFilter(predicate.And(predicate.GenerationChangedPredicate{})).
		Complete(r)
}
