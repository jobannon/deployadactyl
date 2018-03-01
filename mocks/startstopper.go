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
	StopCall struct {
		Received struct {
			AppName       string
			FoundationURL string
		}
		Returns struct {
			Error error
		}
	}
}

func (s *StartStopper) Login(foundationURL string) error {
	s.LoginCall.Received.FoundationURL = foundationURL

	return s.LoginCall.Returns.Error
}

func (s *StartStopper) Stop(appName, foundationURL string) error {
	s.StopCall.Received.AppName = appName
	s.StopCall.Received.FoundationURL = foundationURL

	return s.StopCall.Returns.Error
}
