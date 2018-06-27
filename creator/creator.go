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

	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"

	"bytes"

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
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
	"github.com/spf13/afero"
)

// ENDPOINT is used by the handler to define the deployment endpoint.
const v2ENDPOINT = "/v2/deploy/:environment/:org/:space/:appName"
const ENDPOINT = "/v3/apps/:environment/:org/:space/:appName"

type InvalidRequestError struct{}

func (e InvalidRequestError) Error() string {
	return "invalid request"
}

type InvalidRequestProcessor struct {
	Err error
}

func (p InvalidRequestProcessor) Process() I.DeployResponse {
	return I.DeployResponse{
		StatusCode: 400,
		Error:      p.Err,
	}
}

type CreatorModuleProvider struct {
	NewCourier               courier.CourierConstructor
	NewPrechecker            prechecker.PrecheckerConstructor
	NewFetcher               artifetcher.ArtifetcherConstructor
	NewExtractor             extractor.ExtractorConstructor
	NewEventManager          eventmanager.EventManagerConstructor
	NewPushController        push.PushControllerConstructor
	NewStartController       start.StartControllerConstructor
	NewStopController        stop.StopControllerConstructor
	NewAuthResolver          state.AuthResolverConstructor
	NewEnvResolver           state.EnvResolverConstructor
	NewDeployer              deployer.DeployerConstructor
	NewPushManager           push.PushManagerConstructor
	NewStopManager           stop.StopManagerConstructor
	NewStartManager          start.StartManagerConstructor
	NewBlueGreen             bluegreen.BlueGreenConstructor
	NewPushRequestProcessor  push.PushRequestProcessorConstructor
	NewPushRequestCreator    PushRequestCreatorConstructor
	NewStopRequestProcessor  stop.StopRequestProcessorConstructor
	NewStopRequestCreator    StopRequestCreatorConstructor
	NewStartRequestProcessor start.StartRequestProcessorConstructor
	NewStartRequestCreator   StartRequestCreatorConstructor
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

	r.POST(v2ENDPOINT, controller.PostRequestHandler)
	r.POST(ENDPOINT, controller.PostRequestHandler)
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
		RequestProcessorFactory: c.CreateRequestProcessor,
		Config:                  c.CreateConfig(),
		EventManager:            c.CreateEventManager(),
		ErrorFinder:             c.createErrorFinder(),
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

func (c Creator) CreateRequestProcessor(uuid string, request interface{}, buffer *bytes.Buffer) I.RequestProcessor {
	requestCreator, err := c.CreateRequestCreator(uuid, request, buffer)
	if err != nil {
		return InvalidRequestProcessor{Err: err}
	}
	return requestCreator.CreateRequestProcessor()
}

func (c Creator) CreateRequestCreator(uuid string, request interface{}, buffer *bytes.Buffer) (I.RequestCreator, error) {
	post, ok := request.(I.PostDeploymentRequest)
	if ok {
		if c.provider.NewPushRequestCreator != nil {
			return c.provider.NewPushRequestCreator(c, uuid, post, buffer), nil
		}
		return NewPushRequestCreator(c, uuid, post, buffer), nil
	}
	put, ok := request.(I.PutDeploymentRequest)
	if ok {
		if put.Request.State == "stopped" {
			if c.provider.NewStopRequestCreator != nil {
				return c.provider.NewStopRequestCreator(c, uuid, put, buffer), nil
			}
			return NewStopRequestCreator(c, uuid, put, buffer), nil
		} else if put.Request.State == "started" {
			if c.provider.NewStartRequestCreator != nil {
				return c.provider.NewStartRequestCreator(c, uuid, put, buffer), nil
			}
			return NewStartRequestCreator(c, uuid, put, buffer), nil
		}
	}
	return nil, InvalidRequestError{}
}

func (c Creator) createSilentDeployer() I.Deployer {
	return deployer.SilentDeployer{}
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
