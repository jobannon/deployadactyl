package error_finder

import "github.com/compozed/deployadactyl/interfaces"

func CreateDeploymentError(description string, details []string, solution string) interfaces.DeploymentError {
	return &CFDeploymentError{description: description, details: details, solution: solution}
}

type CFDeploymentError struct {
	description string
	details     []string
	solution    string
}

func (e *CFDeploymentError) Error() string {
	return e.description
}

func (e *CFDeploymentError) Details() []string {
	return e.details
}

func (e *CFDeploymentError) Solution() string {
	return e.solution
}
