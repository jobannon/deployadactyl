package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/compozed/deployadactyl/creator"
	"github.com/compozed/deployadactyl/logger"
	"github.com/op/go-logging"
)

const (
	default_config_file_path = "./config.yml"
	config_file_arg          = "config"
	default_log_level        = "DEBUG"
	log_level_env_var_name   = "DEPLOYADACTYL_LOGLEVEL"
)


func main() {
	config := flag.String(config_file_arg, default_config_file_path, "location of the config file")
	flag.Parse()

	level := os.Getenv(log_level_env_var_name)
	if level == "" {
		level = default_log_level
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
