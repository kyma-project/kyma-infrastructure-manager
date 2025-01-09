package shoot

import "github.com/gardener/gardener/pkg/apis/core/v1beta1"

// FilterOutFields creates a new slice with workers containing only the fields KEB sets
func FilterOutFields(workers []v1beta1.Worker) []v1beta1.Worker {
	newWorkers := make([]v1beta1.Worker, 0)

	for _, worker := range workers {
		newWorker := v1beta1.Worker{
			Machine:        worker.Machine,
			Maximum:        worker.Maximum,
			Minimum:        worker.Minimum,
			MaxSurge:       worker.MaxSurge,
			MaxUnavailable: worker.MaxUnavailable,
			Name:           worker.Name,
			Volume:         worker.Volume,
			Zones:          worker.Zones,
		}

		newWorkers = append(newWorkers, newWorker)
	}

	return newWorkers
}
