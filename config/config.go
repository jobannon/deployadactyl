package config

import (
	"strconv"

	"github.com/compozed/deployadactyl/geterrors"
	"github.com/go-errors/errors"
)

const (
	unableToGetLogLevel = "unable to get log level"
	cannotParsePort     = "cannot parse $PORT"
)

type Config struct {
	Username     string
	Password     string
	Environments map[string]Environment
	Port         int
}

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
