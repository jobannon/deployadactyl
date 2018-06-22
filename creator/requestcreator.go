package creator

import (
	"bytes"

	"github.com/compozed/deployadactyl/artifetcher"
	"github.com/compozed/deployadactyl/artifetcher/extractor"
	"github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/state/push"
	"github.com/compozed/deployadactyl/state/start"
	"github.com/compozed/deployadactyl/state/stop"
	"github.com/compozed/deployadactyl/structs"
)

type RequestCreator struct {
	Creator
	Buffer *bytes.Buffer
	Log    I.DeploymentLogger
}

func (r *RequestCreator) CreateDeployer() I.Deployer {
	if r.provider.NewDeployer != nil {
		return r.provider.NewDeployer(r.CreateConfig(), r.CreateBlueGreener(), r.createPrechecker(), r.CreateEventManager(), r.createRandomizer(), r.createErrorFinder(), r.Log)
	}
	return deployer.NewDeployer(r.CreateConfig(), r.CreateBlueGreener(), r.createPrechecker(), r.CreateEventManager(), r.createRandomizer(), r.createErrorFinder(), r.Log)
}

func (r RequestCreator) CreateBlueGreener() I.BlueGreener {
	if r.provider.NewBlueGreen != nil {
		return r.provider.NewBlueGreen(r.Log)
	}
	return bluegreen.NewBlueGreen(r.Log)
}

func (r RequestCreator) CreateFetcher() I.Fetcher {
	if r.provider.NewFetcher != nil {
		return r.provider.NewFetcher(r.CreateFileSystem(), r.CreateExtractor(), r.Log)
	}
	return artifetcher.NewArtifetcher(r.CreateFileSystem(), r.CreateExtractor(), r.Log)
}

func (r RequestCreator) CreateExtractor() I.Extractor {
	if r.provider.NewExtractor != nil {
		return r.provider.NewExtractor(r.Log, r.CreateFileSystem())
	}
	return extractor.NewExtractor(r.Log, r.CreateFileSystem())
}

type PushRequestCreatorConstructor func(creator Creator, uuid string, request I.PostDeploymentRequest, buffer *bytes.Buffer) I.RequestCreator

func NewPushRequestCreator(creator Creator, uuid string, request I.PostDeploymentRequest, buffer *bytes.Buffer) I.RequestCreator {
	return &PushRequestCreator{
		RequestCreator: RequestCreator{
			Creator: creator,
			Buffer:  buffer,
			Log:     I.DeploymentLogger{UUID: uuid, Log: creator.GetLogger()},
		},
		Request: request,
	}
}

type PushRequestCreator struct {
	RequestCreator
	Request I.PostDeploymentRequest
}

func (r PushRequestCreator) CreateRequestProcessor() I.RequestProcessor {
	if r.provider.NewPushRequestProcessor != nil {
		return r.provider.NewPushRequestProcessor(r.Log, r.CreatePushController(), r.Request, r.Buffer)
	}
	return push.NewPushRequestProcessor(r.Log, r.CreatePushController(), r.Request, r.Buffer)
}

func (r PushRequestCreator) CreatePushController() I.PushController {
	if r.provider.NewPushController != nil {
		return r.provider.NewPushController(r.Log, r.CreateDeployer(), r.createSilentDeployer(), r.CreateEventManager(), r.createErrorFinder(), r, r.CreateAuthResolver(), r.CreateEnvResolver())
	}
	return push.NewPushController(r.Log, r.CreateDeployer(), r.createSilentDeployer(), r.CreateEventManager(), r.createErrorFinder(), r, r.CreateAuthResolver(), r.CreateEnvResolver())
}

func (r PushRequestCreator) PushManager(deployEventData structs.DeployEventData, auth I.Authorization, env structs.Environment, envVars map[string]string) I.ActionCreator {
	if r.provider.NewPushManager != nil {
		return r.provider.NewPushManager(r.Creator, r.CreateEventManager(), r.Log, r.CreateFetcher(), deployEventData, r.CreateFileSystem(), r.Request.CFContext, auth, env, envVars)
	} else {
		return push.NewPushManager(r.Creator, r.CreateEventManager(), r.Log, r.CreateFetcher(), deployEventData, r.CreateFileSystem(), r.Request.CFContext, auth, env, envVars)
	}
}

type StopRequestCreatorConstructor func(creator Creator, uuid string, request I.PutDeploymentRequest, buffer *bytes.Buffer) I.RequestCreator

func NewStopRequestCreator(creator Creator, uuid string, request I.PutDeploymentRequest, buffer *bytes.Buffer) I.RequestCreator {
	return &StopRequestCreator{
		RequestCreator: RequestCreator{
			Creator: creator,
			Buffer:  buffer,
			Log:     I.DeploymentLogger{UUID: uuid, Log: creator.GetLogger()},
		},
		Request: request,
	}
}

type StopRequestCreator struct {
	RequestCreator
	Request I.PutDeploymentRequest
}

func (r StopRequestCreator) CreateRequestProcessor() I.RequestProcessor {
	if r.provider.NewStopRequestProcessor != nil {
		return r.provider.NewStopRequestProcessor(r.Log, r.CreateStopController(), r.Request, r.Buffer)
	}
	return stop.NewStopRequestProcessor(r.Log, r.CreateStopController(), r.Request, r.Buffer)
}

func (r StopRequestCreator) CreateStopController() I.StopController {
	if r.provider.NewStopController != nil {
		return r.provider.NewStopController(r.Log, r.CreateDeployer(), r.CreateEventManager(), r.createErrorFinder(), r, r.CreateAuthResolver(), r.CreateEnvResolver())
	}
	return stop.NewStopController(r.Log, r.CreateDeployer(), r.CreateEventManager(), r.createErrorFinder(), r, r.CreateAuthResolver(), r.CreateEnvResolver())
}

func (r StopRequestCreator) StopManager(deployEventData structs.DeployEventData) I.ActionCreator {
	if r.provider.NewStopManager != nil {
		return r.provider.NewStopManager(r.Creator, r.CreateEventManager(), r.Log, deployEventData)
	} else {
		return stop.NewStopManager(r.Creator, r.CreateEventManager(), r.Log, deployEventData)
	}
}

type StartRequestCreatorConstructor func(creator Creator, uuid string, request I.PutDeploymentRequest, buffer *bytes.Buffer) I.RequestCreator

func NewStartRequestCreator(creator Creator, uuid string, request I.PutDeploymentRequest, buffer *bytes.Buffer) I.RequestCreator {
	return &StartRequestCreator{
		RequestCreator: RequestCreator{
			Creator: creator,
			Buffer:  buffer,
			Log:     I.DeploymentLogger{UUID: uuid, Log: creator.GetLogger()},
		},
		Request: request,
	}
}

type StartRequestCreator struct {
	RequestCreator
	Request I.PutDeploymentRequest
}

func (r StartRequestCreator) CreateRequestProcessor() I.RequestProcessor {
	if r.provider.NewStartRequestProcessor != nil {
		return r.provider.NewStartRequestProcessor(r.Log, r.CreateStartController(), r.Request, r.Buffer)
	}
	return start.NewStartRequestProcessor(r.Log, r.CreateStartController(), r.Request, r.Buffer)
}

func (r StartRequestCreator) CreateStartController() I.StartController {
	if r.provider.NewStartController != nil {
		return r.provider.NewStartController(r.Log, r.CreateDeployer(), r.CreateEventManager(), r.createErrorFinder(), r, r.CreateAuthResolver(), r.CreateEnvResolver())
	}
	return start.NewStartController(r.Log, r.CreateDeployer(), r.CreateEventManager(), r.createErrorFinder(), r, r.CreateAuthResolver(), r.CreateEnvResolver())
}

func (r StartRequestCreator) StartManager(deployEventData structs.DeployEventData) I.ActionCreator {
	if r.provider.NewStartManager != nil {
		return r.provider.NewStartManager(r.Creator, r.CreateEventManager(), r.Log, deployEventData)
	} else {
		return start.NewStartManager(r.Creator, r.CreateEventManager(), r.Log, deployEventData)
	}
}
