package struct2struct

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

var appliers []applier

func init() {
	appliers = []applier{
		interfaceApplier,
		settableTestApplier,
		matchedTypeApplier,
		pointerApplier,
		sliceApplier,
		mapApplier,
		structApplier,
		intApplier,
		uintApplier,
		floatApplier,
		stringApplier,
	}
}

type applier func(reflect.Value, reflect.Value) (bool, error)

func applyField(iField reflect.Value, vField reflect.Value) error {
	for _, applier := range appliers {
		applied, err := applier(iField, vField)
		if applied || err != nil {
			return err
		}
	}
	if !iField.IsValid() || !vField.IsValid() {
		return fmt.Errorf("could not apply types")
	}
	return fmt.Errorf("could not apply type '%v' to '%v'", iField.Type(), vField.Type())
}

func intApplier(iField reflect.Value, vField reflect.Value) (bool, error) {
	if !iField.IsValid() || !vField.IsValid() {
		return false, nil
	}

	switch vField.Type().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
	default:
		return false, nil
	}

	var value int64

	switch iField.Type().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = iField.Int()
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8:
		value = int64(iField.Uint())
	case reflect.Float32, reflect.Float64:
		value = int64(iField.Float())
	case reflect.String:
		valInt, err := strconv.Atoi(iField.String())
		if err != nil {
			return false, err
		}
		value = int64(valInt)
	default:
		return false, nil
	}

	vField.SetInt(value)
	return true, nil
}

func uintApplier(iField reflect.Value, vField reflect.Value) (bool, error) {
	if !iField.IsValid() || !vField.IsValid() {
		return false, nil
	}

	switch vField.Type().Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
	default:
		return false, nil
	}

	var value uint64

	switch iField.Type().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = uint64(iField.Int())
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8:
		value = iField.Uint()
	case reflect.Float32, reflect.Float64:
		value = uint64(iField.Float())
	case reflect.String:
		valInt, err := strconv.Atoi(iField.String())
		if err != nil {
			return false, err
		}
		value = uint64(valInt)
	default:
		return false, nil
	}

	vField.SetUint(value)
	return true, nil
}

func floatApplier(iField reflect.Value, vField reflect.Value) (bool, error) {
	if !iField.IsValid() || !vField.IsValid() {
		return false, nil
	}

	var bitSize = 32
	switch vField.Type().Kind() {
	case reflect.Float32:
	case reflect.Float64:
		bitSize = 64
	default:
		return false, nil
	}

	var value float64
	var err error

	switch iField.Type().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = float64(iField.Int())
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8:
		value = float64(iField.Uint())
	case reflect.Float32:
		value, _ = strconv.ParseFloat(fmt.Sprint(float32(iField.Float())), bitSize)
	case reflect.Float64:
		value = iField.Float()
	case reflect.String:
		value, err = strconv.ParseFloat(iField.String(), bitSize)
		if err != nil {
			return false, err
		}
	default:
		return false, nil
	}

	vField.SetFloat(value)
	return true, nil
}

func stringApplier(iField reflect.Value, vField reflect.Value) (bool, error) {
	if !iField.IsValid() || !vField.IsValid() {
		return false, nil
	}
	if vField.Type().Kind() != reflect.String {
		return false, nil
	}

	vField.SetString(fmt.Sprint(iField.Interface()))
	return true, nil
}

func interfaceApplier(iField reflect.Value, vField reflect.Value) (bool, error) {
	if !iField.IsValid() || !vField.IsValid() {
		return false, nil
	}
	if vField.Type().Kind() != reflect.Interface {
		return false, nil
	}

	vField.Set(iField)
	return true, nil
}

func sliceApplier(iField reflect.Value, vField reflect.Value) (bool, error) {
	if !iField.IsValid() || !vField.IsValid() {
		return false, nil
	}
	if iField.Type().Kind() != reflect.Slice && vField.Type().Kind() != reflect.Slice {
		return false, nil
	}
	if iField.Type().Kind() != reflect.Slice || vField.Type().Kind() != reflect.Slice {
		return false, errors.New("cannot apply a non-slice value to a slice")
	}

	for i := 0; i < iField.Len(); i++ {
		iValue := iField.Index(i)
		appendVal := reflect.New(vField.Type().Elem())
		err := applyField(iValue, appendVal.Elem())
		if err != nil {
			return false, err
		}
		vField.Set(reflect.Append(vField, appendVal.Elem()))
	}
	return true, nil
}

// settableTestApplier drops handling for any unsettable fields
func settableTestApplier(iField reflect.Value, vField reflect.Value) (bool, error) {
	if !vField.CanSet() {
		return true, nil
	}
	return false, nil
}

func matchedTypeApplier(iField reflect.Value, vField reflect.Value) (bool, error) {
	if !iField.IsValid() || !vField.IsValid() {
		return false, nil
	}
	if iField.Type() == vField.Type() {
		vField.Set(iField)
		return true, nil
	}
	return false, nil
}

func structApplier(iField reflect.Value, vField reflect.Value) (bool, error) {
	if !iField.IsValid() || !vField.IsValid() {
		return false, nil
	}
	if iField.Type().Kind() != reflect.Struct && vField.Type().Kind() != reflect.Struct {
		return false, nil
	}
	if iField.Type().Kind() != reflect.Struct || vField.Type().Kind() != reflect.Struct {
		return false, errors.New("cannot apply a struct type to a non-struct")
	}
	newPtr := reflect.New(vField.Type())
	newPtr.Elem().Set(vField)
	err := marshalStruct(iField.Interface(), newPtr.Interface())
	vField.Set(newPtr.Elem())
	return err == nil, err
}

func marshalStruct(i interface{}, v interface{}) error {
	iFields := mapFields(i, v)
	vFields := mapFields(v, i)

	for name, iField := range iFields {
		if vField, ok := vFields[name]; ok {
			err := applyField(iField, vField)
			if err != nil {
				return fmt.Errorf("%v: %v", name, err)
			}
		}
	}
	return nil
}

func pointerApplier(iField reflect.Value, vField reflect.Value) (bool, error) {
	if !iField.IsValid() || !vField.IsValid() {
		return false, nil
	}
	if iField.Type().Kind() == reflect.Ptr {
		err := applyField(reflect.Indirect(iField), vField)
		return err == nil, err
	}
	iPtrType := reflect.PtrTo(iField.Type())
	if vField.Type().Kind() == reflect.Ptr {
		if iPtrType == vField.Type() {
			newPtr := reflect.New(iField.Type())
			newPtr.Elem().Set(iField)
			err := applyField(newPtr, vField)
			return err == nil, err
		}
		t := reflect.TypeOf(vField.Interface())
		if iField.Kind() == reflect.Struct && t.Elem().Kind() == reflect.Struct {
			newPtr := reflect.New(t.Elem())
			err := applyField(iField, newPtr.Elem())
			if err == nil {
				vField.Set(newPtr)
			}
			return err == nil, err
		}
	}
	return false, nil
}

func mapApplier(iField reflect.Value, vField reflect.Value) (bool, error) {
	if !iField.IsValid() || !vField.IsValid() {
		return false, nil
	}
	if iField.Type().Kind() != reflect.Map && vField.Type().Kind() != reflect.Map {
		return false, nil
	}
	if iField.Type().Kind() != reflect.Map || vField.Type().Kind() != reflect.Map {
		return false, errors.New("cannot apply a map type to a non-map")
	}

	vKeyType := vField.Type().Key()
	vElemType := vField.Type().Elem()

	newMap := reflect.MakeMap(vField.Type())

	for _, key := range iField.MapKeys() {
		newKey := reflect.New(vKeyType)
		newElem := reflect.New(vElemType)
		err := applyField(key, newKey.Elem())
		if err != nil {
			return false, err
		}
		err = applyField(iField.MapIndex(key), newElem.Elem())
		if err != nil {
			return false, err
		}

		newMap.SetMapIndex(newKey.Elem(), newElem.Elem())
	}
	vField.Set(newMap)

	return true, nil
}
