package structs

import "github.com/compozed/deployadactyl/config"

type PrecheckerEventData struct {
	Environment config.Environment
	Description string
}
