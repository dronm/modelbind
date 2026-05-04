package metadata

import (
	"reflect"
)

type FieldBoolMetadata struct {
	FieldMetadata
}

func NewFieldBoolMedata(modelFieldID, id string) *FieldBoolMetadata {
	return &FieldBoolMetadata{
		FieldMetadata: FieldMetadata{
			modelID: modelFieldID, 
			id: id, dataType: FieldTypeBool,
		},
	}
}

func (f FieldBoolMetadata) Validate(field reflect.Value) (bool, error) {
	if field.Kind() == reflect.Pointer && field.IsNil() {
		// Pointer exists, but value is not set.
		// Required validation should be handled by ValidateRequired().
		return false, nil
	}

	if field.Kind() == reflect.Pointer {
		field = field.Elem()

		if !field.IsValid() {
			return true, NewMessageError(
				MsgValidationCast,
				map[string]any{
					"Field": f.ModelID(),
					"Type":  "bool",
				},
			)
		}
	}

	if field.Kind() != reflect.Bool {
		return true, NewMessageError(
			MsgValidationCast,
			map[string]any{
				"Field": f.ModelID(),
				"Type":  "bool",
			},
		)
	}

	return true, nil
}
