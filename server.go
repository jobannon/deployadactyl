package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/compozed/deployadactyl/creator"
	"github.com/compozed/deployadactyl/state/push"
	"github.com/op/go-logging"
	"github.com/compozed/deployadactyl/interfaces"
)

const (
	defaultConfigFilePath = "./config.yml"
	defaultLogLevel       = "DEBUG"
	logLevelEnvVarName    = "DEPLOYADACTYL_LOGLEVEL"
)

func main() {
	var (
		config               = flag.String("config", defaultConfigFilePath, "location of the config file")
		envVarHandlerEnabled = flag.Bool("env", false, "enable environment variable handling")
		routeMapperEnabled   = flag.Bool("route-mapper", false, "enables route mapper to map additional routes from a manifest")
	)
	flag.Parse()

	level := os.Getenv(logLevelEnvVarName)
	if level == "" {
		level = defaultLogLevel
	}

	logLevel, err := logging.LogLevel(level)
	if err != nil {
		log.Fatal(err)
	}

	log := interfaces.DefaultLogger(os.Stdout, logLevel, "deployadactyl")
	log.Infof("log level : %s", level)

	c, err := creator.Custom(level, *config, creator.CreatorModuleProvider{})
	if err != nil {
		log.Fatal(err)
	}

	em := c.CreateEventManager()

	if *envVarHandlerEnabled {
		envVarHandler := c.CreateEnvVarHandler()
		log.Infof("registering environment variable event handler")
		em.AddBinding(push.NewArtifactRetrievalSuccessEventBinding(envVarHandler.ArtifactRetrievalSuccessEventHandler))
	}

	healthHandler := c.CreateHealthChecker()
	log.Infof("registering health check handler")
	em.AddBinding(push.NewPushFinishedEventBinding(healthHandler.PushFinishedEventHandler))

	if *routeMapperEnabled {
		routeMapper := c.CreateRouteMapper()

		log.Infof("registering health check handler")
		em.AddBinding(push.NewPushFinishedEventBinding(routeMapper.PushFinishedEventHandler))
	}

	l := c.CreateListener()
	controller := c.CreateController()

	deploy := c.CreateControllerHandler(controller)

	log.Infof("Listening on Port %d", c.CreateConfig().Port)

	err = http.Serve(l, deploy)
	if err != nil {
		log.Fatal(err)
	}
}
