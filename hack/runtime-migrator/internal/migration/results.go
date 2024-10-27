package migration

import (
	"fmt"
	migrator "github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal"
)

const runtimeCrFullPath = "%sshoot-%s.yaml"

type MigrationResults struct {
	Results            []migrator.MigrationResult
	Succeeded          int
	Failed             int
	DifferenceDetected int
	OutputDirectory    string
}

func NewMigratorResults(outputDirectory string) MigrationResults {
	return MigrationResults{
		Results:         make([]migrator.MigrationResult, 0),
		OutputDirectory: outputDirectory,
	}
}

func (mr MigrationResults) ErrorOccurred(runtimeID, shootName string, errorMsg string) {
	result := migrator.MigrationResult{
		RuntimeID:    runtimeID,
		ShootName:    shootName,
		Status:       migrator.StatusError,
		ErrorMessage: errorMsg,
		PathToCRYaml: mr.getYamlPath(shootName),
	}

	mr.Failed++
	mr.Results = append(mr.Results, result)
}

func (mr MigrationResults) ValidationFailed(runtimeID, shootName string) {
	result := migrator.MigrationResult{
		RuntimeID:    runtimeID,
		ShootName:    shootName,
		Status:       migrator.StatusRuntimeCRCanCauseUnwantedUpdate,
		ErrorMessage: "Runtime may cause unwanted update in Gardener. Please verify the runtime CR.",
		PathToCRYaml: mr.getYamlPath(runtimeID),
	}

	mr.DifferenceDetected++
	mr.Results = append(mr.Results, result)
}

func (mr MigrationResults) OperationSucceeded(runtimeID string, shootName string) {
	result := migrator.MigrationResult{
		RuntimeID:    runtimeID,
		ShootName:    shootName,
		Status:       migrator.StatusSuccess,
		PathToCRYaml: mr.getYamlPath(runtimeID),
	}

	mr.Succeeded++
	mr.Results = append(mr.Results, result)
}

func (mr MigrationResults) getYamlPath(runtimeID string) string {
	return fmt.Sprintf(runtimeCrFullPath, mr.OutputDirectory, runtimeID)
}
