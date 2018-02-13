// Package prechecker checks that all the Cloud Foundry instances are running before a deploy.
package prechecker

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
)

// Prechecker has an eventmanager used to manage event if prechecks fail.
type Prechecker struct {
	EventManager I.EventManager
}

// AssertAllFoundationsUp will send a request to each Cloud Foundry instance and check that the response status code is 200 OK.
func (p Prechecker) AssertAllFoundationsUp(environment S.Environment) error {
	precheckerEventData := S.PrecheckerEventData{Environment: environment}

	if len(environment.Foundations) == 0 {
		precheckerEventData.Description = "no foundations configured"

		p.EventManager.Emit(I.Event{Type: "validate.foundationsUnavailable", Data: precheckerEventData})

		return NoFoundationsConfiguredError{}
	}

	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			ResponseHeaderTimeout: 15 * time.Second,
		},
	}

	for _, foundationURL := range environment.Foundations {
		resp, err := insecureClient.Get(fmt.Sprintf("%s/v2/info", foundationURL))
		if err != nil {
			return InvalidGetRequestError{foundationURL, err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			err := FoundationUnavailableError{foundationURL, resp.Status}

			precheckerEventData.Description = err.Error()

			p.EventManager.Emit(I.Event{Type: "validate.foundationsUnavailable", Data: precheckerEventData})

			return err
		}
	}

	return nil
}
