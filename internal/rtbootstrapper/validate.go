package rtbootstrapper

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
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
	if err := verifyManifests(v.config.ManifestsPath); err != nil {
		return err
	}

	if err := verifyDeploymentName(v.config.DeploymentNamespacedName); err != nil {
		return err
	}

	if err := verifyConfigMap(ctx, v.config.ConfigName, v.kcpClient); err != nil {
		return err
	}

	return verifyPullSecret(ctx, v.config.PullSecretName, v.kcpClient)
}

func verifyManifests(manifestPath string) error {
	if manifestPath == "" {
		return errors.New("manifests path is required")
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return errors.New(fmt.Sprintf("manifests file does not exists under path %s", manifestPath))
	}

	documents := strings.Split(string(data), "---")
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
