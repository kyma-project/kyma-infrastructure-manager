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
			Description: "First description - retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorRetryableConfigurationProblem,
			},
		},
		{
			Description: "Second description - retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorRetryableInfraDependencies,
			},
		},
		{
			Description: "Third description - non-retryable error according to gardener API which we deliberately consider as retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorInfraRateLimitsExceeded,
			},
		},
	}
}

func fixNonRetryableErrors() []gardener.LastError {
	return []gardener.LastError{
		{
			Description: "First description - non-retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorInfraDependencies,
			},
		},
		{
			Description: "Second description - non-retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorInfraQuotaExceeded,
			},
		},
		{
			Description: "Third description - non-retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorInfraUnauthenticated,
			},
		},
		{
			Description: "Fourth description - non-retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorInfraUnauthorized,
			},
		},
		{
			Description: "Fifth description - non-retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorConfigurationProblem,
			},
		},
		{
			Description: "Sixth description - non-retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorProblematicWebhook,
			},
		},
	}
}

func fixMixtureOfErrors() []gardener.LastError {
	return []gardener.LastError{
		{
			Description: "First description - non-retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorInfraDependencies,
			},
		},
		{
			Description: "Second description - retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorRetryableConfigurationProblem,
			},
		},
		{
			Description: "Third description - non-retryable",
			Codes: []gardener.ErrorCode{
				gardener.ErrorInfraQuotaExceeded,
			},
		},
	}
}
