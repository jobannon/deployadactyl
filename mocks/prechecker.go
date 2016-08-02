package mocks

import "github.com/compozed/deployadactyl/config"

// Prechecker handmade mock for tests.
type Prechecker struct {
	AssertAllFoundationsUpCall struct {
		Received struct {
			Environment config.Environment
		}
		Returns struct {
			Error error
		}
	}
}

// AssertAllFoundationsUp mock method.
func (p *Prechecker) AssertAllFoundationsUp(environment config.Environment) error {
	p.AssertAllFoundationsUpCall.Received.Environment = environment

	return p.AssertAllFoundationsUpCall.Returns.Error
}
