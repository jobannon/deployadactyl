package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/compozed/deployadactyl/creator"
	"github.com/compozed/deployadactyl/logger"
	"github.com/op/go-logging"
	C "github.com/compozed/deployadactyl/constants"
)

const (
	defaultConfigFilePath    = "./config.yml"
	configFileArg            = "config"
	defaultLogLevel          = "DEBUG"
	logLevelEnvVarName       = "DEPLOYADACTYL_LOGLEVEL"
	envVarHandlerEnabledFlag = "env"
)

func main() {
	config := flag.String(configFileArg, defaultConfigFilePath, "location of the config file")
	envVarHandlerEnabled := flag.Bool(envVarHandlerEnabledFlag, false, "enable environment variable handling")
	flag.Parse()

	level := os.Getenv(logLevelEnvVarName)
	if level == "" {
		level = defaultLogLevel
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
		log.Infof("Adding Environment Variable Event Handler")
		em.AddHandler(c.CreateEnvVarHandler(), C.DeployStartEvent)
	} else {
		log.Info("No Event Handlers added...")
	}

	l := c.CreateListener()
	deploy := c.CreateControllerHandler()

	log.Infof("Listening on Port %d", c.CreateConfig().Port)

	err = http.Serve(l, deploy)
	if err != nil {
		log.Fatal(err)
	}
}
