package logger

import (
	"io"

	"github.com/op/go-logging"
)

func DefaultLogger(out io.Writer, level logging.Level, module string) *logging.Logger {

	var log = logging.MustGetLogger(module)

	var format = logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)

	backend := logging.NewLogBackend(out, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	backendLeveledFormatter := logging.AddModuleLevel(backendFormatter)
	backendLeveledFormatter.SetLevel(level, module)
	logging.SetBackend(backendLeveledFormatter)

	return log
}
