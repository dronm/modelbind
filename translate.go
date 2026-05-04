package modelbind

import (
	"bytes"
	"errors"
	"text/template"

	"github.com/dronm/modelbind/metadata"
)

// Language is shared with the metadata package.
type Language = metadata.Language

const (
	LanguageEN Language = metadata.LanguageEN
	LanguageRU Language = metadata.LanguageRU
)

// RussianMessageTemplates contains Russian templates for all root package
// messages.
var RussianMessageTemplates = map[MessageID]string{
	MsgMetadataFieldNotFound:  "{{ .Func }}() ошибка: поле {{ .Field }} не найдено в метаданных",
	MsgInvalidFieldExpression: "{{ .Func }}() ошибка: недопустимое выражение поля {{ .Expression }}",
	MsgModelFieldNotInterface: "reflect.CanInterface() вернул false для поля {{ .Field }}",

	MsgNoInsertFields: "{{ .Func }}() ошибка: нет полей для вставки",
	MsgNoUpdateFields: "{{ .Func }}() ошибка: нет полей для обновления",
	MsgNoKeys:         "{{ .Func }}() ошибка: ключевые поля не найдены",

	MsgAggregationFieldNotDefined:    "{{ .Func }}() ошибка: поле агрегации не задано для индекса {{ .Index }}",
	MsgAggregationFunctionNotDefined: "{{ .Func }}() ошибка: функция агрегации не задана для поля {{ .Field }}",

	MsgFilterNotInitialized: "{{ .Func }}() ошибка: фильтр должен быть инициализирован",
	MsgSorterNotInitialized: "{{ .Func }}() ошибка: сортировка должна быть инициализирована",
	MsgLimitNotInitialized:  "{{ .Func }}() ошибка: лимит должен быть инициализирован",

	MsgJSONMarshalFailed: "{{ .Func }}() ошибка: поле {{ .Field }} нельзя преобразовать в JSON: {{ .Error }}",
}

// MessageTemplates contains all built-in root package templates by language.
var MessageTemplates = map[Language]map[MessageID]string{
	LanguageEN: DefaultMessageTemplates,
	LanguageRU: RussianMessageTemplates,
}

// NormalizeLanguage converts a language value to a supported base language.
func NormalizeLanguage(lang Language) Language {
	return metadata.NormalizeLanguage(lang)
}

// TemplateFor returns the template for messageID in lang.
//
// Root package templates are checked first. If messageID belongs to the
// metadata subpackage, this function delegates to metadata.TemplateFor(...).
func TemplateFor(lang Language, messageID MessageID) (string, bool) {
	lang = NormalizeLanguage(lang)

	if templates, ok := MessageTemplates[lang]; ok {
		if text, ok := templates[messageID]; ok {
			return text, true
		}
	}

	if text, ok := DefaultMessageTemplates[messageID]; ok {
		return text, true
	}

	return metadata.TemplateFor(lang, messageID)
}

// Translate renders messageID using language-specific template data.
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

// TranslateError translates quvalid and metadata errors that expose a message
// ID and named template data.
//
// For ordinary errors it returns err.Error(). For nil errors it returns an
// empty string.
func TranslateError(lang Language, err error) string {
	if err == nil {
		return ""
	}

	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return validationErr.Translate(lang)
	}

	var translatable metadata.TranslatableError
	if errors.As(err, &translatable) {
		return Translate(lang, translatable.MessageID(), translatable.MessageData())
	}

	return err.Error()
}
