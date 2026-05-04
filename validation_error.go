package modelbind

import "strings"

// ValidationError groups multiple field validation errors.
//
// ErrText is kept for backward compatibility with the previous implementation.
// New code should populate Errors so TranslateError(lang, err) can localize the
// individual messages.
type ValidationError struct {
	ErrText string
	Errors  []error
}

func NewValidationError(errText string) *ValidationError {
	return &ValidationError{ErrText: errText}
}

func NewValidationErrors(errs ...error) *ValidationError {
	validationErr := &ValidationError{}
	for _, err := range errs {
		validationErr.Add(err)
	}

	return validationErr
}

func (e *ValidationError) Add(err error) {
	if err == nil {
		return
	}

	e.Errors = append(e.Errors, err)
}

func (e *ValidationError) HasErrors() bool {
	return e != nil && (e.ErrText != "" || len(e.Errors) > 0)
}

func (e *ValidationError) Error() string {
	return e.Translate(LanguageEN)
}

func (e *ValidationError) Translate(lang Language) string {
	if e == nil {
		return ""
	}

	parts := make([]string, 0, len(e.Errors)+1)
	if e.ErrText != "" {
		parts = append(parts, e.ErrText)
	}

	for _, err := range e.Errors {
		if err == nil {
			continue
		}

		parts = append(parts, TranslateError(lang, err))
	}

	return strings.Join(parts, " ")
}
