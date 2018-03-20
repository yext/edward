package struct2struct

import (
	"errors"
	"fmt"
	"reflect"
)

// Marshal processes i and applies its values to v.
// Fields are matched first by s2s tags, then by field names.
func Marshal(i interface{}, v interface{}) error {
	if v == nil {
		return errors.New("nil target")
	}
	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		return applyField(reflect.ValueOf(i), reflect.ValueOf(v).Elem())
	}
	return errors.New("expect target to be a pointer")
}

func mapFields(i interface{}, other interface{}) map[string]reflect.Value {

	var outFields = make(map[string]reflect.Value)
	iValue := reflect.Indirect(reflect.ValueOf(i))
	iType := iValue.Type()

	var otherType reflect.Type
	if other != nil {
		otherValue := reflect.ValueOf(other)
		if reflect.TypeOf(other).Kind() == reflect.Ptr {
			otherValue = reflect.Indirect(otherValue)
		}
		otherType = otherValue.Type()
	}

	for i := 0; i < iValue.NumField(); i++ {
		fType := iType.Field(i)
		fValue := iValue.Field(i)
		tags := fType.Tag
		if otherType != nil {
			if name, ok := tags.Lookup(fmt.Sprintf("%v.%v", otherType.PkgPath(), otherType.Name())); ok {
				outFields[name] = fValue
				continue
			}
			if name, ok := tags.Lookup(otherType.String()); ok {
				outFields[name] = fValue
				continue
			}
			if name, ok := tags.Lookup(otherType.Name()); ok {
				outFields[name] = fValue
				continue
			}
		}
		outFields[iType.Field(i).Name] = fValue
	}
	return outFields
}

// Marshaler allows a struct to provide custom marshalling to other types.
type Marshaler interface {
	MarshalStruct(v interface{}) error
}

// Unmarshaler allows a struct to provude custom unmarshalling from other types.
type Unmarshaler interface {
	UnmarshalStruct(v interface{}) error
}
