package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/compozed/deployadactyl/creator"
	C "github.com/compozed/deployadactyl/constants"
	"github.com/compozed/deployadactyl/logger"
	"github.com/op/go-logging"
)

func main() {
	config := flag.String("config", C.DEFAULT_CONFIG_FILE_PATH, "location of the config file")
	envVarHandlerEnabled := flag.Bool(C.ENABLE_ENV_VAR_HANDLER_FLAG_ARG, false, "enable the environment variable handler (default: false)")
	flag.Parse()

	level := os.Getenv("DEPLOYADACTYL_LOGLEVEL")
	if level == "" {
		level = C.DEFAULT_LOG_LEVEL
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
		em.AddHandler(c.CreateEnvVarHandler(), C.DEPLOY_START_EVENT)
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
