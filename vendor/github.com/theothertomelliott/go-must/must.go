package must

// TestingT is an interface wrapper around *testing.T
type TestingT interface {
	Errorf(format string, args ...interface{})
}

type MustTester interface {
	BeEqual(expected, got interface{}, message string) bool
	BeEqualErrors(expected, got error, message string) bool
}
