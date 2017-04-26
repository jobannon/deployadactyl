package mocks

import (
	"fmt"
	"io"
)

// Pusher handmade mock for tests.
type Pusher struct {
	Response io.ReadWriter

	LoginCall struct {
		Received struct {
			FoundationURL string
		}
		Write struct {
			Output string
		}
		Returns struct {
			Error error
		}
	}

	PushCall struct {
		Received struct {
			AppPath       string
			FoundationURL string
			AppExists     bool
			Out           io.ReadWriter
		}
		Write struct {
			Output string
		}
		Returns struct {
			Error error
		}
	}

	UndoPushCall struct {
		Received struct {
			AppExists bool
		}
		Returns struct {
			Error error
		}
	}

	FinishPushCall struct {
		Returns struct {
			Error error
		}
	}

	CleanUpCall struct {
		Returns struct {
			Error error
		}
	}
}

// Login mock method.
func (p *Pusher) Login(foundationURL string) error {
	p.LoginCall.Received.FoundationURL = foundationURL

	fmt.Fprint(p.Response, p.LoginCall.Write.Output)

	return p.LoginCall.Returns.Error
}

// Push mock method.
func (p *Pusher) Push(appPath, foundationURL string) error {
	p.PushCall.Received.AppPath = appPath
	p.PushCall.Received.FoundationURL = foundationURL

	fmt.Fprint(p.Response, p.PushCall.Write.Output)

	return p.PushCall.Returns.Error
}

// FinishPush mock method.
func (p *Pusher) FinishPush() error {
	return p.FinishPushCall.Returns.Error
}

// UndoPush mock method.
func (p *Pusher) UndoPush() error {
	return p.UndoPushCall.Returns.Error
}

// CleanUp mock method.
func (p *Pusher) CleanUp() error {
	return p.CleanUpCall.Returns.Error
}
