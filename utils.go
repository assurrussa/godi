package godi

import (
	"fmt"
	"reflect"
	"strings"
)

func validateConstructor(constructor any) error {
	funcType := reflect.TypeOf(constructor)
	if funcType == nil {
		return fmt.Errorf("constructor must be a function, got %T", constructor)
	}
	if funcType.Kind() != reflect.Func {
		return fmt.Errorf("constructor must be a function, got %T", constructor)
	}

	outTypes := make([]reflect.Type, 0, funcType.NumOut())
	for i := range funcType.NumOut() {
		outTypes = append(outTypes, funcType.Out(i))
	}

	// Allowed:
	// 1) func(...) T
	// 2) func(...) (T, error)
	//
	// Disallowed:
	// - returning only error
	// - two returns where the first is error (e.g. (error, error))
	// - two returns where the second is not error
	isOutInvalid := len(outTypes) < 1 ||
		len(outTypes) > 2 ||
		len(outTypes) == 1 && outTypes[0].AssignableTo(reflect.TypeFor[error]()) ||
		len(outTypes) == 2 && outTypes[0].AssignableTo(reflect.TypeFor[error]()) ||
		len(outTypes) == 2 && !outTypes[1].AssignableTo(reflect.TypeFor[error]())

	if isOutInvalid {
		formattedOutTypes := make([]string, 0, len(outTypes))
		for _, o := range outTypes {
			formattedOutTypes = append(formattedOutTypes, o.String())
		}
		return fmt.Errorf("constructor must return value or (value, error), returns (%s)", strings.Join(formattedOutTypes, ", "))
	}

	return nil
}

func isPointerToInterface(a any) bool {
	pType := reflect.TypeOf(a)
	if pType == nil || pType.Kind() != reflect.Pointer {
		return false
	}

	return pType.Elem().Kind() == reflect.Interface
}
