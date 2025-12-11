package rtbootstrapper

import (
	"context"
	"github.com/pkg/errors"
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

func NewValidator(config Config, kcpClient client.Client, runtimeClientGetter RuntimeClientGetter) *Validator {
	return &Validator{
		config:       config,
		kcpClient:    kcpClient,
		configurator: NewConfigurator(kcpClient, runtimeClientGetter, config),
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

	if !v.configurator.ValidateConfigMap(ctx) {
		return errors.New("unable to find Runtime Bootstrapper ConfigMap in KCP cluster")
	}

	if !v.configurator.ValidatePullSecretConfig(ctx, v.config) {
		return errors.New("unable to find Runtime Bootstrapper PullSecret in KCP cluster")
	}

	return nil
}
