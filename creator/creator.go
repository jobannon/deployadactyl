// Package creator creates dependencies upon initialization.
package creator

import (
	"io"
	"log"
	"net"
	"os"
	"os/exec"

	"github.com/compozed/deployadactyl"
	"github.com/compozed/deployadactyl/artifetcher"
	"github.com/compozed/deployadactyl/artifetcher/extractor"
	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/deployer"
	"github.com/compozed/deployadactyl/deployer/bluegreen"
	"github.com/compozed/deployadactyl/deployer/bluegreen/pusher"
	"github.com/compozed/deployadactyl/deployer/bluegreen/pusher/courier"
	"github.com/compozed/deployadactyl/deployer/bluegreen/pusher/courier/executor"
	"github.com/compozed/deployadactyl/deployer/eventmanager"
	"github.com/compozed/deployadactyl/deployer/prechecker"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/gin-gonic/gin"
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
	"github.com/spf13/afero"
)

const (
	ENDPOINT      = "/v1/apps/:environment/:org/:space/:appName" // ENDPOINT is used by the handler to define the deployment endpoint.
	defaultConfig = "./config.yml"
	defaultLevel  = "DEBUG"
)

// EnsureCLI looks for the Cloud Foundary binary, otherwise it returns an error.
func EnsureCLI() error {
	_, err := exec.LookPath("cf")
	return err
}

// Default returns a default Creator and an Error.
func Default() (Creator, error) {
	l, err := getLevel(defaultLevel)
	if err != nil {
		return Creator{}, err
	}

	cfg, err := config.Default(os.Getenv)
	if err != nil {
		return Creator{}, err
	}
	return createCreator(l, cfg)
}

// Custom returns a custom Creator with an Error.
func Custom(level string, configFilename string) (Creator, error) {
	l, err := getLevel(level)
	if err != nil {
		return Creator{}, err
	}

	cfg, err := config.Custom(os.Getenv, configFilename)
	if err != nil {
		return Creator{}, err
	}
	return createCreator(l, cfg)
}

type Creator struct {
	config       config.Config
	eventManager I.EventManager
	logger       *logging.Logger
	writer       io.Writer
}

// CreateDeployadactylHandler returns a gin.Engine that implements http.Handler.
// Sets up the deployadactyl endpoint.
func (c Creator) CreateDeployadactylHandler() *gin.Engine {
	d := c.CreateDeployadactyl()

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithWriter(c.CreateWriter()))
	r.Use(gin.ErrorLogger())

	r.POST(ENDPOINT, d.Deploy)

	return r
}

// CreateDeployadactyl returns a Deployadactyl.
func (c Creator) CreateDeployadactyl() deployadactyl.Deployadactyl {
	return deployadactyl.Deployadactyl{
		Deployer:     c.CreateDeployer(),
		Log:          c.CreateLogger(),
		Config:       c.CreateConfig(),
		EventManager: c.CreateEventManager(),
		Randomizer:   c.CreateRandomizer(),
	}
}

// CreateRandomizer returns a randomizer.
func (c Creator) CreateRandomizer() I.Randomizer {
	return randomizer.Randomizer{}
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

// CreateDeployer creates and returns a deployer.
func (c Creator) CreateDeployer() I.Deployer {
	return deployer.Deployer{
		Environments: c.config.Environments,
		BlueGreener:  c.CreateBlueGreener(),
		Fetcher: &artifetcher.Artifetcher{
			FileSystem: &afero.Afero{Fs: afero.NewOsFs()},
			Extractor: &extractor.Extractor{
				Log:        c.CreateLogger(),
				FileSystem: &afero.Afero{Fs: afero.NewOsFs()},
			},
			Log: c.CreateLogger(),
		},
		Prechecker:   c.CreatePrechecker(),
		EventManager: c.CreateEventManager(),
		Log:          c.CreateLogger(),
	}
}

// CreatePusher is used by the BlueGreener.
//
// Returns a pusher and error.
func (c Creator) CreatePusher() (I.Pusher, error) {
	fs := &afero.Afero{Fs: afero.NewOsFs()}
	ex, err := executor.New(fs)
	if err != nil {
		return nil, err
	}

	p := pusher.Pusher{
		Courier: courier.Courier{
			Executor: ex,
		},
		Log: c.CreateLogger(),
	}

	return p, nil
}

// CreateEventManager returns an EventManager.
func (c Creator) CreateEventManager() I.EventManager {
	return c.eventManager
}

// CreateLogger returns a Logger.
func (c Creator) CreateLogger() *logging.Logger {
	return c.logger
}

// CreateConfig returns a Config.
func (c Creator) CreateConfig() config.Config {
	return c.config
}

// CreatePrechecker returns a Prechecker.
// EventManager is used to handle events within the prechecker.
func (c Creator) CreatePrechecker() I.Prechecker {
	return prechecker.Prechecker{c.CreateEventManager()}
}

// CreateWriter returns a Writer.
func (c Creator) CreateWriter() io.Writer {
	return c.writer
}

// CreateBlueGreener returns a BlueGreener.
func (c Creator) CreateBlueGreener() I.BlueGreener {
	return bluegreen.BlueGreen{
		PusherCreator: c,
		Log:           c.CreateLogger(),
	}
}

func createCreator(l logging.Level, cfg config.Config) (Creator, error) {
	err := EnsureCLI()
	if err != nil {
		return Creator{}, err
	}

	eventManager := eventmanager.NewEventManager()

	logger := logger.DefaultLogger(os.Stdout, l, "deployadactyl")
	return Creator{
		cfg,
		eventManager,
		logger,
		os.Stdout,
	}, nil

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
