package structs

import "github.com/compozed/deployadactyl/config"

// PrecheckerEventData has Environment variables and a description.
type PrecheckerEventData struct {
	Environment config.Environment
	Description string
}
