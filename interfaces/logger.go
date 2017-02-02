package interfaces

type Logger interface {
	Error(...interface{})
	Errorf(string, ...interface{})
	Debug(...interface{})
	Debugf(string, ...interface{})
	Info(...interface{})
	Infof(string, ...interface{})
	Fatal(...interface{})
}
