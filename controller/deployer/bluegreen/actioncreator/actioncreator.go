package actioncreator

import (
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/pusher"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/startstopper"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	S "github.com/compozed/deployadactyl/structs"
	"io"
)

type PusherCreator struct {
	Courier      I.Courier
	EventManager I.EventManager
	Logger       I.Logger
}

type StopperCreator struct {
	Courier      I.Courier
	EventManager I.EventManager
	Logger       I.Logger
}

func (a PusherCreator) CreatePusher(deploymentInfo S.DeploymentInfo, response io.ReadWriter, foundationURL, appPath string) (I.Action, error) {

	p := &pusher.Pusher{
		Courier:        a.Courier,
		DeploymentInfo: deploymentInfo,
		EventManager:   a.EventManager,
		Response:       response,
		Log:            logger.DeploymentLogger{a.Logger, deploymentInfo.UUID},
		FoundationURL:  foundationURL,
		AppPath:        appPath,
	}

	return p, nil
}

func (a StopperCreator) CreateStopper(cfContext I.CFContext, authorization I.Authorization, deploymentInfo S.DeploymentInfo, response io.ReadWriter, foundationURL string) (I.Action, error) {

	p := &startstopper.Stopper{
		Courier:       a.Courier,
		CFContext:     cfContext,
		Authorization: authorization,
		EventManager:  a.EventManager,
		Response:      response,
		Log:           logger.DeploymentLogger{a.Logger, deploymentInfo.UUID},
		FoundationURL: foundationURL,
		AppName:       deploymentInfo.AppName,
	}

	return p, nil
}
