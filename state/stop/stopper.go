package stop

import (
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/state"
	"io"
)

type Stopper struct {
	Courier       I.Courier
	CFContext     I.CFContext
	Authorization I.Authorization
	EventManager  I.EventManager
	Response      io.ReadWriter
	Log           I.Logger
	FoundationURL string
	AppName       string
}

func (s Stopper) Verify() error {
	return nil
}

func (s Stopper) Success() error {
	return nil
}

func (s Stopper) Finally() error {
	return nil
}

// Login will login to a Cloud Foundry instance.
func (s Stopper) Initially() error {
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
		return state.LoginError{s.FoundationURL, output}
	}

	s.Log.Infof("logged into cloud foundry %s", s.FoundationURL)

	return nil
}

func (s Stopper) Execute() error {

	if s.Courier.Exists(s.AppName) != true {
		s.Log.Errorf("failed to stop app on foundation %s: application doesn't exist", s.FoundationURL)
		return state.ExistsError{ApplicationName: s.AppName}
	}

	s.Log.Infof("stopping app %s", s.AppName)

	output, err := s.Courier.Stop(s.AppName)
	if err != nil {
		s.Log.Errorf("failed to stop app on foundation %s: %s", s.FoundationURL, err.Error())
		return state.StopError{ApplicationName: s.AppName, Out: output}
	}
	s.Response.Write(output)

	s.Log.Infof("successfully stopped app %s", s.AppName)

	return nil
}

func (s Stopper) Undo() error {

	if s.Courier.Exists(s.AppName) != true {
		return nil
	}

	s.Log.Infof("starting app %s", s.AppName)

	output, err := s.Courier.Start(s.AppName)
	if err != nil {
		return state.StartError{ApplicationName: s.AppName, Out: output}
	}
	s.Response.Write(output)

	s.Log.Infof("successfully restarted app %s", s.AppName)

	return nil
}
