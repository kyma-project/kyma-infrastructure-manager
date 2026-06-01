package registrycache

import (
	"context"
	"fmt"
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
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
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
	fieldManagerName        = "customconfigcontroller"
	RegistryCacheModuleName = "registry-cache"
	RuntimeIDLabel          = "kyma-project.io/runtime-id"
)

func (r *RegistryCacheConfigReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	r.Log.V(log_level.TRACE).Info(request.String())

	var secret corev1.Secret
	if err := r.KcpClient.Get(ctx, request.NamespacedName, &secret); err != nil {
		return requeueOnError(err)
	}

	if !secretControlledByKIM(secret) {
		return ctrl.Result{}, nil
	}

	runtimeID := secret.Labels[RuntimeIDLabel]

	log := r.Log.WithValues("runtimeID", runtimeID, "secretName", request.Name)

	runtimeClient, err := r.RuntimeClientGetter(secret)
	if err != nil {
		log.Error(err, "Failed to get runtime client for runtime")
		return requeueOnError(err)
	}

	enabled, err := moduleEnabled(ctx, runtimeClient)
	if err != nil {
		log.Error(err, "Failed to verify whether Registry Cache should be enabled")
		return requeueOnError(err)
	}

	var runtimeList imv1.RuntimeList

	err = r.KcpClient.List(ctx, &runtimeList,
		client.MatchingLabels(map[string]string{RuntimeIDLabel: runtimeID}))

	if err != nil {
		log.Error(err, "Failed to find runtime")
		return requeueOnError(err)
	}

	if len(runtimeList.Items) == 0 || len(runtimeList.Items) > 1 {
		e := fmt.Errorf("expected to find one runtime for given runtime ID, found %d", len(runtimeList.Items))
		log.Error(e, "unexpected number of runtimes found", "count", len(runtimeList.Items))
		return ctrl.Result{}, nil
	}

	runtimeToUpdate := runtimeList.Items[0]

	if enabled || len(runtimeToUpdate.Spec.Caching) > 0 {
		return r.applyRegistryCacheConfig(ctx, log, runtimeClient, runtimeToUpdate, enabled)
	}

	return ctrl.Result{
		RequeueAfter: r.ReconcilePeriod,
	}, err
}

func (r *RegistryCacheConfigReconciler) applyRegistryCacheConfig(ctx context.Context, log logr.Logger, runtimeClient client.Client, runtime imv1.Runtime, enabled bool) (ctrl.Result, error) {

	var caches []imv1.ImageRegistryCache

	if enabled {
		var registryCacheConfigs registrycache.RegistryCacheConfigList
		err := runtimeClient.List(ctx, &registryCacheConfigs, &client.ListOptions{})
		if err != nil {
			log.Error(err, "Failed to list registry cache configs")
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
	log.Info("Updating runtime with registry cache config")
	runtime.Spec.Caching = caches
	runtime.ManagedFields = nil
	//nolint:staticcheck // SA1019: client.Apply is used with Patch, which is the correct API for this version
	err := r.KcpClient.Patch(ctx, &runtime, client.Apply, &client.PatchOptions{
		FieldManager: fieldManagerName,
		Force:        ptr.To(true),
	})

	if err != nil {
		log.Error(err, "Failed to patch runtime")
		return ctrl.Result{}, err
	}

	return ctrl.Result{
		RequeueAfter: r.ReconcilePeriod,
	}, err
}

func moduleEnabled(ctx context.Context, runtimeClient client.Client) (bool, error) {

	var kymacrd apiextensions.CustomResourceDefinition
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
