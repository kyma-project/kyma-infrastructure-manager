package internal

type StatusType string

const (
	StatusSuccess                         StatusType = "Success"
	StatusError                           StatusType = "nError"
	StatusAlreadyExists                   StatusType = "AlreadyExists"
	StatusRuntimeIDNotFound               StatusType = "RuntimeIDNotFound"
	StatusFailedToCreateRuntimeCR         StatusType = "FailedToCreateRuntimeCR"
	StatusRuntimeCRCanCauseUnwantedUpdate StatusType = "RuntimeCRCanCauseUnwantedUpdate"
)

type MigrationResult struct {
	RuntimeID                string     `json:"runtimeId"`
	ShootName                string     `json:"shootName"`
	Status                   StatusType `json:"status"`
	ErrorMessage             string     `json:"errorMessage,omitempty"`
	RuntimeCRFilePath        string     `json:"runtimeCRFilePath,omitempty"`
	ComparisonResultsDirPath string     `json:"comparisonResultDirPath,omitempty"`
}
