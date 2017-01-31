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

func main() {
	config := flag.String(C.CONFIG_FILE_ARG, C.DEFAULT_CONFIG_FILE_PATH, "location of the config file")
	flag.Parse()

	level := os.Getenv(C.LOG_LEVEL_ENV_VAR_NAME)
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

	// uncomment the next two lines to add your event handlers
	// em := c.CreateEventManager()
	// em.AddHandler(myInstanceHandler, "deploy.start")

	l := c.CreateListener()
	deploy := c.CreateControllerHandler()

	log.Infof("Listening on Port %d", c.CreateConfig().Port)

	err = http.Serve(l, deploy)
	if err != nil {
		log.Fatal(err)
	}
}
