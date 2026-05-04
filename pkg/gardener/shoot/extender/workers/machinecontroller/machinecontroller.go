package machinecontroller

import (
	"fmt"
	"strconv"
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func ApplyMachineControllerManagerConfig(workers []gardener.Worker, defaultDrainTimeout, defaultEvictRetries string) error {
	evictRetries, err := strconv.ParseInt(defaultEvictRetries, 10, 32)
	if err != nil {
		return fmt.Errorf("cannot parse the value for evict retries: %w", err)
	}

	drainTimeout, err := time.ParseDuration(defaultDrainTimeout)
	if err != nil {
		return fmt.Errorf("cannot parse drain timeout: %w", err)
	}

	for i := range workers {
		if workers[i].MachineControllerManagerSettings == nil {
			workers[i].MachineControllerManagerSettings = &gardener.MachineControllerManagerSettings{}
		}
		if workers[i].MachineControllerManagerSettings.MaxEvictRetries == nil {
			workers[i].MachineControllerManagerSettings.MaxEvictRetries = ptr.To(int32(evictRetries))
		}
		if workers[i].MachineControllerManagerSettings.MachineDrainTimeout == nil {
			workers[i].MachineControllerManagerSettings.MachineDrainTimeout = &v1.Duration{Duration: drainTimeout}
		}
	}

	return nil
}
