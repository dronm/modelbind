package metadata

// MessageID is a stable identifier for a message template.
type MessageID string

const (
	// Internal/model metadata errors.
	MsgModelNotPointerOrStruct MessageID = "model.not_pointer_or_struct"
	MsgModelNotPointer         MessageID = "model.not_pointer"
	MsgModelInvalidField       MessageID = "model.invalid_field"

	// Generic validation errors.
	MsgValidation     MessageID = "validation.error"
	MsgValidationCast MessageID = "validation.cast"
	MsgFieldNotSlice  MessageID = "validation.field_not_slice"
	MsgValueRequired  MessageID = "validation.required"
	MsgValueNotInList MessageID = "validation.not_in_list"
	MsgValueTooSmall  MessageID = "validation.too_small"
	MsgValueTooBig    MessageID = "validation.too_big"

	// Text validation errors.
	MsgTextTooLong     MessageID = "validation.text.too_long"
	MsgTextTooShort    MessageID = "validation.text.too_short"
	MsgTextRegExp      MessageID = "validation.text.regexp"
	MsgTextFixedLength MessageID = "validation.text.fixed_length"
	MsgTextValueList   MessageID = "validation.text.value_list"

	// Float validation errors.
	MsgFloatPrecision MessageID = "validation.float.precision"

	// Array validation errors.
	MsgArrayInvalid     MessageID = "validation.array.invalid"
	MsgArrayTooLong     MessageID = "validation.array.too_long"
	MsgArrayTooShort    MessageID = "validation.array.too_short"
	MsgArrayFixedLength MessageID = "validation.array.fixed_length"
)

// DefaultMessageTemplates contains English fallback templates for all package
// messages.
//
// Templates intentionally use named data instead of fmt-style positional
// arguments. This is better for i18n because different languages often need
// different word order.
//
// Common template data keys:
//   - Func: function name, for internal errors
//   - Field: user-facing field name or field ID
//   - Type: expected Go/data type
//   - Error: nested error text
//   - Min, Max, Length, Precision: validation constraint values
//   - Values: allowed values
//   - RegExp: regular expression text
var DefaultMessageTemplates = map[MessageID]string{
	// Internal/model metadata errors.
	MsgModelNotPointerOrStruct: "{{ .Func }}() failed: model must be a struct or pointer to a struct",
	MsgModelNotPointer:         "{{ .Func }}() failed: model must be a pointer to a struct",
	MsgModelInvalidField:       "reflect.IsValid() failed for field {{ .Field }}",

	// Generic validation errors.
	MsgValidation:     "validation error: {{ .Error }}",
	MsgValidationCast: "value of field {{ .Field }} can not be cast to {{ .Type }}",
	MsgFieldNotSlice:  "field {{ .Field }} is not a slice",
	MsgValueRequired:  "field '{{ .Field }}', value is required",
	MsgValueNotInList: "field '{{ .Field }}', value is not in the list",
	MsgValueTooSmall:  "field '{{ .Field }}', value is too small",
	MsgValueTooBig:    "field '{{ .Field }}', value is too big",

	// Text validation errors.
	MsgTextTooLong:     "field '{{ .Field }}', value is too long",
	MsgTextTooShort:    "field '{{ .Field }}', value is too short",
	MsgTextRegExp:      "field '{{ .Field }}', value should comply with regular expression",
	MsgTextFixedLength: "field '{{ .Field }}', value length should be fixed",
	MsgTextValueList:   "field '{{ .Field }}', value should be in list of values",

	// Float validation errors.
	MsgFloatPrecision: "field '{{ .Field }}', float precision is exceeded maximum value",

	// Array validation errors.
	MsgArrayInvalid:     "field '{{ .Field }}', array format is invalid",
	MsgArrayTooLong:     "field '{{ .Field }}', array count exceeded maximum value",
	MsgArrayTooShort:    "field '{{ .Field }}', array count less than minimal value",
	MsgArrayFixedLength: "field '{{ .Field }}', array count should be of fixed length",
}
