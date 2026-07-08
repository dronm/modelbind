package modelbind

import (
	"encoding/json"
	"reflect"

	"github.com/dronm/modelbind/metadata"
)

func dbFieldValue(funcName string, fieldID string, field reflect.Value, fieldMd metadata.FieldValidator) (any, error) {
	if isNilReflectValue(field) {
		return nil, nil
	}

	if fieldMd.DataType() == metadata.FieldTypeUndefined {
		b, err := json.Marshal(field.Interface())
		if err != nil {
			return nil, NewMessageError(
				MsgJSONMarshalFailed,
				map[string]any{
					"Func":  funcName,
					"Field": fieldID,
					"Error": err.Error(),
				},
			)
		}

		return b, nil
	}

	// Convert named string aliases to plain string before passing them to pgx.
	// This covers PostgreSQL enum values represented as Go string aliases.
	if fieldMd.DataType() == metadata.FieldTypeText {
		value := field
		if value.Kind() == reflect.Pointer {
			value = value.Elem()
		}

		if value.IsValid() && value.Kind() == reflect.String {
			return value.String(), nil
		}
	}

	return field.Interface(), nil
}
