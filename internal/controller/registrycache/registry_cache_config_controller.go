package registrycache

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	kyma "github.com/kyma-project/lifecycle-manager/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
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
	KcpClient client.Client
	// RuntimeClient        client.Client // shouldn't be needed if we use RuntimeConfigurationManager?
	Scheme               *runtime.Scheme
	Log                  logr.Logger
	Cfg                  fsm.RCCfg
	EventRecorder        record.EventRecorder
	RequestID            atomic.Uint64
	RegistryCacheCreator RegistryCacheCreator
}

const (
	fieldManagerName        = "customconfigcontroller"
	RegistryCacheModuleName = "registry-cache"
)

func (r *RegistryCacheConfigReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	r.Log.V(log_level.TRACE).Info(request.String())

	var secret corev1.Secret
	if err := r.KcpClient.Get(ctx, request.NamespacedName, &secret); err != nil {
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
	if err := r.KcpClient.Get(ctx, types.NamespacedName{Name: runtimeID,
		Namespace: request.Namespace,
	}, &runtime); err != nil {
		return requeueOnError(err)
	}

	runtimeClient, err := gardener.GetRuntimeClient(secret)
	if err != nil {
		return requeueOnError(err)
	}

	return r.reconcileRegistryCacheConfig(ctx, secret, runtime)
}

func (r *RegistryCacheConfigReconciler) reconcileRegistryCacheConfig(ctx context.Context, secret corev1.Secret, runtime imv1.Runtime) (ctrl.Result, error) {

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

	//Synchronize runtime.spec.imageRegistryCache with valid Registry Cache configs (replace the old list in Runtime CR with a new one)
	caches := make([]imv1.ImageRegistryCache, 0, len(registryCacheConfigs))
	for _, config := range registryCacheConfigs {
		runtimeRegistryCacheConfig := imv1.ImageRegistryCache{
			Name:      config.Name,
			Namespace: config.Namespace,
			UID:       string(config.UID),
			Config:    config.Spec,
		}
		caches = append(caches, runtimeRegistryCacheConfig)
	}
	runtime.Spec.Caching = caches
	r.Log.Info(fmt.Sprintf("Updating runtime %s with registry cache config", runtime.Name))
	runtime.ManagedFields = nil
	err = r.KcpClient.Patch(ctx, &runtime, client.Apply, &client.PatchOptions{
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

func registryCacheEnabled(ctx context.Context, runtimeClient client.Client) (bool, error) {
	var kyma kyma.Kyma
	err := runtimeClient.Get(ctx, types.NamespacedName{Name: "kyma", Namespace: "kyma-system"}, &kyma)
	if err != nil {
		return false, err
	}

	kymaModules := kyma.Status.Modules

	for _, v := range kymaModules {
		if v.Name == RegistryCacheModuleName {
			return true, nil
		}
	}

	// Fallback: search for crd
	// This is a temporary solution until module is available to be installed

}

func requeueOnError(err error) (ctrl.Result, error) {

	if err != nil && apierrors.IsNotFound(err) {
		return ctrl.Result{
			Requeue: false,
		}, nil
	}

	return ctrl.Result{}, err
}

func secretControlledByKIM(secret corev1.Secret) bool {
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

type RegistryCacheCreator func(secret corev1.Secret) (RegistryCache, error)

func NewRegistryCacheConfigReconciler(mgr ctrl.Manager, logger logr.Logger, registryCacheCreator RegistryCacheCreator) *RegistryCacheConfigReconciler {
	return &RegistryCacheConfigReconciler{
		KcpClient:            mgr.GetClient(),
		Scheme:               mgr.GetScheme(),
		EventRecorder:        mgr.GetEventRecorderFor("runtime-controller"),
		Log:                  logger,
		RegistryCacheCreator: registryCacheCreator,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RegistryCacheConfigReconciler) SetupWithManager(mgr ctrl.Manager, numberOfWorkers int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: numberOfWorkers}).
		WithEventFilter(predicate.Or(
			predicate.GenerationChangedPredicate{},
			predicate.LabelChangedPredicate{},
			predicate.AnnotationChangedPredicate{},
		)).
		Named("registry-config-controller").
		Complete(r)
}
