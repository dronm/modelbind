package metadata

import (
	"reflect"
	"regexp"
	"slices"
)

type FieldTextMetadata struct {
	FieldMetadata
	maxLength *int64
	minLength *int64
	fixLength *int64
	regExp    string
	valList   []string
}

func NewFieldTextMedata(modelFieldID, id string) *FieldTextMetadata {
	return &FieldTextMetadata{
		FieldMetadata: FieldMetadata{
			modelID:  modelFieldID,
			id:       id,
			dataType: FieldTypeText,
		},
	}
}

func (f FieldTextMetadata) MaxLength() *int64 {
	return f.maxLength
}

func (f FieldTextMetadata) MinLength() *int64 {
	return f.minLength
}

func (f FieldTextMetadata) FixLength() *int64 {
	return f.fixLength
}

func (f FieldTextMetadata) RegExp() string {
	return f.regExp
}

func (f FieldTextMetadata) ValList() []string {
	return f.valList
}

// Validate validates a text field and returns a flag indicating whether the value is set.
func (f FieldTextMetadata) Validate(field reflect.Value) (bool, error) {
	if !field.IsValid() {
		return true, NewMessageError(
			MsgValidationCast,
			map[string]any{
				"Field": f.ModelID(),
				"Type":  "string",
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
					"Type":  "string",
				},
			)
		}
	}

	for field.Kind() == reflect.Interface {
		if field.IsNil() {
			return false, nil
		}

		field = field.Elem()

		if !field.IsValid() {
			return true, NewMessageError(
				MsgValidationCast,
				map[string]any{
					"Field": f.ModelID(),
					"Type":  "string",
				},
			)
		}
	}

	val, ok := field.Interface().(string)
	if !ok {
		return true, NewMessageError(
			MsgValidationCast,
			map[string]any{
				"Field": f.ModelID(),
				"Type":  "string",
			},
		)
	}

	return true, f.CheckValue(val)
}

// CheckValue does the actual validation of the given string value.
func (f FieldTextMetadata) CheckValue(val string) error {
	valLen := int64(len([]rune(val)))

	if f.MinLength() != nil && valLen < *f.MinLength() {
		return NewMessageError(
			MsgTextTooShort,
			map[string]any{
				"Field":  f.Descr(),
				"Min":    *f.MinLength(),
				"Length": valLen,
				"Value":  val,
			},
		)
	}

	if f.MaxLength() != nil && valLen > *f.MaxLength() {
		return NewMessageError(
			MsgTextTooLong,
			map[string]any{
				"Field":  f.Descr(),
				"Max":    *f.MaxLength(),
				"Length": valLen,
				"Value":  val,
			},
		)
	}

	if f.FixLength() != nil && valLen != *f.FixLength() {
		return NewMessageError(
			MsgTextFixedLength,
			map[string]any{
				"Field":  f.Descr(),
				"Length": valLen,
				"Fixed":  *f.FixLength(),
				"Value":  val,
			},
		)
	}

	if f.RegExp() != "" {
		match, err := regexp.MatchString(f.RegExp(), val)
		if err != nil {
			return NewMessageError(
				MsgValidation,
				map[string]any{
					"Error": err.Error(),
				},
			)
		}

		if !match {
			return NewMessageError(
				MsgTextRegExp,
				map[string]any{
					"Field":  f.Descr(),
					"RegExp": f.RegExp(),
					"Value":  val,
				},
			)
		}
	}

	if len(f.ValList()) > 0 {
		if !slices.Contains(f.ValList(), val) {
			return NewMessageError(
				MsgTextValueList,
				map[string]any{
					"Field":  f.Descr(),
					"Values": f.ValList(),
					"Value":  val,
				},
			)
		}
	}

	return nil
}
