package customconfig

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	"github.com/kyma-project/infrastructure-manager/internal/registrycache"
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

// CustomConfigReconciler reconciles a secret object
// nolint:revive
type CustomSKRConfigReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Log           logr.Logger
	Cfg           fsm.RCCfg
	EventRecorder record.EventRecorder
	RequestID     atomic.Uint64
}

const fieldManagerName = "customconfigcontroller"

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list,namespace=kcp-system
func (r *CustomSKRConfigReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
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

func (r *CustomSKRConfigReconciler) reconcileRegistryCacheConfig(ctx context.Context, secret v1.Secret, runtime imv1.Runtime) (ctrl.Result, error) {
	enableRegistryCache, err := customConfigExists(ctx, secret)
	if err != nil {
		r.Log.V(log_level.TRACE).Error(err, "Failed to check if custom config exists")

		return ctrl.Result{
			Requeue: true,
		}, err
	}

	cachingAlreadyEnabled := runtime.Spec.Caching != nil && runtime.Spec.Caching.Enabled

	if cachingAlreadyEnabled != enableRegistryCache {
		runtime.Spec.Caching = &imv1.ImageRegistryCache{
			Enabled: enableRegistryCache,
		}

		r.Log.Info(fmt.Sprintf("Updating runtime %s with caching enabled: %t", runtime.Name, enableRegistryCache))

		err := r.Patch(ctx, &runtime, client.Apply, &client.PatchOptions{
			FieldManager: fieldManagerName,
			Force:        ptr.To(true),
		})

		if err != nil {
			r.Log.V(log_level.TRACE).Error(err, "Failed to patch runtime with caching enabled")
			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: time.Minute,
			}, err
		}
	}

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: 1 * time.Minute,
	}, err
}

func customConfigExists(ctx context.Context, kubeconfigSecret v1.Secret) (bool, error) {
	customConfigExplorer, err := registrycache.NewConfigExplorer(ctx, kubeconfigSecret)
	if err != nil {
		return false, err
	}

	return customConfigExplorer.RegistryCacheConfigExists()
}

func requeueOnError(err error) (ctrl.Result, error) {

	if err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{
			Requeue: false,
		}, err
	}

	if err != nil {
		return ctrl.Result{
			Requeue: true,
		}, err
	}

	return ctrl.Result{
		Requeue: false,
	}, err
}

func secretControlledByKIM(secret v1.Secret) bool {
	if secret.Labels == nil {
		return false
	}

	_, ok := secret.Labels["kyma-project.io/runtime-id"]

	return ok && secret.Labels["operator.kyma-project.io/managed-by"] == "infrastructure-manager"
}

func NewCustomConfigReconciler(mgr ctrl.Manager, logger logr.Logger) *CustomSKRConfigReconciler {
	return &CustomSKRConfigReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetEventRecorderFor("runtime-controller"),
		Log:           logger,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomSKRConfigReconciler) SetupWithManager(mgr ctrl.Manager, numberOfWorkers int) error {
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
