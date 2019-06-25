package abi

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/helper"
	"math/big"
	"reflect"
)

var (
	errBadBool                     = errors.New("abi: improperly encoded boolean value")
	errEmptyInput                  = errors.New("unmarshalling empty output")
	errCouldNotLocateNamedMethod   = errors.New("abi: could not locate named method")
	errCouldNotLocateNamedEvent    = errors.New("abi: could not locate named event")
	errCouldNotLocateNamedVariable = errors.New("abi: could not locate named variable")
	errMethodIdNotSpecified        = errors.New("method id is not specified")
	errInvalidEmptyVariableInput   = errors.New("abi: variable inputs should not be empty")
	errInvalidZeroVariableSize     = errors.New("abi: invalid zero variable size")
	errInvalidArrayTypeFormatting  = errors.New("invalid formatting of array type")
	errPackFailed                  = errors.New("abi: pack element failed")
	errPureUnderscoredOutput       = errors.New("abi: purely underscored output cannot unpack to struct")
	errInvalidlFixedBytesType      = errors.New("abi: invalid type in call to make fixed byte array")
	errInvalidlArrayType           = errors.New("abi: invalid type in array/slice unpacking stage")
)

// parse json errors
func errArgumentJsonErr(err error) error {
	return fmt.Errorf("argument json err: %v", err)
}

// method not found errors
func errMethodNotFound(name string) error {
	return fmt.Errorf("method '%s' not found", name)
}
func errNoMethodId(sigdata []byte) error {
	return fmt.Errorf("no method with id: %#x", sigdata[:4])
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

// type errors
func errType(expected, got interface{}) error {
	return fmt.Errorf("abi: cannot use %v as type %v as argument", got, expected)
}
func errParsingVariableSize(err error) error {
	return fmt.Errorf("abi: error parsing variable size: %v", err)
}
func errUnsupportedArgType(t string) error {
	return fmt.Errorf("abi: unsupported arg type: %s", t)
}
func errUnknownType(t Type) error {
	return fmt.Errorf("abi: unknown type %v", t.T)
}

// pack errors
func errWrongPackedLength(marshalledValues []interface{}) error {
	return fmt.Errorf("abi: wrong length, expected single value, got %d", len(marshalledValues))
}
func errArgLengthMismatch(args []interface{}, abiArgs Arguments) error {
	return fmt.Errorf("argument count mismatch: %d for %d", len(args), len(abiArgs))
}

// unpack errors
func errInvalidStruct(v interface{}) error {
	return fmt.Errorf("abi: Unpack(non-pointer %T)", v)
}
func errInsufficientArgumentSize(arguments Arguments, value reflect.Value) error {
	return fmt.Errorf("abi: insufficient number of arguments for unpack, want %d, got %d", len(arguments), value.Len())
}
func errInsufficientElementSize(minLen int, v reflect.Value) error {
	return fmt.Errorf("abi: insufficient number of elements in the list/array for unpack, want %d, got %d",
		minLen, v.Len())
}
func errUnmarshalTypeFailed(src, dst reflect.Value) error {
	return fmt.Errorf("abi: cannot unmarshal %v in to %v", src.Type(), dst.Type())
}
func errInvalidTuple(typ reflect.Type) error {
	return fmt.Errorf("abi: cannot unmarshal tuple into %v", typ)
}
func errEmptyTagName(structFieldName string) error {
	return fmt.Errorf("struct: abi tag in '%s' is empty", structFieldName)
}
func errTagAlreadyMapped(structFieldName string) error {
	return fmt.Errorf("struct: abi tag in '%s' already mapped", structFieldName)
}
func errTagNotFound(tagName string) error {
	return fmt.Errorf("struct: abi tag '%s' defined but not found in abi", tagName)
}
func errMultipleVariable(abiFieldName string) error {
	return fmt.Errorf("abi: multiple variables maps to the same abi field '%s'", abiFieldName)
}
func errMultipleOutput(structFieldName string) error {
	return fmt.Errorf("abi: multiple outputs mapping to the same struct field '%s'", structFieldName)
}
func errNegativeInputSize(size int) error {
	return fmt.Errorf("cannot marshal input to array, size is negative (%d)", size)
}
func errArrayOffsetOverflow(output []byte, start, size int) error {
	return fmt.Errorf("abi: cannot marshal in to go array: offset %d would go over slice boundary (len=%d)", len(output), start+helper.WordSize*size)
}
func errInsufficientLength(outputSize []byte, index int) error {
	return fmt.Errorf("abi: cannot marshal in to go type: length insufficient %d require %d", outputSize, index+helper.WordSize)
}
func errBigSliceOffsetOverflow(bigOffsetEnd, outputLength *big.Int) error {
	return fmt.Errorf("abi: cannot marshal in to go slice: offset %v would go over slice boundary (len=%v)", bigOffsetEnd, outputLength)
}
func errBigOffsetOverflow(bigOffsetEnd *big.Int) error {
	return fmt.Errorf("abi offset larger than int64: %v", bigOffsetEnd)
}
func errBigLengthOverflow(totalSize *big.Int) error {
	return fmt.Errorf("abi length larger than int64: %v", totalSize)
}
func errInsufficientBigLength(outputLength, totalSize *big.Int) error {
	return fmt.Errorf("abi: cannot marshal in to go type: length insufficient %v require %v", outputLength, totalSize)
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
