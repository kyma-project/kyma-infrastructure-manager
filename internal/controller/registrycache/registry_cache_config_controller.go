package registrycache

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sync/atomic"
	"time"
)

// RegistryCacheConfigReconciler reconciles a secret object
// nolint:revive
type RegistryCacheConfigReconciler struct {
	client.Client
	Scheme               *runtime.Scheme
	Log                  logr.Logger
	Cfg                  fsm.RCCfg
	EventRecorder        record.EventRecorder
	RequestID            atomic.Uint64
	RegistryCacheCreator RegistryCacheCreator
}

const fieldManagerName = "customconfigcontroller"

func (r *RegistryCacheConfigReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	r.Log.V(log_level.TRACE).Info(request.String())

	var secret v1.Secret
	if err := r.Get(ctx, request.NamespacedName, &secret); err != nil {
		return requeueOnError(err)
	}

	if !secretControlledByKIM(secret) {
		r.Log.V(log_level.TRACE).Info("Secret doesn't contain kubeconfig for runtime", "Name", request.Name, "Namespace", request.Namespace)
		return ctrl.Result{
			Requeue: false,
		}, nil
	}

	runtimeID := secret.Labels["kyma-project.io/runtime-id"]

	r.Log.V(log_level.TRACE).Info("Getting runtime", "Name", runtimeID, "Namespace", request.Namespace)

	var runtime imv1.Runtime
	if err := r.Get(ctx, types.NamespacedName{Name: runtimeID,
		Namespace: request.Namespace,
	}, &runtime); err != nil {
		return requeueOnError(err)
	}

	return r.reconcileRegistryCacheConfig(ctx, secret, runtime)
}

func (r *RegistryCacheConfigReconciler) reconcileRegistryCacheConfig(ctx context.Context, secret v1.Secret, runtime imv1.Runtime) (ctrl.Result, error) {

	registryCache, err := r.RegistryCacheCreator(secret)
	if err != nil {
		r.Log.V(log_level.TRACE).Error(err, "Failed to get runtime client for runtime", "RuntimeID", runtime.Name, "Namespace", runtime.Namespace)

		return ctrl.Result{}, err
	}

	registryCacheConfigs, err := registryCache.GetRegistryCacheConfig()
	if err != nil {
		r.Log.V(log_level.TRACE).Error(err, "Failed to get registry cache config", "RuntimeID", runtime.Name, "Namespace", runtime.Namespace)

		return ctrl.Result{}, err
	}

	caches := make([]imv1.ImageRegistryCache, 0, len(registryCacheConfigs))

	for _, config := range registryCacheConfigs {
		runtimeRegistryCacheConfig := imv1.ImageRegistryCache{
			Name:      config.Name,
			Namespace: config.Namespace,
			Config:    config.Spec,
		}
		caches = append(caches, runtimeRegistryCacheConfig)
	}
	runtime.Spec.Caching = caches

	r.Log.Info(fmt.Sprintf("Updating runtime %s with registry cache config", runtime.Name))

	runtime.ManagedFields = nil

	err = r.Patch(ctx, &runtime, client.Apply, &client.PatchOptions{
		FieldManager: fieldManagerName,
		Force:        ptr.To(true),
	})

	if err != nil {
		r.Log.V(log_level.TRACE).Error(err, "Failed to patch runtime")
		return ctrl.Result{}, err
	}

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: 5 * time.Minute,
	}, err
}

func requeueOnError(err error) (ctrl.Result, error) {

	if err != nil && apierrors.IsNotFound(err) {
		return ctrl.Result{
			Requeue: false,
		}, nil
	}

	return ctrl.Result{}, err
}

func secretControlledByKIM(secret v1.Secret) bool {
	if secret.Labels == nil {
		return false
	}

	_, ok := secret.Labels["kyma-project.io/runtime-id"]

	return ok && secret.Labels["operator.kyma-project.io/managed-by"] == "infrastructure-manager"
}

//go:generate mockery --name=RegistryCache
type RegistryCache interface {
	GetRegistryCacheConfig() ([]registrycache.RegistryCacheConfig, error)
}

type RegistryCacheCreator func(secret v1.Secret) (RegistryCache, error)

func NewRegistryCacheConfigReconciler(mgr ctrl.Manager, logger logr.Logger, registryCacheCreator RegistryCacheCreator) *RegistryCacheConfigReconciler {
	return &RegistryCacheConfigReconciler{
		Client:               mgr.GetClient(),
		Scheme:               mgr.GetScheme(),
		EventRecorder:        mgr.GetEventRecorderFor("runtime-controller"),
		Log:                  logger,
		RegistryCacheCreator: registryCacheCreator,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RegistryCacheConfigReconciler) SetupWithManager(mgr ctrl.Manager, numberOfWorkers int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Secret{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: numberOfWorkers}).
		WithEventFilter(predicate.Or(
			predicate.GenerationChangedPredicate{},
			predicate.LabelChangedPredicate{},
			predicate.AnnotationChangedPredicate{},
		)).
		Named("custom-config-controller").
		Complete(r)
}
