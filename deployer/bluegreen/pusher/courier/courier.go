package courier

import I "github.com/compozed/deployadactyl/interfaces"

type Courier struct {
	Executor I.Executor
}

func (c Courier) Login(api, username, password, org, space string) ([]byte, error) {
	return c.Executor.Execute("login", "-a", api, "-u", username, "-p", password, "-o", org, "-s", space, "--skip-ssl-validation")
}

func (c Courier) Delete(appName string) ([]byte, error) {
	return c.Executor.Execute("delete", appName, "-f")
}

func (c Courier) Push(appName, appLocation string) ([]byte, error) {
	return c.Executor.ExecuteInDirectory(appLocation, "push", appName)
}

func (c Courier) Rename(appName, newAppName string) ([]byte, error) {
	return c.Executor.Execute("rename", appName, newAppName)
}

func (c Courier) MapRoute(appName, domain string) ([]byte, error) {
	return c.Executor.Execute("map-route", appName, domain, "-n", appName)
}

func (c Courier) Exists(appName string) bool {
	_, err := c.Executor.Execute("app", appName)
	return err == nil
}

func (c Courier) CleanUp() error {
	return c.Executor.CleanUp()
}
