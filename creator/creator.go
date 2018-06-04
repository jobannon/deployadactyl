// Package creator creates dependencies upon initialization.
package creator

import (
	"crypto/tls"
	"fmt"
	"github.com/compozed/deployadactyl/artifetcher"
	"github.com/compozed/deployadactyl/artifetcher/extractor"
	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/controller"
	"github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/state/push"

	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/courier"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/courier/executor"
	"github.com/compozed/deployadactyl/controller/deployer/error_finder"
	"github.com/compozed/deployadactyl/controller/deployer/prechecker"
	"github.com/compozed/deployadactyl/eventmanager"
	"github.com/compozed/deployadactyl/eventmanager/handlers/envvar"
	"github.com/compozed/deployadactyl/eventmanager/handlers/healthchecker"
	"github.com/compozed/deployadactyl/eventmanager/handlers/routemapper"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/compozed/deployadactyl/state"
	"github.com/compozed/deployadactyl/state/start"
	"github.com/compozed/deployadactyl/state/stop"
	"github.com/compozed/deployadactyl/structs"
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
	"github.com/spf13/afero"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
)

// ENDPOINT is used by the handler to define the deployment endpoint.
const v2ENDPOINT = "/v2/deploy/:environment/:org/:space/:appName"
const ENDPOINT = "/v3/apps/:environment/:org/:space/:appName"

type CreatorModuleProvider struct {
	NewCourier         courier.CourierConstructor
	NewPrechecker      prechecker.PrecheckerConstructor
	NewFetcher         artifetcher.ArtifetcherConstructor
	NewExtractor       extractor.ExtractorConstructor
	NewEventManager    eventmanager.EventManagerConstructor
	NewPushController  push.PushControllerConstructor
	NewStartController start.StartControllerConstructor
	NewStopController  stop.StopControllerConstructor
	NewAuthResolver    state.AuthResolverConstructor
	NewEnvResolver     state.EnvResolverConstructor
	NewDeployer        deployer.DeployerConstructor
	NewPushManager     push.PushManagerConstructor
	NewStopManager     stop.StopManagerConstructor
	NewStartManager    start.StartManagerConstructor
	NewBlueGreen       bluegreen.BlueGreenConstructor
}

// Creator has a config, eventManager, logger and writer for creating dependencies.
type Creator struct {
	config       config.Config
	eventManager I.EventManager
	logger       I.Logger
	writer       io.Writer
	fileSystem   *afero.Afero
	provider     CreatorModuleProvider
}

// Default returns a default Creator and an Error.
func Default() (Creator, error) {
	cfg, err := config.Default(os.Getenv)
	if err != nil {
		return Creator{}, err
	}
	return createCreator(logging.DEBUG, cfg, CreatorModuleProvider{})
}

// Custom returns a custom Creator with an Error.
func Custom(level string, configFilename string, provider CreatorModuleProvider) (Creator, error) {
	l, err := getLevel(level)
	if err != nil {
		return Creator{}, err
	}

	cfg, err := config.Custom(os.Getenv, configFilename)
	if err != nil {
		return Creator{}, err
	}
	return createCreator(l, cfg, provider)
}

// CreateControllerHandler returns a gin.Engine that implements http.Handler.
// Sets up the controller endpoint.
func (c Creator) CreateControllerHandler(controller I.Controller) *gin.Engine {

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithWriter(c.createWriter()))
	r.Use(gin.ErrorLogger())

	r.POST(v2ENDPOINT, controller.RunDeploymentViaHttp)
	r.POST(ENDPOINT, controller.RunDeploymentViaHttp)
	r.PUT(ENDPOINT, controller.PutRequestHandler)

	return r
}

// CreateListener creates a listener TCP and listens for all incoming requests.
func (c Creator) CreateListener() net.Listener {
	ls, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: c.config.Port,
		Zone: "",
	})
	if err != nil {
		log.Fatal(err)
	}
	return ls
}

// CreateCourier returns a courier with an executor.
func (c Creator) CreateCourier() (I.Courier, error) {
	ex, err := executor.New(c.CreateFileSystem())
	if err != nil {
		return nil, err
	}

	if c.provider.NewCourier != nil {
		return c.provider.NewCourier(ex), nil
	}

	return courier.NewCourier(ex), nil
}

func (c Creator) GetLogger() I.Logger {
	return c.logger
}

// CreateConfig returns a Config
func (c Creator) CreateConfig() config.Config {
	return c.config
}

// CreateEventManager returns an EventManager.
func (c Creator) CreateEventManager() I.EventManager {
	return c.eventManager
}

// CreateFileSystem returns a file system.
func (c Creator) CreateFileSystem() *afero.Afero {
	return c.fileSystem
}

// CreateHTTPClient return an http client.
func (c Creator) CreateHTTPClient() *http.Client {
	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return insecureClient
}

func (c Creator) CreateController() I.Controller {
	return &controller.Controller{
		Log: c.logger,
		PushControllerFactory:  c.CreatePushController,
		StopControllerFactory:  c.CreateStopController,
		StartControllerFactory: c.CreateStartController,
		Config:                 c.CreateConfig(),
		EventManager:           c.CreateEventManager(),
		ErrorFinder:            c.createErrorFinder(),
	}
}
func (c Creator) CreateAuthResolver() I.AuthResolver {
	if c.provider.NewAuthResolver != nil {
		return c.provider.NewAuthResolver(c.CreateConfig())
	}
	return state.NewAuthResolver(c.CreateConfig())
}

func (c Creator) CreateEnvResolver() I.EnvResolver {
	if c.provider.NewEnvResolver != nil {
		return c.provider.NewEnvResolver(c.CreateConfig())
	}
	return state.NewEnvResolver(c.CreateConfig())
}

func (c Creator) CreatePushController(log I.DeploymentLogger) I.PushController {
	if c.provider.NewPushController != nil {
		return c.provider.NewPushController(log, c.createDeployer(log), c.createSilentDeployer(), c.CreateEventManager(), c.createErrorFinder(), c, c.CreateAuthResolver(), c.CreateEnvResolver())
	}
	return push.NewPushController(log, c.createDeployer(log), c.createSilentDeployer(), c.CreateEventManager(), c.createErrorFinder(), c, c.CreateAuthResolver(), c.CreateEnvResolver())
}

func (c Creator) CreateStopController(log I.DeploymentLogger) I.StopController {
	if c.provider.NewStopController != nil {
		return c.provider.NewStopController(log, c.createDeployer(log), c.CreateEventManager(), c.createErrorFinder(), c, c.CreateAuthResolver(), c.CreateEnvResolver())
	}
	return stop.NewStopController(log, c.createDeployer(log), c.CreateEventManager(), c.createErrorFinder(), c, c.CreateAuthResolver(), c.CreateEnvResolver())
}

func (c Creator) CreateStartController(log I.DeploymentLogger) I.StartController {
	if c.provider.NewStartController != nil {
		return c.provider.NewStartController(log, c.createDeployer(log), c.CreateEventManager(), c.createErrorFinder(), c, c.CreateAuthResolver(), c.CreateEnvResolver())
	}
	return start.NewStartController(log, c.createDeployer(log), c.CreateEventManager(), c.createErrorFinder(), c, c.CreateAuthResolver(), c.CreateEnvResolver())
}

func (c Creator) createDeployer(log I.DeploymentLogger) I.Deployer {
	if c.provider.NewDeployer != nil {
		return c.provider.NewDeployer(c.CreateConfig(), c.createBlueGreener(log), c.createPrechecker(), c.CreateEventManager(), c.createRandomizer(), c.createErrorFinder(), log)
	}
	return deployer.NewDeployer(c.CreateConfig(), c.createBlueGreener(log), c.createPrechecker(), c.CreateEventManager(), c.createRandomizer(), c.createErrorFinder(), log)
}

func (c Creator) PushManager(log I.DeploymentLogger, deployEventData structs.DeployEventData, cf I.CFContext, auth I.Authorization, env structs.Environment, envVars map[string]string) I.ActionCreator {
	if c.provider.NewPushManager != nil {
		return c.provider.NewPushManager(c, c.CreateEventManager(), log, c.createFetcher(log), deployEventData, c.CreateFileSystem(), cf, auth, env, envVars)
	}
	return push.NewPushManager(c, c.CreateEventManager(), log, c.createFetcher(log), deployEventData, c.CreateFileSystem(), cf, auth, env, envVars)
}

func (c Creator) StopManager(log I.DeploymentLogger, deployEventData structs.DeployEventData) I.ActionCreator {
	if c.provider.NewStopManager != nil {
		return c.provider.NewStopManager(c, c.CreateEventManager(), log, deployEventData)
	}
	return stop.NewStopManager(c, c.CreateEventManager(), log, deployEventData)
}

func (c Creator) StartManager(log I.DeploymentLogger, deployEventData structs.DeployEventData) I.ActionCreator {
	if c.provider.NewStartManager != nil {
		return c.provider.NewStartManager(c, c.CreateEventManager(), log, deployEventData)
	}
	return start.NewStartManager(c, c.CreateEventManager(), log, deployEventData)
}

func (c Creator) CreateEnvVarHandler() envvar.Envvarhandler {
	return envvar.Envvarhandler{FileSystem: c.CreateFileSystem()}
}

func (c Creator) CreateHealthChecker() healthchecker.HealthChecker {
	silentUrl := os.Getenv("SILENT_DEPLOY_URL")
	silentEnv := os.Getenv("SILENT_DEPLOY_ENVIRONMENT")
	return healthchecker.NewHealthChecker("api.cf", "apps", silentUrl, silentEnv, c.CreateHTTPClient())
}

func (c Creator) CreateRouteMapper() routemapper.RouteMapper {
	return routemapper.RouteMapper{
		FileSystem: c.CreateFileSystem(),
	}
}

func (c Creator) createSilentDeployer() I.Deployer {
	return deployer.SilentDeployer{}
}

func (c Creator) createExtractor(log I.DeploymentLogger) I.Extractor {
	if c.provider.NewExtractor != nil {
		return c.provider.NewExtractor(log, c.CreateFileSystem())
	}
	return extractor.NewExtractor(log, c.CreateFileSystem())
}

func (c Creator) createFetcher(log I.DeploymentLogger) I.Fetcher {
	if c.provider.NewFetcher != nil {
		return c.provider.NewFetcher(c.CreateFileSystem(), c.createExtractor(log), log)
	}
	return artifetcher.NewArtifetcher(c.CreateFileSystem(), c.createExtractor(log), log)
}

func (c Creator) createRandomizer() I.Randomizer {
	return randomizer.Randomizer{}
}

func (c Creator) createPrechecker() I.Prechecker {
	if c.provider.NewPrechecker != nil {
		return c.provider.NewPrechecker(c.CreateEventManager())
	}
	return prechecker.NewPrechecker(c.CreateEventManager())
}

func (c Creator) createWriter() io.Writer {
	return c.writer
}

func (c Creator) createBlueGreener(log I.DeploymentLogger) I.BlueGreener {
	if c.provider.NewBlueGreen != nil {
		return c.provider.NewBlueGreen(log)
	}
	return bluegreen.NewBlueGreen(log)
}

func (c Creator) createErrorFinder() I.ErrorFinder {
	return &error_finder.ErrorFinder{
		Matchers: c.config.ErrorMatchers,
	}
}

/*
Skip refactoring the functions below
*/
func createCreator(l logging.Level, cfg config.Config, provider CreatorModuleProvider) (Creator, error) {
	err := ensureCLI()
	if err != nil {
		return Creator{}, err
	}

	logger := I.DefaultLogger(os.Stdout, l, "controller")
	var eventManager I.EventManager
	if provider.NewEventManager != nil {
		eventManager = provider.NewEventManager(logger)
	} else {
		eventManager = eventmanager.NewEventManager(logger)
	}

	return Creator{
		cfg,
		eventManager,
		logger,
		os.Stdout,
		&afero.Afero{Fs: afero.NewOsFs()},
		provider,
	}, nil

}

func ensureCLI() error {
	_, err := exec.LookPath("cf")
	return err
}

func getLevel(level string) (logging.Level, error) {
	if level != "" {
		l, err := logging.LogLevel(level)
		if err != nil {
			return 0, fmt.Errorf("unable to get log level: %s. error: %s", level, err.Error())
		}
		return l, nil
	}

	return logging.INFO, nil
}
