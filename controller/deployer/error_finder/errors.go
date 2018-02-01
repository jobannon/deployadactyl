package error_finder

import "github.com/compozed/deployadactyl/interfaces"

func CreateDeploymentError(description string, details []string) interfaces.DeploymentError {
	return &CFDeploymentError{description: description, details: details}
}

type CFDeploymentError struct {
	description string
	details     []string
}

func (e *CFDeploymentError) Error() string {
	return e.description
}

func (e *CFDeploymentError) Details() []string {
	return e.details
}

type TrustStoreError struct{}

func (t TrustStoreError) Error() string {
	return "TrustStore error detected"
}
