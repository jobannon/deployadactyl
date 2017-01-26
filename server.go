package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/compozed/deployadactyl/creator"
	"github.com/compozed/deployadactyl/logger"
	"github.com/op/go-logging"
	I "github.com/compozed/deployadactyl/interfaces"
)

const (
	defaultConfig = "./config.yml"
	defaultLevel  = "DEBUG"
)

func main() {
	config := flag.String("config", defaultConfig, "location of the config file")
	envVarHandlerEnabled := flag.Bool(I.ENABLE_ENV_VAR_HANDLER_FLAG_ARG, false, "enable the environment variable handler (default: false)")
	flag.Bool(I.ENABLE_DISABLE_FILESYSTEM_CLEANUP_ON_DEPLOY_FAILURE_FLAG_ARG, true, "enable/disable cleanup of temp file system on deploy failure. (default: true")
	flag.Parse()

	level := os.Getenv("DEPLOYADACTYL_LOGLEVEL")
	if level == "" {
		level = defaultLevel
	}

	logLevel, err := logging.LogLevel(level)
	if err != nil {
		log.Fatal(err)
	}

	log := logger.DefaultLogger(os.Stdout, logLevel, "deployadactyl")
	log.Infof("log level : %s", level)

	c, err := creator.Custom(level, *config)
	if err != nil {
		log.Fatal(err)
	}

	em := c.CreateEventManager()

	if *envVarHandlerEnabled {
		em.AddHandler(c.CreateEnvVarHandler(), I.ENV_VARS_FOUND_EVENT)
	}

	l := c.CreateListener()
	deploy := c.CreateControllerHandler()

	log.Infof("Listening on Port %d", c.CreateConfig().Port)

	err = http.Serve(l, deploy)
	if err != nil {
		log.Fatal(err)
	}
}
