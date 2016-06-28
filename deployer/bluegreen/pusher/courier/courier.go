// Package courier interfaces with the Executor to run specific Cloud Foundry CLI commands.
package courier

import I "github.com/compozed/deployadactyl/interfaces"

type Courier struct {
	Executor I.Executor
}

<<<<<<< 509e5bea158cd9696fb413ab07a9dd7eab72afc1
// Login runs the Cloud Foundry login command.
//
// Returns the combined standard output and standard error.
func (c Courier) Login(api, username, password, org, space string) ([]byte, error) {
	return c.Executor.Execute("login", "-a", api, "-u", username, "-p", password, "-o", org, "-s", space, "--skip-ssl-validation")
=======
func (c Courier) Login(api, username, password, org, space string, skipSSL bool) ([]byte, error) {
	var s string
	if skipSSL {
		s = "--skip-ssl-validation"
	}

	return c.Executor.Execute("login", "-a", api, "-u", username, "-p", password, "-o", org, "-s", space, s)
>>>>>>> Make skip-ssl-validation optional
}

// Delete runs the Cloud Foundry delete command.
//
// Returns the combined standard output and standard error.
func (c Courier) Delete(appName string) ([]byte, error) {
	return c.Executor.Execute("delete", appName, "-f")
}

// Push runs the Cloud Foundry push command.
//
// Returns the combined standard output and standard error.
func (c Courier) Push(appName, appLocation string) ([]byte, error) {
	return c.Executor.ExecuteInDirectory(appLocation, "push", appName)
}

// Rename runs the Cloud Foundry rename command.
//
// Returns the combined standard output and standard error.
func (c Courier) Rename(appName, newAppName string) ([]byte, error) {
	return c.Executor.Execute("rename", appName, newAppName)
}

// MapRoute runs the Cloud Foundry map-route command.
//
// Returns the combined standard output and standard error.
func (c Courier) MapRoute(appName, domain string) ([]byte, error) {
	return c.Executor.Execute("map-route", appName, domain, "-n", appName)
}

// Exists checks to see whether the app name exists already.
//
// Returns true if the app exists.
func (c Courier) Exists(appName string) bool {
	_, err := c.Executor.Execute("app", appName)
	return err == nil
}

// CleanUp removes the temporary directory created by the Executor.
func (c Courier) CleanUp() error {
	return c.Executor.CleanUp()
}
