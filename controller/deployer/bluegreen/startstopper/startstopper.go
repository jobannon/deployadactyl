package startstopper

import (
	C "github.com/compozed/deployadactyl/constants"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/pusher"
	I "github.com/compozed/deployadactyl/interfaces"
	"io"
)

type StartStopper struct {
	Courier       I.Courier
	CFContext     I.CFContext
	Authorization I.Authorization
	EventManager  I.EventManager
	Response      io.ReadWriter
	Log           I.Logger
}

type StopperAction struct {
	Stopper       I.StartStopper
	FoundationURL string
	AppName       string
}

func (s StopperAction) Initially() error {
	return s.Stopper.Login(s.FoundationURL)
}

func (s StopperAction) Execute() error {
	return s.Stopper.Stop(s.AppName, s.FoundationURL)
}

func (s StopperAction) Verify() error {
	return nil
}

func (s StopperAction) Success() error {
	return nil
}

func (s StopperAction) Undo() error {
	return s.Stopper.Start(s.AppName, s.FoundationURL)
}

func (s StopperAction) Finally() error {
	return nil
}

// Login will login to a Cloud Foundry instance.
func (s StartStopper) Login(foundationURL string) error {
	s.Log.Debugf(
		`logging into cloud foundry with parameters:
		foundation URL: %+v
		username: %+v
		org: %+v
		space: %+v`,
		foundationURL, s.Authorization.Username, s.CFContext.Organization, s.CFContext.Space,
	)

	output, err := s.Courier.Login(
		foundationURL,
		s.Authorization.Username,
		s.Authorization.Password,
		s.CFContext.Organization,
		s.CFContext.Space,
		s.CFContext.SkipSSL,
	)
	s.Response.Write(output)
	if err != nil {
		s.Log.Errorf("could not login to %s", foundationURL)
		return pusher.LoginError{foundationURL, output}
	}

	s.Log.Infof("logged into cloud foundry %s", foundationURL)

	return nil
}

func (s StartStopper) Start(appName, foundationURL string) error {

	if s.Courier.Exists(appName) != true {
		return ExistsError{ApplicationName: appName}
	}

	s.Log.Infof("starting app %s", appName)

	output, err := s.Courier.Start(appName)
	if err != nil {
		return StartError{ApplicationName: appName, Out: output}
	}
	s.Response.Write(output)

	s.Log.Debugf("emitting a %s event", C.StartFinishedEvent)
	startData := I.StartStopEventData{
		FoundationURL: foundationURL,
		Context:       s.CFContext,
		Courier:       s.Courier,
		Response:      s.Response,
	}

	err = s.EventManager.Emit(I.Event{Type: C.StartFinishedEvent, Data: startData})

	s.Log.Infof("successfully started app %s", appName)

	return nil
}

func (s StartStopper) Stop(appName, foundationURL string) error {

	if s.Courier.Exists(appName) != true {
		return ExistsError{ApplicationName: appName}
	}

	s.Log.Infof("stopping app %s", appName)

	output, err := s.Courier.Stop(appName)
	if err != nil {
		return StopError{ApplicationName: appName, Out: output}
	}
	s.Response.Write(output)

	s.Log.Debugf("emitting a %s event", C.StopFinishedEvent)
	stopData := I.StartStopEventData{
		FoundationURL: foundationURL,
		Context:       s.CFContext,
		Courier:       s.Courier,
		Response:      s.Response,
	}

	err = s.EventManager.Emit(I.Event{Type: C.StopFinishedEvent, Data: stopData})

	s.Log.Infof("successfully stopped app %s", appName)

	return nil
}
