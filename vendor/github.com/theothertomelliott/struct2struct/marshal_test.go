package struct2struct_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/theothertomelliott/struct2struct"
)

type Untagged struct {
	MatchString          string
	MappedNameString     string
	MappedShortPkgString string
	MappedPkgPathString  string
}

type TwoIntsA struct {
	First  int
	Second int `TwoIntsB:"SecondB"`
}

type TwoIntsB struct {
	SecondB int
	First   int
}

type marshalTest struct {
	name       string
	in         interface{}
	other      interface{}
	expected   interface{}
	comparator func(e interface{}, g interface{}) (bool, string)
	err        error
}

func TestMarshalStructs(t *testing.T) {
	var tests = []marshalTest{
		{
			name: "Untagged, other nil",
			in: struct {
				Str string
			}{
				Str: "string",
			},
			err: errors.New("nil target"),
		},
		{
			name: "Tagged - partial",
			in: struct {
				MatchString string
			}{
				MatchString: "match",
			},
			other: &Untagged{},
			expected: &Untagged{
				MatchString: "match",
			},
		},
		{
			name: "Tagged - complete",
			in: struct {
				MatchString    string
				NameString     string `Untagged:"MappedNameString"`
				ShortPkgString string `struct2struct_test.Untagged:"MappedShortPkgString"`
				PkgPathString  string `github.com/theothertomelliott/struct2struct_test.Untagged:"MappedPkgPathString"`
			}{
				MatchString:    "match",
				NameString:     "name",
				ShortPkgString: "shortPkg",
				PkgPathString:  "pkgPath",
			},
			other: &Untagged{},
			expected: &Untagged{
				MatchString:          "match",
				MappedNameString:     "name",
				MappedShortPkgString: "shortPkg",
				MappedPkgPathString:  "pkgPath",
			},
		},
		{
			name: "Target not a pointer",
			in: struct {
				MatchString    string
				NameString     string `Untagged:"MappedNameString"`
				ShortPkgString string `struct2struct.Untagged:"MappedShortPkgString"`
				PkgPathString  string `github.com/theothertomelliott/struct2struct.Untagged:"MappedPkgPathString"`
			}{
				MatchString:    "match",
				NameString:     "name",
				ShortPkgString: "shortPkg",
				PkgPathString:  "pkgPath",
			},
			other: Untagged{},
			err:   errors.New("expect target to be a pointer"),
		},
		{
			name: "String pointer to string pointer",
			in: struct {
				MatchString *string
			}{
				MatchString: stringPtr("match"),
			},
			other: &struct {
				MatchString *string
			}{},
			expected: &struct {
				MatchString *string
			}{
				MatchString: stringPtr("match"),
			},
		},
		{
			name: "String pointer to string",
			in: struct {
				MatchString *string
			}{
				MatchString: stringPtr("match"),
			},
			other: &struct {
				MatchString string
			}{},
			expected: &struct {
				MatchString string
			}{
				MatchString: "match",
			},
		},
		{
			name: "String to string pointer",
			in: struct {
				MatchString string
			}{
				MatchString: "match",
			},
			other: &struct {
				MatchString *string
			}{},
			expected: &struct {
				MatchString *string
			}{
				MatchString: stringPtr("match"),
			},
		},
		{
			name: "String to int pointer",
			in: struct {
				MatchString string
			}{
				MatchString: "match",
			},
			other: &struct {
				MatchString *int
			}{},
			err: errors.New("MatchString: could not apply type 'string' to '*int'"),
		},
		{
			name: "Struct field, matching",
			in: struct {
				SubStruct struct{ num int }
			}{
				SubStruct: struct{ num int }{num: 100},
			},
			other: &struct {
				SubStruct struct{ num int }
			}{},
			expected: &struct {
				SubStruct struct{ num int }
			}{
				SubStruct: struct{ num int }{num: 100},
			},
		},
		{
			name: "Struct field, not matching",
			in: struct {
				SubStruct TwoIntsA
			}{
				SubStruct: TwoIntsA{
					First:  10,
					Second: 20,
				},
			},
			other: &struct {
				SubStruct TwoIntsB
			}{},
			expected: &struct {
				SubStruct TwoIntsB
			}{
				SubStruct: TwoIntsB{
					SecondB: 20,
					First:   10,
				},
			},
		},
		{
			name: "Struct field, pointer to pointer",
			in: struct {
				SubStruct *TwoIntsA
			}{
				SubStruct: &TwoIntsA{
					First:  10,
					Second: 20,
				},
			},
			other: &struct {
				SubStruct *TwoIntsB
			}{},
			expected: &struct {
				SubStruct *TwoIntsB
			}{
				SubStruct: &TwoIntsB{
					SecondB: 20,
					First:   10,
				},
			},
		},
		{
			name: "Struct field, pointer to non-pointer",
			in: struct {
				SubStruct *TwoIntsA
			}{
				SubStruct: &TwoIntsA{
					First:  10,
					Second: 20,
				},
			},
			other: &struct {
				SubStruct TwoIntsB
			}{},
			expected: &struct {
				SubStruct TwoIntsB
			}{
				SubStruct: TwoIntsB{
					SecondB: 20,
					First:   10,
				},
			},
		},
		{
			name: "Struct field, non-pointer to pointer",
			in: struct {
				SubStruct TwoIntsA
			}{
				SubStruct: TwoIntsA{
					First:  10,
					Second: 20,
				},
			},
			other: &struct {
				SubStruct *TwoIntsB
			}{},
			expected: &struct {
				SubStruct *TwoIntsB
			}{
				SubStruct: &TwoIntsB{
					SecondB: 20,
					First:   10,
				},
			},
		},
		{
			name: "Struct fields, error",
			in: struct {
				SubStruct struct{ First string }
			}{
				SubStruct: struct{ First string }{First: "first"},
			},
			other: &struct {
				SubStruct TwoIntsB
			}{},
			err: errors.New("SubStruct: First: strconv.Atoi: parsing \"first\": invalid syntax"),
		},
	}
	executeTests(t, tests)
}

func TestMarshalUnsettable(t *testing.T) {
	var tests = []marshalTest{
		{
			name: "Unsettable struct fields",
			in: struct {
				Settable   string
				unsettable string
			}{
				Settable:   "abc",
				unsettable: "def",
			},
			other: &struct {
				Settable   string
				unsettable string
				Extra      string
			}{},
			expected: &struct {
				Settable   string
				unsettable string
				Extra      string
			}{
				Settable: "abc",
			},
		},
	}
	executeTests(t, tests)
}

func TestMarshalSlices(t *testing.T) {
	var tests = []marshalTest{
		{
			name: "Matching slice types",
			in: []string{
				"a", "b",
			},
			other: &[]string{},
			expected: &[]string{
				"a", "b",
			},
		},
		{
			name: "Non-matching slice types",
			in: []TwoIntsA{
				{
					First:  10,
					Second: 20,
				},
			},
			other: &[]TwoIntsB{},
			expected: &[]TwoIntsB{
				{
					First:   10,
					SecondB: 20,
				},
			},
		},
		{
			name: "Non-matching slice types in struct",
			in: struct {
				Arr []TwoIntsA
			}{
				Arr: []TwoIntsA{
					{
						First:  10,
						Second: 20,
					},
				},
			},
			other: &struct {
				Arr []TwoIntsB
			}{},
			expected: &struct {
				Arr []TwoIntsB
			}{
				Arr: []TwoIntsB{
					{
						First:   10,
						SecondB: 20,
					},
				},
			},
		},
		{
			name: "Slice to non-slice error",
			in: []string{
				"a", "b",
			},
			other: &struct{}{},
			err:   errors.New("cannot apply a non-slice value to a slice"),
		},
		{
			name: "Invalid value mapping",
			in: []string{
				"a", "b",
			},
			other: &[]int{},
			err:   errors.New("strconv.Atoi: parsing \"a\": invalid syntax"),
		},
		{
			name: "Non-slice to slice error",
			in:   &struct{}{},
			other: &[]string{
				"a", "b",
			},
			err: errors.New("cannot apply a non-slice value to a slice"),
		},
	}
	executeTests(t, tests)
}

func TestMarshalFunc(t *testing.T) {
	var f1 = func() int {
		return 1
	}
	var f2 = func() int {
		return 0
	}
	var f3 = func() string {
		return "hello"
	}

	var tests = []marshalTest{
		{
			name:     "Matching func types",
			in:       f1,
			other:    &f2,
			expected: f1,
			comparator: func(e interface{}, g interface{}) (bool, string) {
				fe := e.(func() int)
				pfg := g.(*func() int)
				fg := *pfg
				if fe() != fg() {
					return false, "didn't match"
				}
				return true, ""
			},
		},
		{
			name:  "Non-matching func types",
			in:    f1,
			other: &f3,
			err:   errors.New("could not apply type 'func() int' to 'func() string'"),
		},
	}
	executeTests(t, tests)
}

func TestMarshalMaps(t *testing.T) {
	var tests = []marshalTest{
		{
			name: "Matching map types",
			in: map[string]string{
				"key-a": "val-a",
				"key-b": "val-b",
			},
			other: &map[string]string{},
			expected: &map[string]string{
				"key-a": "val-a",
				"key-b": "val-b",
			},
		},
		{
			name: "Map to non-map",
			in: map[string]string{
				"key-a": "val-a",
				"key-b": "val-b",
			},
			other: stringPtr("not a map"),
			err:   errors.New("cannot apply a map type to a non-map"),
		},
		{
			name: "Invalid key mapping",
			in: map[string]string{
				"abc": "val-a",
			},
			other: &map[int]interface{}{},
			err:   errors.New("strconv.Atoi: parsing \"abc\": invalid syntax"),
		},
		{
			name: "Invalid value mapping",
			in: map[string]string{
				"key-a": "abc",
			},
			other: &map[string]int{},
			err:   errors.New("strconv.Atoi: parsing \"abc\": invalid syntax"),
		},
		{
			name: "string->string to string->interface{}",
			in: map[string]string{
				"key-a": "val-a",
				"key-b": "val-b",
			},
			other: &map[string]interface{}{},
			expected: &map[string]interface{}{
				"key-a": "val-a",
				"key-b": "val-b",
			},
		},
	}
	executeTests(t, tests)
}

func TestMarshalToInt(t *testing.T) {
	var tests = []marshalTest{
		{
			name:     "string to int",
			in:       []string{"1"},
			other:    &[]int{},
			expected: &[]int{1},
		},
		{
			name:     "string to int8",
			in:       []string{"2"},
			other:    &[]int8{},
			expected: &[]int8{2},
		},
		{
			name:     "string to int16",
			in:       []string{"3"},
			other:    &[]int16{},
			expected: &[]int16{3},
		},
		{
			name:     "string to int32",
			in:       []string{"4"},
			other:    &[]int32{},
			expected: &[]int32{4},
		},
		{
			name:     "string to int64",
			in:       []string{"5"},
			other:    &[]int64{},
			expected: &[]int64{5},
		},
		{
			name:     "uint to int",
			in:       []uint{1, 2},
			other:    &[]int{},
			expected: &[]int{1, 2},
		},
		{
			name:     "uint8 to int",
			in:       []uint8{1, 2},
			other:    &[]int{},
			expected: &[]int{1, 2},
		},
		{
			name:     "uint16 to int",
			in:       []uint16{1, 2},
			other:    &[]int{},
			expected: &[]int{1, 2},
		},
		{
			name:     "uint32 to int",
			in:       []uint32{1, 2},
			other:    &[]int{},
			expected: &[]int{1, 2},
		},
		{
			name:     "uint64 to int",
			in:       []uint64{1, 2},
			other:    &[]int{},
			expected: &[]int{1, 2},
		},
		{
			name:     "int32 to int",
			in:       []int32{1, 2},
			other:    &[]int{},
			expected: &[]int{1, 2},
		},
		{
			name:     "float32 to int",
			in:       []float32{1, 2.2},
			other:    &[]int{},
			expected: &[]int{1, 2},
		},
		{
			name:     "float64 to int",
			in:       []float64{1, 2.2},
			other:    &[]int{},
			expected: &[]int{1, 2},
		},
		{
			name:  "Bool to int",
			in:    []bool{true, false},
			other: &[]int{},
			err:   errors.New("could not apply type 'bool' to 'int'"),
		},
	}
	executeTests(t, tests)
}

func TestMarshalToUint(t *testing.T) {
	var tests = []marshalTest{
		{
			name:     "string to uint",
			in:       []string{"1"},
			other:    &[]uint{},
			expected: &[]uint{1},
		},
		{
			name:     "string to uint8",
			in:       []string{"2"},
			other:    &[]uint8{},
			expected: &[]uint8{2},
		},
		{
			name:     "string to uint16",
			in:       []string{"3"},
			other:    &[]uint16{},
			expected: &[]uint16{3},
		},
		{
			name:     "string to uint32",
			in:       []string{"4"},
			other:    &[]uint32{},
			expected: &[]uint32{4},
		},
		{
			name:     "string to uint64",
			in:       []string{"5"},
			other:    &[]uint64{},
			expected: &[]uint64{5},
		},
		{
			name:     "uint32 to uint",
			in:       []uint32{1, 2},
			other:    &[]uint{},
			expected: &[]uint{1, 2},
		},
		{
			name:     "int8 to uint",
			in:       []int8{1, 2},
			other:    &[]uint{},
			expected: &[]uint{1, 2},
		},
		{
			name:     "int16 to uint",
			in:       []int16{1, 2},
			other:    &[]uint{},
			expected: &[]uint{1, 2},
		},
		{
			name:     "uint64 to int",
			in:       []uint64{1, 2},
			other:    &[]int{},
			expected: &[]int{1, 2},
		},
		{
			name:     "float32 to uint",
			in:       []float32{1, 2.2},
			other:    &[]uint{},
			expected: &[]uint{1, 2},
		},
		{
			name:     "float64 to uint",
			in:       []float64{1, 2.2},
			other:    &[]uint{},
			expected: &[]uint{1, 2},
		},
		{
			name:     "uint64 to uint32",
			in:       []uint64{1, 2},
			other:    &[]uint32{},
			expected: &[]uint32{1, 2},
		},
		{
			name:  "Bool to uint",
			in:    []bool{true, false},
			other: &[]uint{},
			err:   errors.New("could not apply type 'bool' to 'uint'"),
		},
		{
			name:  "Invalid string to uint",
			in:    []string{"abc"},
			other: &[]uint{},
			err:   errors.New("strconv.Atoi: parsing \"abc\": invalid syntax"),
		},
	}
	executeTests(t, tests)
}

func TestMarshalToFloat(t *testing.T) {
	var tests = []marshalTest{
		{
			name:     "string to float32",
			in:       []string{"1.11"},
			other:    &[]float32{},
			expected: &[]float32{1.11},
		},
		{
			name:     "string to float32",
			in:       []string{"2.22"},
			other:    &[]float64{},
			expected: &[]float64{2.22},
		},
		{
			name:     "uint32 to float32",
			in:       []uint32{1, 2},
			other:    &[]float32{},
			expected: &[]float32{1, 2},
		},
		{
			name:     "int8 to float64",
			in:       []int8{1, 2},
			other:    &[]float64{},
			expected: &[]float64{1, 2},
		},
		{
			name:     "int16 to float32",
			in:       []int16{1, 2},
			other:    &[]float32{},
			expected: &[]float32{1, 2},
		},
		{
			name:     "uint64 to int",
			in:       []uint64{1, 2},
			other:    &[]int{},
			expected: &[]int{1, 2},
		},
		{
			name:     "float32 to float64",
			in:       []float32{1, 2.2},
			other:    &[]float64{},
			expected: &[]float64{1, 2.2},
		},
		{
			name:     "float64 to float32",
			in:       []float64{1, 2.2},
			other:    &[]float32{},
			expected: &[]float32{1, 2.2},
		},
		{
			name:  "Bool to float32",
			in:    []bool{true, false},
			other: &[]float32{},
			err:   errors.New("could not apply type 'bool' to 'float32'"),
		},
		{
			name:  "Invalid string to float32",
			in:    []string{"abc"},
			other: &[]float32{},
			err:   errors.New("strconv.ParseFloat: parsing \"abc\": invalid syntax"),
		},
	}
	executeTests(t, tests)
}

func TestMarshalToString(t *testing.T) {
	var tests = []marshalTest{
		{
			name: "Int to string",
			in: struct {
				MatchString int
			}{
				MatchString: 100,
			},
			other: &Untagged{},
			expected: &Untagged{
				MatchString: "100",
			},
		},
		{
			name:     "bool to string",
			in:       []bool{true},
			other:    &[]string{},
			expected: &[]string{"true"},
		},
		{
			name:     "float64 to string",
			in:       []float64{2.2},
			other:    &[]string{},
			expected: &[]string{"2.2"},
		},
		{
			name: "struct to string",
			in: []struct {
				Field string
			}{
				{
					Field: "abc",
				},
			},
			other: &[]string{},
			err:   errors.New("cannot apply a struct type to a non-struct"),
		},
		{
			name: "slice to string",
			in: [][]string{
				[]string{},
			},
			other: &[]string{},
			err:   errors.New("cannot apply a non-slice value to a slice"),
		},
		{
			name: "map to string",
			in: []map[string]string{
				make(map[string]string),
			},
			other: &[]string{},
			err:   errors.New("cannot apply a map type to a non-map"),
		},
	}
	executeTests(t, tests)
}

func executeTests(t *testing.T, tests []marshalTest) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := struct2struct.Marshal(
				test.in,
				test.other,
			)
			if test.err == nil && err != nil {
				t.Error(err)
			}
			if test.err != nil && err == nil {
				t.Error("expected an error")
			}
			if test.err != nil && err != nil && test.err.Error() != err.Error() {
				t.Errorf("errors did not match, expected '%v', got '%v'", test.err, err)
			}
			if err != nil {
				return
			}
			if test.comparator != nil {
				if ok, message := test.comparator(test.expected, test.other); !ok {
					t.Errorf("comparison failed: %v", message)
				}
				return
			}
			if !reflect.DeepEqual(test.expected, test.other) {
				t.Errorf("values did not match, expected '%v', got '%v'", test.expected, test.other)
			}
		})
	}
}

func stringPtr(in string) *string {
	return &in
}
