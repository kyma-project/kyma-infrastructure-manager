package rtbootstrapper

import (
	"context"
	"github.com/pkg/errors"
	"k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type Validator struct {
	config       Config
	kcpClient    client.Client
	configurator *Configurator
}

func NewValidator(config Config, kcpClient client.Client) *Validator {
	return &Validator{
		config:       config,
		kcpClient:    kcpClient,
		configurator: NewConfigurator(kcpClient, nil, config),
	}
}

func (v Validator) Validate(ctx context.Context) error {
	if err := verifyManifestsConfigMap(ctx, v.config.ManifestsConfigMapName, v.kcpClient); err != nil {
		return err
	}

	if err := verifyDeploymentName(v.config.DeploymentNamespacedName); err != nil {
		return err
	}

	if err := verifyConfigMap(ctx, v.config.ConfigName, v.kcpClient); err != nil {
		return err
	}

	if err := verifyPullSecret(ctx, v.config.PullSecretName, v.kcpClient); err != nil {
		return err
	}

	return verifyClusterTrustBundle(ctx, v.config.ClusterTrustBundleName, v.kcpClient)
}

func verifyDeploymentName(deploymentNamespacedName string) error {
	deploymentNameParts := strings.Split(deploymentNamespacedName, string(types.Separator))

	if len(deploymentNameParts) != 2 || deploymentNameParts[0] == "" || deploymentNameParts[1] == "" {
		return errors.New("deployment namespaced name is invalid")
	}

	return nil
}

func verifyConfigMap(ctx context.Context, configMapName string, kcpClient client.Client) error {
	if configMapName == "" {
		return errors.New("config name is required")
	}

	var configMap corev1.ConfigMap
	if err := getResource(ctx, kcpClient, configMapName, &configMap); err != nil {
		return errors.New("unable to find Runtime Bootstrapper ConfigMap in KCP cluster")
	}

	return nil
}

func verifyClusterTrustBundle(ctx context.Context, clusterTrustBundleName string, kcpClient client.Client) error {
	if clusterTrustBundleName != "" {
		var clusterTrustBundle v1beta1.ClusterTrustBundle

		if err := kcpClient.Get(ctx, client.ObjectKey{Name: clusterTrustBundleName}, &clusterTrustBundle); err != nil {
			return errors.New("unable to find Cluster Trust Bundle")
		}
	}

	return nil
}

func verifyPullSecret(ctx context.Context, pullSecretName string, kcpClient client.Client) error {
	if pullSecretName != "" {
		var secret corev1.Secret
		if err := getResource(ctx, kcpClient, pullSecretName, &secret); err != nil {
			return errors.New("unable to find Runtime Bootstrapper pull secret in KCP cluster")
		}

		if secret.Type != corev1.SecretTypeDockerConfigJson {
			return errors.New("pull secret has invalid type, expected kubernetes.io/dockerconfigjson")
		}
	}

	return nil
}

func verifyManifestsConfigMap(ctx context.Context, configMapName string, kcpClient client.Client) error {
	if configMapName == "" {
		return errors.New("manifests config map name is required")
	}

	var configMap corev1.ConfigMap
	if err := getResource(ctx, kcpClient, configMapName, &configMap); err != nil {
		return errors.New("unable to find Manifests ConfigMap in KCP cluster")
	}

	return verifyManifests(string(configMap.Data["manifests.yaml"]))
}

func verifyManifests(manifests string) error {
	documents := strings.Split(manifests, "---")
	for _, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var obj interface{}
		if err := yaml.Unmarshal([]byte(doc), &obj); err != nil {
			return errors.Wrap(err, "invalid YAML manifests file")
		}
	}

	return nil
}
