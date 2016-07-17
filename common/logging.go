package common

type Logger interface {
	Printf(format string, v ...interface{})
}

type NullLogger struct{}

func (n NullLogger) Printf(_ string, _ ...interface{}) {}

func MaskLogger(logger Logger) Logger {
	if logger != nil {
		return logger
	}
	return NullLogger{}
}
