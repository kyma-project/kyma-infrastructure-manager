package registrycache

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	kyma "github.com/kyma-project/lifecycle-manager/api/v1beta2"
	registrycache "github.com/kyma-project/registry-cache/api/v1beta1"
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
	RuntimeClientGetter  RuntimeClientGetter
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
		return ctrl.Result{}, nil
	}

	runtimeID := secret.Labels["kyma-project.io/runtime-id"]

	runtimeClient, err := r.RuntimeClientGetter(secret)
	if err != nil {
		r.Log.V(log_level.TRACE).Error(err, "Failed to get runtime client for runtime", "RuntimeID", runtimeID, "Namespace", secret.Namespace)
		return requeueOnError(err)
	}

	registryCacheEnabled, err := registryCacheEnabled(ctx, runtimeClient)
	if err != nil {
		r.Log.V(log_level.TRACE).Error(err, "Failed to verify whether Registry Cache should be enabled", "RuntimeID", runtimeID, "Namespace", secret.Namespace)
		return requeueOnError(err)
	}

	var runtime imv1.Runtime
	if err := r.KcpClient.Get(ctx, types.NamespacedName{Name: runtimeID,
		Namespace: request.Namespace,
	}, &runtime); err != nil {
		return requeueOnError(err)
	}

	if registryCacheEnabled ||
		(registryCacheEnabled == false && len(runtime.Spec.Caching) > 0) {
		r.Log.V(log_level.TRACE).Info("Getting runtime", "Name", runtimeID, "Namespace", request.Namespace)

		return r.reconcileRegistryCacheConfig(ctx, runtimeClient, runtime, registryCacheEnabled)
	}

	return ctrl.Result{
		RequeueAfter: 5 * time.Minute,
	}, err
}

func (r *RegistryCacheConfigReconciler) reconcileRegistryCacheConfig(ctx context.Context, runtimeClient client.Client, runtime imv1.Runtime, enabled bool) (ctrl.Result, error) {

	var caches []imv1.ImageRegistryCache

	if enabled {
		var registryCacheConfigs registrycache.RegistryCacheConfigList
		err := runtimeClient.List(ctx, &registryCacheConfigs, &client.ListOptions{})
		if err != nil {
			r.Log.V(log_level.TRACE).Error(err, "Failed to list registry cache configs", "RuntimeID", runtime.Name, "Namespace", runtime.Namespace)

			return ctrl.Result{}, err
		}

		for _, config := range registryCacheConfigs.Items {
			runtimeRegistryCacheConfig := imv1.ImageRegistryCache{
				Name:      config.Name,
				Namespace: config.Namespace,
				UID:       string(config.UID),
				Config:    config.Spec,
			}
			caches = append(caches, runtimeRegistryCacheConfig)
		}
	}
	r.Log.Info(fmt.Sprintf("Updating runtime %s with registry cache config", runtime.Name))
	runtime.Spec.Caching = caches
	runtime.ManagedFields = nil
	err := r.KcpClient.Patch(ctx, &runtime, client.Apply, &client.PatchOptions{
		FieldManager: fieldManagerName,
		Force:        ptr.To(true),
	})

	if err != nil {
		r.Log.V(log_level.TRACE).Error(err, "Failed to patch runtime")
		return ctrl.Result{}, err
	}

	return ctrl.Result{
		RequeueAfter: 5 * time.Minute,
	}, err
}

func registryCacheEnabled(ctx context.Context, runtimeClient client.Client) (bool, error) {
	var defaultKyma kyma.Kyma
	err := runtimeClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: "kyma-system"}, &defaultKyma)
	if err != nil {
		return false, err
	}

	kymaModules := defaultKyma.Spec.Modules

	for _, m := range kymaModules {
		if m.Name == RegistryCacheModuleName {
			return true, nil
		}
	}

	// Fallback: search for CRD
	// This is a temporary solution until module is available to be installed
	var crd apiextensions.CustomResourceDefinition
	crdErr := runtimeClient.Get(ctx, types.NamespacedName{Name: "registrycacheconfigs.core.kyma-project.io"}, &crd)
	if crdErr != nil {
		if apierrors.IsNotFound(crdErr) {
			return false, nil
		}
		return false, crdErr
	}

	return true, nil
}

func requeueOnError(err error) (ctrl.Result, error) {

	if err != nil && apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
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

type RegistryCacheCreator func(runtimeClient client.Client) (RegistryCache, error)
type RuntimeClientGetter func(secret corev1.Secret) (client.Client, error)

func NewRegistryCacheConfigReconciler(mgr ctrl.Manager, logger logr.Logger, runtimeClientGetter RuntimeClientGetter) *RegistryCacheConfigReconciler {
	return &RegistryCacheConfigReconciler{
		KcpClient:           mgr.GetClient(),
		Scheme:              mgr.GetScheme(),
		EventRecorder:       mgr.GetEventRecorderFor("runtime-controller"),
		Log:                 logger,
		RuntimeClientGetter: runtimeClientGetter,
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
