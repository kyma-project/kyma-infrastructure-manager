package machinecontroller

import (
	"testing"
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestApplyMachineControllerManagerConfig(t *testing.T) {
	tests := []struct {
		name                 string
		workers              []gardener.Worker
		defaultDrainTimeout  string
		defaultEvictRetries  string
		expectedSettings     []*gardener.MachineControllerManagerSettings
		expectError          bool
		expectedErrorContain string
	}{
		{
			name: "nil settings get defaults applied",
			workers: []gardener.Worker{
				{MachineControllerManagerSettings: nil},
			},
			defaultDrainTimeout: "15m",
			defaultEvictRetries: "2",
			expectedSettings: []*gardener.MachineControllerManagerSettings{
				{
					MaxEvictRetries:     ptr.To(int32(2)),
					MachineDrainTimeout: &v1.Duration{Duration: 15 * time.Minute},
				},
			},
		},
		{
			name: "existing settings with all fields set unchanged",
			workers: []gardener.Worker{
				{
					MachineControllerManagerSettings: &gardener.MachineControllerManagerSettings{
						MaxEvictRetries:     ptr.To(int32(5)),
						MachineDrainTimeout: &v1.Duration{Duration: 30 * time.Minute},
					},
				},
			},
			defaultDrainTimeout: "15m",
			defaultEvictRetries: "2",
			expectedSettings: []*gardener.MachineControllerManagerSettings{
				{
					MaxEvictRetries:     ptr.To(int32(5)),
					MachineDrainTimeout: &v1.Duration{Duration: 30 * time.Minute},
				},
			},
		},
		{
			name: "partial settings get nil fields filled",
			workers: []gardener.Worker{
				{
					MachineControllerManagerSettings: &gardener.MachineControllerManagerSettings{
						MaxEvictRetries: ptr.To(int32(10)),
					},
				},
			},
			defaultDrainTimeout: "20m",
			defaultEvictRetries: "3",
			expectedSettings: []*gardener.MachineControllerManagerSettings{
				{
					MaxEvictRetries:     ptr.To(int32(10)),
					MachineDrainTimeout: &v1.Duration{Duration: 20 * time.Minute},
				},
			},
		},
		{
			name: "partial settings with only drain timeout set",
			workers: []gardener.Worker{
				{
					MachineControllerManagerSettings: &gardener.MachineControllerManagerSettings{
						MachineDrainTimeout: &v1.Duration{Duration: 45 * time.Minute},
					},
				},
			},
			defaultDrainTimeout: "15m",
			defaultEvictRetries: "2",
			expectedSettings: []*gardener.MachineControllerManagerSettings{
				{
					MaxEvictRetries:     ptr.To(int32(2)),
					MachineDrainTimeout: &v1.Duration{Duration: 45 * time.Minute},
				},
			},
		},
		{
			name: "multiple workers all get defaults",
			workers: []gardener.Worker{
				{MachineControllerManagerSettings: nil},
				{MachineControllerManagerSettings: nil},
				{
					MachineControllerManagerSettings: &gardener.MachineControllerManagerSettings{
						MaxEvictRetries: ptr.To(int32(7)),
					},
				},
			},
			defaultDrainTimeout: "10m",
			defaultEvictRetries: "4",
			expectedSettings: []*gardener.MachineControllerManagerSettings{
				{
					MaxEvictRetries:     ptr.To(int32(4)),
					MachineDrainTimeout: &v1.Duration{Duration: 10 * time.Minute},
				},
				{
					MaxEvictRetries:     ptr.To(int32(4)),
					MachineDrainTimeout: &v1.Duration{Duration: 10 * time.Minute},
				},
				{
					MaxEvictRetries:     ptr.To(int32(7)),
					MachineDrainTimeout: &v1.Duration{Duration: 10 * time.Minute},
				},
			},
		},
		{
			name:                "empty workers slice no error",
			workers:             []gardener.Worker{},
			defaultDrainTimeout: "15m",
			defaultEvictRetries: "2",
			expectedSettings:    []*gardener.MachineControllerManagerSettings{},
		},
		{
			name: "invalid evict retries returns error",
			workers: []gardener.Worker{
				{MachineControllerManagerSettings: nil},
			},
			defaultDrainTimeout:  "15m",
			defaultEvictRetries:  "not-a-number",
			expectError:          true,
			expectedErrorContain: "cannot parse the value for evict retries",
		},
		{
			name: "empty evict retries returns error",
			workers: []gardener.Worker{
				{MachineControllerManagerSettings: nil},
			},
			defaultDrainTimeout:  "15m",
			defaultEvictRetries:  "",
			expectError:          true,
			expectedErrorContain: "cannot parse the value for evict retries",
		},
		{
			name: "invalid drain timeout returns error",
			workers: []gardener.Worker{
				{MachineControllerManagerSettings: nil},
			},
			defaultDrainTimeout:  "not-a-duration",
			defaultEvictRetries:  "2",
			expectError:          true,
			expectedErrorContain: "cannot parse drain timeout",
		},
		{
			name: "empty drain timeout returns error",
			workers: []gardener.Worker{
				{MachineControllerManagerSettings: nil},
			},
			defaultDrainTimeout:  "",
			defaultEvictRetries:  "2",
			expectError:          true,
			expectedErrorContain: "cannot parse drain timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ApplyMachineControllerManagerConfig(tt.workers, tt.defaultDrainTimeout, tt.defaultEvictRetries)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorContain)
				return
			}
			require.NoError(t, err)
			require.Len(t, tt.workers, len(tt.expectedSettings))
			for i, w := range tt.workers {
				assert.Equal(t, tt.expectedSettings[i], w.MachineControllerManagerSettings)
			}
		})
	}
}
