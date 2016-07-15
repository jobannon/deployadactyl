package main

import (
	"flag"
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
	level := flag.String("loglevel", defaultLevel, "available levels: DEBUG, INFO, NOTI, WARN, ERROR, CRIT")
	flag.Parse()

	logLevel, _ := logging.LogLevel(*level)
	log := logger.DefaultLogger(os.Stdout, logLevel, "deployadactyl")

	c, err := creator.Custom(defaultLevel, *config)
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
