package reconciler

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestShouldForceReconciliation(t *testing.T) {
	for _, testCase := range []struct {
		name           string
		annotations    map[string]string
		expectedResult bool
	}{
		{
			name:           "Should force reconciliation for `operator.kyma-project.io/force-patch-reconciliation` set to `true",
			annotations:    map[string]string{"operator.kyma-project.io/force-patch-reconciliation": "true"},
			expectedResult: true,
		},
		{
			name:           "Should not force reconciliation for `operator.kyma-project.io/force-patch-reconciliation` set to `kaloryfer",
			annotations:    map[string]string{"operator.kyma-project.io/force-patch-reconciliation": "kaloryfer"},
			expectedResult: false,
		},
		{
			name:           "Should not force reconciliation for nil annotations",
			annotations:    nil,
			expectedResult: false,
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
		name           string
		annotations    map[string]string
		expectedResult bool
	}{
		{
			name:           "Should suspend reconciliation for `operator.kyma-project.io/suspend-patch-reconciliation` set to `true",
			annotations:    map[string]string{"operator.kyma-project.io/suspend-patch-reconciliation": "true"},
			expectedResult: true,
		},
		{
			name:           "Should not suspend reconciliation for `operator.kyma-project.io/suspend-patch-reconciliation` set to `kaloryfer",
			annotations:    map[string]string{"operator.kyma-project.io/suspend-patch-reconciliation": "kaloryfer"},
			expectedResult: false,
		},
		{
			name:           "Should not suspend reconciliation for nil annotations",
			annotations:    nil,
			expectedResult: false,
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
