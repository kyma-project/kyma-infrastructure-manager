package backup

import (
	"fmt"
)

type StatusType string

const (
	StatusSuccess StatusType = "Success"
	StatusError   StatusType = "Error"
)

type RuntimeResult struct {
	RuntimeID     string     `json:"runtimeId"`
	ShootName     string     `json:"shootName"`
	Status        StatusType `json:"status"`
	ErrorMessage  string     `json:"errorMessage,omitempty"`
	BackupDirPath string     `json:"backupDirPath,omitempty"`
}

type Results struct {
	Results         []RuntimeResult
	Succeeded       int
	Failed          int
	OutputDirectory string
}

func NewBackupResults(outputDirectory string) Results {
	return Results{
		Results:         make([]RuntimeResult, 0),
		OutputDirectory: outputDirectory,
	}
}

func (br *Results) ErrorOccurred(runtimeID, shootName string, errorMsg string) {
	result := RuntimeResult{
		RuntimeID:    runtimeID,
		ShootName:    shootName,
		Status:       StatusError,
		ErrorMessage: errorMsg,
	}

	br.Failed++
	br.Results = append(br.Results, result)
}

func (br *Results) OperationSucceeded(runtimeID string, shootName string) {
	result := RuntimeResult{
		RuntimeID:     runtimeID,
		ShootName:     shootName,
		Status:        StatusSuccess,
		BackupDirPath: br.getBackupDirPath(runtimeID),
	}

	br.Succeeded++
	br.Results = append(br.Results, result)
}

func (br *Results) getBackupDirPath(runtimeID string) string {
	return fmt.Sprintf("%s/%s/%s", br.OutputDirectory, backupFolderName, runtimeID)
}
