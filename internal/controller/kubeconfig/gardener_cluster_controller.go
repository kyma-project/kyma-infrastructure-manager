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

package kubeconfig

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	pkgctrl "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	lastKubeconfigSyncAnnotation      = "operator.kyma-project.io/last-sync"
	forceKubeconfigRotationAnnotation = "operator.kyma-project.io/force-kubeconfig-rotation"
	clusterCRNameLabel                = "operator.kyma-project.io/cluster-name"

	rotationPeriodRatio = 0.95
)

// GardenerClusterController reconciles a GardenerCluster object
type GardenerClusterController struct {
	client.Client
	Scheme                   *runtime.Scheme
	KubeconfigProvider       KubeconfigProvider
	log                      logr.Logger
	rotationPeriod           time.Duration
	minimalRotationTimeRatio float64
	gardenerRequestTimeout   time.Duration
	metrics                  metrics.Metrics
}

func NewGardenerClusterController(mgr ctrl.Manager, kubeconfigProvider KubeconfigProvider, logger logr.Logger, rotationPeriod time.Duration, minimalRotationTimeRatio float64, gardenerRequestTimeout time.Duration, metrics metrics.Metrics) *GardenerClusterController {
	return &GardenerClusterController{
		Client:                   mgr.GetClient(),
		Scheme:                   mgr.GetScheme(),
		KubeconfigProvider:       kubeconfigProvider,
		log:                      logger,
		rotationPeriod:           rotationPeriod,
		minimalRotationTimeRatio: minimalRotationTimeRatio,
		gardenerRequestTimeout:   gardenerRequestTimeout,
		metrics:                  metrics,
	}
}

// nolint:revive
//
//go:generate mockery --name=KubeconfigProvider
type KubeconfigProvider interface {
	Fetch(ctx context.Context, shootName string) (string, error)
}

//+kubebuilder:rbac:groups=infrastructuremanager.kyma-project.io,resources=gardenerclusters,verbs=get;list;watch;create;update;patch;delete,namespace=kcp-system
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;delete,namespace=kcp-system
//+kubebuilder:rbac:groups=infrastructuremanager.kyma-project.io,resources=gardenerclusters/finalizers,verbs=get;list;delete;create;update;patch,namespace=kcp-system
//+kubebuilder:rbac:groups=infrastructuremanager.kyma-project.io,resources=gardenerclusters/status,verbs=get;list;delete;create;update;patch,namespace=kcp-system

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the GardenerCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (controller *GardenerClusterController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) { //nolint:revive
	controller.log.Info("Starting reconciliation.", loggingContext(req)...)
	reconciliationContext, cancel := context.WithTimeout(ctx, controller.gardenerRequestTimeout)
	defer cancel()

	var cluster imv1.GardenerCluster

	err := controller.Get(reconciliationContext, req.NamespacedName, &cluster)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			controller.unsetMetrics(req)
			err = controller.deleteKubeconfigSecret(reconciliationContext, req.Name)
		}

		if err == nil {
			controller.log.Info("The gardener cluster does not exist the coresponding secret has been deleted.", loggingContext(req)...)
		}

		return controller.resultWithoutRequeue(&cluster), err
	}

	secret, err := controller.getSecret(reconciliationContext, cluster.Spec.Shoot.Name)
	if err != nil && !k8serrors.IsNotFound(err) {
		cluster.UpdateConditionForErrorState(imv1.ConditionTypeKubeconfigManagement, imv1.ConditionReasonFailedToGetSecret, err)
		_ = controller.persistStatusChange(reconciliationContext, &cluster)
		return controller.resultWithoutRequeue(&cluster), err
	}

	var annotations map[string]string
	if secret != nil {
		annotations = secret.Annotations
	}

	lastSyncTime, _ := findLastSyncTime(annotations)
	now := time.Now().UTC()
	requeueAfter := nextRequeue(now, lastSyncTime, controller.rotationPeriod, rotationPeriodRatio)

	controller.log.V(log_level.DEBUG).WithValues(loggingContextFromCluster(&cluster)...).Info("rotation params",
		"lastSync", lastSyncTime.Format("2006-01-02 15:04:05"),
		"requeueAfter", requeueAfter.String(),
		"gardenerRequestTimeout", controller.gardenerRequestTimeout.String(),
	)

	kubeconfigStatus, err := controller.handleKubeconfig(reconciliationContext, secret, &cluster, now)
	if err != nil {
		_ = controller.persistStatusChange(reconciliationContext, &cluster)
		// if a claster was not found in gardener,
		// CRD should not be rereconciled
		if k8serrors.IsNotFound(err) {
			err = nil
		}
		return controller.resultWithoutRequeue(&cluster), err
	}

	// there was a request to rotate the kubeconfig
	if kubeconfigStatus == ksRotated {
		err = controller.removeForceRotationAnnotation(reconciliationContext, &cluster)
		if err != nil {
			return controller.resultWithoutRequeue(&cluster), err
		}
	}

	if err := controller.persistStatusChange(reconciliationContext, &cluster); err != nil {
		return controller.resultWithoutRequeue(&cluster), err
	}

	return controller.resultWithRequeue(&cluster, requeueAfter), nil
}

func (controller *GardenerClusterController) unsetMetrics(req ctrl.Request) {
	controller.metrics.CleanUpGardenerClusterGauge(req.NamespacedName.Name)
	controller.metrics.CleanUpKubeconfigExpiration(req.NamespacedName.Name)
}

func loggingContextFromCluster(cluster *imv1.GardenerCluster) []any {
	return []any{"GardenerCluster", cluster.Name, "Namespace", cluster.Namespace}
}

func loggingContext(req ctrl.Request) []any {
	return []any{"GardenerCluster", req.Name, "Namespace", req.Namespace}
}

func (controller *GardenerClusterController) resultWithRequeue(cluster *imv1.GardenerCluster, requeueAfter time.Duration) ctrl.Result {
	controller.metrics.SetGardenerClusterStates(*cluster)

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: requeueAfter,
	}
}

func (controller *GardenerClusterController) resultWithoutRequeue(cluster *imv1.GardenerCluster) ctrl.Result { //nolint:unparam
	controller.metrics.SetGardenerClusterStates(*cluster)
	return ctrl.Result{}
}

func (controller *GardenerClusterController) persistStatusChange(reconciliationContext context.Context, cluster *imv1.GardenerCluster) error {
	err := controller.Status().Update(reconciliationContext, cluster)
	if err != nil {
		controller.log.Error(err, "status update failed")
	}
	return err
}

func (controller *GardenerClusterController) deleteKubeconfigSecret(reconciliationContext context.Context, clusterCRName string) error {
	selector := client.MatchingLabels(map[string]string{
		clusterCRNameLabel: clusterCRName,
	})

	var secretList corev1.SecretList
	err := controller.List(reconciliationContext, &secretList, selector)

	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	// secret was already deleted, nothing to do
	if len(secretList.Items) < 1 {
		return nil
	}

	if len(secretList.Items) > 1 {
		return errors.Errorf("unexpected numer of secrets found for cluster CR `%s`", clusterCRName)
	}

	return controller.Delete(reconciliationContext, &secretList.Items[0])
}

func (controller *GardenerClusterController) getSecret(ctx context.Context, shootName string) (*corev1.Secret, error) {
	var secretList corev1.SecretList

	shootNameSelector := client.MatchingLabels(map[string]string{
		"kyma-project.io/shoot-name": shootName,
	})

	err := controller.List(ctx, &secretList, shootNameSelector)
	if err != nil {
		return nil, err
	}

	size := len(secretList.Items)

	if size == 0 {
		return nil, k8serrors.NewNotFound(schema.GroupResource{}, "") //nolint:all TODO: update with valid GroupResource
	}

	if size > 1 {
		return nil, fmt.Errorf("unexpected numer of secrets found for shoot `%s`", shootName)
	}

	return &secretList.Items[0], nil
}

type kubeconfigStatus int

const (
	ksZero kubeconfigStatus = iota
	ksCreated
	ksModified
	ksRotated
)

func (controller *GardenerClusterController) handleKubeconfig(ctx context.Context, secret *corev1.Secret, cluster *imv1.GardenerCluster, now time.Time) (kubeconfigStatus, error) {
	kubeconfig, err := controller.KubeconfigProvider.Fetch(ctx, cluster.Spec.Shoot.Name)
	if err != nil {
		cluster.UpdateConditionForErrorState(imv1.ConditionTypeKubeconfigManagement, imv1.ConditionReasonFailedToGetKubeconfig, err)
		return ksZero, err
	}

	if secretRotationForced(cluster) {
		message := fmt.Sprintf("Rotation of secret %s in namespace %s forced.", cluster.Spec.Kubeconfig.Secret.Name, cluster.Spec.Kubeconfig.Secret.Namespace)
		controller.log.V(log_level.DEBUG).Info(message, loggingContextFromCluster(cluster)...)

		// delete secret containing kubeconfig to be rotated
		if err := controller.removeKubeconfig(ctx, cluster, secret); err != nil {
			cluster.UpdateConditionForErrorState(imv1.ConditionTypeKubeconfigManagement, imv1.ConditionReasonFailedToDeleteSecret, err)
			return ksZero, err
		}

		controller.metrics.CleanUpKubeconfigExpiration(cluster.Name)

		return ksRotated, nil
	}

	if !secretNeedsToBeRotated(cluster, secret, controller.rotationPeriod, now) {
		message := fmt.Sprintf("Secret %s in namespace %s does not need to be rotated yet.", cluster.Spec.Kubeconfig.Secret.Name, cluster.Spec.Kubeconfig.Secret.Namespace)
		controller.log.V(log_level.DEBUG).Info(message, loggingContextFromCluster(cluster)...)
		cluster.UpdateConditionForReadyState(imv1.ConditionTypeKubeconfigManagement, imv1.ConditionReasonKubeconfigSecretCreated, metav1.ConditionTrue)
		controller.metrics.SetKubeconfigExpiration(*secret, controller.rotationPeriod, controller.minimalRotationTimeRatio)
		return ksZero, nil
	}

	if secret != nil {
		return ksModified, controller.updateExistingSecret(ctx, kubeconfig, cluster, secret, now)
	}

	return ksCreated, controller.createNewSecret(ctx, kubeconfig, cluster, now)
}

func secretNeedsToBeRotated(cluster *imv1.GardenerCluster, secret *corev1.Secret, rotationPeriod time.Duration, now time.Time) bool {
	return secretRotationTimePassed(secret, rotationPeriod, now) || secretRotationForced(cluster)
}

func secretRotationTimePassed(secret *corev1.Secret, rotationPeriod time.Duration, now time.Time) bool {
	if secret == nil {
		return true
	}

	annotations := secret.GetAnnotations()
	lastSyncTime, found := findLastSyncTime(annotations)
	if !found {
		return true
	}

	minutesToRotate := now.Sub(lastSyncTime).Minutes()
	minutesInRotationPeriod := rotationPeriodRatio * rotationPeriod.Minutes()

	return minutesToRotate >= minutesInRotationPeriod
}

func secretRotationForced(cluster *imv1.GardenerCluster) bool {
	annotations := cluster.GetAnnotations()
	if annotations == nil {
		return false
	}

	_, found := annotations[forceKubeconfigRotationAnnotation]
	return found
}

func (controller *GardenerClusterController) createNewSecret(ctx context.Context, kubeconfig string, cluster *imv1.GardenerCluster, now time.Time) error {
	newSecret := controller.newSecret(*cluster, kubeconfig, now)
	err := controller.Create(ctx, &newSecret)
	if err != nil {
		cluster.UpdateConditionForErrorState(imv1.ConditionTypeKubeconfigManagement, imv1.ConditionReasonFailedToCreateSecret, err)
		return err
	}

	cluster.UpdateConditionForReadyState(imv1.ConditionTypeKubeconfigManagement, imv1.ConditionReasonKubeconfigSecretCreated, metav1.ConditionTrue)
	controller.metrics.SetKubeconfigExpiration(newSecret, controller.rotationPeriod, controller.minimalRotationTimeRatio)
	message := fmt.Sprintf("Secret %s has been created in %s namespace.", newSecret.Name, newSecret.Namespace)
	controller.log.V(log_level.DEBUG).Info(message, loggingContextFromCluster(cluster)...)

	return nil
}

func (controller *GardenerClusterController) removeKubeconfig(ctx context.Context, cluster *imv1.GardenerCluster, existingSecret *corev1.Secret) error {
	if existingSecret == nil {
		return nil
	}

	delete(existingSecret.Data, cluster.Spec.Kubeconfig.Secret.Key)

	if annotations := existingSecret.GetAnnotations(); annotations != nil {
		delete(annotations, lastKubeconfigSyncAnnotation)
	}

	return controller.Update(ctx, existingSecret)
}

func (controller *GardenerClusterController) updateExistingSecret(ctx context.Context, kubeconfig string, cluster *imv1.GardenerCluster, existingSecret *corev1.Secret, lastSyncTime time.Time) error {
	if existingSecret.Data == nil {
		existingSecret.Data = map[string][]byte{}
	}
	existingSecret.Data[cluster.Spec.Kubeconfig.Secret.Key] = []byte(kubeconfig)
	annotations := existingSecret.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	annotations[lastKubeconfigSyncAnnotation] = lastSyncTime.UTC().Format(time.RFC3339)
	existingSecret.SetAnnotations(annotations)

	err := controller.Update(ctx, existingSecret)
	if err != nil {
		cluster.UpdateConditionForErrorState(imv1.ConditionTypeKubeconfigManagement, imv1.ConditionReasonFailedToUpdateSecret, err)

		return err
	}

	cluster.UpdateConditionForReadyState(imv1.ConditionTypeKubeconfigManagement, imv1.ConditionReasonKubeconfigSecretRotated, metav1.ConditionTrue)
	controller.metrics.SetKubeconfigExpiration(*existingSecret, controller.rotationPeriod, controller.minimalRotationTimeRatio)

	message := fmt.Sprintf("Secret %s has been updated in %s namespace.", existingSecret.Name, existingSecret.Namespace)
	controller.log.V(log_level.DEBUG).Info(message, loggingContextFromCluster(cluster)...)

	return nil
}

func (controller *GardenerClusterController) removeForceRotationAnnotation(ctx context.Context, cluster *imv1.GardenerCluster) error {
	secretRotationForced := secretRotationForced(cluster)

	if secretRotationForced {
		key := types.NamespacedName{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
		}
		var clusterToUpdate imv1.GardenerCluster

		err := controller.Client.Get(ctx, key, &clusterToUpdate)
		if err != nil {
			return err
		}

		annotations := clusterToUpdate.GetAnnotations()
		delete(annotations, forceKubeconfigRotationAnnotation)
		clusterToUpdate.SetAnnotations(annotations)

		return controller.Update(ctx, &clusterToUpdate)
	}

	return nil
}

func (controller *GardenerClusterController) newSecret(cluster imv1.GardenerCluster, kubeconfig string, now time.Time) corev1.Secret {
	labels := map[string]string{}

	for key, val := range cluster.Labels {
		labels[key] = val
	}
	labels["operator.kyma-project.io/managed-by"] = "infrastructure-manager"
	labels[clusterCRNameLabel] = cluster.Name

	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        cluster.Spec.Kubeconfig.Secret.Name,
			Namespace:   cluster.Spec.Kubeconfig.Secret.Namespace,
			Labels:      labels,
			Annotations: map[string]string{lastKubeconfigSyncAnnotation: now.UTC().Format(time.RFC3339)},
		},
		StringData: map[string]string{cluster.Spec.Kubeconfig.Secret.Key: kubeconfig},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (controller *GardenerClusterController) SetupWithManager(mgr ctrl.Manager, numberOfWorkers int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&imv1.GardenerCluster{}, builder.WithPredicates(predicate.Or(
			predicate.LabelChangedPredicate{},
			predicate.AnnotationChangedPredicate{},
			predicate.GenerationChangedPredicate{}),
		)).
		WithOptions(pkgctrl.Options{MaxConcurrentReconciles: numberOfWorkers}).
		Complete(controller)
}
