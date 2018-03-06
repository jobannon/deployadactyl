package statemanager

import (
	"fmt"
	"io"
	"net/http"

	"github.com/compozed/deployadactyl/config"
	C "github.com/compozed/deployadactyl/constants"
	E "github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	S "github.com/compozed/deployadactyl/structs"
	"regexp"
)

const (
	stopOutput = `Stop Parameters:
Username:     %s,
Environment:  %s,
Org:          %s,
Space:        %s,
AppName:      %s`
)

type StateManager struct {
	Prechecker   interfaces.Prechecker
	Config       config.Config
	Log          interfaces.Logger
	EventManager interfaces.EventManager
	BlueGreener  interfaces.BlueGreener
}

//func (s *StateManager) Start(environment environment.Environment) {
// errors = []error
// foreach environment.Foundations as foundation
//     foundation.Login(user)
//     state = foundation.State(app)
//     if (state == stopped)
//         append(errors, foundation.Start(app))
// if errors.length > 0
//     rollback
// return success
//}

func (s *StateManager) Stop(context interfaces.CFContext, uuid string, auth interfaces.Authorization, response io.ReadWriter) (statusCode int, deploymentInfo *S.DeploymentInfo, err error) {

	environments := s.Config.Environments
	deploymentInfo = &S.DeploymentInfo{}
	deploymentLogger := logger.DeploymentLogger{s.Log, uuid}

	e, ok := environments[context.Environment]
	if !ok {
		fmt.Fprintln(response, E.EnvironmentNotFoundError{context.Environment}.Error())
		return http.StatusInternalServerError, deploymentInfo, E.EnvironmentNotFoundError{context.Environment}
	}

	err = s.Prechecker.AssertAllFoundationsUp(environments[context.Environment])
	if err != nil {
		deploymentLogger.Error(err)
		return http.StatusInternalServerError, deploymentInfo, err
	}

	deploymentInfo.Username = auth.Username
	deploymentInfo.Password = auth.Password
	deploymentInfo.Environment = context.Environment
	deploymentInfo.Org = context.Organization
	deploymentInfo.Space = context.Space
	deploymentInfo.AppName = context.Application
	deploymentInfo.UUID = uuid
	deploymentInfo.SkipSSL = environments[context.Environment].SkipSSL

	deploymentMessage := fmt.Sprintf(stopOutput, deploymentInfo.Username, deploymentInfo.Environment, deploymentInfo.Org, deploymentInfo.Space, deploymentInfo.AppName)
	deploymentLogger.Info(deploymentMessage)
	fmt.Fprintln(response, deploymentMessage)

	stopEventData := S.StopEventData{Response: response, DeploymentInfo: deploymentInfo}

	defer emitStopFinish(s, stopEventData, response, &err, &statusCode, deploymentLogger)
	defer emitStopSuccess(s, stopEventData, response, &err, &statusCode, deploymentLogger)

	deploymentLogger.Debugf("emitting a %s event", C.StopStartEvent)
	err = s.EventManager.Emit(interfaces.Event{Type: C.StopStartEvent, Data: stopEventData})
	if err != nil {
		deploymentLogger.Error(err)
		err = &bluegreen.StartStopError{Err: err}
		return http.StatusInternalServerError, deploymentInfo, E.EventError{Type: C.StopStartEvent, Err: err}
	}

	err = s.BlueGreener.Stop(e, *deploymentInfo, response)

	if err != nil {
		if matched, _ := regexp.MatchString("login failed", err.Error()); matched {
			return http.StatusBadRequest, deploymentInfo, err
		}
		return http.StatusInternalServerError, deploymentInfo, err
	}

	return http.StatusOK, deploymentInfo, nil
}

func emitStopFinish(s *StateManager, stopEventData S.StopEventData, response io.ReadWriter, err *error, statusCode *int, deploymentLogger logger.DeploymentLogger) {
	deploymentLogger.Debugf("emitting a %s event", C.StopFinishEvent)

	finishErr := s.EventManager.Emit(interfaces.Event{Type: C.StopFinishEvent, Data: stopEventData})
	if finishErr != nil {
		fmt.Fprintln(response, finishErr)
		*err = bluegreen.FinishStopError{Err: fmt.Errorf("%s: %s", *err, E.EventError{C.StopFinishEvent, finishErr})}
		*statusCode = http.StatusInternalServerError
	}
}

func emitStopSuccess(s *StateManager, stopEventData S.StopEventData, response io.ReadWriter, err *error, statusCode *int, deploymentLogger logger.DeploymentLogger) {
	stopEvent := interfaces.Event{Type: C.StopSuccessEvent, Data: stopEventData}
	if *err != nil {
		fmt.Fprintln(response, *err)

		stopEvent.Type = C.StopFailureEvent
		stopEvent.Error = *err
	}

	deploymentLogger.Debug(fmt.Sprintf("emitting a %s event", stopEvent.Type))
	eventErr := s.EventManager.Emit(stopEvent)
	if eventErr != nil {
		deploymentLogger.Errorf("an error occurred when emitting a %s event: %s", stopEvent.Type, eventErr)
		fmt.Fprintln(response, eventErr)
	}
}
