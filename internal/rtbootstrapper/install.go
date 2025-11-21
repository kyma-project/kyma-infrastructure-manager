package rtbootstrapper

import "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm"

type RuntimeInstaller struct {
	config              Config
	runtimeClientGetter fsm.RuntimeClientGetter
}

type Config struct {
	PullSecretName         string
	ClusterTrustBundleName string
	ManifestsPath          string
	ConfigPath             string
}

func NewInstaller(config Config, runtimeClientGetter fsm.RuntimeClientGetter) (*RuntimeInstaller, error) {
	err := validate(config)
	if err != nil {
		return nil, err
	}

	return &RuntimeInstaller{
		config: config,
	}, nil
}

func validate(config Config) error {
	return nil
}

func (r *RuntimeInstaller) Install(runtimeID string) error {
	return nil
}

func (r *RuntimeInstaller) Ready() (bool, error) {
	return true, nil
}
