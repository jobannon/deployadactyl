// Package logger is used for logging.
package logger

import (
	"io"

	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/op/go-logging"
)

// DefaultLogger returns a logging.Logger with a specific logging format.
func DefaultLogger(out io.Writer, level logging.Level, module string) I.Logger {

	var log = logging.MustGetLogger(module)

	var format = logging.MustStringFormatter(
		`%{time:2006/01/02 15:04:05} %{level:.4s} ▶ (%{shortfunc}) %{message}`,
	)

	backend := logging.NewLogBackend(out, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	backendLeveledFormatter := logging.AddModuleLevel(backendFormatter)
	backendLeveledFormatter.SetLevel(level, module)
	logging.SetBackend(backendLeveledFormatter)

	return log
}

type DeploymentLogger struct {
	Log  I.Logger
	UUID string
}

func (l DeploymentLogger) Error(args ...interface{}) {
	args = append([]interface{}{l.UUID}, args...)
	l.Log.Error(args...)
}

func (l DeploymentLogger) Errorf(str string, args ...interface{}) {
	l.Log.Errorf(l.UUID+" "+str, args...)
}

func (l DeploymentLogger) Debug(args ...interface{}) {
	args = append([]interface{}{l.UUID}, args...)
	l.Log.Debug(args...)
}

func (l DeploymentLogger) Debugf(str string, args ...interface{}) {
	l.Log.Debugf(l.UUID+" "+str, args...)
}

func (l DeploymentLogger) Info(args ...interface{}) {
	args = append([]interface{}{l.UUID}, args...)
	l.Log.Info(args...)
}

func (l DeploymentLogger) Infof(str string, args ...interface{}) {
	l.Log.Infof(l.UUID+" "+str, args...)
}

func (l DeploymentLogger) Fatal(args ...interface{}) {
	args = append([]interface{}{l.UUID}, args...)
	l.Log.Fatal(args...)
}