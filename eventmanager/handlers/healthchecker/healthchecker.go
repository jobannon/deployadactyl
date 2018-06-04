package healthchecker

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/state/push"
)

func NewHealthChecker(oldURL, newURL, silentDeployURL, silentDeployEnvironment string, client I.Client) HealthChecker {
	return HealthChecker{
		OldURL:                  oldURL,
		NewURL:                  newURL,
		SilentDeployURL:         silentDeployURL,
		SilentDeployEnvironment: silentDeployEnvironment,
		Client:                  client,
	}
}

type HealthChecker struct {
	// OldURL is the prepend on the foundationURL to replace in order to build the
	// newly pushed application URL.
	// Eg: "api.run.pivotal"
	OldURL string

	// NewUrl is what replaces OldURL in the OnEvent function.
	// Eg: "cfapps"
	NewURL string

	//SilentDeployURL represents any other url that doesn't match cfapps
	SilentDeployURL         string
	SilentDeployEnvironment string

	Client  I.Client
	Courier I.Courier
}

func (h HealthChecker) PushFinishedEventHandler(event push.PushFinishedEvent) error {

	var (
		newFoundationURL string
		domain           string
	)

	if event.HealthCheckEndpoint == "" {
		return nil
	}

	h.Courier = event.Courier

	event.Log.Debugf("starting health check")

	if event.CFContext.Environment != h.SilentDeployEnvironment {
		newFoundationURL = strings.Replace(event.FoundationURL, h.OldURL, h.NewURL, 1)
		domain = regexp.MustCompile(fmt.Sprintf("%s.*", h.NewURL)).FindString(newFoundationURL)
	} else {
		newFoundationURL = strings.Replace(event.FoundationURL, h.OldURL, h.SilentDeployURL, 1)
		domain = regexp.MustCompile(fmt.Sprintf("%s.*", h.SilentDeployURL)).FindString(newFoundationURL)
	}

	err := h.mapTemporaryRoute(event.TempAppWithUUID, domain, event.Log)
	if err != nil {
		return err
	}

	// unmapTemporaryRoute will be called before deleteTemporaryRoute
	defer h.deleteTemporaryRoute(event.TempAppWithUUID, domain, event.Log)
	defer h.unmapTemporaryRoute(event.TempAppWithUUID, domain, event.Log)

	newFoundationURL = strings.Replace(newFoundationURL, h.NewURL, fmt.Sprintf("%s.%s", event.TempAppWithUUID, h.NewURL), 1)

	return h.Check(newFoundationURL, event.HealthCheckEndpoint, event.Log)
}

// Check takes a url and endpoint. It does an http.Get to get the response
// status and returns an error if it is not http.StatusOK.
func (h HealthChecker) Check(url, endpoint string, log I.DeploymentLogger) error {
	trimmedEndpoint := strings.TrimPrefix(endpoint, "/")

	log.Debugf("checking route %s%s", url, endpoint)

	resp, err := h.Client.Get(fmt.Sprintf("%s/%s", url, trimmedEndpoint))
	if err != nil {
		log.Error(ClientError{err})
		return ClientError{err}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Errorf("health check failed for %s/%s", url, trimmedEndpoint)
		return HealthCheckError{resp.StatusCode, endpoint, body}
	}

	log.Infof("health check successful for %s%s", url, endpoint)
	return nil
}

func (h HealthChecker) mapTemporaryRoute(tempAppWithUUID, domain string, log I.DeploymentLogger) error {
	log.Debugf("mapping temporary route %s.%s", tempAppWithUUID, domain)

	out, err := h.Courier.MapRoute(tempAppWithUUID, domain, tempAppWithUUID)
	if err != nil {
		log.Errorf("failed to map temporary route: %s", out)
		return MapRouteError{tempAppWithUUID, domain}
	}
	log.Infof("mapped temporary route %s.%s", tempAppWithUUID, domain)

	return nil
}

func (h HealthChecker) deleteTemporaryRoute(tempAppWithUUID, domain string, log I.DeploymentLogger) error {
	log.Debugf("deleting temporary route %s.%s", tempAppWithUUID, domain)

	out, err := h.Courier.DeleteRoute(domain, tempAppWithUUID)
	if err != nil {
		log.Errorf("failed to delete temporary route: %s", out)
		return DeleteRouteError{tempAppWithUUID, domain}
	}

	log.Infof("deleted temporary route %s.%s", tempAppWithUUID, domain)

	return nil
}

func (h HealthChecker) unmapTemporaryRoute(tempAppWithUUID, domain string, log I.DeploymentLogger) {
	log.Debugf("unmapping temporary route %s.%s", tempAppWithUUID, domain)

	out, err := h.Courier.UnmapRoute(tempAppWithUUID, domain, tempAppWithUUID)
	if err != nil {
		log.Errorf("failed to unmap temporary route: %s", out)
	} else {
		log.Infof("unmapped temporary route %s.%s", tempAppWithUUID, domain)
	}

	log.Infof("finished health check")
}
