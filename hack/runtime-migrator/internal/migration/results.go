package migration

import (
	"fmt"
)

type StatusType string

const (
	StatusSuccess         StatusType = "Success"
	StatusError           StatusType = "Error"
	StatusValidationError StatusType = "ValidationError"
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
		Status:                   StatusValidationError,
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
	return fmt.Sprintf("%s/%s/%s.yaml", mr.OutputDirectory, runtimesFolderName, runtimeID)
}

func (mr *Results) getComparisonResultPath(runtimeID string) string {
	return fmt.Sprintf("%s/%s/%s", mr.OutputDirectory, comparisonFolderName, runtimeID)
}
