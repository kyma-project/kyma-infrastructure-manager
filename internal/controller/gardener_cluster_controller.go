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
	"time"

	"github.com/go-logr/logr"
	infrastructuremanagerv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	lastKubeconfigSyncAnnotation = "operator.kyma-project.io/last-sync"
	clusterCRNameLabel           = "operator.kyma-project.io/cluster-name"
	defaultRequeuInSeconds       = 60 * 1000
)

// GardenerClusterController reconciles a GardenerCluster object
type GardenerClusterController struct {
	client.Client
	Scheme             *runtime.Scheme
	KubeconfigProvider KubeconfigProvider
	log                logr.Logger
}

func NewGardenerClusterController(mgr ctrl.Manager, kubeconfigProvider KubeconfigProvider, logger logr.Logger) *GardenerClusterController {
	return &GardenerClusterController{
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		KubeconfigProvider: kubeconfigProvider,
		log:                logger,
	}
}

//go:generate mockery --name=KubeconfigProvider
type KubeconfigProvider interface {
	Fetch(shootName string) (string, error)
}

//+kubebuilder:rbac:groups=infrastructuremanager.kyma-project.io,resources=gardenerclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups=infrastructuremanager.kyma-project.io,resources=gardenerclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the GardenerCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *GardenerClusterController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) { //nolint:revive
	r.log.Info(fmt.Sprintf("Starting reconciliation loop for GardenerCluster resource: %v", req.NamespacedName))

	var cluster infrastructuremanagerv1.GardenerCluster

	err := r.Client.Get(ctx, req.NamespacedName, &cluster)
	if err != nil {
		cluster.UpdateState(infrastructuremanagerv1.ErrorState, infrastructuremanagerv1.ConditionReasonGardenClusterNotRetrieved, "Couldn't retrieve the Garden Cluster CR")

		if k8serrors.IsNotFound(err) {
			cluster.UpdateState(infrastructuremanagerv1.DeletingState, infrastructuremanagerv1.ConditionReasonGardenClusterNotFound, "CR not found, secret will be deleted")
			err = r.deleteSecret(req.NamespacedName.Name)
			if err != nil {
				r.log.Error(err, "failed to delete secret")
			}
		}

		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: defaultRequeuInSeconds,
		}, err
	}

	secret, err := r.getSecret(cluster.Spec.Shoot.Name)
	if err != nil {
		r.log.Error(err, "could not get the Secret for "+cluster.Spec.Shoot.Name)
		cluster.UpdateState(infrastructuremanagerv1.ErrorState, infrastructuremanagerv1.ConditionReasonSecretNotFound, "Secret not found, and will be created")
		if !k8serrors.IsNotFound(err) {

			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: defaultRequeuInSeconds,
			}, err
		}
	}

	if secret == nil {
		r.log.Error(err, "Secret not found, and will be created")
		cluster.UpdateState(infrastructuremanagerv1.ErrorState, infrastructuremanagerv1.ConditionReasonSecretNotFound, "Secret not found, and will be created")
		err = r.createSecret(ctx, cluster)
		if err != nil {
			return r.ResultWithoutRequeue(), err
		}

	}

	cluster.UpdateState(infrastructuremanagerv1.ConditionReasonSecretCreated, infrastructuremanagerv1.ReadyState, "GardenCluster is ready")
	return ctrl.Result{}, nil
}

func (r *GardenerClusterController) deleteSecret(clusterCRName string) error {
	selector := client.MatchingLabels(map[string]string{
		clusterCRNameLabel: clusterCRName,
	})

	var secretList corev1.SecretList
	err := r.Client.List(context.TODO(), &secretList, selector)
	if err != nil {
		return err
	}

	if len(secretList.Items) != 1 {
		return errors.Errorf("unexpected numer of secrets found for cluster CR `%s`", clusterCRName)
	}

	return r.Client.Delete(context.TODO(), &secretList.Items[0])
}

func (r *GardenerClusterController) ResultWithoutRequeue() ctrl.Result {
	return ctrl.Result{
		Requeue: false,
	}
}

func (r *GardenerClusterController) getSecret(shootName string) (*corev1.Secret, error) {
	var secretList corev1.SecretList

	shootNameSelector := client.MatchingLabels(map[string]string{
		"kyma-project.io/shoot-name": shootName,
	})

	err := r.Client.List(context.Background(), &secretList, shootNameSelector)
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

func (r *GardenerClusterController) createSecret(ctx context.Context, cluster infrastructuremanagerv1.GardenerCluster) error {
	secret, err := r.newSecret(cluster)
	if err != nil {
		return err
	}

	return r.Client.Create(ctx, &secret)
}

func (r *GardenerClusterController) newSecret(cluster infrastructuremanagerv1.GardenerCluster) (corev1.Secret, error) {
	labels := map[string]string{}

	for key, val := range cluster.Labels {
		labels[key] = val
	}
	labels["operator.kyma-project.io/managed-by"] = "infrastructure-manager"
	labels[clusterCRNameLabel] = cluster.Name

	kubeconfig, err := r.KubeconfigProvider.Fetch(cluster.Spec.Shoot.Name)
	if err != nil {
		return corev1.Secret{}, err
	}

	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        cluster.Spec.Kubeconfig.Secret.Name,
			Namespace:   cluster.Spec.Kubeconfig.Secret.Namespace,
			Labels:      labels,
			Annotations: map[string]string{lastKubeconfigSyncAnnotation: time.Now().UTC().String()},
		},
		StringData: map[string]string{cluster.Spec.Kubeconfig.Secret.Key: kubeconfig},
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GardenerClusterController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructuremanagerv1.GardenerCluster{}).
		Complete(r)
}
