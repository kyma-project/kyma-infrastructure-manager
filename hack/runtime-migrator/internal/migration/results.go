package migration

import (
	"fmt"
	migrator "github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal"
)

type Results struct {
	Results            []migrator.MigrationResult
	Succeeded          int
	Failed             int
	DifferenceDetected int
	OutputDirectory    string
}

func NewMigratorResults(outputDirectory string) Results {
	return Results{
		Results:         make([]migrator.MigrationResult, 0),
		OutputDirectory: outputDirectory,
	}
}

func (mr *Results) ErrorOccurred(runtimeID, shootName string, errorMsg string) {
	result := migrator.MigrationResult{
		RuntimeID:         runtimeID,
		ShootName:         shootName,
		Status:            migrator.StatusError,
		ErrorMessage:      errorMsg,
		RuntimeCRFilePath: mr.getRuntimeCRPath(shootName),
	}

	mr.Failed++
	mr.Results = append(mr.Results, result)
}

func (mr *Results) ValidationFailed(runtimeID, shootName string) {
	result := migrator.MigrationResult{
		RuntimeID:                runtimeID,
		ShootName:                shootName,
		Status:                   migrator.StatusRuntimeCRCanCauseUnwantedUpdate,
		ErrorMessage:             "Runtime may cause unwanted update in Gardener. Please verify the runtime CR.",
		RuntimeCRFilePath:        mr.getRuntimeCRPath(runtimeID),
		ComparisonResultsDirPath: mr.getComparisonResultPath(runtimeID),
	}

	mr.DifferenceDetected++
	mr.Results = append(mr.Results, result)
}

func (mr *Results) OperationSucceeded(runtimeID string, shootName string) {
	result := migrator.MigrationResult{
		RuntimeID:         runtimeID,
		ShootName:         shootName,
		Status:            migrator.StatusSuccess,
		RuntimeCRFilePath: mr.getRuntimeCRPath(runtimeID),
	}

	mr.Succeeded++
	mr.Results = append(mr.Results, result)
}

func (mr *Results) getRuntimeCRPath(runtimeID string) string {
	return fmt.Sprintf("%s/runtimes/%s.yaml", mr.OutputDirectory, runtimeID)
}

func (mr *Results) getComparisonResultPath(runtimeID string) string {
	return fmt.Sprintf("%s/comparison-results/%s", mr.OutputDirectory, runtimeID)
}
