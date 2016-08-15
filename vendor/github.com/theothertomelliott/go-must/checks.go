package must

import "github.com/kylelemons/godebug/pretty"

func BeEqual(t TestingT, expected, got interface{}, message string) bool {
	if diff := pretty.Compare(expected, got); diff != "" {
		t.Errorf("%s: diff: (-got +want)\n%s", message, diff)
		return false
	}
	return true
}

func getErrMessage(err error) string {
	if err != nil {
		return err.Error()
	}
	return "<nil>"
}

func BeEqualErrors(t TestingT, expected, got error, message string) bool {
	if expected == nil && got == nil {
		return true
	}
	if (expected == nil || got == nil) || expected.Error() != got.Error() {
		t.Errorf("%v\nExpected '%v', got '%v'", message, getErrMessage(expected), getErrMessage(got))
		return false
	}
	return true
}
