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
	timeout, err := parseDrainTimeoutOrDefault(drainTimeout)
	if err != nil {
		return fmt.Errorf("cannot parse drain timeout: %w", err)
	}

	retries, err := parseEvictRetriesOrDefault(evictRetries)
	if err != nil {
		return fmt.Errorf("cannot parse the value for evict retries: %w", err)
	}

	setMachineControllerManagerConfig(workers, timeout, retries)

	return nil
}

func setMachineControllerManagerConfig(workers []gardener.Worker, drainTimeout time.Duration, evictRetries int64) {
	for i := range workers {
		machineSettings := workers[i].MachineControllerManagerSettings
		if machineSettings == nil {
			machineSettings = &gardener.MachineControllerManagerSettings{}
			workers[i].MachineControllerManagerSettings = machineSettings
		}
		if machineSettings.MaxEvictRetries == nil {
			machineSettings.MaxEvictRetries = ptr.To(int32(evictRetries))
		}
		if machineSettings.MachineDrainTimeout == nil {
			machineSettings.MachineDrainTimeout = &v1.Duration{Duration: drainTimeout}
		}
	}
}

func parseDrainTimeoutOrDefault(drainTimeout string) (time.Duration, error) {
	if drainTimeout == "" {
		drainTimeout = defaultDrainTimeout
	}
	return time.ParseDuration(drainTimeout)
}

func parseEvictRetriesOrDefault(evictRetries string) (int64, error) {
	if evictRetries == "" {
		evictRetries = defaultEvictRetries
	}
	return strconv.ParseInt(evictRetries, 10, 32)
}
