package metadata

import (
	"math"
	"reflect"
)

type FieldFloatMetadata struct {
	FieldMetadata
	maxValue  *float64
	minValue  *float64
	precision int64
}

func NewFieldFloatMedata(modelFieldID, id string) *FieldFloatMetadata {
	return &FieldFloatMetadata{
		FieldMetadata: FieldMetadata{
			modelID:  modelFieldID,
			id:       id,
			dataType: FieldTypeFloat,
		},
	}
}

func (f FieldFloatMetadata) MaxValue() *float64 {
	return f.maxValue
}

func (f FieldFloatMetadata) MinValue() *float64 {
	return f.minValue
}

func (f FieldFloatMetadata) Precision() int64 {
	return f.precision
}

func castFloat(fieldID string, field reflect.Value) (float64, error) {
	if !field.IsValid() {
		return 0, NewMessageError(
			MsgValidationCast,
			map[string]any{
				"Field": fieldID,
				"Type":  "float64",
			},
		)
	}

	for field.Kind() == reflect.Interface {
		if field.IsNil() {
			return 0, NewMessageError(
				MsgValidationCast,
				map[string]any{
					"Field": fieldID,
					"Type":  "float64",
				},
			)
		}

		field = field.Elem()

		if !field.IsValid() {
			return 0, NewMessageError(
				MsgValidationCast,
				map[string]any{
					"Field": fieldID,
					"Type":  "float64",
				},
			)
		}
	}

	switch field.Kind() {
	case reflect.Float32, reflect.Float64:
		return field.Float(), nil

	default:
		return 0, NewMessageError(
			MsgValidationCast,
			map[string]any{
				"Field": fieldID,
				"Type":  "float64",
			},
		)
	}
}

func (f FieldFloatMetadata) Validate(field reflect.Value) (bool, error) {
	if !field.IsValid() {
		return true, NewMessageError(
			MsgValidationCast,
			map[string]any{
				"Field": f.ModelID(),
				"Type":  "float64",
			},
		)
	}

	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return false, nil
		}

		field = field.Elem()

		if !field.IsValid() {
			return true, NewMessageError(
				MsgValidationCast,
				map[string]any{
					"Field": f.ModelID(),
					"Type":  "float64",
				},
			)
		}
	}

	val, err := castFloat(f.ModelID(), field)
	if err != nil {
		return true, err
	}

	if f.MinValue() != nil && val < *f.MinValue() {
		return true, NewMessageError(
			MsgValueTooSmall,
			map[string]any{
				"Field": f.Descr(),
				"Min":   *f.MinValue(),
				"Value": val,
			},
		)
	}

	if f.MaxValue() != nil && val > *f.MaxValue() {
		return true, NewMessageError(
			MsgValueTooBig,
			map[string]any{
				"Field": f.Descr(),
				"Max":   *f.MaxValue(),
				"Value": val,
			},
		)
	}

	prec := f.Precision()
	if prec > 0 {
		factor := math.Pow(10, float64(prec))
		roundedValue := math.Round(val*factor) / factor

		if math.Abs(val-roundedValue) >= 1e-9 {
			return true, NewMessageError(
				MsgFloatPrecision,
				map[string]any{
					"Field":     f.Descr(),
					"Precision": prec,
					"Value":     val,
				},
			)
		}
	}

	return true, nil
}
