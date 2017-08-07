package must

import (
	"errors"
	"reflect"
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
		format:     "%v: diff\n%s",
		message:    "Message1",
	},
	{
		name:       "Matching strings",
		expected:   "string",
		got:        "string",
		shouldPass: true,
	},
	{
		name:       "Different multiline strings",
		expected:   "string1\nstring2",
		got:        "string2\nstring1",
		shouldPass: false,
		format:     "%v: diff\n%s",
		message:    "Message",
	},
	{
		name:       "Matching multiline strings",
		expected:   "string1\nstring2",
		got:        "string1\nstring2",
		shouldPass: true,
	},
	{
		name:       "Different arrays",
		expected:   []string{"string1", "string2"},
		got:        []string{"string3", "string4"},
		shouldPass: false,
		format:     "%v: diff\n%s",
		message:    "Message2",
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
		test := test
		t.Run(test.name, func(t *testing.T) {
			m := &MockTesting{}
			tester := Tester{
				T: m,
			}
			result := tester.BeEqual(test.expected, test.got, test.message)
			if test.shouldPass && !result {
				t.Error("Check did not pass as expected.")
			} else if !test.shouldPass && result {
				t.Error("Check did not fail as expected")
			} else {
				if test.format != m.format {
					t.Errorf("Incorrect error format. Expected '%v', got '%v'. errorCalled=%v", test.format, m.format, m.errorCalled)
				}

				if !result {
					if len(m.args) < 2 {
						t.Errorf("Expected 2 error args, got %d", len(m.args))
					}

					if test.message != m.args[0] {
						t.Errorf("Incorrect message. Expected '%v', got '%v'", test.message, m.args[0])
					}
				}
			}
		})
	}
}

func TestBeEqualCustomCompare(t *testing.T) {
	for _, test := range beEqualTests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			m := &MockTesting{}
			tester := Tester{
				T: m,
				InterfaceComparison: func(expected, got interface{}) bool {
					if !reflect.DeepEqual(expected, test.expected) {
						t.Error("Wrong expected sent to compare")
					}
					if !reflect.DeepEqual(got, test.got) {
						t.Error("Wrong got sent to compare")
					}
					return true
				},
			}
			if !tester.BeEqual(test.expected, test.got, test.message) {
				t.Error("Forced true comparison did not suceed as expected")
			}

			tester = Tester{
				T: m,
				InterfaceComparison: func(expected, got interface{}) bool {
					return false
				},
			}
			if tester.BeEqual(test.expected, test.got, test.message) {
				t.Errorf("Forced false comparison did not fail as expected")
			}
		})
	}
}

func TestBeEqualCustomDiff(t *testing.T) {
	for _, test := range beEqualTests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			m := &MockTesting{}
			tester := Tester{
				T: m,
				InterfaceDiff: func(expected, got interface{}) string {
					return "forced diff"
				},
			}
			result := tester.BeEqual(test.expected, test.got, test.message)
			if test.shouldPass && !result {
				t.Error("Check did not pass as expected.")
			} else if !test.shouldPass && result {
				t.Error("Check did not fail as expected.")
			} else {
				if test.format != m.format {
					t.Errorf("Incorrect error format. Expected '%v', got '%v'. errorCalled=%v", test.format, m.format, m.errorCalled)
				}

				if !result {
					if len(m.args) < 2 {
						t.Errorf("Expected 2 error args, got %d", len(m.args))
					}

					if test.message != m.args[0] {
						t.Errorf("Incorrect message. Expected '%v', got '%v'", test.message, m.args[0])
					}

					if "forced diff" != m.args[1] {
						t.Errorf("Custom diff func was not used, got '%v'", m.args[1])
					}
				}
			}
		})
	}
}

func TestBeEqualErrors(t *testing.T) {
	var tests = []struct {
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
			format:     "%v: Expected '%v', got '%v'",
		},
		{
			name:       "Got nil, expected not",
			expected:   errors.New("Message one"),
			got:        nil,
			shouldPass: false,
			format:     "%v: Expected '%v', got '%v'",
		},
		{
			name:       "Expected nil, got not",
			expected:   nil,
			got:        errors.New("Message one"),
			shouldPass: false,
			format:     "%v: Expected '%v', got '%v'",
		},
		{
			name:       "Matching errors",
			expected:   errors.New("Message"),
			got:        errors.New("Message"),
			shouldPass: true,
		},
		{
			name:       "Both nil",
			expected:   nil,
			got:        nil,
			shouldPass: true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			m := &MockTesting{}
			tester := Tester{
				T: m,
			}
			result := tester.BeEqualErrors(test.expected, test.got, test.message)
			if test.shouldPass && !result {
				t.Error("Expected check would pass.")
			} else if !test.shouldPass && result {
				t.Error("Expected check would not pass.")
			}

			if test.format != m.format {
				t.Errorf("Incorrect error format. Expected '%v', got '%v'. errorCalled=%v", test.format, m.format, m.errorCalled)
			}
		})
	}
}

func TestBeNoError(t *testing.T) {
	var tests = []struct {
		name       string
		got        error
		message    string
		shouldPass bool
		format     string
	}{
		{
			name:       "No error exists",
			shouldPass: true,
		},
		{
			name:       "Error exists",
			got:        errors.New("Message"),
			shouldPass: false,
			format:     "%v: error: %s",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			m := &MockTesting{}
			tester := Tester{
				T: m,
			}
			result := tester.BeNoError(test.got, test.message)
			if test.shouldPass && !result {
				t.Error("Expected check would pass.")
			} else if !test.shouldPass && result {
				t.Error("Expected check would not pass.")
			}

			if test.format != m.format {
				t.Errorf("Incorrect error format. Expected '%v', got '%v'. errorCalled=%v", test.format, m.format, m.errorCalled)
			}
		})
	}
}

func TestBeSameLength(t *testing.T) {
	var tests = []struct {
		name       string
		expected   interface{}
		got        interface{}
		message    string
		shouldPass bool
		format     string
	}{
		{
			name:       "Strings, same length",
			expected:   "abcdefg",
			got:        "hijklmn",
			shouldPass: true,
		},
		{
			name:       "Strings, different length",
			expected:   "abc",
			got:        "defg",
			shouldPass: false,
			format:     "%v: expected length %d, got length %d",
		},
		{
			name:       "Arrays, same length",
			expected:   []int{1, 2, 3},
			got:        []int{4, 5, 6},
			shouldPass: true,
		},
		{
			name:       "Arrays, different length",
			expected:   []int{1, 2, 3, 7},
			got:        []int{8},
			shouldPass: false,
			format:     "%v: expected length %d, got length %d",
		},
		{
			name:       "String and string pointer, same length",
			expected:   stringToPointer("abcdefg"),
			got:        "hijklmn",
			shouldPass: true,
		},
		{
			name:       "String and string pointer, different length",
			expected:   "abc",
			got:        stringToPointer("defg"),
			shouldPass: false,
			format:     "%v: expected length %d, got length %d",
		},
		{
			name:       "String and struct",
			expected:   "abc",
			got:        struct{ content string }{content: "test"},
			shouldPass: false,
			format:     "%v: could not test lengths - %v",
		},
		{
			name:       "Struct and string",
			got:        "abc",
			expected:   struct{ content string }{content: "test"},
			shouldPass: false,
			format:     "%v: could not test lengths - %v",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			m := &MockTesting{}
			tester := Tester{
				T: m,
			}
			result := tester.BeSameLength(test.expected, test.got, test.message)
			if test.shouldPass && !result {
				t.Error("Check did not pass as expected.")
			} else if !test.shouldPass && result {
				t.Error("Check did not fail as expected")
			}

			if test.format != m.format {
				t.Errorf("Incorrect error format. Expected '%v', got '%v'. args=%v, errorCalled=%v", test.format, m.format, m.args, m.errorCalled)
			}

			if !result {
				if len(m.args) < 2 {
					t.Errorf("Expected at least 2 error args, got %d", len(m.args))
				}

				if test.message != m.args[0] {
					t.Errorf("Incorrect message. Expected '%v', got '%v'", test.message, m.args)
				}
			}
		})
	}
}

func stringToPointer(val string) *string {
	return &val
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
