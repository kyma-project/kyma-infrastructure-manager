package registrycache

import (
	"context"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RuntimeConfigurationManager is responsible for managing runtime configurations (RegistryCacheConfig and corresponding Secrets).
type RuntimeConfigurationManager struct {
	runtimeClient client.Client
	Context       context.Context
}

type GetSecretFunc func() (corev1.Secret, error)

func NewRuntimeConfigurationManager(ctx context.Context, runtimeClient client.Client) *RuntimeConfigurationManager {
	return &RuntimeConfigurationManager{
		runtimeClient: runtimeClient,
		Context:       ctx,
	}
}

func (c *RuntimeConfigurationManager) RegistryCacheConfigExists() (bool, error) {
	registryCaches, err := c.GetRegistryCacheConfig()

	if err != nil {
		return false, err
	}

	return len(registryCaches) > 0, nil
}

func (c *RuntimeConfigurationManager) GetRegistryCacheConfig() ([]registrycache.RegistryCacheConfig, error) {
	var registryCacheConfigList registrycache.RegistryCacheConfigList
	err := c.runtimeClient.List(c.Context, &registryCacheConfigList)
	if err != nil {
		return nil, err
	}
	registryCacheConfigs := make([]registrycache.RegistryCacheConfig, 0)

	return append(registryCacheConfigs, registryCacheConfigList.Items...), nil
}

func GetRegistryCacheSecrets() ([]corev1.Secret, []corev1.Secret) {
	var secretsToBeMarkedForDeletion []corev1.Secret
	var secretsToBeCreated []corev1.Secret

	//TODO: Implement logic to retrieve secrets that need to be marked for deletion and those that need to be created.

	return secretsToBeMarkedForDeletion, secretsToBeCreated
}

func GetReferenceSecretNames(configs []registrycache.RegistryCacheConfig) []string {
	referenceNames := make([]string, 0)
	for _, config := range configs {
		if config.Spec.SecretReferenceName != nil {
			referenceNames = append(referenceNames, *config.Spec.SecretReferenceName)
		} else {
			return nil // No reference secret name found, nothing to do
		}
	}
	return referenceNames
}

func (c *RuntimeConfigurationManager) GetRegistryCacheSecrets(configs []registrycache.RegistryCacheConfig) []corev1.Secret {
	referenceSecretNames := GetReferenceSecretNames(configs)

	var allSecretsList corev1.SecretList
	var referencedSecrets []corev1.Secret

	//TODO: pass using label selector like name in `refname1 refname2 refname3` instead of iterating if possible
	c.runtimeClient.List(c.Context, &allSecretsList, client.MatchingLabels{})
	for _, referenceSecretName := range referenceSecretNames {
		for _, runtimeSecret := range allSecretsList.Items {
			if referenceSecretName == runtimeSecret.Name {
				referencedSecrets = append(referencedSecrets, runtimeSecret)
			}
		}
	}

	return referencedSecrets
}

func (c *RuntimeConfigurationManager) CreateSecrets(secrets []corev1.Secret) error {
	for _, secret := range secrets {
		if err := c.runtimeClient.Create(c.Context, &secret); err != nil {
			return err
		}
	}
	return nil
}

func (c *RuntimeConfigurationManager) MarkSecretForDeletionUsingLabel(toBeMarked corev1.Secret) error {
	toBeMarked.Labels["kyma-project.io/registry-cache-config-marked-for-deletion"] = "true"

	if err := c.runtimeClient.Update(c.Context, &toBeMarked); err != nil {
		return err
	}
	return nil
}
