package must

import (
	"errors"
	"testing"
)

var beEqualTests = []struct {
	name       string
	expected   interface{}
	got        interface{}
	message    string
	shouldPass bool
	format     string
}{
	{
		name:       "Different strings",
		expected:   "string1",
		got:        "string2",
		shouldPass: false,
		format:     "%s: diff: (-got +want)\n%s",
	},
	{
		name:       "Matching strings",
		expected:   "string",
		got:        "string",
		shouldPass: true,
	},
	{
		name:       "Different arrays",
		expected:   []string{"string1", "string2"},
		got:        []string{"string3", "string4"},
		shouldPass: false,
		format:     "%s: diff: (-got +want)\n%s",
	},
	{
		name:       "Matching arrays",
		expected:   []string{"string1", "string2"},
		got:        []string{"string1", "string2"},
		shouldPass: true,
	},
}

func TestBeEqual(t *testing.T) {
	for _, test := range beEqualTests {
		m := &MockTesting{}
		tester := Tester{
			T: m,
		}
		result := tester.BeEqual(test.expected, test.got, test.message)
		if test.shouldPass && !result {
			t.Errorf("%s: Expected check would pass.", test.name)
		} else if !test.shouldPass && result {
			t.Errorf("%s: Expected check would not pass.", test.name)
		}

		if test.format != m.format {
			t.Errorf("%s: Incorrect error format. Expected '%v', got '%v'. errorCalled=%v", test.name, test.format, m.format, m.errorCalled)
		}
	}
}

var beEqualErrorsTests = []struct {
	name       string
	expected   error
	got        error
	message    string
	shouldPass bool
	format     string
}{
	{
		name:       "Different errors",
		expected:   errors.New("Message one"),
		got:        errors.New("Message two"),
		shouldPass: false,
		format:     "%v\nExpected '%v', got '%v'",
	},
	{
		name:       "Matching errors",
		expected:   errors.New("Message"),
		got:        errors.New("Message"),
		shouldPass: true,
	},
}

func TestBeEqualErrors(t *testing.T) {
	for _, test := range beEqualErrorsTests {
		m := &MockTesting{}
		tester := Tester{
			T: m,
		}
		result := tester.BeEqualErrors(test.expected, test.got, test.message)
		if test.shouldPass && !result {
			t.Errorf("%s: Expected check would pass.", test.name)
		} else if !test.shouldPass && result {
			t.Errorf("%s: Expected check would not pass.", test.name)
		}

		if test.format != m.format {
			t.Errorf("%s: Incorrect error format. Expected '%v', got '%v'. errorCalled=%v", test.name, test.format, m.format, m.errorCalled)
		}
	}
}

type MockTesting struct {
	errorCalled bool
	format      string
	args        []interface{}
}

func (m *MockTesting) Errorf(format string, args ...interface{}) {
	m.errorCalled = true
	m.format = format
	m.args = args
}
