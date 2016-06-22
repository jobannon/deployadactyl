package prechecker

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/compozed/deployadactyl/config"
	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/go-errors/errors"
)

const (
	apiPathExtension        = "/v2/info"
	unavailableFoundation   = "deploy aborted, one or more CF foundations unavailable"
	noFoundationsConfigured = "no foundations configured"
	anApiEndpointFailed     = "An api endpoint failed"
)

type Prechecker struct {
	EventManager I.EventManager
}

func (p Prechecker) AssertAllFoundationsUp(environment config.Environment) error {
	var insecureClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			ResponseHeaderTimeout: 15 * time.Second,
		},
	}

	precheckerEventData := S.PrecheckerEventData{
		Environment: environment,
	}

	if len(environment.Foundations) == 0 {
		precheckerEventData.Description = noFoundationsConfigured

		p.EventManager.Emit(S.Event{Type: "validate.foundationsUnavailable", Data: precheckerEventData})
		return errors.Errorf(noFoundationsConfigured)
	}

	for _, foundationURL := range environment.Foundations {

		resp, err := insecureClient.Get(foundationURL + apiPathExtension)

		if err != nil {
			return errors.WrapPrefix(err, unavailableFoundation, 0)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			precheckerEventData.Description = unavailableFoundation

			p.EventManager.Emit(S.Event{Type: "validate.foundationsUnavailable", Data: precheckerEventData})
			return errors.Errorf("%s: %s: %s", anApiEndpointFailed, foundationURL, resp.Status)
		}
	}
	return nil
}
