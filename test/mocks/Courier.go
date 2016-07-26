package mocks

type Courier struct {
	LoginCall struct {
		Received struct {
			API      string
			Username string
			Password string
			Org      string
			Space    string
			SkipSSL  bool
		}
		Returns struct {
			Output []byte
			Error  error
		}
	}

	DeleteCall struct {
		Received struct {
			AppName string
		}
		Returns struct {
			Output []byte
			Error  error
		}
	}

	PushCall struct {
		Received struct {
			AppName     string
			AppLocation string
		}
		Returns struct {
			Output []byte
			Error  error
		}
	}

	RenameCall struct {
		Received struct {
			AppName    string
			NewAppName string
		}
		Returns struct {
			Output []byte
			Error  error
		}
	}

	MapRouteCall struct {
		Received struct {
			AppName string
			Domain  string
		}
		Returns struct {
			Output []byte
			Error  error
		}
	}

	ExistsCall struct {
		Received struct {
			AppName string
		}
		Returns struct {
			Bool bool
		}
	}

	CleanUpCall struct {
		Returns struct {
			Error error
		}
	}
}

func (c *Courier) Login(api, username, password, org, space string, skipSSL bool) ([]byte, error) {
	c.LoginCall.Received.API = api
	c.LoginCall.Received.Username = username
	c.LoginCall.Received.Password = password
	c.LoginCall.Received.Org = org
	c.LoginCall.Received.Space = space
	c.LoginCall.Received.SkipSSL = skipSSL

	return c.LoginCall.Returns.Output, c.LoginCall.Returns.Error
}

func (c *Courier) Delete(appName string) ([]byte, error) {
	c.DeleteCall.Received.AppName = appName

	return c.DeleteCall.Returns.Output, c.DeleteCall.Returns.Error
}

func (c *Courier) Push(appName, appLocation string) ([]byte, error) {
	c.PushCall.Received.AppName = appName
	c.PushCall.Received.AppLocation = appLocation

	return c.PushCall.Returns.Output, c.PushCall.Returns.Error
}

func (c *Courier) Rename(appName, newAppName string) ([]byte, error) {
	c.RenameCall.Received.AppName = appName
	c.RenameCall.Received.NewAppName = newAppName

	return c.RenameCall.Returns.Output, c.RenameCall.Returns.Error
}

func (c *Courier) MapRoute(appName, domain string) ([]byte, error) {
	c.MapRouteCall.Received.AppName = appName
	c.MapRouteCall.Received.Domain = domain

	return c.MapRouteCall.Returns.Output, c.MapRouteCall.Returns.Error
}

func (c *Courier) Exists(appName string) bool {
	c.ExistsCall.Received.AppName = appName
	return c.ExistsCall.Returns.Bool
}

func (c *Courier) CleanUp() error {
	return c.CleanUpCall.Returns.Error
}
