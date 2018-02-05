package error_finder

import (
	"errors"
	"regexp"

	"github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/structs"
)

type ErrorMatcherFactory struct {
}

func (f *ErrorMatcherFactory) CreateErrorMatcher(descriptor structs.ErrorMatcherDescriptor) (interfaces.ErrorMatcher, error) {
	if descriptor.Pattern == "" {
		return &RegExErrorMatcher{}, errors.New("error matcher requires a pattern")
	}

	regex, err := regexp.Compile(descriptor.Pattern)
	if err != nil {
		return &RegExErrorMatcher{}, err
	}

	description := descriptor.Description
	if description == "" {
		description = "This error does not have a description."
	}

	solution := descriptor.Solution
	if solution == "" {
		solution = "No recommended solution available."
	}

	return &RegExErrorMatcher{
		description: description,
		regex:       regex,
		pattern:     descriptor.Pattern,
		solution:    solution}, nil
}

type RegExErrorMatcher struct {
	pattern     string
	description string
	solution    string
	regex       *regexp.Regexp
}

func (m *RegExErrorMatcher) Descriptor() string {
	return m.description + ": " + m.pattern + ": " + m.solution
}

func (m *RegExErrorMatcher) Match(matchTo []byte) interfaces.DeploymentError {
	matches := m.regex.FindAllString(string(matchTo), -1)
	if len(matches) > 0 {
		return &CFDeploymentError{details: matches, description: m.description, solution: m.solution}
	}
	return nil
}
