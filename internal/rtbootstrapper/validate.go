package rtbootstrapper

import (
	"context"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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
	if v.config.ManifestsPath == "" {
		return errors.New("manifests path is required")
	}

	if _, err := os.Stat(v.config.ManifestsPath); err != nil {
		return errors.Wrapf(err, "manifests path %s is invalid", v.config.ManifestsPath)
	}

	deploymentNameParts := strings.Split(v.config.DeploymentNamespacedName, string(types.Separator))

	if len(deploymentNameParts) != 2 || deploymentNameParts[0] == "" || deploymentNameParts[1] == "" {
		return errors.New("deployment namespaced name is invalid")
	}
	// TODO: consider validating is file contains valid yaml

	if v.config.ConfigName == "" {
		return errors.New("config name is required")
	}

	var configMap corev1.ConfigMap
	if err := getResource(ctx, v.kcpClient, v.config.ConfigName, &configMap); err != nil {
		return errors.New("unable to find Runtime Bootstrapper ConfigMap in KCP cluster")
	}

	if v.config.PullSecretName != "" {
		var secret corev1.Secret
		if err := getResource(ctx, v.kcpClient, v.config.PullSecretName, &secret); err != nil {
			return errors.New("unable to find Runtime Bootstrapper PullSecret in KCP cluster")
		}

		if secret.Type != corev1.SecretTypeDockercfg {
			return errors.New("pull secret has invalid type, expected kubernetes.io/dockerconfigjson")
		}
	}

	return nil
}
