package backup

import (
	"fmt"
	v12 "k8s.io/api/rbac/v1"
)

type StatusType string

const (
	StatusSuccess StatusType = "Success"
	StatusError   StatusType = "Error"
)

type RuntimeResult struct {
	RuntimeID          string     `json:"runtimeId"`
	ShootName          string     `json:"shootName"`
	Status             StatusType `json:"status"`
	ErrorMessage       string     `json:"errorMessage,omitempty"`
	BackupDirPath      string     `json:"backupDirPath,omitempty"`
	DeprecatedCRBs     []string   `json:"deprecatedCRBs,omitempty"`
	SetControlledByKIM bool       `json:"setControlledByKIM"`
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

func (br *Results) OperationSucceeded(runtimeID string, shootName string, deprecatedCRBs []v12.ClusterRoleBinding, setControlledByKIM bool) {

	var deprecatedCRBsString []string
	for _, crb := range deprecatedCRBs {
		deprecatedCRBsString = append(deprecatedCRBsString, crb.Name)
	}

	result := RuntimeResult{
		RuntimeID:          runtimeID,
		ShootName:          shootName,
		Status:             StatusSuccess,
		BackupDirPath:      br.getBackupDirPath(runtimeID),
		DeprecatedCRBs:     deprecatedCRBsString,
		SetControlledByKIM: setControlledByKIM,
	}

	br.Succeeded++
	br.Results = append(br.Results, result)
}

func (br *Results) getBackupDirPath(runtimeID string) string {
	return fmt.Sprintf("%s/%s/%s", br.OutputDirectory, backupFolderName, runtimeID)
}
