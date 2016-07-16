package common

type Logger interface {
	Printf(format string, v ...interface{})
}

type nullLogger struct{}

func (n nullLogger) Printf(_ string, _ ...interface{}) {}

func MaskLogger(logger Logger) Logger {
	if logger != nil {
		return logger
	}
	return nullLogger{}
}
