package access

import (
	"fmt"
	"math"
	"reflect"
	"time"
)

// NormalizeValue takes a scalar snapshot suitable for a policy. Pointer values
// and named scalar types are dereferenced; integers retain exact precision.
// Maps, slices, custom database values and SQL expressions are not supported.
func NormalizeValue(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	v := reflect.ValueOf(value)
	for depth := 0; v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface; depth++ {
		if depth > 64 {
			return nil, fmt.Errorf("%w: pointer nesting too deep", ErrUnsupportedValue)
		}
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}
	if v.Type() == reflect.TypeOf(time.Time{}) {
		return v.Interface().(time.Time), nil
	}
	switch v.Kind() {
	case reflect.Bool:
		return v.Bool(), nil
	case reflect.String:
		return v.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint(), nil
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		if !math.IsNaN(f) && !math.IsInf(f, 0) {
			return f, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedValue, v.Type())
}

// ValuesEqual compares supported policy scalars. Integer widths and signedness
// may differ, but strings are never parsed as numbers and floats are not coerced
// to integers. Times compare by instant. Neither value is included in errors.
func ValuesEqual(a, b any) (bool, error) {
	x, err := NormalizeValue(a)
	if err != nil {
		return false, err
	}
	y, err := NormalizeValue(b)
	if err != nil {
		return false, err
	}
	if x == nil || y == nil {
		return x == nil && y == nil, nil
	}
	switch v := x.(type) {
	case int64:
		if u, ok := y.(uint64); ok {
			return v >= 0 && uint64(v) == u, nil
		}
	case uint64:
		if i, ok := y.(int64); ok {
			return i >= 0 && v == uint64(i), nil
		}
	case time.Time:
		u, ok := y.(time.Time)
		return ok && v.Equal(u), nil
	}
	return x == y, nil
}
