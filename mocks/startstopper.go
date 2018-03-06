package mocks

type StartStopper struct {
	LoginCall struct {
		Received struct {
			FoundationURL string
		}
		Returns struct {
			Error error
		}
	}
	StartCall struct {
		Received struct {
			AppName       string
			FoundationURL string
		}
		Returns struct {
			Error error
		}
	}
	StopCall struct {
		Received struct {
			AppName       string
			FoundationURL string
		}
		Write   string
		Returns struct {
			Error error
		}
	}
}

func (s *StartStopper) Login(foundationURL string) error {
	s.LoginCall.Received.FoundationURL = foundationURL

	return s.LoginCall.Returns.Error
}

func (s *StartStopper) Start(appName, foundationURL string) error {
	s.StartCall.Received.AppName = appName
	s.StartCall.Received.FoundationURL = foundationURL

	return s.StartCall.Returns.Error
}

func (s *StartStopper) Stop(appName, foundationURL string) error {
	s.StopCall.Received.AppName = appName
	s.StopCall.Received.FoundationURL = foundationURL

	return s.StopCall.Returns.Error
}
