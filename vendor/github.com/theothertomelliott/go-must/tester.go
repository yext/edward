package must

var _ MustTester = Tester{}

type Tester struct {
	T TestingT
}

func (tester Tester) BeEqual(expected, got interface{}, message string) bool {
	return BeEqual(tester.T, expected, got, message)
}

func (tester Tester) BeEqualErrors(expected, got error, message string) bool {
	return BeEqualErrors(tester.T, expected, got, message)
}
