package metadata

import (
	"fmt"
	"reflect"
)

// FieldValidator is a base interface.
type FieldValidator interface {
	ModelID() string
	ID() string
	Alias() string
	SetAlias(string)
	Descr() string
	Required() bool
	SetRequired(bool)
	DataType() FieldDataType
	PrimaryKey() bool
	SetPrimaryKey(bool)
	SrvCalc() bool
	SetSrvCalc(bool)
	Validate(field reflect.Value) (bool, error) //true if set
	ValidateRequired(field reflect.Value) error
}

func ValidateModel(model any, fieldTagName string) error {
	modelValue := reflect.ValueOf(model)
	if modelValue.Kind() == reflect.Pointer {
		modelValue = modelValue.Elem() // Dereference pointer types
	}
	if modelValue.Kind() != reflect.Struct {
		return NewMessageError(
			MsgModelNotPointerOrStruct,
			map[string]any{
				"Func": "ValidateModel",
			},
		)
	}

	modelMd, err := NewModelMetadata(model)
	if err != nil {
		return err
	}

	for _, fieldMd := range modelMd.Fields {
		field := modelValue.FieldByName(fieldMd.ModelID())
		if !field.IsValid() {
			return fmt.Errorf("reflect.IsValid() failed for field %s", fieldMd.ModelID())
		}
		if _, err := fieldMd.Validate(field); err != nil {
			return err
		}

		if err := fieldMd.ValidateRequired(field); err != nil {
			return err
		}
	}

	return nil
}
