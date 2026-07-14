package modelbind

import (
	"reflect"

	"github.com/dronm/modelbind/metadata"
	"github.com/dronm/modelbind/types"
)

func BindInsertModelInput[T any](input ModelInput[T], dbInsert types.DBInserter) error {
	return bindInsertModelInput("BindInsertModelInput", input, dbInsert)
}

func BindUpdateModelInput[T any](keyModel any, input ModelInput[T], dbUpdate types.DBUpdater) error {
	return bindUpdateModelInput("BindUpdateModelInput", keyModel, input, dbUpdate)
}

func ValidateModelInput[T any](input ModelInput[T], forInsert bool) error {
	return validateModelInput("ValidateModelInput", input, forInsert)
}

func (in ModelInput[T]) BindInsert(dbInsert types.DBInserter) error {
	return BindInsertModelInput(in, dbInsert)
}

func (in ModelInput[T]) BindUpdate(keyModel any, dbUpdate types.DBUpdater) error {
	return BindUpdateModelInput(keyModel, in, dbUpdate)
}

func validateModelInput[T any](funcName string, input ModelInput[T], forInsert bool) error {
	modelType, modelVal, err := modelPointerValue(input.Model, funcName)
	if err != nil {
		return err
	}

	modelMd, err := metadata.NewModelMetadata(input.Model)
	if err != nil {
		return err
	}

	validationErr := &ValidationError{}
	for i := 0; i < modelVal.NumField(); i++ {
		fieldType := modelType.Field(i)
		fieldID := metadata.FieldAnnotationValue(fieldType, metadata.FieldAnnotationName)
		if fieldID == "-" || fieldID == "" {
			continue
		}

		fieldMd, ok := modelMd.Fields[fieldID]
		if !ok {
			return NewMessageError(
				MsgMetadataFieldNotFound,
				map[string]any{
					"Func":  funcName,
					"Field": fieldID,
				},
			)
		}

		if fieldMd.SrvCalc() {
			continue
		}

		field := modelVal.Field(i)
		if !field.CanInterface() {
			return NewMessageError(
				MsgModelFieldNotInterface,
				map[string]any{
					"Field": fieldID,
				},
			)
		}

		if input.AbsentFields.IsTracked() && input.AbsentFields.IsAbsent(fieldID) {
			if forInsert && fieldMd.Required() {
				validationErr.Add(NewMessageError(
					metadata.MsgValueRequired,
					map[string]any{
						"Field": fieldMd.Descr(),
					},
				))
			}

			continue
		}

		isSet, err := fieldMd.Validate(field)
		if err != nil {
			validationErr.Add(err)
			continue
		}

		if !isSet && !forInsert {
			if err := fieldMd.ValidateRequired(field); err != nil {
				validationErr.Add(err)
			}

			continue
		}

		if err := fieldMd.ValidateRequired(field); err != nil {
			validationErr.Add(err)
			continue
		}
	}

	if validationErr.HasErrors() {
		return validationErr
	}

	return nil
}

func bindInsertModelInput[T any](funcName string, input ModelInput[T], dbInsert types.DBInserter) error {
	modelType, modelVal, err := modelPointerValue(input.Model, funcName)
	if err != nil {
		return err
	}

	modelMd, err := metadata.NewModelMetadata(input.Model)
	if err != nil {
		return err
	}

	validationErr := &ValidationError{}
	for i := 0; i < modelVal.NumField(); i++ {
		fieldType := modelType.Field(i)
		fieldID := metadata.FieldAnnotationValue(fieldType, metadata.FieldAnnotationName)
		if fieldID == "-" || fieldID == "" {
			continue
		}

		field := modelVal.Field(i)
		if !field.CanInterface() {
			return NewMessageError(
				MsgModelFieldNotInterface,
				map[string]any{
					"Field": fieldID,
				},
			)
		}

		if !field.IsValid() {
			return NewMessageError(
				metadata.MsgModelInvalidField,
				map[string]any{
					"Field": fieldID,
				},
			)
		}

		fieldMd, ok := modelMd.Fields[fieldID]
		if !ok {
			return NewMessageError(
				MsgMetadataFieldNotFound,
				map[string]any{
					"Func":  funcName,
					"Field": fieldID,
				},
			)
		}

		if fieldMd.SrvCalc() {
			dbInsert.AddRetField(fieldID, field.Addr().Interface())
			continue
		}

		if input.AbsentFields.IsTracked() && input.AbsentFields.IsAbsent(fieldID) {
			if fieldMd.Required() {
				validationErr.Add(NewMessageError(
					metadata.MsgValueRequired,
					map[string]any{
						"Field": fieldMd.Descr(),
					},
				))
			}

			continue
		}

		isSet, err := fieldMd.Validate(field)
		if err != nil {
			validationErr.Add(err)
			continue
		}

		if err := fieldMd.ValidateRequired(field); err != nil {
			validationErr.Add(err)
			continue
		}

		if !isSet {
			dbInsert.AddField(fieldID, nil)
			continue
		}

		value, err := dbFieldValue(funcName, fieldID, field, fieldMd)
		if err != nil {
			validationErr.Add(err)
			continue
		}

		dbInsert.AddField(fieldID, value)
	}

	if validationErr.HasErrors() {
		return validationErr
	}

	if insert, ok := dbInsert.(interface{ InsertFieldLen() int }); ok && insert.InsertFieldLen() == 0 {
		return NewMessageError(
			MsgNoInsertFields,
			map[string]any{
				"Func": funcName,
			},
		)
	}

	return nil
}

func bindUpdateModelInput[T any](funcName string, keyModel any, input ModelInput[T], dbUpdate types.DBUpdater) error {
	if err := ModelToDBFilters(keyModel, dbUpdate.Filter(), types.SQLFilterOperatorEq, types.SQLFilterJoinAnd, ""); err != nil {
		return err
	}

	modelType, modelVal, err := modelPointerValue(input.Model, funcName)
	if err != nil {
		return err
	}

	modelMd, err := metadata.NewModelMetadata(input.Model)
	if err != nil {
		return err
	}

	validationErr := &ValidationError{}
	for i := 0; i < modelVal.NumField(); i++ {
		fieldType := modelType.Field(i)
		fieldID := metadata.FieldAnnotationValue(fieldType, metadata.FieldAnnotationName)
		if fieldID == "-" || fieldID == "" {
			continue
		}

		if input.AbsentFields.IsTracked() && input.AbsentFields.IsAbsent(fieldID) {
			continue
		}

		field := modelVal.Field(i)
		if !field.CanInterface() {
			return NewMessageError(
				MsgModelFieldNotInterface,
				map[string]any{
					"Field": fieldID,
				},
			)
		}

		if !field.IsValid() {
			return NewMessageError(
				metadata.MsgModelInvalidField,
				map[string]any{
					"Field": fieldID,
				},
			)
		}

		fieldMd, ok := modelMd.Fields[fieldID]
		if !ok {
			return NewMessageError(
				MsgMetadataFieldNotFound,
				map[string]any{
					"Func":  funcName,
					"Field": fieldID,
				},
			)
		}

		isSet, err := fieldMd.Validate(field)
		if err != nil {
			validationErr.Add(err)
			continue
		}

		if err := fieldMd.ValidateRequired(field); err != nil {
			validationErr.Add(err)
			continue
		}

		if !isSet {
			dbUpdate.AddField(fieldID, nil)
			continue
		}

		value, err := dbFieldValue(funcName, fieldID, field, fieldMd)
		if err != nil {
			validationErr.Add(err)
			continue
		}

		dbUpdate.AddField(fieldID, value)
	}

	if validationErr.HasErrors() {
		return validationErr
	}

	if assigner, ok := dbUpdate.(interface{ AssignerLen() int }); ok && assigner.AssignerLen() == 0 {
		return NewMessageError(
			MsgNoUpdateFields,
			map[string]any{
				"Func": funcName,
			},
		)
	}

	return nil
}

func isNilReflectValue(field reflect.Value) bool {
	if !field.IsValid() {
		return true
	}

	switch field.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
		return field.IsNil()
	default:
		return false
	}
}
