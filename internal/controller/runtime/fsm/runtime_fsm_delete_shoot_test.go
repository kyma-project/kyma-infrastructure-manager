package fsm

import (
	"fmt"
	"testing"

	v1 "github.com/kyma-project/kyma-infrastructure-manager/api/v1"
)

func Test_addGardenerCloudDelConfirmation(t *testing.T) {
	instance := v1.Runtime{}
	actual := addGardenerCloudDelConfirmation(instance.Annotations)
	if _, found := actual[v1.AnnotationGardenerCloudDelConfirmation]; !found {
		t.Errorf("actual map should contain '%s' annotation", v1.AnnotationGardenerCloudDelConfirmation)
	}
}

func Test_IsGardenerCloudDelConfirmation(t *testing.T) {
	var cases = []struct {
		annotations map[string]string
		expected    bool
	}{
		{
			expected: false,
		},
		{
			annotations: (map[string]string{}),
			expected:    false,
		},
		{
			annotations: map[string]string{"test": "me"},
			expected:    false,
		},
		{
			annotations: map[string]string{
				v1.AnnotationGardenerCloudDelConfirmation: "true",
				"test": "me",
			},
			expected: true,
		},
		{
			annotations: map[string]string{
				v1.AnnotationGardenerCloudDelConfirmation: "anything",
				"test": "me",
			},
			expected: false,
		},
		{
			annotations: map[string]string{
				v1.AnnotationGardenerCloudDelConfirmation: "",
			},
			expected: false,
		},
	}

	for i, tt := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			if actual := isGardenerCloudDelConfirmationSet(tt.annotations); actual != tt.expected {
				t.Errorf("expected IsGardenerCloudDelConfirmation == %t",
					tt.expected)
			}
		})
	}
}
