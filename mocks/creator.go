package mocks

import (
	"io"
	"os"

	"github.com/compozed/deployadactyl/artifetcher"
	"github.com/compozed/deployadactyl/artifetcher/extractor"
	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/controller"
	"github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/controller/deployer/error_finder"
	"github.com/compozed/deployadactyl/eventmanager"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/compozed/deployadactyl/state/push"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	logging "github.com/op/go-logging"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// ENDPOINT is used by the handler to define the deployment endpoint.
const ENDPOINT = "/v2/deploy/:environment/:org/:space/:appName"

// Handmade Creator mock.
// Uses a mock prechecker to skip verifying the foundations are up and running.
// Uses a mock Courier and Executor to mock pushing an application.
// Uses a mock FileSystem to mock writing to the operating system.
type Creator struct {
	CourierCreatorFn func() (I.Courier, error)
	config           config.Config
	eventManager     I.EventManager
	logger           I.Logger
	writer           io.Writer
	fileSystem       *afero.Afero
}

func NewCreator(level string, configFilename string) (Creator, error) {
	cfg, err := config.Custom(os.Getenv, configFilename)
	if err != nil {
		return Creator{}, err
	}

	l, err := getLevel(level)
	if err != nil {
		return Creator{}, err
	}

	logger := logger.DefaultLogger(GinkgoWriter, l, "creator")

	eventManager := eventmanager.NewEventManager(logger)

	return Creator{
		config:       cfg,
		eventManager: eventManager,
		logger:       logger,
		writer:       GinkgoWriter,
		fileSystem:   &afero.Afero{Fs: afero.NewMemMapFs()},
	}, nil
}

func (c Creator) CreateControllerHandler() *gin.Engine {
	d := c.CreateController()

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithWriter(c.CreateWriter()))
	r.Use(gin.ErrorLogger())

	r.POST(ENDPOINT, d.RunDeploymentViaHttp)

	return r
}

func (c Creator) CreateController() controller.Controller {
	return controller.Controller{
		Deployer:       c.CreateDeployer(),
		SilentDeployer: c.CreateSilentDeployer(),
		Log:            c.CreateLogger(),
		PushController: c.CreatePushController(),
		Config:         c.CreateConfig(),
		EventManager:   c.CreateEventManager(),
		ErrorFinder:    c.createErrorFinder(),
	}
}

func (c Creator) CreateRandomizer() I.Randomizer {
	return randomizer.Randomizer{}
}

func (c Creator) CreateDeployer() I.Deployer {
	return deployer.Deployer{
		Config:       c.CreateConfig(),
		BlueGreener:  c.CreateBlueGreener(),
		Prechecker:   c.CreatePrechecker(),
		EventManager: c.CreateEventManager(),
		Randomizer:   c.CreateRandomizer(),
		Log:          c.CreateLogger(),
		ErrorFinder:  c.createErrorFinder(),
	}
}
func (c Creator) CreateCourier() (I.Courier, error) {
	if c.CourierCreatorFn != nil {
		return c.CourierCreatorFn()
	}

	courier := &Courier{}

	courier.LoginCall.Returns.Output = []byte("logged in\t")
	courier.DeleteCall.Returns.Output = []byte("deleted app\t")
	courier.PushCall.Returns.Output = []byte("pushed app\t")
	courier.RenameCall.Returns.Output = []byte("renamed app\t")
	courier.MapRouteCall.Returns.Output = append(courier.MapRouteCall.Returns.Output, []byte("mapped route\t"))
	courier.ExistsCall.Returns.Bool = true

	return courier, nil
}

func (c Creator) CreatePushController() I.PushController {
	return &push.PushController{
		Deployer:           c.CreateDeployer(),
		Log:                c.CreateLogger(),
		Config:             c.CreateConfig(),
		EventManager:       c.CreateEventManager(),
		ErrorFinder:        c.createErrorFinder(),
		PushManagerFactory: c,
	}
}

func (c Creator) PushManager(deployEventData S.DeployEventData, cf I.CFContext, auth I.Authorization, env S.Environment) I.ActionCreator {
	deploymentLogger := logger.DeploymentLogger{c.CreateLogger(), deployEventData.DeploymentInfo.UUID}
	return &push.PushManager{
		CourierCreator:    c,
		EventManager:      c.CreateEventManager(),
		Logger:            deploymentLogger,
		Fetcher:           c.CreateFetcher(),
		DeployEventData:   deployEventData,
		FileSystemCleaner: c.CreateFileSystem(),
		CFContext:         cf,
		Auth:              auth,
		Environment:       env,
	}
}

func (c Creator) CreateStopperCreator() I.ActionCreator {
	return &StopManager{}
}

func (c Creator) CreateSilentDeployer() I.Deployer {
	return deployer.SilentDeployer{}
}

func (c Creator) CreateFetcher() I.Fetcher {
	return &artifetcher.Artifetcher{
		FileSystem: c.CreateFileSystem(),
		Extractor: &extractor.Extractor{
			Log:        c.CreateLogger(),
			FileSystem: c.CreateFileSystem(),
		},
		Log: c.CreateLogger(),
	}
}

func (c Creator) CreatePusher(deploymentInfo S.DeploymentInfo, response io.ReadWriter, foundationURL, appPath string) (I.Action, error) {
	courier := &Courier{}

	courier.LoginCall.Returns.Output = []byte("logged in\t")
	courier.DeleteCall.Returns.Output = []byte("deleted app\t")
	courier.PushCall.Returns.Output = []byte("pushed app\t")
	courier.RenameCall.Returns.Output = []byte("renamed app\t")
	courier.MapRouteCall.Returns.Output = append(courier.MapRouteCall.Returns.Output, []byte("mapped route\t"))
	courier.ExistsCall.Returns.Bool = true

	p := &push.Pusher{
		Courier:        courier,
		DeploymentInfo: deploymentInfo,
		EventManager:   c.CreateEventManager(),
		Response:       response,
		Log:            c.CreateLogger(),
		FoundationURL:  foundationURL,
		AppPath:        appPath,
		Fetcher:        c.CreateFetcher(),
	}

	return p, nil
}

func (c Creator) CreateEventManager() I.EventManager {
	return c.eventManager
}

func (c Creator) CreateLogger() I.Logger {
	return c.logger
}

func (c Creator) CreateConfig() config.Config {
	return c.config
}

func (c Creator) CreatePrechecker() I.Prechecker {
	return &Prechecker{}
}

func (c Creator) CreateWriter() io.Writer {
	return c.writer
}

func (c Creator) CreateBlueGreener() I.BlueGreener {
	return bluegreen.BlueGreen{
		Log: c.CreateLogger(),
	}
}

func (c Creator) CreateFileSystem() *afero.Afero {
	return c.fileSystem
}

func (c Creator) createErrorFinder() I.ErrorFinder {
	return &error_finder.ErrorFinder{
		Matchers: c.config.ErrorMatchers,
	}
}

func getLevel(level string) (logging.Level, error) {
	if level != "" {
		l, err := logging.LogLevel(level)
		if err != nil {
			return 0, errors.Errorf("unable to get log level: %s. error: %s", level, err.Error())
		}
		return l, nil
	}

	return logging.INFO, nil
}
