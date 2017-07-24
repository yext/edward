package common

// Logger provides a simple logging interface with minimal print function(s)
type Logger interface {
	Printf(format string, v ...interface{})
}

// NullLogger implements the Logger interface with no-op functions
type NullLogger struct{}

var _ Logger = NullLogger{}

// Printf is a no-op print function
func (n NullLogger) Printf(_ string, _ ...interface{}) {}

// MaskLogger takes a Logger and returns the Logger if not nil, or a NullLogger
// if it is nil.
func MaskLogger(logger Logger) Logger {
	if logger != nil {
		return logger
	}
	return NullLogger{}
}
