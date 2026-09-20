// Package access defines resolved, database-independent row policies and scalar
// write rules. Authentication, role lookup and policy registration belong to the
// application, not to this package.
package access

import (
	"errors"
	"fmt"
)

var (
	ErrDenied           = errors.New("access denied")
	ErrInvalidPolicy    = errors.New("invalid access policy")
	ErrInvalidPredicate = errors.New("invalid access predicate")
	ErrInvalidRule      = errors.New("invalid write rule")
	ErrInvalidField     = errors.New("invalid access field")
	ErrUnsupportedValue = errors.New("unsupported access value")
	ErrWriteConflict    = errors.New("write violates access policy")
)

// FieldError identifies the logical field and reason, without disclosing policy
// values (such as another customer's ID). Kind supports errors.Is; Code is a
// stable, untranslated reason that applications can translate for their UI.
type FieldError struct {
	Field string
	Code  string
	Kind  error
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("%v: field %q (%s)", e.Kind, e.Field, e.Code)
}

func (e *FieldError) Unwrap() error { return e.Kind }
