// Package metadata
package metadata

import (
	"reflect"
	"sync"
	"time"
)

var ControledTags []string // defined at startup, tags to cache

// ModelMetadata holds the mapping of json tags to field types.
type ModelMetadata struct {
	ID           string
	Fields       map[string]FieldValidator    // FieldMetadata, key is an sql field
	FieldList    []string                     // structure field Names in original order,
	FieldTagList []string                     // sql fields in original order for retrieving metadata from Firlds
	Tags         map[string]map[string]string // controled tag values. List of controled tags is defined in ControledTags
}

// cache to store metadata for different types.
var (
	metadataCache = make(map[reflect.Type]ModelMetadata)
	cacheMutex    sync.RWMutex
)

// NewModelMetadata returns the Metadata structure for a given type.
// Internally it stores a race-safe cache.
func NewModelMetadata(model any) (*ModelMetadata, error) {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Pointer {
		modelType = modelType.Elem() // Dereference pointer types
	}

	if modelType.Kind() != reflect.Struct {
		return nil, NewMessageError(
			MsgModelNotPointer,
			map[string]any{
				"Func": "NewModelMetadata",
			},
		)
	}

	// Check the cache for existing metadata
	cacheMutex.RLock()
	if meta, found := metadataCache[modelType]; found {
		cacheMutex.RUnlock()
		return &meta, nil
	}
	cacheMutex.RUnlock()

	// Build metadata if not found in cache
	meta := ModelMetadata{ID: modelType.String(), Fields: make(map[string]FieldValidator)}
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		fieldID := field.Name
		fieldTagVal := FieldAnnotationValue(field, FieldAnnotationName)

		// Skip fields without a FieldFilterAnnotationName tag
		if fieldTagVal == "-" || fieldTagVal == "" {
			// check filter tag
			fieldTagVal = FieldAnnotationValue(field, FieldFilterAnnotationName)
			if fieldTagVal == "-" || fieldTagVal == "" {
				continue
			}
		}

		for _, tag := range ControledTags {
			tagVal := field.Tag.Get(tag)
			if tagVal != "" {
				if meta.Tags == nil {
					meta.Tags = make(map[string]map[string]string)
				}
				if meta.Tags[fieldID] == nil {
					meta.Tags[fieldID] = make(map[string]string)
				}
				meta.Tags[fieldID][tag] = tagVal
			}
		}

		meta.FieldList = append(meta.FieldList, fieldID)
		meta.FieldTagList = append(meta.FieldTagList, fieldTagVal)

		fieldType := field.Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}

		dateType := ParseDateFieldType(annotationTagStringVal(field, AnnotTagDateType))
		if dateType != FieldTypeUndefined {
			meta.Fields[fieldTagVal] = NewFieldDateMedata(fieldID, fieldTagVal, dateType)
		} else if fieldType == reflect.TypeOf(time.Time{}) {
			meta.Fields[fieldTagVal] = NewFieldDateMedata(fieldID, fieldTagVal, FieldTypeDatetime)
		} else {
			switch ParseReflectFieldType(fieldType) {
			case FieldTypeArray:
				meta.Fields[fieldTagVal] = NewFieldArrayMetadata(fieldID, fieldTagVal)

			case FieldTypeBool:
				// no constraints
				meta.Fields[fieldTagVal] = NewFieldBoolMedata(fieldID, fieldTagVal)

			case FieldTypeText:
				validator := NewFieldTextMedata(fieldID, fieldTagVal)
				if err := setTextValidatorConstraints(field, validator); err != nil {
					return nil, err
				}
				meta.Fields[fieldTagVal] = validator

			case FieldTypeInt:
				validator := NewFieldIntMedata(fieldID, fieldTagVal)
				if err := setIntValidatorConstraints(field, validator); err != nil {
					return nil, err
				}
				meta.Fields[fieldTagVal] = validator

			case FieldTypeFloat:
				validator := NewFieldFloatMedata(fieldID, fieldTagVal)
				if err := setFloatValidatorConstraints(field, validator); err != nil {
					return nil, err
				}
				meta.Fields[fieldTagVal] = validator

			default:
				meta.Fields[fieldTagVal] = &FieldMetadata{modelID: fieldID, id: fieldTagVal}
			}
		}

		// common tags
		meta.Fields[fieldTagVal].SetAlias(annotationTagStringVal(field, AnnotTagAlias))
		meta.Fields[fieldTagVal].SetRequired(annotationTagBoolVal(field, AnnotTagRequired))
		meta.Fields[fieldTagVal].SetPrimaryKey(annotationTagBoolVal(field, AnnotTagPrimKey))
		meta.Fields[fieldTagVal].SetSrvCalc(annotationTagBoolVal(field, AnnotTagSrvCalc))
	}

	// Save to cache
	cacheMutex.Lock()
	metadataCache[modelType] = meta
	cacheMutex.Unlock()

	return &meta, nil
}
