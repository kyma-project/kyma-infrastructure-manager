package auditlog

import (
	"context"
	"testing"

	auditlogv1 "github.com/kyma-project/infrastructure-manager/pkg/auditlog/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestContainsRegion(t *testing.T) {
	tests := []struct {
		name     string
		regions  []string
		region   string
		expected bool
	}{
		{
			name:     "region found in list",
			regions:  []string{"eu-central-1", "eu-west-2", "us-east-1"},
			region:   "eu-west-2",
			expected: true,
		},
		{
			name:     "region not found in list",
			regions:  []string{"eu-central-1", "eu-west-2"},
			region:   "us-east-1",
			expected: false,
		},
		{
			name:     "empty list",
			regions:  []string{},
			region:   "eu-central-1",
			expected: false,
		},
		{
			name:     "single region match",
			regions:  []string{"eu-central-1"},
			region:   "eu-central-1",
			expected: true,
		},
		{
			name:     "single region no match",
			regions:  []string{"eu-central-1"},
			region:   "us-east-1",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := containsRegion(tc.regions, tc.region)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFindAvailableAuditLogCR(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, auditlogv1.AddToScheme(scheme))
	logger := zap.New(zap.UseDevMode(true))

	tests := []struct {
		name         string
		auditLogs    []auditlogv1.AuditLog
		region       string
		expectFound  bool
		expectedName string
	}{
		{
			name: "finds available CR for matching region",
			auditLogs: []auditlogv1.AuditLog{
				createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"eu-central-1", "eu-west-2"}, nil),
			},
			region:       "eu-central-1",
			expectFound:  true,
			expectedName: "al-1",
		},
		{
			name: "finds CR in RegistrationReady state",
			auditLogs: []auditlogv1.AuditLog{
				createAuditLogCR("al-1", auditlogv1.StateRegistrationReady, "", []string{"eu-central-1"}, nil),
			},
			region:       "eu-central-1",
			expectFound:  true,
			expectedName: "al-1",
		},
		{
			name: "skips CR with non-matching region",
			auditLogs: []auditlogv1.AuditLog{
				createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"us-east-1", "us-west-2"}, nil),
			},
			region:      "eu-central-1",
			expectFound: false,
		},
		{
			name: "skips CR already assigned",
			auditLogs: []auditlogv1.AuditLog{
				createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "other-runtime", []string{"eu-central-1"}, nil),
			},
			region:      "eu-central-1",
			expectFound: false,
		},
		{
			name: "skips CR with reservation label",
			auditLogs: []auditlogv1.AuditLog{
				createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"eu-central-1"}, map[string]string{
					LabelReservedForRuntimeID: "other-runtime",
				}),
			},
			region:      "eu-central-1",
			expectFound: false,
		},
		{
			name: "skips CR in wrong state (Pending, Assigned, Orphaned)",
			auditLogs: []auditlogv1.AuditLog{
				createAuditLogCR("al-1", auditlogv1.StatePending, "", []string{"eu-central-1"}, nil),
				createAuditLogCR("al-2", auditlogv1.StateAssigned, "", []string{"eu-central-1"}, nil),
				createAuditLogCR("al-3", auditlogv1.StateOrphaned, "", []string{"eu-central-1"}, nil),
			},
			region:      "eu-central-1",
			expectFound: false,
		},
		{
			name: "finds first available among multiple",
			auditLogs: []auditlogv1.AuditLog{
				createAuditLogCR("al-1", auditlogv1.StatePending, "", []string{"eu-central-1"}, nil),
				createAuditLogCR("al-2", auditlogv1.StateSiemApproved, "assigned", []string{"eu-central-1"}, nil),
				createAuditLogCR("al-3", auditlogv1.StateSiemApproved, "", []string{"eu-central-1"}, nil),
				createAuditLogCR("al-4", auditlogv1.StateSiemApproved, "", []string{"eu-central-1"}, nil),
			},
			region:       "eu-central-1",
			expectFound:  true,
			expectedName: "al-3",
		},
		{
			name:        "empty list returns nil",
			auditLogs:   []auditlogv1.AuditLog{},
			region:      "eu-central-1",
			expectFound: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tc.auditLogs))
			for i := range tc.auditLogs {
				objects[i] = &tc.auditLogs[i]
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objects...).
				Build()

			provider := &DefaultDataProvider{
				client: fakeClient,
				logger: logger,
			}

			result, err := provider.findAvailableAuditLogCR(context.Background(), tc.region)
			require.NoError(t, err)

			if tc.expectFound {
				require.NotNil(t, result)
				assert.Equal(t, tc.expectedName, result.Name)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestReserveAuditLogCR(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, auditlogv1.AddToScheme(scheme))
	logger := zap.New(zap.UseDevMode(true))

	t.Run("reserves available CR", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"eu-central-1"}, nil)

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := &DefaultDataProvider{
			client: fakeClient,
			logger: logger,
		}

		err := provider.reserveAuditLogCR(context.Background(), "test-runtime", "eu-central-1")
		require.NoError(t, err)

		// Verify reservation label was added
		var updated auditlogv1.AuditLog
		err = fakeClient.Get(context.Background(),
			namespacedName("al-1"), &updated)
		require.NoError(t, err)
		assert.Equal(t, "test-runtime", updated.Labels[LabelReservedForRuntimeID])
		assert.NotEmpty(t, updated.Labels[LabelReservedAt])
	})

	t.Run("succeeds when CR already reserved for same runtime (idempotent)", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"eu-central-1"}, map[string]string{
			LabelReservedForRuntimeID: "test-runtime",
			LabelReservedAt:           "2026-06-01T00:00:00Z",
		})

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := &DefaultDataProvider{
			client: fakeClient,
			logger: logger,
		}

		err := provider.reserveAuditLogCR(context.Background(), "test-runtime", "eu-central-1")
		require.NoError(t, err)
	})

	t.Run("fails when no available CR for region", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"us-east-1"}, nil)

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := &DefaultDataProvider{
			client: fakeClient,
			logger: logger,
		}

		err := provider.reserveAuditLogCR(context.Background(), "test-runtime", "eu-central-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no available AuditLogCR in the pool for region eu-central-1")
	})
}

func TestFindAuditLogCRByReservation(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, auditlogv1.AddToScheme(scheme))
	logger := zap.New(zap.UseDevMode(true))

	t.Run("finds CR by reservation label", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"eu-central-1"}, map[string]string{
			LabelReservedForRuntimeID: "test-runtime",
		})

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := &DefaultDataProvider{
			client: fakeClient,
			logger: logger,
		}

		result, err := provider.findAuditLogCRByReservation(context.Background(), "test-runtime")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "al-1", result.Name)
	})

	t.Run("returns nil when no reservation found", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"eu-central-1"}, nil)

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := &DefaultDataProvider{
			client: fakeClient,
			logger: logger,
		}

		result, err := provider.findAuditLogCRByReservation(context.Background(), "test-runtime")
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestGetOrClaimAuditLogCR(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, auditlogv1.AddToScheme(scheme))
	logger := zap.New(zap.UseDevMode(true))

	t.Run("claims reserved CR", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"eu-central-1"}, map[string]string{
			LabelReservedForRuntimeID: "test-runtime",
		})

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := &DefaultDataProvider{
			client: fakeClient,
			logger: logger,
		}

		result, err := provider.getOrClaimAuditLogCR(context.Background(), "test-runtime")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "test-runtime", result.Spec.AssignedToRuntimeID)
	})

	t.Run("returns already claimed CR (idempotent)", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "test-runtime", []string{"eu-central-1"}, nil)

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := &DefaultDataProvider{
			client: fakeClient,
			logger: logger,
		}

		result, err := provider.getOrClaimAuditLogCR(context.Background(), "test-runtime")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "al-1", result.Name)
	})

	t.Run("fails when no reservation found", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		provider := &DefaultDataProvider{
			client: fakeClient,
			logger: logger,
		}

		result, err := provider.getOrClaimAuditLogCR(context.Background(), "test-runtime")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no reserved AuditLogCR found")
	})
}

func TestReleaseAuditLogCR(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, auditlogv1.AddToScheme(scheme))
	logger := zap.New(zap.UseDevMode(true))

	t.Run("marks assigned CR as orphaned", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateAssigned, "test-runtime", []string{"eu-central-1"}, nil)

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := &DefaultDataProvider{
			client: fakeClient,
			logger: logger,
		}

		err := provider.releaseAuditLogCR(context.Background(), &auditLog)
		require.NoError(t, err)

		// Verify orphaned flag was set
		var updated auditlogv1.AuditLog
		err = fakeClient.Get(context.Background(), namespacedName("al-1"), &updated)
		require.NoError(t, err)
		assert.True(t, updated.Spec.Orphaned)
	})

	t.Run("skips non-assigned CR", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"eu-central-1"}, nil)

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := &DefaultDataProvider{
			client: fakeClient,
			logger: logger,
		}

		err := provider.releaseAuditLogCR(context.Background(), &auditLog)
		require.NoError(t, err)

		// Verify orphaned flag was NOT set
		var updated auditlogv1.AuditLog
		err = fakeClient.Get(context.Background(), namespacedName("al-1"), &updated)
		require.NoError(t, err)
		assert.False(t, updated.Spec.Orphaned)
	})
}

// Helper functions

func createAuditLogCR(name string, state auditlogv1.State, assignedTo string, regions []string, labels map[string]string) auditlogv1.AuditLog {
	return auditlogv1.AuditLog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels:    labels,
		},
		Spec: auditlogv1.AuditLogSpec{
			PlatformRegion:      "eu10",
			Regions:             regions,
			AssignedToRuntimeID: assignedTo,
			SubaccountID:        "test-subaccount-id",
			Config: auditlogv1.AuditLogConfig{
				ServiceURL:         "https://auditlog.example.com",
				GardenerSecretName: "auditlog-secret",
			},
		},
		Status: auditlogv1.AuditLogStatus{
			State: state,
		},
	}
}

func namespacedName(name string) types.NamespacedName {
	return types.NamespacedName{Name: name, Namespace: "default"}
}
