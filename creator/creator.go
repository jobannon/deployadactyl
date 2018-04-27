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
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/randomizer"
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
	NewCourier courier.CourierConstructor
	NewPrechecker prechecker.PrecheckerConstructor
	NewFetcher artifetcher.ArtifetcherConstructor
	NewExtractor extractor.ExtractorConstructor
	NewEventManager eventmanager.EventManagerConstructor
}

// Creator has a config, eventManager, logger and writer for creating dependencies.
type Creator struct {
	config       config.Config
	eventManager I.EventManager
	logger       I.Logger
	writer       io.Writer
	fileSystem   *afero.Afero
	provider CreatorModuleProvider
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

// CreateLogger returns a Logger.
func (c Creator) CreateLogger() I.Logger {
	return c.logger
}

// CreateConfig returns a Config.
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
		Deployer:        c.createDeployer(),
		SilentDeployer:  c.createSilentDeployer(),
		Log:             c.CreateLogger(),
		PushController:  c.CreatePushController(),
		StopController:  c.CreateStopController(),
		StartController: c.CreateStartController(),
		Config:          c.CreateConfig(),
		EventManager:    c.CreateEventManager(),
		ErrorFinder:     c.createErrorFinder(),
	}
}

func (c Creator) CreatePushController() I.PushController {
	return &push.PushController{
		Deployer:           c.createDeployer(),
		Log:                c.CreateLogger(),
		Config:             c.CreateConfig(),
		EventManager:       c.CreateEventManager(),
		ErrorFinder:        c.createErrorFinder(),
		PushManagerFactory: c,
	}
}

func (c Creator) CreateStopController() I.StopController {
	return &stop.StopController{
		Deployer:           c.createDeployer(),
		Log:                c.CreateLogger(),
		Config:             c.CreateConfig(),
		EventManager:       c.CreateEventManager(),
		ErrorFinder:        c.createErrorFinder(),
		StopManagerFactory: c,
	}
}

func (c Creator) CreateStartController() I.StartController {
	return &start.StartController{
		Deployer:            c.createDeployer(),
		Log:                 c.CreateLogger(),
		Config:              c.CreateConfig(),
		EventManager:        c.CreateEventManager(),
		ErrorFinder:         c.createErrorFinder(),
		StartManagerFactory: c,
	}
}

func (c Creator) createDeployer() I.Deployer {
	return deployer.Deployer{
		Config:       c.CreateConfig(),
		BlueGreener:  c.createBlueGreener(),
		Prechecker:   c.createPrechecker(),
		EventManager: c.CreateEventManager(),
		Randomizer:   c.createRandomizer(),
		ErrorFinder:  c.createErrorFinder(),
		Log:          c.CreateLogger(),
	}
}

func (c Creator) PushManager(deployEventData structs.DeployEventData, cf I.CFContext, auth I.Authorization, env structs.Environment, envVars map[string]string) I.ActionCreator {
	deploymentLogger := logger.DeploymentLogger{c.CreateLogger(), deployEventData.DeploymentInfo.UUID}
	return &push.PushManager{
		CourierCreator:       c,
		EventManager:         c.CreateEventManager(),
		Logger:               deploymentLogger,
		Fetcher:              c.createFetcher(),
		DeployEventData:      deployEventData,
		FileSystemCleaner:    c.CreateFileSystem(),
		CFContext:            cf,
		Auth:                 auth,
		Environment:          env,
		EnvironmentVariables: envVars,
	}
}

func (c Creator) StopManager(deployEventData structs.DeployEventData) I.ActionCreator {
	deploymentLogger := logger.DeploymentLogger{c.CreateLogger(), deployEventData.DeploymentInfo.UUID}
	return stop.StopManager{
		CourierCreator:  c,
		EventManager:    c.CreateEventManager(),
		Log:             deploymentLogger,
		DeployEventData: deployEventData,
	}
}

func (c Creator) StartManager(deployEventData structs.DeployEventData) I.ActionCreator {
	deploymentLogger := logger.DeploymentLogger{c.CreateLogger(), deployEventData.DeploymentInfo.UUID}
	return start.StartManager{
		CourierCreator:  c,
		EventManager:    c.CreateEventManager(),
		Logger:          deploymentLogger,
		DeployEventData: deployEventData,
	}
}

func (c Creator) createSilentDeployer() I.Deployer {
	return deployer.SilentDeployer{}
}

func (c Creator) createExtractor() I.Extractor {
	if c.provider.NewExtractor != nil {
		return c.provider.NewExtractor(c.CreateLogger(), c.CreateFileSystem())
	}
	return extractor.NewExtractor(c.CreateLogger(), c.CreateFileSystem())
}

func (c Creator) createFetcher() I.Fetcher {
	if c.provider.NewFetcher != nil {
		return c.provider.NewFetcher(c.CreateFileSystem(), c.createExtractor(), c.CreateLogger())
	}
	return artifetcher.NewArtifetcher(c.CreateFileSystem(), c.createExtractor(), c.CreateLogger())
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

func (c Creator) createBlueGreener() I.BlueGreener {
	return bluegreen.BlueGreen{
		Log: c.CreateLogger(),
	}
}

func (c Creator) createErrorFinder() I.ErrorFinder {
	return &error_finder.ErrorFinder{
		Matchers: c.config.ErrorMatchers,
	}
}

func createCreator(l logging.Level, cfg config.Config, provider CreatorModuleProvider) (Creator, error) {
	err := ensureCLI()
	if err != nil {
		return Creator{}, err
	}

	logger := logger.DefaultLogger(os.Stdout, l, "controller")
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
