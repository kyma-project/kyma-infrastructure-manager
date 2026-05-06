package machinecontroller

import (
	"fmt"
	"strconv"
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	defaultDrainTimeout = "15m"
	defaultEvictRetries = "2"
)

func ApplyMachineControllerManagerConfig(workers []gardener.Worker, drainTimeout, evictRetries string) error {
	if drainTimeout == "" {
		drainTimeout = defaultDrainTimeout
	}

	if evictRetries == "" {
		evictRetries = defaultEvictRetries
	}

	retries, err := strconv.ParseInt(evictRetries, 10, 32)
	if err != nil {
		return fmt.Errorf("cannot parse the value for evict retries: %w", err)
	}

	timeout, err := time.ParseDuration(drainTimeout)
	if err != nil {
		return fmt.Errorf("cannot parse drain timeout: %w", err)
	}

	for i := range workers {
		machineSettings := workers[i].MachineControllerManagerSettings
		if machineSettings == nil {
			machineSettings = &gardener.MachineControllerManagerSettings{}
			workers[i].MachineControllerManagerSettings = machineSettings
		}
		if machineSettings.MaxEvictRetries == nil {
			machineSettings.MaxEvictRetries = ptr.To(int32(retries))
		}
		if machineSettings.MachineDrainTimeout == nil {
			machineSettings.MachineDrainTimeout = &v1.Duration{Duration: timeout}
		}
	}

	return nil
}
