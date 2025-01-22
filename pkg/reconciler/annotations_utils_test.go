package reconciler

import (
	"github.com/stretchr/testify/assert"
	"testing"
)


func TestShouldForceReconciliation(t *testing.T) {
	for _, testCase := range []struct {
		name            string
		annotations    map[string]string
		expectedResult bool
	}{
		{
			name:            "Should force reconciliation for `operator.kyma-project.io/force-shoot-reconciliation` set to `true",
			annotations:     map[string]string{"operator.kyma-project.io/force-shoot-reconciliation": "true"},
			expectedResult:  true,
		},
		{
			name:            "Should force reconciliation for `operator.kyma-project.io/force-shoot-reconciliation` set to `kaloryfer",
			annotations:     map[string]string{"operator.kyma-project.io/force-shoot-reconciliation": "kaloryfer"},
			expectedResult:  false,
		},
		{
			name:            "Should not force reconciliation for  annotations",
			annotations:     nil,
			expectedResult:  false,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// given

			// when
			forceReconciliation := ShouldForceReconciliation(testCase.annotations)

			// then
			assert.Equal(t, testCase.expectedResult, forceReconciliation)
		})
	}
}

func TestShouldSuspendReconciliation(t *testing.T) {
	for _, testCase := range []struct {
		name            string
		annotations    map[string]string
		expectedResult bool
	}{
		{
			name:            "Should suspend reconciliation for `operator.kyma-project.io/suspend-shoot-reconciliation` set to `true",
			annotations:     map[string]string{"operator.kyma-project.io/suspend-shoot-reconciliation": "true"},
			expectedResult:  true,
		},
		{
			name:            "Should suspend reconciliation for `operator.kyma-project.io/suspend-shoot-reconciliation` set to `kaloryfer",
			annotations:     map[string]string{"operator.kyma-project.io/suspend-shoot-reconciliation": "kaloryfer"},
			expectedResult:  false,
		},
		{
			name:            "Should not suspend reconciliation for nil annotations",
			annotations:     nil,
			expectedResult:  false,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// given

			// when
			forceReconciliation := ShouldSuspendReconciliation(testCase.annotations)

			// then
			assert.Equal(t, testCase.expectedResult, forceReconciliation)
		})
	}
}
