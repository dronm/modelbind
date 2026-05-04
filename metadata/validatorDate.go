package metadata

import (
	"reflect"
	"time"
)

type FieldDateMetadata struct {
	FieldMetadata
}

func NewFieldDateMedata(modelFieldID, id string, dataType FieldDataType) *FieldDateMetadata {
	return &FieldDateMetadata{
		FieldMetadata: FieldMetadata{
			modelID:  modelFieldID,
			id:       id,
			dataType: dataType,
		},
	}
}

func (f FieldDateMetadata) Validate(field reflect.Value) (bool, error) {
	if !field.IsValid() {
		return true, f.castError()
	}

	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return false, nil
		}

		field = field.Elem()
		if !field.IsValid() {
			return true, f.castError()
		}
	}

	for field.Kind() == reflect.Interface {
		if field.IsNil() {
			return false, nil
		}

		field = field.Elem()
		if !field.IsValid() {
			return true, f.castError()
		}
	}

	if txtField, ok := field.Interface().(string); ok {
		if txtField == "" {
			return false, nil
		}

		if err := f.validateString(txtField); err != nil {
			return true, err
		}

		return true, nil
	}

	timeField, ok := field.Interface().(time.Time)
	if !ok {
		return true, f.castError()
	}

	if timeField.IsZero() {
		return false, nil
	}

	if f.DataType() == FieldTypeDate {
		if timeField.Hour() != 0 || timeField.Minute() != 0 || timeField.Second() != 0 || timeField.Nanosecond() != 0 {
			return true, NewMessageError(
				MsgValidationCast,
				map[string]any{
					"Field": f.ModelID(),
					"Type":  f.expectedTypeName(),
				},
			)
		}
	}

	return true, nil
}

func (f FieldDateMetadata) validateString(val string) error {
	var layouts []string

	switch f.DataType() {
	case FieldTypeDate:
		layouts = []string{"2006-01-02"}

	case FieldTypeTime:
		layouts = []string{"15:04"}

	case FieldTypeDatetime:
		layouts = []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
		}

	case FieldTypeDatetimeTZ:
		layouts = []string{
			time.RFC3339,
			time.RFC3339Nano,
		}

	default:
		return f.castError()
	}

	for _, layout := range layouts {
		if _, err := time.Parse(layout, val); err == nil {
			return nil
		}
	}

	return NewMessageError(
		MsgValidationCast,
		map[string]any{
			"Field": f.ModelID(),
			"Type":  f.expectedTypeName(),
			"Value": val,
		},
	)
}

func (f FieldDateMetadata) castError() error {
	return NewMessageError(
		MsgValidationCast,
		map[string]any{
			"Field": f.ModelID(),
			"Type":  f.expectedTypeName(),
		},
	)
}

func (f FieldDateMetadata) expectedTypeName() string {
	switch f.DataType() {
	case FieldTypeDate:
		return "date"
	case FieldTypeTime:
		return "time"
	case FieldTypeDatetime:
		return "datetime"
	case FieldTypeDatetimeTZ:
		return "datetime with timezone"
	default:
		return "time.Time"
	}
}
