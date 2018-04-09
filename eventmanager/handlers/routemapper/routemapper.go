package routemapper

import (
	"strings"

	"github.com/cloudfoundry-incubator/candiedyaml"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/state/push"
	"github.com/spf13/afero"
)

// RouteMapper will map additional routes to an application at
// deploy time if they are specified in the manifest.
type RouteMapper struct {
	Courier    I.Courier
	FileSystem *afero.Afero
	Log        I.Logger
}

type manifest struct {
	Applications []application
}

type application struct {
	CustomRoutes []route `yaml:"custom-routes"`
}

type route struct {
	Route string
}

func (r RouteMapper) PushFinishedEventHandler(event push.PushFinishedEvent) error {
	r.Log.Debugf("starting route mapper")

	r.Courier = event.Courier

	manifestBytes, err := r.readManifest(event.Manifest, event.AppPath)
	if err != nil || manifestBytes == nil {
		return err
	}

	m := &manifest{}

	r.Log.Debugf("looking for routes in the manifest")
	err = candiedyaml.Unmarshal(manifestBytes, m)
	if err != nil {
		r.Log.Errorf("failed to parse manifest: %s", err.Error())
		return err
	}

	if m.Applications == nil || len(m.Applications[0].CustomRoutes) == 0 {
		r.Log.Info("finished mapping routes: no routes to map")
		return nil
	}

	r.Log.Infof("found %d routes in the manifest", len(m.Applications[0].CustomRoutes))

	domains, _ := r.Courier.Domains()

	r.Log.Debugf("mapping routes to %s", event.TempAppWithUUID)
	return r.routeMapper(m, event.TempAppWithUUID, domains, event.CFContext.Application)
}

func isRouteADomainInTheFoundation(route string, domains []string) bool {
	for _, domain := range domains {
		if route == domain {
			return true
		}
	}
	return false
}

func (r RouteMapper) readManifest(manifest, appPath string) ([]byte, error) {
	var (
		manifestBytes []byte
		err           error
	)
	if manifest != "" {
		manifestBytes = []byte(manifest)
		return manifestBytes, nil
	} else if appPath != "" {
		manifestBytes, err = r.FileSystem.ReadFile(appPath + "/manifest.yml")
		if err != nil {
			r.Log.Errorf("failed to read manifest file: %s", err.Error())
			return nil, ReadFileError{err}
		}
		return manifestBytes, nil
	} else {
		r.Log.Info("finished mapping routes: no manifest found")
		return nil, nil
	}
}

// routeMapper is used to decide how to map an applications routes that are given to it from the manifest.
// if the route does not include appname or path it will map the given domain to the given application by default
// if the route has an app name it will remove the app name so it can map it with the given domain
// if the route has an app name and a path it will remove the app name so it can map it with the given domain and the path as well
func (r RouteMapper) routeMapper(manifest *manifest, tempAppWithUUID string, domains []string, appName string) error {

	for _, route := range manifest.Applications[0].CustomRoutes {
		var domainAndPath []string

		appNameAndDomain := strings.SplitN(route.Route, ".", 2)

		if len(appNameAndDomain) >= 2 {
			domainAndPath = strings.SplitN(appNameAndDomain[1], "/", 2)
		}

		if isRouteADomainInTheFoundation(route.Route, domains) {
			output, err := r.Courier.MapRoute(tempAppWithUUID, route.Route, appName)
			if err != nil {
				r.Log.Errorf("failed to map route: %s: %s", route.Route, string(output))
				return MapRouteError{route.Route, output}
			}
		} else if len(appNameAndDomain) >= 2 && isRouteADomainInTheFoundation(appNameAndDomain[1], domains) {
			output, err := r.Courier.MapRoute(tempAppWithUUID, appNameAndDomain[1], appNameAndDomain[0])
			if err != nil {
				r.Log.Errorf("failed to map route: %s: %s", route.Route, string(output))
				return MapRouteError{route.Route, output}
			}
		} else if domainAndPath != nil && isRouteADomainInTheFoundation(domainAndPath[0], domains) {
			output, err := r.Courier.MapRouteWithPath(tempAppWithUUID, domainAndPath[0], appNameAndDomain[0], domainAndPath[1])
			if err != nil {
				r.Log.Error(MapRouteError{route.Route, output})
				return MapRouteError{route.Route, output}
			}
		} else {
			return InvalidRouteError{route.Route}
		}

		r.Log.Infof("mapped route %s to %s", route.Route, tempAppWithUUID)
	}

	r.Log.Info("route mapping successful: finished mapping routes")
	return nil
}
