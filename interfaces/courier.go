package interfaces

type Courier interface {
	Login(api, username, password, org, space string, skipSSL bool) ([]byte, error)
	Delete(appName string) ([]byte, error)
	Push(appName, appLocation string) ([]byte, error)
	Rename(oldName, newName string) ([]byte, error)
	MapRoute(appName, domain string) ([]byte, error)
	Exists(appName string) bool
	CleanUp() error
}
