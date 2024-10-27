package migration

import (
	"fmt"
)

type StatusType string

const (
	StatusSuccess                         StatusType = "Success"
	StatusError                           StatusType = "nError"
	StatusRuntimeCRCanCauseUnwantedUpdate StatusType = "RuntimeCRCanCauseUnwantedUpdate"
)

type RuntimeResult struct {
	RuntimeID                string     `json:"runtimeId"`
	ShootName                string     `json:"shootName"`
	Status                   StatusType `json:"status"`
	ErrorMessage             string     `json:"errorMessage,omitempty"`
	RuntimeCRFilePath        string     `json:"runtimeCRFilePath,omitempty"`
	ComparisonResultsDirPath string     `json:"comparisonResultDirPath,omitempty"`
}

type Results struct {
	Results            []RuntimeResult
	Succeeded          int
	Failed             int
	DifferenceDetected int
	OutputDirectory    string
}

func NewMigratorResults(outputDirectory string) Results {
	return Results{
		Results:         make([]RuntimeResult, 0),
		OutputDirectory: outputDirectory,
	}
}

func (mr *Results) ErrorOccurred(runtimeID, shootName string, errorMsg string) {
	result := RuntimeResult{
		RuntimeID:         runtimeID,
		ShootName:         shootName,
		Status:            StatusError,
		ErrorMessage:      errorMsg,
		RuntimeCRFilePath: mr.getRuntimeCRPath(shootName),
	}

	mr.Failed++
	mr.Results = append(mr.Results, result)
}

func (mr *Results) ValidationFailed(runtimeID, shootName string) {
	result := RuntimeResult{
		RuntimeID:                runtimeID,
		ShootName:                shootName,
		Status:                   StatusRuntimeCRCanCauseUnwantedUpdate,
		ErrorMessage:             "Runtime may cause unwanted update in Gardener. Please verify the runtime CR.",
		RuntimeCRFilePath:        mr.getRuntimeCRPath(runtimeID),
		ComparisonResultsDirPath: mr.getComparisonResultPath(runtimeID),
	}

	mr.DifferenceDetected++
	mr.Results = append(mr.Results, result)
}

func (mr *Results) OperationSucceeded(runtimeID string, shootName string) {
	result := RuntimeResult{
		RuntimeID:         runtimeID,
		ShootName:         shootName,
		Status:            StatusSuccess,
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
