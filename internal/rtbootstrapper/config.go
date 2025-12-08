package rtbootstrapper

import (
	"context"
	"fmt"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const fieldManagerName = "kim"

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
	configMap, err := c.prepareConfigMap(ctx)
	if err != nil {
		return fmt.Errorf("failed to prepare bootstrapper ConfigMap: %w", err)
	}

	var pullSecret *corev1.Secret
	if c.config.PullSecretName != "" {
		pullSecret, err = c.preparePullSecret(ctx)
		if err != nil {
			return fmt.Errorf("failed to prepare bootstrapper PullSecret: %w", err)
		}
	}

	// TODO: in future, add support for config.ClusterTrustBundleName, as optional parameter

	runtimeClient, err := c.runtimeClientGetter.Get(ctx, runtime)
	if err != nil {
		return fmt.Errorf("failed to get runtimeClient: %w", err)
	}

	return c.applyResourcesToRuntimeCluster(ctx, runtimeClient, pullSecret, configMap)
}

func getControlPlaneResource[T client.Object](ctx context.Context, kcpClient client.Client, name string) (T, error) {
	var resource T
	if err := kcpClient.Get(ctx, client.ObjectKey{Name: name, Namespace: "kcp-system"}, resource); err != nil {
		return resource, fmt.Errorf("failed to get resource %s: %w", name, err)
	}
	return resource, nil
}

func (c *Configurator) prepareConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	resource, err := getControlPlaneResource[*corev1.ConfigMap](ctx, c.kcpClient, c.config.ConfigName)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (c *Configurator) preparePullSecret(ctx context.Context) (*corev1.Secret, error) {
	resource, err := getControlPlaneResource[*corev1.Secret](ctx, c.kcpClient, c.config.PullSecretName)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (c *Configurator) applyResourcesToRuntimeCluster(ctx context.Context, runtimeClient client.Client, secret *corev1.Secret, configMap *corev1.ConfigMap) error {
	runtimeConfigMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMap.Name,
			Namespace: configMap.Namespace,
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
				Name:      secret.Name,
				Namespace: secret.Namespace,
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
	return nil
}
