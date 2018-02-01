package error_finder

import (
	"github.com/compozed/deployadactyl/interfaces"
	"regexp"
)

type ErrorMatcherFactory struct {
}

func (f *ErrorMatcherFactory) CreateErrorMatcher(description, regex string) (interfaces.ErrorMatcher, error) {
	pattern, err := regexp.Compile(regex)
	if err != nil {
		return &RegExErrorMatcher{}, err
	}
	return &RegExErrorMatcher{description: description, regex: pattern, pattern: regex}, nil
}

type RegExErrorMatcher struct {
	pattern     string
	description string
	regex       *regexp.Regexp
}

func (m *RegExErrorMatcher) Descriptor() string {
	return m.description + ": " + m.pattern
}
func (m *RegExErrorMatcher) Match(matchTo []byte) interfaces.DeploymentError {
	matches := m.regex.FindAllString(string(matchTo), -1)
	if len(matches) > 0 {
		return &CFDeploymentError{details: matches, description: m.description}
	}
	return nil
}
