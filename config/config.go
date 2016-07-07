// Package config holds all specified configuration information aggregated from across all possible inputs (config yaml file and user-defined environment variables).
package config

import (
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/compozed/deployadactyl/geterrors"
	"github.com/go-errors/errors"
)

const (
	unableToGetLogLevel    = "unable to get log level"
	cannotParsePort        = "cannot parse $PORT"
	cannotCreateGetRequest = "cannot create GET request"
	cannotSendGetRequest   = "cannot send GET request"
	cannotReadResponseBody = "cannot read response body"
	cannotParseYamlFile    = "cannot parse yaml file"
)

type Config struct {
	Username     string
	Password     string
	Environments map[string]Environment
	Port         int
}

type Environment struct {
	Name         string
	Domain       string
	Foundations  []string `yaml:",flow"`
	Authenticate bool
	SkipSSL      bool `yaml:"skip_ssl"`
}

type configYaml struct {
	Environments []Environment `yaml:",flow"`
}

type foundationYaml struct {
	Foundations []string
}

// New returns a new Config struct with information from environment variables and the config file.
func New(getenv func(string) string, configFilename string) (Config, error) {
	getter := geterrors.WrapFunc(getenv)

	username := getter.Get("CF_USERNAME")
	password := getter.Get("CF_PASSWORD")

	if err := getter.Err("missing environment variables"); err != nil {
		return Config{}, errors.New(err)
	}

	port, err := getPortFromEnv(getenv)
	if err != nil {
		return Config{}, errors.New(err)
	}

	environments, err := getEnvironmentsFromFile(configFilename)
	if err != nil {
		return Config{}, errors.New(err)
	}

	config := Config{
		Username:     username,
		Password:     password,
		Port:         port,
		Environments: environments,
	}
	return config, nil
}

func getPortFromEnv(getenv func(string) string) (int, error) {
	envPort := getenv("PORT")
	if envPort == "" {
		envPort = "8080"
	}

	cfgPort, err := strconv.Atoi(envPort)
	if err != nil {
		return 0, errors.Errorf("%s: %s: %s", cannotParsePort, envPort, err)
	}

	return cfgPort, nil
}

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
