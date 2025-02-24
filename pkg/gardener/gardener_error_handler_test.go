package gardener

import (
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestGardenerErrorHandler(t *testing.T) {
	t.Run("Should combine error descriptions", func(t *testing.T) {
		// given
		lastErrors := []gardener.LastError{
			{
				Description: "First description",
			},
			{
				Description: "Second description",
			},
		}

		// when
		combinedDescritpion := CombineErrorDescriptions(lastErrors)

		// then
		assert.Equal(t, "1) First description 2) Second description ", combinedDescritpion)
	})

	for _, testCase := range []struct {
		name              string
		lastErrors        []gardener.LastError
		expectedRetryable bool
	}{
		{
			name:              "Should return true for retryable gardener errors",
			lastErrors:        fixRetryableErrors(),
			expectedRetryable: true,
		},
		{
			name: "Should return true for retryable error not yet defined in Gardener",
			lastErrors: []gardener.LastError{
				{
					Description: "New retryable error not yet defined in Gardener",
					Codes: []gardener.ErrorCode{
						"NEW_RETRYABLE_ERROR",
					},
				},
			},
			expectedRetryable: true,
		},
		{
			name:              "Should return false for retryable gardener errors",
			lastErrors:        fixNonRetryableErrors(),
			expectedRetryable: false,
		},
		{
			name:              "Should return false for mixture of retryable and non-retryable gardener errors",
			lastErrors:        fixMixtureOfErrors(),
			expectedRetryable: false,
		},
		{
			name:              "Should return false empty list",
			lastErrors:        []gardener.LastError{},
			expectedRetryable: false,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// given

			// when
			retryable := IsRetryable(testCase.lastErrors)

			// then
			assert.Equal(t, testCase.expectedRetryable, retryable)
		})
	}
}

func fixRetryableErrors() []gardener.LastError {
	return []gardener.LastError{
		{
			Description: "ErrorRetryableConfigurationProblem - retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorRetryableConfigurationProblem,
			},
		},
		{
			Description: "ErrorConfigurationProblem -  non-retryable error according to gardener API which we deliberately consider as retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorConfigurationProblem,
			},
		},
		{
			Description: "ErrorRetryableInfraDependencies - retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorRetryableInfraDependencies,
			},
		},
		{
			Description: "ErrorInfraRateLimitsExceeded - non-retryable error according to gardener API which we deliberately consider as retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorInfraRateLimitsExceeded,
			},
		},
		{
			Description: "ErrorInfraQuotaExceeded - non-retryable error according to gardener API which we deliberately consider as retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorInfraQuotaExceeded,
			},
		},
		{
			Description: "ErrorProblematicWebhook - non-retryable error according to gardener API which we deliberately consider as retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorProblematicWebhook,
			},
		},
		{
			Description: "ErrorInfraDependencies - non-retryable error according to gardener API which we deliberately consider as retryable as it can occur during deletion",
			Codes: []gardener.ErrorCode{
				gardener.ErrorInfraDependencies,
			},
		},
		{
			Description: "New retryable error not yet defined in Gardener",
			Codes: []gardener.ErrorCode{
				"NEW_RETRYABLE_ERROR",
			},
		},
	}
}

func fixNonRetryableErrors() []gardener.LastError {
	return []gardener.LastError{
		{
			Description: "ErrorInfraUnauthenticated - non-retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorInfraUnauthenticated,
			},
		},
		{
			Description: "ErrorInfraUnauthorized - non-retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorInfraUnauthorized,
			},
		},
	}
}

func fixMixtureOfErrors() []gardener.LastError {
	return []gardener.LastError{
		{
			Description: "ErrorInfraDependencies - non-retryable error according to gardener API which we deliberately consider as retryable as it can occur during deletion",
			Codes: []gardener.ErrorCode{
				gardener.ErrorInfraDependencies,
			},
		},
		{
			Description: "ErrorRetryableConfigurationProblem - retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorRetryableConfigurationProblem,
			},
		},
		{
			Description: "ErrorInfraDependencies - non-retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorInfraDependencies,
			},
		},
		{
			Description: "ErrorInfraUnauthenticated - non-retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorInfraUnauthenticated,
			},
		},
		{
			Description: "ErrorInfraQuotaExceeded - non-retryable error according to gardener API which we deliberately consider as retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorInfraQuotaExceeded,
			},
		},
		{
			Description: "ErrorProblematicWebhook - non-retryable error according to gardener API which we deliberately consider as retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorProblematicWebhook,
			},
		},
		{
			Description: "New retryable error not yet defined in Gardener",
			Codes: []gardener.ErrorCode{
				"NEW_RETRYABLE_ERROR",
			},
		},
	}
}
