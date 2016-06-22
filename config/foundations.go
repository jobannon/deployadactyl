package config

import (
	"io/ioutil"
	"strings"

	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/go-errors/errors"
)

const (
	cannotCreateGetRequest = "cannot create GET request"
	cannotSendGetRequest   = "cannot send GET request"
	cannotReadResponseBody = "cannot read response body"
	cannotParseYamlFile    = "cannot parse yaml file"
)

func getEnvironmentsFromFile(filename string) (map[string]Environment, error) {

	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.New(err)
	}

	foundationConfig, err := parseYamlFromBody(file)
	if err != nil {
		return nil, errors.New(err)
	}

	environments := map[string]Environment{}
	for _, environment := range foundationConfig.Environments {
		environments[strings.ToLower(environment.Name)] = environment
	}

	return environments, nil
}

func parseYamlFromBody(data []byte) (configYaml, error) {
	var foundationConfig configYaml
	err := candiedyaml.Unmarshal(data, &foundationConfig)
	if err != nil {
		return configYaml{}, errors.Errorf("%s: %s", cannotParseYamlFile, err)
	}

	return foundationConfig, nil
}

type Environment struct {
	Name         string
	Domain       string
	Foundations  []string `yaml:",flow"`
	Authenticate bool
}

type configYaml struct {
	Environments []Environment `yaml:",flow"`
}

type foundationYaml struct {
	Foundations []string
}
