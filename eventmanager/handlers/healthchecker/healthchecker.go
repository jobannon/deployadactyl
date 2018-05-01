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

// HealthChecker will check an endpoint for a http.StatusOK
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
	Log     I.Logger
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

	h.Log.Debugf("starting health check")

	if event.CFContext.Environment != h.SilentDeployEnvironment {
		newFoundationURL = strings.Replace(event.FoundationURL, h.OldURL, h.NewURL, 1)
		domain = regexp.MustCompile(fmt.Sprintf("%s.*", h.NewURL)).FindString(newFoundationURL)
	} else {
		newFoundationURL = strings.Replace(event.FoundationURL, h.OldURL, h.SilentDeployURL, 1)
		domain = regexp.MustCompile(fmt.Sprintf("%s.*", h.SilentDeployURL)).FindString(newFoundationURL)
	}

	err := h.mapTemporaryRoute(event.TempAppWithUUID, domain)
	if err != nil {
		return err
	}

	// unmapTemporaryRoute will be called before deleteTemporaryRoute
	defer h.deleteTemporaryRoute(event.TempAppWithUUID, domain)
	defer h.unmapTemporaryRoute(event.TempAppWithUUID, domain)

	newFoundationURL = strings.Replace(newFoundationURL, h.NewURL, fmt.Sprintf("%s.%s", event.TempAppWithUUID, h.NewURL), 1)

	return h.Check(newFoundationURL, event.HealthCheckEndpoint)
}

// Check takes a url and endpoint. It does an http.Get to get the response
// status and returns an error if it is not http.StatusOK.
func (h HealthChecker) Check(url, endpoint string) error {
	trimmedEndpoint := strings.TrimPrefix(endpoint, "/")

	h.Log.Debugf("checking route %s%s", url, endpoint)

	resp, err := h.Client.Get(fmt.Sprintf("%s/%s", url, trimmedEndpoint))
	if err != nil {
		h.Log.Error(ClientError{err})
		return ClientError{err}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		h.Log.Errorf("health check failed for %s/%s", url, trimmedEndpoint)
		return HealthCheckError{resp.StatusCode, endpoint, body}
	}

	h.Log.Infof("health check successful for %s%s", url, endpoint)
	return nil
}

func (h HealthChecker) mapTemporaryRoute(tempAppWithUUID, domain string) error {
	h.Log.Debugf("mapping temporary route %s.%s", tempAppWithUUID, domain)

	out, err := h.Courier.MapRoute(tempAppWithUUID, domain, tempAppWithUUID)
	if err != nil {
		h.Log.Errorf("failed to map temporary route: %s", out)
		return MapRouteError{tempAppWithUUID, domain}
	}
	h.Log.Infof("mapped temporary route %s.%s", tempAppWithUUID, domain)

	return nil
}

func (h HealthChecker) deleteTemporaryRoute(tempAppWithUUID, domain string) error {
	h.Log.Debugf("deleting temporary route %s.%s", tempAppWithUUID, domain)

	out, err := h.Courier.DeleteRoute(domain, tempAppWithUUID)
	if err != nil {
		h.Log.Errorf("failed to delete temporary route: %s", out)
		return DeleteRouteError{tempAppWithUUID, domain}
	}

	h.Log.Infof("deleted temporary route %s.%s", tempAppWithUUID, domain)

	return nil
}

func (h HealthChecker) unmapTemporaryRoute(tempAppWithUUID, domain string) {
	h.Log.Debugf("unmapping temporary route %s.%s", tempAppWithUUID, domain)

	out, err := h.Courier.UnmapRoute(tempAppWithUUID, domain, tempAppWithUUID)
	if err != nil {
		h.Log.Errorf("failed to unmap temporary route: %s", out)
	} else {
		h.Log.Infof("unmapped temporary route %s.%s", tempAppWithUUID, domain)
	}

	h.Log.Infof("finished health check")
}
