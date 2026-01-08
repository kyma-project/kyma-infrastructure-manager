package rtbootstrapper

import (
	"context"
	"fmt"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const fieldManagerName = "kim-bootstrapper"

type Configurator struct {
	kcpClient           client.Client
	runtimeClientGetter RuntimeClientGetter
	config              Config
}

func NewConfigurator(kcpClient client.Client, runtimeClientGetter RuntimeClientGetter, config Config) *Configurator {
	return &Configurator{
		kcpClient:           kcpClient,
		runtimeClientGetter: runtimeClientGetter,
		config:              config,
	}
}

func (c *Configurator) Configure(ctx context.Context, runtime imv1.Runtime) error {
	configMap, err := c.getConfigMap(ctx)
	if err != nil {
		return fmt.Errorf("failed to prepare bootstrapper ConfigMap: %w", err)
	}

	var pullSecret *corev1.Secret
	if c.config.PullSecretName != "" {
		pullSecret, err = c.getPullSecret(ctx)
		if err != nil {
			return fmt.Errorf("failed to prepare bootstrapper PullSecret: %w", err)
		}
	}

	var clusterTrustBundle *certificatesv1beta1.ClusterTrustBundle
	if c.config.ClusterTrustBundleName != "" {
		clusterTrustBundle, err = c.getClusterTrustBundle(ctx)
		if err != nil {
			return fmt.Errorf("failed to prepare ClusterTrustBundle: %w", err)
		}
	}

	runtimeClient, err := c.runtimeClientGetter.Get(ctx, runtime)
	if err != nil {
		return fmt.Errorf("failed to get runtimeClient: %w", err)
	}

	return c.applyResourcesToRuntimeCluster(ctx, runtimeClient, pullSecret, configMap, clusterTrustBundle)
}

func getResource[T client.Object](ctx context.Context, kcpClient client.Client, name string, resource T) error {
	if err := kcpClient.Get(ctx, client.ObjectKey{Name: name, Namespace: "kcp-system"}, resource); err != nil {
		return fmt.Errorf("failed to get resource %s: %w", name, err)
	}
	return nil
}

func (c *Configurator) getConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	if err := getResource[*corev1.ConfigMap](ctx, c.kcpClient, c.config.ConfigName, cm); err != nil {
		return nil, err
	}
	return cm, nil
}

func (c *Configurator) getPullSecret(ctx context.Context) (*corev1.Secret, error) {
	sec := &corev1.Secret{}
	if err := getResource[*corev1.Secret](ctx, c.kcpClient, c.config.PullSecretName, sec); err != nil {
		return nil, err
	}
	return sec, nil
}

func (c *Configurator) getClusterTrustBundle(ctx context.Context) (*certificatesv1beta1.ClusterTrustBundle, error) {
	ctb := &certificatesv1beta1.ClusterTrustBundle{}
	if err := c.kcpClient.Get(ctx, client.ObjectKey{Name: c.config.ClusterTrustBundleName}, ctb); err != nil {
		return nil, fmt.Errorf("failed to get ClusterTrustBundle %s: %w", c.config.ClusterTrustBundleName, err)
	}
	return ctb, nil
}

func (c *Configurator) applyResourcesToRuntimeCluster(ctx context.Context, runtimeClient client.Client, secret *corev1.Secret, configMap *corev1.ConfigMap, clusterTrustBundle *certificatesv1beta1.ClusterTrustBundle) error {
	runtimeConfigMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			// TODO: make the name configurable
			Name:      "rt-bootstrapper-config",
			Namespace: "kyma-system",
		},
		Data: configMap.Data,
	}

	err := runtimeClient.Patch(ctx, runtimeConfigMap, client.Apply, &client.PatchOptions{
		Force:        ptr.To(true),
		FieldManager: fieldManagerName,
	})

	if err != nil {
		return fmt.Errorf("failed to apply bootstrapper ConfigMap to runtime cluster: %w", err)
	}

	if secret != nil {
		secretToApply := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				// TODO: make the name configurable
				Name:      "registry-credentials",
				Namespace: "kyma-system",
			},
			Data: secret.Data,
			Type: secret.Type,
		}

		err = runtimeClient.Patch(ctx, secretToApply, client.Apply, &client.PatchOptions{
			Force:        ptr.To(true),
			FieldManager: fieldManagerName,
		})
		if err != nil {
			return fmt.Errorf("failed to apply bootstrapper PullSecret to runtime cluster: %w", err)
		}
	}

	if clusterTrustBundle != nil {
		ctbToApply := &certificatesv1beta1.ClusterTrustBundle{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterTrustBundle",
				APIVersion: "certificates.k8s.io/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterTrustBundle.Name,
			},
			Spec: clusterTrustBundle.Spec,
		}

		err = runtimeClient.Patch(ctx, ctbToApply, client.Apply, &client.PatchOptions{
			Force:        ptr.To(true),
			FieldManager: fieldManagerName,
		})
		if err != nil {
			return fmt.Errorf("failed to apply ClusterTrustBundle to runtime cluster: %w", err)
		}
	}

	return nil
}
