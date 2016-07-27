package mocks

type Executor struct {
	ExecuteCall struct {
		Received struct {
			Args []string
		}
		Returns struct {
			Output []byte
			Error  error
		}
	}

	ExecuteInDirectoryCall struct {
		Received struct {
			AppLocation string
			Args        []string
		}
		Returns struct {
			Output []byte
			Error  error
		}
	}

	CleanUpCall struct {
		Returns struct {
			Error error
		}
	}
}

func (e *Executor) Execute(args ...string) ([]byte, error) {
	e.ExecuteCall.Received.Args = args

	return e.ExecuteCall.Returns.Output, e.ExecuteCall.Returns.Error
}

func (e *Executor) ExecuteInDirectory(appLocation string, args ...string) ([]byte, error) {
	e.ExecuteInDirectoryCall.Received.AppLocation = appLocation
	e.ExecuteInDirectoryCall.Received.Args = args

	return e.ExecuteInDirectoryCall.Returns.Output, e.ExecuteInDirectoryCall.Returns.Error
}

func (e *Executor) CleanUp() error {
	return e.CleanUpCall.Returns.Error
}
