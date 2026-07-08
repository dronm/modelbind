package metadata

import (
	"reflect"
	"strings"
	"time"
)

type FieldDataType byte

const (
	FieldTypeUndefined FieldDataType = iota
	FieldTypeBool
	FieldTypeInt
	FieldTypeDate
	FieldTypeDatetime
	FieldTypeDatetimeTZ
	FieldTypeTime
	FieldTypeFloat
	FieldTypeText
	FieldTypeArray  // slice
	FieldTypeObjecT // struct
)

func ParseDateFieldType(fieldTypeName string) FieldDataType {
	switch strings.ToLower(fieldTypeName) {
	case "date":
		return FieldTypeDate
	case "time":
		return FieldTypeTime
	case "datetime", "date_time", "timestamp":
		return FieldTypeDatetime
	case "datetimetz", "datetime_tz", "datetimez", "timestamp_tz", "timestamptz":
		return FieldTypeDatetimeTZ
	}

	return FieldTypeUndefined
}

func ParseFieldType(fieldTypeName string) FieldDataType {
	switch fieldTypeName {
	case "FieldFloat", "float64", "float32":
		return FieldTypeFloat
	case "FieldInt", "int", "int0", "int8", "int16", "int32", "int64":
		return FieldTypeInt
	case "FieldText", "string":
		return FieldTypeText
	}
	return FieldTypeUndefined
}

func ParseReflectFieldType(fieldType reflect.Type) FieldDataType {
	if fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}

	if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
		elemType := fieldType.Elem()
		if elemType.Kind() == reflect.Pointer {
			elemType = elemType.Elem()
		}

		switch elemType.Kind() {
		case reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
			reflect.Float32, reflect.Float64,
			reflect.String:
			return FieldTypeArray
		}

		return FieldTypeUndefined
	}

	if fieldType == reflect.TypeOf(time.Time{}) {
		return FieldTypeDatetime
	}

	switch fieldType.Kind() {
	case reflect.Bool:
		return FieldTypeBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return FieldTypeInt
	case reflect.Float32, reflect.Float64:
		return FieldTypeFloat
	case reflect.String:
		return FieldTypeText
	}

	return FieldTypeUndefined
}

type FieldMetadata struct {
	modelID    string // real structure ID
	id         string // database field id, json/xml tag
	alias      string
	required   bool
	dataType   FieldDataType
	primaryKey bool
	srvCalc    bool // server calculated field, return to client on insert
	// like auto inc for example
}

func NewFieldArrayMetadata(modelFieldID, id string) *FieldMetadata {
	return &FieldMetadata{modelID: modelFieldID, id: id, dataType: FieldTypeArray}
}

func (f FieldMetadata) ModelID() string {
	return f.modelID
}

func (f FieldMetadata) Alias() string {
	return f.alias
}

func (f *FieldMetadata) SetAlias(v string) {
	f.alias = v
}

func (f FieldMetadata) Required() bool {
	return f.required
}

func (f *FieldMetadata) SetRequired(v bool) {
	f.required = v
}

func (f FieldMetadata) PrimaryKey() bool {
	return f.primaryKey
}

func (f *FieldMetadata) SetPrimaryKey(v bool) {
	f.primaryKey = v
}

func (f FieldMetadata) SrvCalc() bool {
	return f.srvCalc
}

func (f *FieldMetadata) SetSrvCalc(v bool) {
	f.srvCalc = v
}

func (f FieldMetadata) ID() string {
	return f.id
}

func (f FieldMetadata) DataType() FieldDataType {
	return f.dataType
}

// Descr returns alias OR id
func (f FieldMetadata) Descr() string {
	if f.alias != "" {
		return f.alias
	}
	return f.id
}

type NullableField interface {
	// GetValue() interface{}
	IsSet() bool
	IsNull() bool
}

func (f FieldMetadata) ValidateRequired(field reflect.Value) error {
	modelField, ok := field.Interface().(NullableField)
	if !ok {
		// standart type, check for ptr
		if field.Kind() == reflect.Pointer && field.IsNil() && f.Required() {
			return NewMessageError(
				MsgValueRequired,
				map[string]any{
					"Field": f.Descr(),
				},
			)
		}
		return nil
	}
	if f.Required() && (!modelField.IsSet() || modelField.IsNull()) {
		return NewMessageError(
			MsgValueRequired,
			map[string]any{
				"Field": f.Descr(),
			},
		)
	}

	return nil
}

// Validate returns true if value is set.
// Value is set/unset can only be checked for pointers.
func (f FieldMetadata) Validate(field reflect.Value) (bool, error) {
	if field.Type().Kind() == reflect.Pointer && field.IsValid() {
		return !field.IsNil(), nil
	}
	return true, nil
}
