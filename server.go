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
	defaultConfig = "./config.yml"
	defaultLevel  = "DEBUG"
)

func main() {
	config := flag.String("config", defaultConfig, "location of the config file")
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

	// add your event handling here

	l := c.CreateListener()
	dh := c.CreateControllerHandler()

	log.Infof("Listening on Port %d", c.CreateConfig().Port)

	err = http.Serve(l, dh)
	if err != nil {
		log.Fatal(err)
	}
}
