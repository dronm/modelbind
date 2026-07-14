// Package modelbind provides helpers for binding model metadata to DB builders.
package modelbind

import (
	"reflect"

	"github.com/dronm/modelbind/metadata"
	"github.com/dronm/modelbind/types"
)

// modelPointerValue returns the concrete struct type and value for a non-nil
// pointer to a struct.
func modelPointerValue(model any, funcName string) (reflect.Type, reflect.Value, error) {
	modelVal := reflect.ValueOf(model)
	if !modelVal.IsValid() || modelVal.Kind() != reflect.Pointer || modelVal.IsNil() {
		return nil, reflect.Value{}, NewMessageError(
			metadata.MsgModelNotPointer,
			map[string]any{
				"Func": funcName,
			},
		)
	}

	modelType := reflect.TypeOf(model).Elem()
	modelVal = modelVal.Elem()
	if !modelVal.IsValid() || modelVal.Kind() != reflect.Struct {
		return nil, reflect.Value{}, NewMessageError(
			metadata.MsgModelNotPointer,
			map[string]any{
				"Func": funcName,
			},
		)
	}

	return modelType, modelVal, nil
}

// modelStructOrPointerValue returns the concrete struct type and value for a
// struct value or a non-nil pointer to a struct.
func modelStructOrPointerValue(model any, funcName string) (reflect.Type, reflect.Value, error) {
	modelVal := reflect.ValueOf(model)
	if !modelVal.IsValid() {
		return nil, reflect.Value{}, NewMessageError(
			metadata.MsgModelNotPointerOrStruct,
			map[string]any{
				"Func": funcName,
			},
		)
	}

	modelType := reflect.TypeOf(model)
	if modelVal.Kind() == reflect.Pointer {
		if modelVal.IsNil() {
			return nil, reflect.Value{}, NewMessageError(
				metadata.MsgModelNotPointerOrStruct,
				map[string]any{
					"Func": funcName,
				},
			)
		}

		modelType = modelType.Elem()
		modelVal = modelVal.Elem()
	}

	if !modelVal.IsValid() || modelVal.Kind() != reflect.Struct {
		return nil, reflect.Value{}, NewMessageError(
			metadata.MsgModelNotPointerOrStruct,
			map[string]any{
				"Func": funcName,
			},
		)
	}

	return modelType, modelVal, nil
}

// ModelToDBFilters converts annotated model fields into DB filters using the
// provided SQL operator, join mode, and table name.
func ModelToDBFilters(model any, filters types.DBFilters, operator types.SQLFilterOperator, join types.FilterJoin, table string) error {
	const funcName = "ModelToDBFilters"

	if filters == nil {
		return NewMessageError(
			MsgFilterNotInitialized,
			map[string]any{
				"Func": funcName,
			},
		)
	}

	modelType, modelVal, err := modelStructOrPointerValue(model, funcName)
	if err != nil {
		return err
	}

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

		filters.Add(table, fieldID, field.Interface(), operator, join)
	}

	return nil
}

// BindUpdateModel validates the update model and binds its set fields plus key
// filters to dbUpdate.
func BindUpdateModel(keyModel any, dbUpdate types.DBUpdater) error {
	return bindUpdateModel("BindUpdateModel", keyModel, dbUpdate)
}

// bindUpdateModel validates the update model, binds key filters, and adds all
// set update fields to dbUpdate.
func bindUpdateModel(funcName string, keyModel any, dbUpdate types.DBUpdater) error {
	if err := ModelToDBFilters(keyModel, dbUpdate.Filter(), types.SQLFilterOperatorEq, types.SQLFilterJoinAnd, ""); err != nil {
		return err
	}

	model := dbUpdate.Model()
	modelType, modelVal, err := modelPointerValue(model, funcName)
	if err != nil {
		return err
	}

	modelMd, err := metadata.NewModelMetadata(model)
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

		isSet, err := fieldMd.Validate(field)
		if err != nil {
			validationErr.Add(err)
			continue
		}

		if !isSet {
			continue
		}

		if err := fieldMd.ValidateRequired(field); err != nil {
			validationErr.Add(err)
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

// BindCollectionSelectModel parses collection params and binds select fields,
// aggregate fields, filters, sorters, and limit to dbSelect.
func BindCollectionSelectModel(dbSelect types.DBSelecter, params CollectionParams) error {
	return bindCollectionSelectModel("BindCollectionSelectModel", dbSelect, params)
}

// BindCollectionSelectModel parses collection parameters and binds select
// fields, aggregate fields, filters, sorters, and limit to dbSelect.
func bindCollectionSelectModel(funcName string, dbSelect types.DBSelecter, params CollectionParams) error {
	if err := ParseFilterParams(dbSelect.Model(), dbSelect.Filter(), params, ""); err != nil {
		return err
	}

	aggModel := dbSelect.Model().CollectionAgg()
	if aggModel != nil {
		aggModelType, aggModelVal, err := modelPointerValue(aggModel, funcName)
		if err != nil {
			return err
		}

		for i := 0; i < aggModelVal.NumField(); i++ {
			aggFieldType := aggModelType.Field(i)
			fieldID := metadata.FieldAnnotationValue(aggFieldType, metadata.FieldAnnotationName)
			if fieldID == "-" || fieldID == "" {
				return NewMessageError(
					MsgAggregationFieldNotDefined,
					map[string]any{
						"Func":  funcName,
						"Index": i,
					},
				)
			}

			aggFunc := aggFieldType.Tag.Get(metadata.AnnotTagAgg)
			if aggFunc == "" {
				return NewMessageError(
					MsgAggregationFunctionNotDefined,
					map[string]any{
						"Func":  funcName,
						"Field": fieldID,
					},
				)
			}

			field := aggModelVal.Field(i)
			dbSelect.AddAggField(aggFunc, field.Addr().Interface())
		}
	}

	if err := ParseSorterParams(dbSelect.Model(), dbSelect.Sorter(), params); err != nil {
		return err
	}

	if err := ParseLimitParams(dbSelect.Limit(), params); err != nil {
		return err
	}

	return bindSelectModel(funcName, dbSelect.(types.PrepareModel), dbSelect.Model())
}

// bindSelectModel binds annotated model fields as scan destinations for a
// select-like query builder.
func bindSelectModel(funcName string, selectModel types.PrepareModel, model any) error {
	modelType, modelVal, err := modelPointerValue(model, funcName)
	if err != nil {
		return err
	}

	for i := 0; i < modelVal.NumField(); i++ {
		fieldType := modelType.Field(i)
		fieldID := metadata.FieldAnnotationValue(fieldType, metadata.FieldAnnotationName)
		if fieldID == "-" || fieldID == "" {
			continue
		}

		field := modelVal.Field(i)
		selectModel.AddField(fieldID, field.Addr().Interface())
	}

	return nil
}

// BindDetailSelectModel binds key filters and scan destinations for a detail
// select.
func BindDetailSelectModel(keyModel any, dbSelect types.DBDetailSelecter) error {
	return bindDetailSelectModel("BindDetailSelectModel", keyModel, dbSelect)
}

// BindDetailSelectModel binds key filters and scan destinations for a detail
// select query.
func bindDetailSelectModel(funcName string, keyModel any, dbSelect types.DBDetailSelecter) error {
	filters := dbSelect.Filter()
	if err := ModelToDBFilters(keyModel, filters, types.SQLFilterOperatorEq, types.SQLFilterJoinAnd, ""); err != nil {
		return err
	}

	if filters.Len() == 0 {
		return NewMessageError(
			MsgNoKeys,
			map[string]any{
				"Func": funcName,
			},
		)
	}

	return bindSelectModel(funcName, dbSelect, dbSelect.Model())
}

// BindInsertModel validates the insert model and binds insert fields plus
// server-calculated return fields to dbInsert.
func BindInsertModel(dbInsert types.DBInserter) error {
	return bindInsertModel("BindInsertModel", dbInsert)
}

// bindInsertModel validates the insert model, adds present insert fields, and
// registers server-calculated fields as return scan destinations.
func bindInsertModel(funcName string, dbInsert types.DBInserter) error {
	model := dbInsert.Model()
	modelType, modelVal, err := modelPointerValue(model, funcName)
	if err != nil {
		return err
	}

	modelMd, err := metadata.NewModelMetadata(model)
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

		if err := fieldMd.ValidateRequired(field); err != nil {
			validationErr.Add(err)
			continue
		}

		if fieldMd.SrvCalc() {
			dbInsert.AddRetField(fieldID, field.Addr().Interface())
			continue
		}

		present, err := fieldMd.Validate(field)
		if err != nil {
			validationErr.Add(err)
			continue
		}

		if !present {
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

// ValidateModel checks the given model.
//
// If forInsert is true, required fields must be set. If forInsert is false,
// missing required fields are ignored, but explicitly set invalid values still
// fail validation.
func ValidateModel(model any, forInsert bool) error {
	const funcName = "ValidateModel"

	modelType, modelVal, err := modelPointerValue(model, funcName)
	if err != nil {
		return err
	}

	modelMd, err := metadata.NewModelMetadata(model)
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
			continue
		}

		isSet, err := fieldMd.Validate(field)
		if err != nil {
			validationErr.Add(err)
			continue
		}

		if !isSet && !forInsert {
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
