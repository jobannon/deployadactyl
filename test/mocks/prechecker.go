package mocks

import "github.com/compozed/deployadactyl/config"

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

func (p *Prechecker) AssertAllFoundationsUp(environment config.Environment) error {
	p.AssertAllFoundationsUpCall.Received.Environment = environment

	return p.AssertAllFoundationsUpCall.Returns.Error
}
