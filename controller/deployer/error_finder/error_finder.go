package error_finder

import (
	"github.com/compozed/deployadactyl/interfaces"
	"io"
	"strings"
)

const TRUST_STORE_ERROR_STRING = "Creating TrustStore with container certificates\nFAILED"

type ErrorFinder struct {
	Matchers []interfaces.ErrorMatcher
}

func (e *ErrorFinder) FindError(responseString string) error {
	if strings.Contains(responseString, TRUST_STORE_ERROR_STRING) {
		return TrustStoreError{}
	}

	return nil
}

func (e *ErrorFinder) FindErrors(responseString string) []interfaces.DeploymentError {
	errors := make([]interfaces.DeploymentError, 0, 0)

	if len(e.Matchers) > 0 {
		for _, matcher := range e.Matchers {
			match := matcher.Match([]byte(responseString))
			if match != nil {
				errors = append(errors, match)
			}
		}
	}
	return errors
}

func (e *ErrorFinder) WriteFormattedErrors(errors []string, response io.ReadWriter) {

}
