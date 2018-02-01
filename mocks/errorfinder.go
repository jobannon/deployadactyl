package mocks

import "github.com/compozed/deployadactyl/interfaces"

type ErrorFinder struct {
	FindErrorCall struct {
		Received struct {
			Response string
		}
		Returns struct {
			Error error
		}
	}
	FindErrorsCall struct {
		Received struct {
			Response string
		}
		Returns struct {
			Errors []interfaces.DeploymentError
		}
	}
}

func (e *ErrorFinder) FindError(responseString string) error {
	e.FindErrorCall.Received.Response = responseString
	return e.FindErrorCall.Returns.Error
}

func (e *ErrorFinder) FindErrors(responseString string) []interfaces.DeploymentError {
	e.FindErrorsCall.Received.Response = responseString
	return e.FindErrorsCall.Returns.Errors
}
