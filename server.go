package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/compozed/deployadactyl/creator"
	"github.com/compozed/deployadactyl/eventmanager/handlers/envvar"
	"github.com/compozed/deployadactyl/eventmanager/handlers/healthchecker"
	"github.com/compozed/deployadactyl/eventmanager/handlers/routemapper"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/state/push"
	"github.com/op/go-logging"
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

	log := logger.DefaultLogger(os.Stdout, logLevel, "deployadactyl")
	log.Infof("log level : %s", level)

	c, err := creator.Custom(level, *config, creator.CreatorModuleProvider{})
	if err != nil {
		log.Fatal(err)
	}

	em := c.CreateEventManager()

	if *envVarHandlerEnabled {
		envVarHandler := envvar.Envvarhandler{Logger: c.CreateLogger(), FileSystem: c.CreateFileSystem()}
		log.Infof("registering environment variable event handler")
		em.AddBinding(push.NewArtifactRetrievalSuccessEventBinding(envVarHandler.ArtifactRetrievalSuccessEventHandler))
	}

	healthHandler := healthchecker.HealthChecker{
		OldURL: "api.cf",
		NewURL: "apps",
		Client: c.CreateHTTPClient(),
		Log:    c.CreateLogger(),
	}
	log.Infof("registering health check handler")
	em.AddBinding(push.NewPushFinishedEventBinding(healthHandler.PushFinishedEventHandler))

	if *routeMapperEnabled {
		routeMapper := routemapper.RouteMapper{
			FileSystem: c.CreateFileSystem(),
			Log:        c.CreateLogger(),
		}

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
