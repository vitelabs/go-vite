package abi

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	errBadBool                    = errors.New("abi: improperly encoded boolean value")
	errEmptyInput                 = errors.New("unmarshalling empty output")
	errCouldNotLocateNamedMethod  = errors.New("abi: could not locate named method")
	errInvalidZeroVariableSize    = errors.New("abi: invalid zero variable size")
	errInvalidArrayTypeFormatting = errors.New("invalid formatting of array type")
	errPackFailed                 = errors.New("abi: pack element failed")
)

func errMethodNotFound(name string) error {
	return fmt.Errorf("method '%s' not found", name)
}
func errCallbackNotFound(name string) error {
	return fmt.Errorf("callback '%s' not found", name)
}
func errOffchainNotFound(name string) error {
	return fmt.Errorf("offchain '%s' not found", name)
}
func errVariableNotFound(name string) error {
	return fmt.Errorf("varible '%s' not found", name)
}
func errEventNotFound(name string) error {
	return fmt.Errorf("event '%s' not found", name)
}
func errType(expected, got interface{}) error {
	return fmt.Errorf("abi: cannot use %v as type %v as argument", got, expected)
}
func errParsingVariableSize(err error) error {
	return fmt.Errorf("abi: error parsing variable size: %v", err)
}
func errUnsupportedArgType(t string) error {
	return fmt.Errorf("abi: unsupported arg type: %s", t)
}

// formatSliceString formats the reflection kind with the given slice size
// and returns a formatted string representation.
func formatSliceString(kind reflect.Kind, sliceSize int) string {
	if sliceSize == -1 {
		return fmt.Sprintf("[]%v", kind)
	}
	return fmt.Sprintf("[%d]%v", sliceSize, kind)
}

// sliceTypeCheck checks that the given slice can by assigned to the reflection
// type in t.
func sliceTypeCheck(t Type, val reflect.Value) error {
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return errType(formatSliceString(t.Kind, t.Size), val.Type())
	}

	if t.T == ArrayTy && val.Len() != t.Size {
		return errType(formatSliceString(t.Elem.Kind, t.Size), formatSliceString(val.Type().Elem().Kind(), val.Len()))
	}

	if t.Elem.T == SliceTy {
		if val.Len() > 0 {
			return sliceTypeCheck(*t.Elem, val.Index(0))
		}
	} else if t.Elem.T == ArrayTy {
		return sliceTypeCheck(*t.Elem, val.Index(0))
	}

	if elemKind := val.Type().Elem().Kind(); (elemKind != reflect.Slice && elemKind != t.Elem.Kind) ||
		(elemKind == reflect.Slice && t.Elem.Kind != reflect.Slice && t.Elem.Kind != reflect.Array) ||
		(elemKind != reflect.Slice && elemKind != reflect.Array && t.Elem.Kind == reflect.Array) {
		return errType(formatSliceString(t.Elem.Kind, t.Size), val.Type())
	}
	return nil
}

// typeCheck checks that the given reflection value can be assigned to the reflection
// type in t.
func typeCheck(t Type, value reflect.Value) error {
	if t.T == SliceTy || t.T == ArrayTy {
		return sliceTypeCheck(t, value)
	}

	// Check base type validity. Element types will be checked later on.
	if t.Kind != reflect.Array && t.Kind != value.Kind() {
		return errType(t.Kind, value.Kind())
	} else if t.T == FixedBytesTy && t.Size != value.Len() {
		return errType(t.Type, value.Type())
	} else {
		return nil
	}

}
