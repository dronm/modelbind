package metadata

import (
	"reflect"
	"strconv"
	"strings"
)

// Enums is a glabal variable holding enum values.
// Enum is a preset list of values to check text fields.
// Variable is not threadsafe. It can only be set at startup,
// before using the package concurrently.
var Enums map[string][]string // Key is a enum ID, values is an array of possible values.

// FieldAnnotationName  is a glabal variable that can be set at startup.
// This annotation is used for select and filter. If not set or set to "-" the
// field is not included in select statement but can be filtered if
// FieldFilterAnnotationName is set.
var FieldAnnotationName = "json"

// FieldFilterAnnotationName is used for filtering. Columns marked with this
// tag are not included in select, but can be filtered.
var FieldFilterAnnotationName = "f"

// ValListSeparator is used as a separator between list values.
// This variable must be set at startup.
var ValListSeparator = "@@"

// All possible annotations.
const (
	// common
	AnnotTagAlias       = "alias"
	AnnotTagRequired    = "required"
	AnnotTagDBRequired = "dbRequired" // required for Database, can be set with trigger/autoinc
	AnnotTagPrimKey    = "primaryKey"
	AnnotTagSrvCalc    = "srvCalc" // server initialized field on insert
	AnnotTagDateType   = "dateType" // date, time, datetime, datetime_tz

	//
	AnnotTagAgg = "agg" // aggregation function, like agg:"count(*)" for agg models

	// text
	AnnotTagMaxLen  = "max"
	AnnotTagMinLen  = "min"
	AnnotTagFixLen  = "fix"
	AnnotTagRegExp  = "regExp"
	AnnotTagValList = "valList" // separated list of values, separator is set at startup in ValListSeparator

	AnnotTagEnum = "enum" // check against predefined list of value,
	// list is set at startup and stored in global variable

	// number
	AnnotTagMaxVal = "max"
	AnnotTagMinVal = "min"
	AnnotTagFixVal = "fix"
)

func annotationTagBoolVal(fieldType reflect.StructField, tagName string) bool {
	_, present := fieldType.Tag.Lookup(tagName)
	return present
}

func annotationTagStringVal(fieldType reflect.StructField, tagName string) string {
	return fieldType.Tag.Get(tagName)
}

func annotationTagIntVal(fieldType reflect.StructField, tagName string) (*int64, error) {
	tagVal := fieldType.Tag.Get(tagName)
	if tagVal == "" {
		return nil, nil
	}
	iVal, err := strconv.ParseInt(tagVal, 10, 64)
	if err != nil {
		return nil, err
	}
	return &iVal, nil
}

func annotationTagFloatVal(fieldType reflect.StructField, tagName string) (*float64, error) {
	tagVal := fieldType.Tag.Get(tagName)
	if tagVal == "" {
		return nil, nil
	}
	fVal, err := strconv.ParseFloat(tagVal, 64)
	if err != nil {
		return nil, err
	}
	return &fVal, nil
}

func setTextValidatorConstraints(field reflect.StructField, validator *FieldTextMetadata) error {
	tagVal, err := annotationTagIntVal(field, AnnotTagMaxLen)
	if err != nil {
		return err
	}
	validator.maxLength = tagVal

	tagVal, err = annotationTagIntVal(field, AnnotTagMinLen)
	if err != nil {
		return err
	}
	validator.minLength = tagVal

	tagVal, err = annotationTagIntVal(field, AnnotTagFixLen)
	if err != nil {
		return err
	}
	validator.fixLength = tagVal

	if tagVal := annotationTagStringVal(field, AnnotTagRegExp); tagVal != "" {
		validator.regExp = tagVal
	}

	if tagVal := annotationTagStringVal(field, AnnotTagEnum); tagVal != "" {
		if vals, ok := Enums[tagVal]; ok {
			// enum exists
			validator.valList = vals
		}
	}

	if tagVal := annotationTagStringVal(field, AnnotTagValList); tagVal != "" {
		validator.valList = strings.Split(tagVal, ValListSeparator)
	}

	return nil
}

func setIntValidatorConstraints(field reflect.StructField, validator *FieldIntMetadata) error {
	tagVal, err := annotationTagIntVal(field, AnnotTagMinVal)
	if err != nil {
		return err
	}
	validator.minValue = tagVal

	tagVal, err = annotationTagIntVal(field, AnnotTagMaxVal)
	if err != nil {
		return err
	}
	validator.maxValue = tagVal

	return nil
}

func setFloatValidatorConstraints(field reflect.StructField, validator *FieldFloatMetadata) error {
	tagVal, err := annotationTagFloatVal(field, AnnotTagMinVal)
	if err != nil {
		return err
	}
	validator.minValue = tagVal

	tagVal, err = annotationTagFloatVal(field, AnnotTagMaxVal)
	if err != nil {
		return err
	}
	validator.maxValue = tagVal

	return nil
}
