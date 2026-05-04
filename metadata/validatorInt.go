package metadata

import (
	"fmt"
	"reflect"
)

type FieldIntMetadata struct {
	FieldMetadata
	maxValue *int64
	minValue *int64
}

func NewFieldIntMedata(modelFieldID, id string) *FieldIntMetadata {
	return &FieldIntMetadata{
		FieldMetadata: FieldMetadata{
			modelID:  modelFieldID,
			id:       id,
			dataType: FieldTypeInt,
		},
	}
}

func (f FieldIntMetadata) MaxValue() *int64 {
	return f.maxValue
}

func (f FieldIntMetadata) MinValue() *int64 {
	return f.minValue
}

func castInt(fieldID string, field reflect.Value) (int64, error) {
	if !field.IsValid() {
		return 0, NewMessageError(
			MsgValidationCast,
			map[string]any{
				"Field": fieldID,
				"Type":  "int64",
			},
		)
	}

	for field.Kind() == reflect.Interface {
		if field.IsNil() {
			return 0, NewMessageError(
				MsgValidationCast,
				map[string]any{
					"Field": fieldID,
					"Type":  "int64",
				},
			)
		}

		field = field.Elem()

		if !field.IsValid() {
			return 0, NewMessageError(
				MsgValidationCast,
				map[string]any{
					"Field": fieldID,
					"Type":  "int64",
				},
			)
		}
	}

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int(), nil

	case reflect.Float64:
		val := field.Float()
		return int64(val), nil

	default:
		return 0, NewMessageError(
			MsgValidationCast,
			map[string]any{
				"Field": fieldID,
				"Type":  "int64",
			},
		)
	}
}

func (f FieldIntMetadata) ValidateSlice(fieldID string, field reflect.Value) error {
	if !field.IsValid() || field.Kind() != reflect.Slice {
		return NewMessageError(
			MsgFieldNotSlice,
			map[string]any{
				"Field": fieldID,
			},
		)
	}

	for i := 0; i < field.Len(); i++ {
		elem := field.Index(i)

		val, err := castInt(fmt.Sprintf("%s[%d]", fieldID, i), elem)
		if err != nil {
			return err
		}

		if err := f.CheckValue(val); err != nil {
			return err
		}
	}

	return nil
}

func (f FieldIntMetadata) Validate(field reflect.Value) (bool, error) {
	if !field.IsValid() {
		return true, NewMessageError(
			MsgValidationCast,
			map[string]any{
				"Field": f.ModelID(),
				"Type":  "int64",
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
					"Type":  "int64",
				},
			)
		}
	}

	if field.Kind() == reflect.Slice {
		if err := f.ValidateSlice(f.ModelID(), field); err != nil {
			return true, err
		}

		return true, nil
	}

	val, err := castInt(f.ModelID(), field)
	if err != nil {
		return true, err
	}

	return true, f.CheckValue(val)
}

// CheckValue does the actual validation of the given int64 value.
func (f FieldIntMetadata) CheckValue(val int64) error {
	if f.MinValue() != nil && val < *f.MinValue() {
		return NewMessageError(
			MsgValueTooSmall,
			map[string]any{
				"Field": f.Descr(),
				"Min":   *f.MinValue(),
				"Value": val,
			},
		)
	}

	if f.MaxValue() != nil && val > *f.MaxValue() {
		return NewMessageError(
			MsgValueTooBig,
			map[string]any{
				"Field": f.Descr(),
				"Max":   *f.MaxValue(),
				"Value": val,
			},
		)
	}

	return nil
}
