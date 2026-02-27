package rtbootstrapper

import (
	"context"
	"fmt"
	"reflect"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	if c.config.KCPConfig.PullSecretName != "" {
		pullSecret, err = c.getPullSecret(ctx)
		if err != nil {
			return fmt.Errorf("failed to prepare bootstrapper PullSecret: %w", err)
		}
	}

	var clusterTrustBundle *certificatesv1beta1.ClusterTrustBundle
	if c.config.KCPConfig.ClusterTrustBundleName != "" {
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
	if err := getResource[*corev1.ConfigMap](ctx, c.kcpClient, c.config.KCPConfig.ConfigName, cm); err != nil {
		return nil, err
	}
	return cm, nil
}

func (c *Configurator) getPullSecret(ctx context.Context) (*corev1.Secret, error) {
	sec := &corev1.Secret{}
	if err := getResource[*corev1.Secret](ctx, c.kcpClient, c.config.KCPConfig.PullSecretName, sec); err != nil {
		return nil, err
	}
	return sec, nil
}

func (c *Configurator) getClusterTrustBundle(ctx context.Context) (*certificatesv1beta1.ClusterTrustBundle, error) {
	ctb := &certificatesv1beta1.ClusterTrustBundle{}
	if err := c.kcpClient.Get(ctx, client.ObjectKey{Name: c.config.KCPConfig.ClusterTrustBundleName}, ctb); err != nil {
		return nil, fmt.Errorf("failed to get ClusterTrustBundle %s: %w", c.config.KCPConfig.ClusterTrustBundleName, err)
	}
	return ctb, nil
}

func (c *Configurator) applyResourcesToRuntimeCluster(ctx context.Context, runtimeClient client.Client, secret *corev1.Secret, configMap *corev1.ConfigMap, clusterTrustBundle *certificatesv1beta1.ClusterTrustBundle) error {
	if err := c.applyConfigMap(ctx, runtimeClient, configMap); err != nil {
		return err
	}

	if secret != nil {
		if err := c.applySecret(ctx, runtimeClient, secret); err != nil {
			return err
		}
	}

	if clusterTrustBundle != nil {
		if err := c.applyClusterTrustBundle(ctx, runtimeClient, clusterTrustBundle); err != nil {
			return err
		}
	}

	return nil
}

func (c *Configurator) applyConfigMap(ctx context.Context, runtimeClient client.Client, configMap *corev1.ConfigMap) error {
	applyCM := true

	var existing corev1.ConfigMap
	err := runtimeClient.Get(ctx, client.ObjectKey{Name: c.config.SKRConfig.ConfigName, Namespace: c.config.SKRConfig.Namespace}, &existing)
	if err == nil {
		if equalConfigMap(existing, *configMap) {
			applyCM = false
		}
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check runtime ConfigMap: %w", err)
	}

	if applyCM {
		runtimeConfigMap := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.config.SKRConfig.ConfigName,
				Namespace: c.config.SKRConfig.Namespace,
			},
			Data: configMap.Data,
		}
		if err := runtimeClient.Patch(ctx, runtimeConfigMap, client.Apply, &client.PatchOptions{
			Force:        ptr.To(true),
			FieldManager: fieldManagerName,
		}); err != nil {
			return fmt.Errorf("failed to apply bootstrapper ConfigMap to runtime cluster: %w", err)
		}
	}
	return nil
}

func (c *Configurator) applySecret(ctx context.Context, runtimeClient client.Client, secret *corev1.Secret) error {
	applySecret := true

	var existing corev1.Secret
	err := runtimeClient.Get(ctx, client.ObjectKey{Name: c.config.SKRConfig.PullSecretName, Namespace: c.config.SKRConfig.Namespace}, &existing)
	if err == nil {
		if equalSecret(existing, *secret) {
			applySecret = false
		}
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check runtime Secret: %w", err)
	}

	if applySecret {
		secretToApply := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.config.SKRConfig.PullSecretName,
				Namespace: c.config.SKRConfig.Namespace,
			},
			Data: secret.Data,
			Type: secret.Type,
		}
		if err := runtimeClient.Patch(ctx, secretToApply, client.Apply, &client.PatchOptions{
			Force:        ptr.To(true),
			FieldManager: fieldManagerName,
		}); err != nil {
			return fmt.Errorf("failed to apply bootstrapper PullSecret to runtime cluster: %w", err)
		}
	}
	return nil
}

func (c *Configurator) applyClusterTrustBundle(ctx context.Context, runtimeClient client.Client, clusterTrustBundle *certificatesv1beta1.ClusterTrustBundle) error {
	applyCTB := true

	var existing certificatesv1beta1.ClusterTrustBundle
	err := runtimeClient.Get(ctx, client.ObjectKey{Name: c.config.SKRConfig.ClusterTrustBundleName}, &existing)
	if err == nil {
		if equalClusterTrustBundle(existing, *clusterTrustBundle) {
			applyCTB = false
		}
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check runtime ClusterTrustBundle: %w", err)
	}

	if applyCTB {
		ctbToApply := &certificatesv1beta1.ClusterTrustBundle{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterTrustBundle",
				APIVersion: "certificates.k8s.io/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: c.config.SKRConfig.ClusterTrustBundleName,
			},
			Spec: clusterTrustBundle.Spec,
		}
		if err := runtimeClient.Patch(ctx, ctbToApply, client.Apply, &client.PatchOptions{
			Force:        ptr.To(true),
			FieldManager: fieldManagerName,
		}); err != nil {
			return fmt.Errorf("failed to apply ClusterTrustBundle to runtime cluster: %w", err)
		}
	}
	return nil
}

// equality helpers (minimal, focused on fields used by tests)
func equalConfigMap(a corev1.ConfigMap, b corev1.ConfigMap) bool {
	return reflect.DeepEqual(a.Data, b.Data)
}

func equalSecret(a corev1.Secret, b corev1.Secret) bool {
	return reflect.DeepEqual(a.Data, b.Data)
}

func equalClusterTrustBundle(a certificatesv1beta1.ClusterTrustBundle, b certificatesv1beta1.ClusterTrustBundle) bool {
	return a.Spec.TrustBundle == b.Spec.TrustBundle
}
