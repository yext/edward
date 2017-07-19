package must

// TestingT is an interface wrapper around *testing.T
type TestingT interface {
	Errorf(format string, args ...interface{})
}

// MustTester defines an interface with functions matching the package level check functions, without the requirement to specify a TestingT.
type MustTester interface {
	BeEqual(expected, got interface{}, a ...interface{}) bool
	BeEqualErrors(expected, got error, a ...interface{}) bool
	BeSameLength(expected, got interface{}, a ...interface{}) bool
}
