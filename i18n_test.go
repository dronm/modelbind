package modelbind

import (
	"errors"
	"strings"
	"testing"

	"github.com/dronm/modelbind/metadata"
)

func TestTranslateRootMessage(t *testing.T) {
	got := Translate(
		LanguageEN,
		MsgNoInsertFields,
		map[string]any{
			"Func": "BindInsertModel",
		},
	)

	want := "BindInsertModel() failed: no fields to insert"
	if got != want {
		t.Fatalf("Translate() = %q, want %q", got, want)
	}
}

func TestTranslateRootMessageRussian(t *testing.T) {
	got := Translate(
		LanguageRU,
		MsgMetadataFieldNotFound,
		map[string]any{
			"Func":  "ParseSorterParams",
			"Field": "name",
		},
	)

	want := "ParseSorterParams() ошибка: поле name не найдено в метаданных"
	if got != want {
		t.Fatalf("Translate() = %q, want %q", got, want)
	}
}

func TestTranslateFallsBackToMetadataMessages(t *testing.T) {
	got := Translate(
		LanguageRU,
		metadata.MsgValueRequired,
		map[string]any{
			"Field": "name",
		},
	)

	want := "поле «name» обязательно для заполнения"
	if got != want {
		t.Fatalf("Translate() = %q, want %q", got, want)
	}
}

func TestMessageErrorUsesRootTranslator(t *testing.T) {
	err := NewMessageError(
		MsgNoUpdateFields,
		map[string]any{
			"Func": "BindUpdateModel",
		},
	)

	want := "BindUpdateModel() failed: no fields to update"
	if err.Error() != want {
		t.Fatalf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestTranslateErrorValidationError(t *testing.T) {
	err := NewValidationErrors(
		NewMessageError(
			MsgNoKeys,
			map[string]any{
				"Func": "BindDetailSelectModel",
			},
		),
		metadata.NewMessageError(
			metadata.MsgValueRequired,
			map[string]any{
				"Field": "name",
			},
		),
	)

	got := TranslateError(LanguageRU, err)
	if !strings.Contains(got, "BindDetailSelectModel() ошибка: ключевые поля не найдены") {
		t.Fatalf("TranslateError() = %q, missing root message", got)
	}
	if !strings.Contains(got, "поле «name» обязательно для заполнения") {
		t.Fatalf("TranslateError() = %q, missing metadata message", got)
	}
}

func TestTranslateErrorPlainError(t *testing.T) {
	err := errors.New("plain error")
	if got := TranslateError(LanguageRU, err); got != "plain error" {
		t.Fatalf("TranslateError() = %q, want plain error", got)
	}
}

func TestRootMessageTemplateCoverage(t *testing.T) {
	for id := range DefaultMessageTemplates {
		if _, ok := RussianMessageTemplates[id]; !ok {
			t.Fatalf("RussianMessageTemplates is missing %q", id)
		}
	}

	for id := range RussianMessageTemplates {
		if _, ok := DefaultMessageTemplates[id]; !ok {
			t.Fatalf("DefaultMessageTemplates is missing %q", id)
		}
	}
}
