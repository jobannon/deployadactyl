// Package logger is used for logging.
package logger

import (
	"io"

	"github.com/op/go-logging"
)

// DefaultLogger returns a logging.Logger with a specific logging format.
func DefaultLogger(out io.Writer, level logging.Level, module string) *logging.Logger {

	var log = logging.MustGetLogger(module)

	var format = logging.MustStringFormatter(
		`%{color}%{time:2006/01/02 15:04:05} %{level:.4s} ▶ (%{shortfunc}) %{color:reset}%{message}`,
	)

	backend := logging.NewLogBackend(out, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	backendLeveledFormatter := logging.AddModuleLevel(backendFormatter)
	backendLeveledFormatter.SetLevel(level, module)
	logging.SetBackend(backendLeveledFormatter)

	return log
}
