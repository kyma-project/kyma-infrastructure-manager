package registrycache

import (
	"context"
	"fmt"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	"github.com/kyma-project/infrastructure-manager/internal/registrycache/runtimewatcher"
	kyma "github.com/kyma-project/lifecycle-manager/api/v1beta2"
	registrycache "github.com/kyma-project/registry-cache/api/v1beta1"
	watcherevent "github.com/kyma-project/runtime-watcher/listener/pkg/v2/event"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// RegistryCacheConfigReconciler reconciles a secret object
// nolint:revive
type RegistryCacheConfigReconciler struct {
	KcpClient           client.Client
	Scheme              *runtime.Scheme
	Log                 logr.Logger
	Cfg                 fsm.RCCfg
	EventRecorder       record.EventRecorder
	RequestID           atomic.Uint64
	RuntimeClientGetter RuntimeClientGetter
	KcpNamespace        string
	ReconcilePeriod     time.Duration
}

const (
	fieldManagerName        = "registrycachecontroller"
	RegistryCacheModuleName = "registry-cache"
	RuntimeIDLabel          = "kyma-project.io/runtime-id"
)

func (r *RegistryCacheConfigReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	r.Log.V(log_level.TRACE).Info(request.String())

	var secret corev1.Secret
	if err := r.KcpClient.Get(ctx, request.NamespacedName, &secret); err != nil {
		return stopIfNotFound(err, r.Log.WithValues("secretName", request.Name), "Failed to get secret", "Secret not found, skipping reconciliation")
	}

	if !secretControlledByKIM(secret) {
		return stop()
	}

	runtimeID := secret.Labels[RuntimeIDLabel]

	log := r.Log.WithValues("runtimeID", runtimeID, "secretName", request.Name)

	var runtime imv1.Runtime
	if err := r.KcpClient.Get(ctx, types.NamespacedName{Name: runtimeID, Namespace: r.KcpNamespace}, &runtime); err != nil {
		return stopIfNotFound(err, log, "Failed to get runtime", "Runtime not found, skipping reconciliation")
	}

	if !runtime.GetDeletionTimestamp().IsZero() || runtime.Status.State == imv1.RuntimeStateTerminating {
		log.Info("Skipping reconciliation, runtime is being deleted")
		return stop()
	}

	runtimeClient, err := r.RuntimeClientGetter(secret)
	if err != nil {
		log.Error(err, "Failed to get runtime client for runtime")
		return ctrl.Result{}, err
	}

	return r.applyRegistryCacheConfig(ctx, log, runtimeClient, runtimeID)
}

func (r *RegistryCacheConfigReconciler) applyRegistryCacheConfig(ctx context.Context, log logr.Logger, runtimeClient client.Client, runtimeID string) (ctrl.Result, error) {
	newRegistryCacheConfig, err := fetchConfigs(ctx, log, runtimeClient)
	if err != nil {
		return ctrl.Result{}, err
	}

	var runtimeToUpdate imv1.Runtime
	err = r.KcpClient.Get(ctx, types.NamespacedName{Name: runtimeID, Namespace: r.KcpNamespace}, &runtimeToUpdate)
	if err != nil {
		return stopIfNotFound(err, log, "Failed to get runtime", "Runtime not found, skipping reconciliation")
	}

	err = r.updateRuntime(ctx, log, runtimeToUpdate, newRegistryCacheConfig)
	if err != nil {
		return stopIfNotFound(err, log, "Failed to update runtime with registry cache config", "Runtime not found, skipping reconciliation")
	}

	return ctrl.Result{
		RequeueAfter: r.ReconcilePeriod,
	}, nil
}

func (r *RegistryCacheConfigReconciler) updateRuntime(ctx context.Context, log logr.Logger, runtimeToUpdate imv1.Runtime, newRegistryCacheConfig []imv1.ImageRegistryCache) error {

	if apiequality.Semantic.DeepEqual(newRegistryCacheConfig, runtimeToUpdate.Spec.Caching) {
		return nil
	}

	log.Info("Updating runtime with registry cache config")
	runtimeToUpdate.Spec.Caching = newRegistryCacheConfig

	return r.KcpClient.Update(ctx, &runtimeToUpdate, &client.UpdateOptions{
		FieldManager: fieldManagerName,
	})
}

func fetchConfigs(ctx context.Context, log logr.Logger, runtimeClient client.Client) ([]imv1.ImageRegistryCache, error) {
	enabled, err := moduleEnabled(ctx, runtimeClient)
	if err != nil {
		log.Error(err, "Failed to verify whether Registry Cache is enabled")
		return nil, err
	}

	if !enabled {
		return nil, nil
	}

	var registryCacheConfigs registrycache.RegistryCacheConfigList
	err = runtimeClient.List(ctx, &registryCacheConfigs, &client.ListOptions{})
	if err != nil {
		log.Error(err, "Failed to list registry cache configs")
		return nil, err
	}

	imageRegistryCaches := make([]imv1.ImageRegistryCache, 0, len(registryCacheConfigs.Items))
	for _, config := range registryCacheConfigs.Items {
		runtimeRegistryCacheConfig := imv1.ImageRegistryCache{
			Name:      config.Name,
			Namespace: config.Namespace,
			UID:       string(config.UID),
			Config:    config.Spec,
		}
		imageRegistryCaches = append(imageRegistryCaches, runtimeRegistryCacheConfig)
	}

	return imageRegistryCaches, nil
}

func moduleEnabled(ctx context.Context, runtimeClient client.Client) (bool, error) {

	var kymacrd apiextensionsv1.CustomResourceDefinition
	crdErr := runtimeClient.Get(ctx, types.NamespacedName{Name: "kymas.operator.kyma-project.io"}, &kymacrd)
	if crdErr != nil {
		if apierrors.IsNotFound(crdErr) {
			return false, nil
		}
		return false, crdErr
	}

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

	return false, nil
}

func stop() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func stopIfNotFound(err error, log logr.Logger, errMsg, infoMsg string) (ctrl.Result, error) {

	if err != nil && apierrors.IsNotFound(err) {
		log.Info(infoMsg)
		return ctrl.Result{}, nil
	}

	log.Error(err, errMsg)

	return ctrl.Result{}, err
}

func secretControlledByKIM(secret corev1.Secret) bool {
	if secret.Labels == nil {
		return false
	}

	_, ok := secret.Labels[RuntimeIDLabel]

	return ok && secret.Labels["operator.kyma-project.io/managed-by"] == "infrastructure-manager"
}

type RuntimeClientGetter func(secret corev1.Secret) (client.Client, error)

func NewRegistryCacheConfigReconciler(mgr ctrl.Manager, logger logr.Logger, kcpNamespace string, runtimeClientGetter RuntimeClientGetter, reconcilePeriod time.Duration) *RegistryCacheConfigReconciler {
	return &RegistryCacheConfigReconciler{
		KcpClient: mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		//nolint:staticcheck // SA1019: GetEventRecorderFor is used, which is the correct API for this version
		EventRecorder:       mgr.GetEventRecorderFor("runtime-controller"),
		Log:                 logger,
		RuntimeClientGetter: runtimeClientGetter,
		KcpNamespace:        kcpNamespace,
		ReconcilePeriod:     reconcilePeriod,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RegistryCacheConfigReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, numberOfWorkers int, registryCacheListenerPort, registryCacheListenerComponentName string) error {
	runnableListener := watcherevent.NewSKREventListener(
		fmt.Sprintf(":%s", registryCacheListenerPort),
		registryCacheListenerComponentName,
	)
	runnableListener.Logger = r.Log

	if err := mgr.Add(runnableListener); err != nil {
		return fmt.Errorf("RegistryCacheReconciler %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: numberOfWorkers}).
		WithEventFilter(predicate.Or(
			predicate.GenerationChangedPredicate{},
			predicate.LabelChangedPredicate{},
			predicate.AnnotationChangedPredicate{},
		)).
		WatchesRawSource(source.Channel(runtimewatcher.AdaptEvents(ctx, runnableListener.ReceivedEvents), runtimewatcher.CreateSkrEventHandler(r.Log, r.KcpNamespace))).
		Named("registry-config-controller").
		Complete(r)
}
