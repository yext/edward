package must

import (
	"fmt"
	"reflect"

	"github.com/kylelemons/godebug/diff"
	"github.com/kylelemons/godebug/pretty"
)

var _ MustTester = Tester{}

/*
Tester implements MustTester and provides a TestingT to be used for all check functions.
*/
type Tester struct {
	T                   TestingT                               // *testing.T or equivalent
	InterfaceComparison func(expected, got interface{}) bool   // Optional custom interface comparison function
	InterfaceDiff       func(expected, got interface{}) string // Optional custom interace diff function
}

/*
BeEqual compares the expected and got interfaces, triggering an error on the Tester's T if they are not equal.

This corresponds to the function BeEqual
*/
func (tester Tester) BeEqual(expected, got interface{}, a ...interface{}) bool {
	if !tester.equal(expected, got) {
		tester.formattedError("diff\n%s", a, tester.diff(expected, got))
		return false
	}
	return true
}

/*
BeEqualErrors compares the expected and got errors, triggering an error on the Tester's T if they are not equal.

This corresponds to the function BeEqualErrors
*/
func (tester Tester) BeEqualErrors(expected, got error, a ...interface{}) bool {
	if expected == nil && got == nil {
		return true
	}
	if (expected == nil || got == nil) || expected.Error() != got.Error() {
		tester.formattedError("Expected '%v', got '%v'", a, getErrMessage(expected), getErrMessage(got))
		return false
	}
	return true
}

/*
BeNoError checks whether got is set, triggering an error on the Tester's T if it is non-nil.

This corresponds to the function BeNoError
*/
func (tester Tester) BeNoError(got error, a ...interface{}) bool {
	if got == nil {
		return true
	}
	tester.formattedError("error: %s", a, got.Error())
	return false
}

/*
BeSameLength checks whether the two inputs have the same length according to the len function.

This corresponds to the function BeSameLength
*/
func (tester Tester) BeSameLength(expected, got interface{}, a ...interface{}) bool {
	lenExpected, err := lenterface(expected)
	if err != nil {
		tester.formattedError("could not test lengths - %v", a, err)
		return false
	}
	lenGot, err := lenterface(got)
	if err != nil {
		tester.formattedError("could not test lengths - %v", a, err)
		return false
	}

	if lenExpected == lenGot {
		return true
	}
	tester.formattedError("expected length %d, got length %d", a, lenExpected, lenGot)
	return false
}

func lenterface(val interface{}) (int, error) {
	kind := reflect.TypeOf(val).Kind()
	switch kind {
	case reflect.Slice, reflect.Map, reflect.String, reflect.Chan, reflect.Array:
		s := reflect.ValueOf(val)
		return s.Len(), nil
	case reflect.Ptr:
		return lenterfacePtr(reflect.ValueOf(val))
	}
	return 0, fmt.Errorf("cannot get the length of type: %v", kind)
}

func lenterfacePtr(val reflect.Value) (int, error) {
	i := reflect.Indirect(val)
	switch i.Kind() {
	case reflect.Slice, reflect.Map, reflect.String, reflect.Chan, reflect.Array:
		return i.Len(), nil
	}
	return 0, fmt.Errorf("cannot get the length of a pointer to type: %v", i.Kind())
}

func (tester Tester) equal(expected, got interface{}) bool {
	if tester.InterfaceComparison != nil {
		return tester.InterfaceComparison(expected, got)
	}
	return pretty.Compare(expected, got) == ""
}

func (tester Tester) diff(expected, got interface{}) string {
	if tester.InterfaceDiff != nil {
		return tester.InterfaceDiff(expected, got)
	}

	// Do string diff if strings. Compare does not handle multiline strings well
	e, eok := expected.(string)
	g, gok := got.(string)
	if eok && gok {
		return fmt.Sprintf("(- expected, + got)\n%v", diff.Diff(e, g))
	}

	return fmt.Sprintf("(- expected, + got)\n%v", pretty.Compare(expected, got))
}

func (tester Tester) formattedError(format string, a []interface{}, following ...interface{}) {
	if len(a) > 0 {
		var args []interface{}
		args = append(args, fmt.Sprint(a...))
		args = append(args, following...)
		tester.T.Errorf("%v: "+format, args...)
	} else {
		tester.T.Errorf(format, following...)
	}
}

func getErrMessage(err error) string {
	if err != nil {
		return err.Error()
	}
	return "<nil>"
}
