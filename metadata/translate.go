package metadata

import (
	"bytes"
	"errors"
	"strings"
	"text/template"
)

// Language is a BCP-47-like language code used by this package.
//
// Currently supported values:
//   - "en"
//   - "ru"
//
// Region-specific values such as "en-US" or "ru-RU" are normalized to their
// base language.
type Language string

const (
	LanguageEN Language = "en"
	LanguageRU Language = "ru"
)

// RussianMessageTemplates contains Russian templates for all package messages.
//
// Templates use named data keys instead of fmt-style positional arguments.
// This lets each language use its own word order.
//
// Common template data keys:
//   - Func: function name, for internal errors
//   - Field: user-facing field name or field ID
//   - Type: expected Go/data type
//   - Error: nested error text
//   - Min, Max, Length, Precision: validation constraint values
//   - Values: allowed values
//   - RegExp: regular expression text
var RussianMessageTemplates = map[MessageID]string{
	// Internal/model metadata errors.
	MsgModelNotPointerOrStruct: "{{ .Func }}() ошибка: модель должна быть структурой или указателем на структуру",
	MsgModelNotPointer:         "{{ .Func }}() ошибка: модель должна быть указателем на структуру",
	MsgModelInvalidField:       "reflect.IsValid() вернул false для поля {{ .Field }}",

	// Generic validation errors.
	MsgValidation:     "ошибка валидации: {{ .Error }}",
	MsgValidationCast: "значение поля «{{ .Field }}» нельзя привести к типу {{ .Type }}",
	MsgFieldNotSlice:  "поле «{{ .Field }}» не является срезом",
	MsgValueRequired:  "поле «{{ .Field }}» обязательно для заполнения",
	MsgValueNotInList: "значение поля «{{ .Field }}» отсутствует в списке допустимых значений",
	MsgValueTooSmall:  "значение поля «{{ .Field }}» слишком маленькое",
	MsgValueTooBig:    "значение поля «{{ .Field }}» слишком большое",

	// Text validation errors.
	MsgTextTooLong:     "значение поля «{{ .Field }}» слишком длинное",
	MsgTextTooShort:    "значение поля «{{ .Field }}» слишком короткое",
	MsgTextRegExp:      "значение поля «{{ .Field }}» не соответствует регулярному выражению",
	MsgTextFixedLength: "длина значения поля «{{ .Field }}» должна быть фиксированной",
	MsgTextValueList:   "значение поля «{{ .Field }}» должно быть одним из допустимых значений",

	// Float validation errors.
	MsgFloatPrecision: "превышена допустимая точность дробного значения поля «{{ .Field }}»",

	// Array validation errors.
	MsgArrayInvalid:     "поле «{{ .Field }}» имеет неверный формат массива",
	MsgArrayTooLong:     "количество элементов массива в поле «{{ .Field }}» превышает максимум",
	MsgArrayTooShort:    "количество элементов массива в поле «{{ .Field }}» меньше минимума",
	MsgArrayFixedLength: "количество элементов массива в поле «{{ .Field }}» должно быть фиксированным",
}

// MessageTemplates contains all built-in message templates by language.
//
// English templates are defined in DefaultMessageTemplates in messages.go.
var MessageTemplates = map[Language]map[MessageID]string{
	LanguageEN: DefaultMessageTemplates,
	LanguageRU: RussianMessageTemplates,
}

// TranslatableError is an optional interface for errors that can expose a
// message ID and named template data.
//
// You can implement it on your structured validation error type and then use
// TranslateError() at the API boundary.
type TranslatableError interface {
	error
	MessageID() MessageID
	MessageData() map[string]any
}

// NormalizeLanguage converts a language value to a supported base language.
//
// Examples:
//   - "en-US" -> "en"
//   - "ru-RU" -> "ru"
//   - ""      -> "en"
func NormalizeLanguage(lang Language) Language {
	value := strings.ToLower(strings.TrimSpace(string(lang)))
	if value == "" {
		return LanguageEN
	}

	if index := strings.IndexAny(value, "-_"); index >= 0 {
		value = value[:index]
	}

	switch Language(value) {
	case LanguageEN, LanguageRU:
		return Language(value)
	default:
		return LanguageEN
	}
}

// TemplateFor returns the template for messageID in lang.
//
// If the language is unknown or the requested message is missing in that
// language, it falls back to English.
func TemplateFor(lang Language, messageID MessageID) (string, bool) {
	lang = NormalizeLanguage(lang)

	if templates, ok := MessageTemplates[lang]; ok {
		if text, ok := templates[messageID]; ok {
			return text, true
		}
	}

	text, ok := DefaultMessageTemplates[messageID]
	return text, ok
}

// Translate renders messageID using language-specific template data.
//
// Unknown language falls back to English. Unknown message ID returns the raw
// message ID string.
func Translate(lang Language, messageID MessageID, data map[string]any) string {
	messageTemplate, ok := TemplateFor(lang, messageID)
	if !ok {
		return string(messageID)
	}

	if data == nil {
		data = make(map[string]any)
	}

	tmpl, err := template.New(string(messageID)).Option("missingkey=zero").Parse(messageTemplate)
	if err != nil {
		return messageTemplate
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return messageTemplate
	}

	return buf.String()
}

// TranslateError translates errors that implement TranslatableError.
//
// For ordinary errors it returns err.Error(). For nil errors it returns an
// empty string.
func TranslateError(lang Language, err error) string {
	if err == nil {
		return ""
	}

	var translatable TranslatableError
	if errors.As(err, &translatable) {
		return Translate(lang, translatable.MessageID(), translatable.MessageData())
	}

	return err.Error()
}
