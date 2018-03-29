package startstopper

/*
import (
	C "github.com/compozed/deployadactyl/constants"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/pusher"
	I "github.com/compozed/deployadactyl/interfaces"
	"io"
)

type Starter struct {
	Courier       I.Courier
	CFContext     I.CFContext
	Authorization I.Authorization
	EventManager  I.EventManager
	Response      io.ReadWriter
	Log           I.Logger
	FoundationURL string
	AppName       string
}

func (s Starter) Verify() error {
	return nil
}

func (s Starter) Success() error {
	return nil
}

func (s Starter) Finally() error {
	return nil
}

// Login will login to a Cloud Foundry instance.
func (s Starter) Initially() error {
	s.Log.Debugf(
		`logging into cloud foundry with parameters:
		foundation URL: %+v
		username: %+v
		org: %+v
		space: %+v`,
		s.FoundationURL, s.Authorization.Username, s.CFContext.Organization, s.CFContext.Space,
	)

	output, err := s.Courier.Login(
		s.FoundationURL,
		s.Authorization.Username,
		s.Authorization.Password,
		s.CFContext.Organization,
		s.CFContext.Space,
		s.CFContext.SkipSSL,
	)
	s.Response.Write(output)
	if err != nil {
		s.Log.Errorf("could not login to %s", s.FoundationURL)
		return pusher.LoginError{s.FoundationURL, output}
	}

	s.Log.Infof("logged into cloud foundry %s", s.FoundationURL)

	return nil
}

func (s Starter) Execute() error {

	if s.Courier.Exists(s.AppName) != true {
		return ExistsError{ApplicationName: s.AppName}
	}

	s.Log.Infof("starting app %s", s.AppName)

	output, err := s.Courier.Start(s.AppName)
	if err != nil {
		return StartError{ApplicationName: s.AppName, Out: output}
	}
	s.Response.Write(output)

	s.Log.Debugf("emitting a %s event", C.StartFinishedEvent)
	startData := I.StartStopEventData{
		FoundationURL: s.FoundationURL,
		Context:       s.CFContext,
		Courier:       s.Courier,
		Response:      s.Response,
	}

	err = s.EventManager.Emit(I.Event{Type: C.StartFinishedEvent, Data: startData})

	s.Log.Infof("successfully started app %s", s.AppName)

	return nil
}

func (s Starter) Undo() error {

	if s.Courier.Exists(s.AppName) != true {
		return ExistsError{ApplicationName: s.AppName}
	}

	s.Log.Infof("stopping app %s", s.AppName)

	output, err := s.Courier.Stop(s.AppName)
	if err != nil {
		return StopError{ApplicationName: s.AppName, Out: output}
	}
	s.Response.Write(output)

	s.Log.Debugf("emitting a %s event", C.StopFinishedEvent)
	stopData := I.StartStopEventData{
		FoundationURL: s.FoundationURL,
		Context:       s.CFContext,
		Courier:       s.Courier,
		Response:      s.Response,
	}

	err = s.EventManager.Emit(I.Event{Type: C.StopFinishedEvent, Data: stopData})

	s.Log.Infof("successfully stopped app %s", s.AppName)

	return nil
}
*/
