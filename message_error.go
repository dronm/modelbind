package modelbind

import "github.com/dronm/modelbind/metadata"

// MessageError is a root-package wrapper around metadata.MessageError.
//
// It reuses the metadata package's structured error shape while rendering
// Error() through the root package translator, so quvalid-specific message IDs
// are readable even before explicit TranslateError(...) calls.
type MessageError struct {
	*metadata.MessageError
}

func NewMessageError(id MessageID, data map[string]any) error {
	if data == nil {
		data = make(map[string]any)
	}

	return &MessageError{
		MessageError: &metadata.MessageError{
			ID:   id,
			Data: data,
		},
	}
}

func (e *MessageError) Error() string {
	if e == nil || e.MessageError == nil {
		return ""
	}

	return Translate(LanguageEN, e.ID, e.Data)
}
